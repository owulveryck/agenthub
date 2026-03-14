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

// UnregisterAgent removes an agent from Cortex's registry.
// This is called when an agent is stopped or unregistered from the broker.
func (c *Cortex) UnregisterAgent(agentID string) {
	c.agentsMu.Lock()
	defer c.agentsMu.Unlock()
	delete(c.registeredAgents, agentID)
}

// GetAvailableAgents returns a map of all registered agents keyed by agent ID.
func (c *Cortex) GetAvailableAgents() map[string]*pb.AgentCard {
	c.agentsMu.RLock()
	defer c.agentsMu.RUnlock()

	agents := make(map[string]*pb.AgentCard, len(c.registeredAgents))
	for id, card := range c.registeredAgents {
		agents[id] = card
	}

	return agents
}

// GetCurrentPrompt returns the system prompt that Cortex would send to the LLM
// with the currently registered agents. Useful for introspection/debugging.
func (c *Cortex) GetCurrentPrompt() string {
	return c.llmClient.BuildPrompt(c.GetAvailableAgents())
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
	for agentID, agent := range availableAgents {
		agentNames = append(agentNames, fmt.Sprintf("%s(%s)", agent.GetName(), agentID))
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

		case "status.request":
			if err := c.executeStatusRequest(actCtx, traceManager, conversationState, action); err != nil {
				traceManager.RecordError(actSpan, err)
				traceManager.AddSpanEvent(actSpan, "action_execution_failed",
					attribute.Int("action_index", i),
					attribute.String("action_type", action.Type),
					attribute.String("target_agent", action.TargetAgent),
					attribute.String("error", err.Error()),
				)
				return fmt.Errorf("failed to execute status request: %w", err)
			}
			traceManager.AddSpanEvent(actSpan, "action_executed_successfully",
				attribute.Int("action_index", i),
				attribute.String("action_type", "status.request"),
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

	// Use the original user message content so the target agent
	// gets the full text (e.g. file paths, instructions).
	content := triggeringMsg.GetContent()
	if len(content) == 0 {
		content = []*pb.Part{
			{Part: &pb.Part_Text{Text: fmt.Sprintf("Task: %s", action.TaskType)}},
		}
	}

	if len(content) > 0 {
		c.logger.InfoContext(ctx, "Dispatching task with content",
			"target_agent", action.TargetAgent,
			"task_type", action.TaskType,
			"content_preview", content[0].GetText(),
		)
	}

	// Create task request message
	taskMsg := &pb.Message{
		MessageId: fmt.Sprintf("task_request_%d", time.Now().UnixNano()),
		ContextId: conversationState.SessionID,
		TaskId:    taskID,
		Role:      pb.Role_ROLE_AGENT,
		Content:   content,
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

// HandleTaskCompletion processes task completion notifications from delegated agents.
// For FAILED or CANCELLED tasks, it routes the error through the LLM pipeline
// so the user receives a meaningful response instead of silence.
func (c *Cortex) HandleTaskCompletion(ctx context.Context, traceManager *observability.TraceManager, taskID, contextID string, status *pb.TaskStatus) {
	var shouldRoute bool
	var errorText string

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

		// For failed/cancelled tasks, extract error info and route through LLM
		// so the user gets a response. Completed tasks are handled by HandleTaskArtifact.
		if status.GetState() == pb.TaskState_TASK_STATE_FAILED || status.GetState() == pb.TaskState_TASK_STATE_CANCELLED {
			shouldRoute = true
			// Extract error text from the status update message
			if updateMsg := status.GetUpdate(); updateMsg != nil {
				for _, part := range updateMsg.GetContent() {
					if text := part.GetText(); text != "" {
						errorText = text
						break
					}
				}
			}
			if errorText == "" {
				errorText = fmt.Sprintf("Task %s with status: %s", taskID, status.GetState().String())
			}
		}

		return nil
	})

	// Route failed/cancelled tasks through LLM so the user gets a response
	if shouldRoute {
		failureMsg := &pb.Message{
			MessageId: fmt.Sprintf("task_failure_%d", time.Now().UnixNano()),
			ContextId: contextID,
			TaskId:    taskID,
			Role:      pb.Role_ROLE_AGENT,
			Content:   []*pb.Part{{Part: &pb.Part_Text{Text: errorText}}},
		}
		c.logger.DebugContext(ctx, "Routing task failure through HandleMessage",
			"task_id", taskID,
			"context_id", contextID,
			"error_text", errorText)
		if err := c.HandleMessage(ctx, traceManager, failureMsg); err != nil {
			c.logger.ErrorContext(ctx, "Failed to route task failure through LLM",
				"error", err,
				"task_id", taskID,
				"context_id", contextID)
		}
	}
}

// HandleTaskArtifact processes task artifact notifications from delegated agents.
// Instead of sending results directly to the user, it routes them through the
// LLM decision pipeline so the LLM can decide whether to chain to another agent
// (e.g., transcription -> summary) or respond to the user.
func (c *Cortex) HandleTaskArtifact(ctx context.Context, traceManager *observability.TraceManager, taskID, contextID string, artifact *pb.Artifact) {
	c.logger.DebugContext(ctx, "HandleTaskArtifact called",
		"task_id", taskID,
		"context_id", contextID,
		"artifact_id", artifact.GetArtifactId())

	var shouldRoute bool
	var responseText string

	// Use WithLock to ensure thread-safe state access
	_ = c.stateManager.WithLock(contextID, func(conversationState *state.ConversationState) error {
		// Check if this task is pending
		taskContext, pending := conversationState.PendingTasks[taskID]
		if !pending {
			c.logger.DebugContext(ctx, "Task not found in pending tasks", "task_id", taskID)
			return nil
		}

		// Store the artifact with the task context
		if taskContext.Artifacts == nil {
			taskContext.Artifacts = make([]*pb.Artifact, 0)
		}
		taskContext.Artifacts = append(taskContext.Artifacts, artifact)

		// Extract text content from artifact
		var textParts []string
		for _, part := range artifact.GetParts() {
			if textPart := part.GetText(); textPart != "" {
				textParts = append(textParts, textPart)
			}
		}

		c.logger.DebugContext(ctx, "Extracted text parts from artifact",
			"part_count", len(textParts))

		if len(textParts) > 0 {
			shouldRoute = true
			if artifact.GetName() != "" && artifact.GetDescription() != "" {
				responseText = fmt.Sprintf("%s: %s", artifact.GetName(), strings.Join(textParts, "\n"))
			} else {
				responseText = strings.Join(textParts, "\n")
			}
			c.logger.DebugContext(ctx, "Will route artifact through LLM", "response_text_length", len(responseText))
		}

		return nil
	})

	// Route through LLM decision pipeline instead of directly sending to user
	if shouldRoute && responseText != "" {
		artifactMsg := &pb.Message{
			MessageId: fmt.Sprintf("artifact_result_%d", time.Now().UnixNano()),
			ContextId: contextID,
			TaskId:    taskID,
			Role:      pb.Role_ROLE_AGENT,
			Content:   []*pb.Part{{Part: &pb.Part_Text{Text: responseText}}},
			Metadata:  artifact.GetMetadata(),
		}
		c.logger.DebugContext(ctx, "Routing artifact through HandleMessage", "context_id", contextID)
		if err := c.HandleMessage(ctx, traceManager, artifactMsg); err != nil {
			c.logger.ErrorContext(ctx, "Failed to route artifact through LLM",
				"error", err,
				"task_id", taskID,
				"context_id", contextID)
		}
	} else {
		c.logger.DebugContext(ctx, "No content to route",
			"should_route", shouldRoute,
			"has_text", responseText != "")
	}
}

// HandleAgentShutdown processes a shutdown notification from an agent.
// It preemptively removes the agent from the registry. The broker's "unregistered"
// event will follow shortly but this prevents any gap.
func (c *Cortex) HandleAgentShutdown(agentID string) {
	c.logger.Info("Agent shutdown notification received", "agent_id", agentID)
	c.UnregisterAgent(agentID)
}

// executeStatusRequest sends a status request message to a specific agent.
func (c *Cortex) executeStatusRequest(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, action llm.Action) error {
	reqCtx, reqSpan := traceManager.StartSpan(ctx, "cortex.status_request",
		attribute.String("session_id", conversationState.SessionID),
		attribute.String("target_agent", action.TargetAgent),
	)
	defer reqSpan.End()

	traceManager.AddComponentAttribute(reqSpan, "cortex_orchestrator")

	msg := &pb.Message{
		MessageId: fmt.Sprintf("status_request_%d", time.Now().UnixNano()),
		ContextId: conversationState.SessionID,
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: fmt.Sprintf("Status request for %s", action.TargetAgent)}},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"type":       structpb.NewStringValue("status_request"),
				"from_agent": structpb.NewStringValue(CortexAgentID),
			},
		},
	}

	routing := &pb.AgentEventMetadata{
		FromAgentId: CortexAgentID,
		ToAgentId:   action.TargetAgent,
		EventType:   "status_request",
		Priority:    pb.Priority_PRIORITY_MEDIUM,
	}

	err := c.messagePublisher.PublishMessage(reqCtx, msg, routing)
	if err != nil {
		traceManager.RecordError(reqSpan, err)
		return fmt.Errorf("failed to publish status request: %w", err)
	}

	traceManager.SetSpanSuccess(reqSpan)
	return nil
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
