package cortex

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/owulveryck/agenthub/agents/cortex/llm"
	"github.com/owulveryck/agenthub/agents/cortex/state"
	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	CortexAgentID = "cortex"
)

// MessagePublisher is an interface for publishing messages to the event bus.
// This abstraction allows for easier testing.
type MessagePublisher interface {
	PublishMessage(ctx context.Context, msg *pb.Message, routing *pb.AgentEventMetadata) error
}

// Cortex is the core orchestrator that manages conversations and tasks.
// It is the "brain" of the system, deciding what actions to take based on
// incoming messages and events.
type Cortex struct {
	stateManager     state.StateManager
	llmClient        llm.Client
	messagePublisher MessagePublisher
	logger           *slog.Logger
	registeredAgents map[string]*pb.AgentCard
	agentsMu         sync.RWMutex
}

// NewCortex creates a new Cortex instance.
func NewCortex(
	stateManager state.StateManager,
	llmClient llm.Client,
	messagePublisher MessagePublisher,
	logger *slog.Logger,
) *Cortex {
	return &Cortex{
		stateManager:     stateManager,
		llmClient:        llmClient,
		messagePublisher: messagePublisher,
		logger:           logger,
		registeredAgents: make(map[string]*pb.AgentCard),
	}
}

// RegisterAgent registers an agent's capabilities with Cortex.
// This is called when an AgentCard is received.
func (c *Cortex) RegisterAgent(agentID string, card *pb.AgentCard) {
	c.agentsMu.Lock()
	defer c.agentsMu.Unlock()

	c.registeredAgents[agentID] = card
}

// GetAvailableAgents returns a list of all registered agents.
func (c *Cortex) GetAvailableAgents() []*pb.AgentCard {
	c.agentsMu.RLock()
	defer c.agentsMu.RUnlock()

	agents := make([]*pb.AgentCard, 0, len(c.registeredAgents))
	for _, card := range c.registeredAgents {
		agents = append(agents, card)
	}

	return agents
}

// HandleMessage is the main entry point for processing messages.
// It handles:
// - Chat requests from users
// - Task results from agents
// - Agent card registrations
func (c *Cortex) HandleMessage(ctx context.Context, traceManager *observability.TraceManager, msg *pb.Message) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	sessionID := msg.ContextId
	if sessionID == "" {
		return fmt.Errorf("message must have a context_id (session ID)")
	}

	// Use WithLock to ensure thread-safe state updates
	return c.stateManager.WithLock(sessionID, func(conversationState *state.ConversationState) error {
		// Add the incoming message to conversation history
		conversationState.Messages = append(conversationState.Messages, msg)

		// Check if this is a task result
		if msg.TaskId != "" && msg.Role == pb.Role_ROLE_AGENT {
			return c.handleTaskResult(ctx, traceManager, conversationState, msg)
		}

		// Otherwise, it's a new chat request
		return c.handleChatRequest(ctx, traceManager, conversationState, msg)
	})
}

// handleChatRequest processes a new chat request from a user.
func (c *Cortex) handleChatRequest(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, msg *pb.Message) error {
	// Start tracing for chat request processing
	reqCtx, reqSpan := traceManager.StartSpan(ctx, "cortex.chat_request",
		attribute.String("session_id", conversationState.SessionID),
		attribute.String("message_id", msg.GetMessageId()),
		attribute.Int("message_history_count", len(conversationState.Messages)),
	)
	defer reqSpan.End()

	traceManager.AddComponentAttribute(reqSpan, "cortex_orchestrator")

	// Get available agents
	availableAgents := c.GetAvailableAgents()
	traceManager.AddSpanEvent(reqSpan, "available_agents_retrieved",
		attribute.Int("agent_count", len(availableAgents)),
	)

	// Call LLM to decide what to do
	llmCtx, llmSpan := traceManager.StartSpan(reqCtx, "cortex.llm_decide",
		attribute.String("message_id", msg.GetMessageId()),
		attribute.Int("available_agents", len(availableAgents)),
		attribute.Int("conversation_history_length", len(conversationState.Messages)),
	)

	// Log LLM input details
	traceManager.AddSpanEvent(llmSpan, "llm_input_prepared",
		attribute.Int("history_messages", len(conversationState.Messages)),
		attribute.Int("available_agents", len(availableAgents)),
		attribute.String("new_message_role", msg.GetRole().String()),
	)

	// Add available agent names to trace
	agentNames := make([]string, 0, len(availableAgents))
	for _, agent := range availableAgents {
		agentNames = append(agentNames, agent.GetName())
	}
	if len(agentNames) > 0 {
		traceManager.AddSpanEvent(llmSpan, "available_agents_list",
			attribute.StringSlice("agent_names", agentNames),
			attribute.Int("count", len(agentNames)),
		)
	}

	decision, err := c.llmClient.Decide(llmCtx, conversationState.Messages, availableAgents, msg)
	if err != nil {
		traceManager.RecordError(llmSpan, err)
		traceManager.RecordError(reqSpan, err)
		traceManager.AddSpanEvent(llmSpan, "llm_decision_failed",
			attribute.String("error", err.Error()),
		)
		llmSpan.End()
		return fmt.Errorf("LLM decision failed: %w", err)
	}

	// Log detailed LLM decision output
	traceManager.SetSpanSuccess(llmSpan)
	traceManager.AddSpanEvent(llmSpan, "llm_decision_made",
		attribute.Int("action_count", len(decision.Actions)),
		attribute.String("reasoning", decision.Reasoning),
	)

	// Log each action type decided by LLM
	for i, action := range decision.Actions {
		attrs := []attribute.KeyValue{
			attribute.Int("action_index", i),
			attribute.String("action_type", action.Type),
		}

		if action.Type == "chat.response" {
			attrs = append(attrs,
				attribute.Int("response_length", len(action.ResponseText)),
				attribute.String("response_preview", truncateString(action.ResponseText, 100)),
			)
		} else if action.Type == "task.request" {
			attrs = append(attrs,
				attribute.String("task_type", action.TaskType),
				attribute.String("target_agent", action.TargetAgent),
			)
		}

		traceManager.AddSpanEvent(llmSpan, "llm_action_decided", attrs...)
	}

	llmSpan.End()

	// Execute the decided actions
	err = c.executeActions(reqCtx, traceManager, conversationState, decision.Actions, msg)
	if err != nil {
		traceManager.RecordError(reqSpan, err)
		return err
	}

	traceManager.SetSpanSuccess(reqSpan)
	return nil
}

// handleTaskResult processes a task result from an agent.
func (c *Cortex) handleTaskResult(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, msg *pb.Message) error {
	// Start tracing for task result processing
	resCtx, resSpan := traceManager.StartSpan(ctx, "cortex.task_result",
		attribute.String("session_id", conversationState.SessionID),
		attribute.String("task_id", msg.GetTaskId()),
		attribute.String("message_id", msg.GetMessageId()),
	)
	defer resSpan.End()

	traceManager.AddComponentAttribute(resSpan, "cortex_orchestrator")

	// Remove the task from pending tasks
	delete(conversationState.PendingTasks, msg.TaskId)
	traceManager.AddSpanEvent(resSpan, "task_completed",
		attribute.String("task_id", msg.GetTaskId()),
		attribute.Int("remaining_tasks", len(conversationState.PendingTasks)),
	)

	// Get available agents
	availableAgents := c.GetAvailableAgents()

	// Call LLM to decide how to synthesize this result
	llmCtx, llmSpan := traceManager.StartSpan(resCtx, "cortex.llm_synthesize",
		attribute.String("task_id", msg.GetTaskId()),
		attribute.Int("available_agents", len(availableAgents)),
		attribute.Int("conversation_history_length", len(conversationState.Messages)),
		attribute.Int("remaining_pending_tasks", len(conversationState.PendingTasks)),
	)

	// Log LLM synthesis input details
	traceManager.AddSpanEvent(llmSpan, "llm_synthesis_input_prepared",
		attribute.String("task_id", msg.GetTaskId()),
		attribute.Int("history_messages", len(conversationState.Messages)),
		attribute.Int("remaining_tasks", len(conversationState.PendingTasks)),
		attribute.String("result_role", msg.GetRole().String()),
	)

	// Add task result preview to trace
	if len(msg.GetContent()) > 0 {
		resultText := msg.GetContent()[0].GetText()
		traceManager.AddSpanEvent(llmSpan, "task_result_content",
			attribute.String("task_id", msg.GetTaskId()),
			attribute.Int("content_length", len(resultText)),
			attribute.String("content_preview", truncateString(resultText, 100)),
		)
	}

	decision, err := c.llmClient.Decide(llmCtx, conversationState.Messages, availableAgents, msg)
	if err != nil {
		traceManager.RecordError(llmSpan, err)
		traceManager.RecordError(resSpan, err)
		traceManager.AddSpanEvent(llmSpan, "llm_synthesis_failed",
			attribute.String("task_id", msg.GetTaskId()),
			attribute.String("error", err.Error()),
		)
		llmSpan.End()
		return fmt.Errorf("LLM decision failed: %w", err)
	}

	// Log detailed LLM synthesis output
	traceManager.SetSpanSuccess(llmSpan)
	traceManager.AddSpanEvent(llmSpan, "llm_synthesis_complete",
		attribute.Int("action_count", len(decision.Actions)),
		attribute.String("reasoning", decision.Reasoning),
	)

	// Log each action from synthesis
	for i, action := range decision.Actions {
		attrs := []attribute.KeyValue{
			attribute.Int("action_index", i),
			attribute.String("action_type", action.Type),
		}

		if action.Type == "chat.response" {
			attrs = append(attrs,
				attribute.Int("response_length", len(action.ResponseText)),
				attribute.String("response_preview", truncateString(action.ResponseText, 100)),
			)
		} else if action.Type == "task.request" {
			attrs = append(attrs,
				attribute.String("task_type", action.TaskType),
				attribute.String("target_agent", action.TargetAgent),
			)
		}

		traceManager.AddSpanEvent(llmSpan, "llm_synthesis_action", attrs...)
	}

	llmSpan.End()

	// Execute the decided actions
	err = c.executeActions(resCtx, traceManager, conversationState, decision.Actions, msg)
	if err != nil {
		traceManager.RecordError(resSpan, err)
		return err
	}

	traceManager.SetSpanSuccess(resSpan)
	return nil
}

// executeActions executes the actions decided by the LLM.
func (c *Cortex) executeActions(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, actions []llm.Action, triggeringMsg *pb.Message) error {
	actCtx, actSpan := traceManager.StartSpan(ctx, "cortex.execute_actions",
		attribute.Int("action_count", len(actions)),
		attribute.String("session_id", conversationState.SessionID),
		attribute.String("triggering_message_id", triggeringMsg.GetMessageId()),
		attribute.Int("pending_tasks_count", len(conversationState.PendingTasks)),
	)
	defer actSpan.End()

	traceManager.AddComponentAttribute(actSpan, "cortex_orchestrator")

	// Log execution plan
	traceManager.AddSpanEvent(actSpan, "execution_plan_started",
		attribute.Int("total_actions", len(actions)),
		attribute.Int("current_pending_tasks", len(conversationState.PendingTasks)),
	)

	for i, action := range actions {
		// Detailed logging for each action before execution
		actionAttrs := []attribute.KeyValue{
			attribute.Int("action_index", i),
			attribute.String("action_type", action.Type),
		}

		if action.Type == "chat.response" {
			actionAttrs = append(actionAttrs,
				attribute.Int("response_length", len(action.ResponseText)),
				attribute.String("response_preview", truncateString(action.ResponseText, 100)),
			)
		} else if action.Type == "task.request" {
			actionAttrs = append(actionAttrs,
				attribute.String("task_type", action.TaskType),
				attribute.String("target_agent", action.TargetAgent),
			)
		}

		traceManager.AddSpanEvent(actSpan, "executing_action", actionAttrs...)

		// Execute the action
		switch action.Type {
		case "chat.response":
			if err := c.executeChatResponse(actCtx, traceManager, conversationState, action, triggeringMsg); err != nil {
				traceManager.RecordError(actSpan, err)
				traceManager.AddSpanEvent(actSpan, "action_execution_failed",
					attribute.Int("action_index", i),
					attribute.String("action_type", action.Type),
					attribute.String("error", err.Error()),
				)
				return fmt.Errorf("failed to execute chat response: %w", err)
			}
			traceManager.AddSpanEvent(actSpan, "action_executed_successfully",
				attribute.Int("action_index", i),
				attribute.String("action_type", "chat.response"),
			)

		case "task.request":
			if err := c.executeTaskRequest(actCtx, traceManager, conversationState, action, triggeringMsg); err != nil {
				traceManager.RecordError(actSpan, err)
				traceManager.AddSpanEvent(actSpan, "action_execution_failed",
					attribute.Int("action_index", i),
					attribute.String("action_type", action.Type),
					attribute.String("task_type", action.TaskType),
					attribute.String("target_agent", action.TargetAgent),
					attribute.String("error", err.Error()),
				)
				return fmt.Errorf("failed to execute task request: %w", err)
			}
			traceManager.AddSpanEvent(actSpan, "action_executed_successfully",
				attribute.Int("action_index", i),
				attribute.String("action_type", "task.request"),
				attribute.String("task_type", action.TaskType),
				attribute.String("target_agent", action.TargetAgent),
			)

		default:
			err := fmt.Errorf("unknown action type: %s", action.Type)
			traceManager.RecordError(actSpan, err)
			traceManager.AddSpanEvent(actSpan, "unknown_action_type",
				attribute.Int("action_index", i),
				attribute.String("action_type", action.Type),
			)
			return err
		}
	}

	// Log execution completion
	traceManager.AddSpanEvent(actSpan, "execution_plan_completed",
		attribute.Int("actions_executed", len(actions)),
		attribute.Int("final_pending_tasks", len(conversationState.PendingTasks)),
	)

	traceManager.SetSpanSuccess(actSpan)
	return nil
}

// executeChatResponse sends a chat response to the user.
func (c *Cortex) executeChatResponse(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, action llm.Action, triggeringMsg *pb.Message) error {
	// Start tracing for chat response execution
	respCtx, respSpan := traceManager.StartSpan(ctx, "cortex.send_chat_response",
		attribute.String("session_id", conversationState.SessionID),
		attribute.String("response_length", fmt.Sprintf("%d", len(action.ResponseText))),
	)
	defer respSpan.End()

	traceManager.AddComponentAttribute(respSpan, "cortex_orchestrator")

	// Create response message
	responseMsg := &pb.Message{
		MessageId: fmt.Sprintf("cortex_response_%d", time.Now().UnixNano()),
		ContextId: conversationState.SessionID,
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: action.ResponseText}},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":           structpb.NewStringValue("chat_response"),
				"from_agent":          structpb.NewStringValue(CortexAgentID),
				"original_message_id": structpb.NewStringValue(triggeringMsg.MessageId),
			},
		},
	}

	traceManager.AddSpanEvent(respSpan, "chat_response_created",
		attribute.String("message_id", responseMsg.MessageId),
		attribute.Int("response_length", len(action.ResponseText)),
	)

	// Add to conversation history
	conversationState.Messages = append(conversationState.Messages, responseMsg)

	// Publish the message (trace context automatically propagated via respCtx)
	routing := &pb.AgentEventMetadata{
		FromAgentId: CortexAgentID,
		EventType:   "a2a.message.chat_response",
		Priority:    pb.Priority_PRIORITY_MEDIUM,
	}

	err := c.messagePublisher.PublishMessage(respCtx, responseMsg, routing)
	if err != nil {
		traceManager.RecordError(respSpan, err)
		return err
	}

	traceManager.SetSpanSuccess(respSpan)
	traceManager.AddSpanEvent(respSpan, "chat_response_published",
		attribute.String("message_id", responseMsg.MessageId),
	)

	return nil
}

// executeTaskRequest dispatches a task request to an agent.
func (c *Cortex) executeTaskRequest(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, action llm.Action, triggeringMsg *pb.Message) error {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	// Start tracing for task request execution
	taskCtx, taskSpan := traceManager.StartSpan(ctx, "cortex.dispatch_task",
		attribute.String("session_id", conversationState.SessionID),
		attribute.String("task_id", taskID),
		attribute.String("task_type", action.TaskType),
		attribute.String("target_agent", action.TargetAgent),
	)
	defer taskSpan.End()

	traceManager.AddComponentAttribute(taskSpan, "cortex_orchestrator")

	// Create task request message
	taskMsg := &pb.Message{
		MessageId: fmt.Sprintf("task_request_%d", time.Now().UnixNano()),
		ContextId: conversationState.SessionID,
		TaskId:    taskID,
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: fmt.Sprintf("Task: %s", action.TaskType)}},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":           structpb.NewStringValue(action.TaskType),
				"from_agent":          structpb.NewStringValue(CortexAgentID),
				"original_message_id": structpb.NewStringValue(triggeringMsg.MessageId),
			},
		},
	}

	traceManager.AddSpanEvent(taskSpan, "task_request_created",
		attribute.String("task_id", taskID),
		attribute.String("message_id", taskMsg.MessageId),
		attribute.String("target_agent", action.TargetAgent),
	)

	// Track this task as pending
	conversationState.PendingTasks[taskID] = &state.TaskContext{
		TaskID:        taskID,
		TaskType:      action.TaskType,
		RequestedAt:   time.Now().Unix(),
		OriginalInput: triggeringMsg,
		UserNotified:  true, // We assume we've already sent an acknowledgment
	}

	traceManager.AddSpanEvent(taskSpan, "task_tracked_as_pending",
		attribute.Int("total_pending_tasks", len(conversationState.PendingTasks)),
	)

	// Publish the task request (trace context automatically propagated via taskCtx)
	routing := &pb.AgentEventMetadata{
		FromAgentId: CortexAgentID,
		ToAgentId:   action.TargetAgent,
		EventType:   fmt.Sprintf("a2a.task.%s", action.TaskType),
		Priority:    pb.Priority_PRIORITY_MEDIUM,
	}

	err := c.messagePublisher.PublishMessage(taskCtx, taskMsg, routing)
	if err != nil {
		traceManager.RecordError(taskSpan, err)
		return err
	}

	traceManager.SetSpanSuccess(taskSpan)
	traceManager.AddSpanEvent(taskSpan, "task_request_published",
		attribute.String("task_id", taskID),
		attribute.String("message_id", taskMsg.MessageId),
	)

	return nil
}

// HandleTaskCompletion processes task completion notifications from delegated agents
func (c *Cortex) HandleTaskCompletion(ctx context.Context, taskID, contextID string, status *pb.TaskStatus) {
	// Use WithLock to ensure thread-safe state access
	_ = c.stateManager.WithLock(contextID, func(conversationState *state.ConversationState) error {
		// Check if this task is pending
		taskContext, pending := conversationState.PendingTasks[taskID]
		if !pending {
			// Task not found or already processed
			return nil
		}

		// Store the task result and update completion time
		taskContext.CompletedAt = time.Now().Unix()
		taskContext.Result = status

		// Note: We don't delete from PendingTasks yet - keep it for potential
		// use in responding to the user with the task results

		return nil
	})
}

// HandleTaskArtifact processes task artifact notifications from delegated agents
func (c *Cortex) HandleTaskArtifact(ctx context.Context, taskID, contextID string, artifact *pb.Artifact) {
	c.logger.DebugContext(ctx, "HandleTaskArtifact called",
		"task_id", taskID,
		"context_id", contextID,
		"artifact_id", artifact.GetArtifactId())

	var shouldRespond bool
	var responseText string

	// Use WithLock to ensure thread-safe state access
	_ = c.stateManager.WithLock(contextID, func(conversationState *state.ConversationState) error {
		// Check if this task is pending
		taskContext, pending := conversationState.PendingTasks[taskID]
		if !pending {
			c.logger.DebugContext(ctx, "Task not found in pending tasks", "task_id", taskID)
			// Task not found or already processed
			return nil
		}

		// Store the artifact with the task context
		if taskContext.Artifacts == nil {
			taskContext.Artifacts = make([]*pb.Artifact, 0)
		}
		taskContext.Artifacts = append(taskContext.Artifacts, artifact)

		// Extract text content from artifact for response
		var textParts []string
		for _, part := range artifact.GetParts() {
			if textPart := part.GetText(); textPart != "" {
				textParts = append(textParts, textPart)
			}
		}

		c.logger.DebugContext(ctx, "Extracted text parts from artifact",
			"part_count", len(textParts))

		// By default, send artifact results back to the user
		if len(textParts) > 0 {
			shouldRespond = true
			if artifact.GetName() != "" && artifact.GetDescription() != "" {
				responseText = fmt.Sprintf("%s: %s", artifact.GetName(), strings.Join(textParts, "\n"))
			} else {
				responseText = strings.Join(textParts, "\n")
			}
			c.logger.DebugContext(ctx, "Will send response to user", "response_text", responseText)
		}

		return nil
	})

	// Send response to user if we have content
	if shouldRespond && responseText != "" {
		c.logger.DebugContext(ctx, "Calling sendTaskResultToUser", "context_id", contextID)
		c.sendTaskResultToUser(ctx, contextID, taskID, responseText)
	} else {
		c.logger.DebugContext(ctx, "Not sending response",
			"should_respond", shouldRespond,
			"has_text", responseText != "")
	}
}

// sendTaskResultToUser sends task results back to the user
func (c *Cortex) sendTaskResultToUser(ctx context.Context, contextID, taskID, resultText string) {
	messageID := fmt.Sprintf("cortex_task_result_%d", time.Now().UnixNano())

	c.logger.DebugContext(ctx, "sendTaskResultToUser called",
		"message_id", messageID,
		"context_id", contextID,
		"task_id", taskID,
		"response_text", resultText)

	responseMsg := &pb.Message{
		MessageId: messageID,
		ContextId: contextID,
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: resultText}},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":  structpb.NewStringValue("task_result"),
				"from_agent": structpb.NewStringValue(CortexAgentID),
				"task_id":    structpb.NewStringValue(taskID),
			},
		},
	}

	// Update conversation state with the response
	_ = c.stateManager.WithLock(contextID, func(conversationState *state.ConversationState) error {
		conversationState.Messages = append(conversationState.Messages, responseMsg)
		c.logger.DebugContext(ctx, "Added response to conversation history",
			"total_messages", len(conversationState.Messages))
		return nil
	})

	// Publish the response - broadcast to all message subscribers (including REPL)
	routing := &pb.AgentEventMetadata{
		FromAgentId: CortexAgentID,
		// No ToAgentId - broadcast to all
		EventType: "a2a.message.task_result",
		Priority:  pb.Priority_PRIORITY_MEDIUM,
	}

	c.logger.DebugContext(ctx, "Publishing message",
		"from_agent", routing.FromAgentId,
		"event_type", routing.EventType)

	err := c.messagePublisher.PublishMessage(ctx, responseMsg, routing)
	if err != nil {
		c.logger.ErrorContext(ctx, "Failed to publish task result to user",
			"error", err,
			"message_id", messageID,
			"task_id", taskID)
	} else {
		c.logger.InfoContext(ctx, "Successfully published task result message",
			"message_id", messageID,
			"context_id", contextID,
			"task_id", taskID)
	}
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
