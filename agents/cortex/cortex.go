package cortex

import (
	"context"
	"fmt"
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
	registeredAgents map[string]*pb.AgentCard
	agentsMu         sync.RWMutex
}

// NewCortex creates a new Cortex instance.
func NewCortex(
	stateManager state.StateManager,
	llmClient llm.Client,
	messagePublisher MessagePublisher,
) *Cortex {
	return &Cortex{
		stateManager:     stateManager,
		llmClient:        llmClient,
		messagePublisher: messagePublisher,
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
	reqCtx, reqSpan := traceManager.StartSpan(ctx, "cortex_chat_request",
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
	llmCtx, llmSpan := traceManager.StartSpan(reqCtx, "cortex_llm_decide",
		attribute.String("message_id", msg.GetMessageId()),
		attribute.Int("available_agents", len(availableAgents)),
	)
	decision, err := c.llmClient.Decide(llmCtx, conversationState.Messages, availableAgents, msg)
	if err != nil {
		traceManager.RecordError(llmSpan, err)
		traceManager.RecordError(reqSpan, err)
		llmSpan.End()
		return fmt.Errorf("LLM decision failed: %w", err)
	}
	traceManager.SetSpanSuccess(llmSpan)
	traceManager.AddSpanEvent(llmSpan, "llm_decision_made",
		attribute.Int("action_count", len(decision.Actions)),
	)
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
	resCtx, resSpan := traceManager.StartSpan(ctx, "cortex_task_result",
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
	llmCtx, llmSpan := traceManager.StartSpan(resCtx, "cortex_llm_synthesize",
		attribute.String("task_id", msg.GetTaskId()),
		attribute.Int("available_agents", len(availableAgents)),
	)
	decision, err := c.llmClient.Decide(llmCtx, conversationState.Messages, availableAgents, msg)
	if err != nil {
		traceManager.RecordError(llmSpan, err)
		traceManager.RecordError(resSpan, err)
		llmSpan.End()
		return fmt.Errorf("LLM decision failed: %w", err)
	}
	traceManager.SetSpanSuccess(llmSpan)
	traceManager.AddSpanEvent(llmSpan, "llm_synthesis_complete",
		attribute.Int("action_count", len(decision.Actions)),
	)
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
	actCtx, actSpan := traceManager.StartSpan(ctx, "cortex_execute_actions",
		attribute.Int("action_count", len(actions)),
		attribute.String("session_id", conversationState.SessionID),
	)
	defer actSpan.End()

	traceManager.AddComponentAttribute(actSpan, "cortex_orchestrator")

	for i, action := range actions {
		traceManager.AddSpanEvent(actSpan, "executing_action",
			attribute.Int("action_index", i),
			attribute.String("action_type", action.Type),
		)

		switch action.Type {
		case "chat.response":
			if err := c.executeChatResponse(actCtx, traceManager, conversationState, action, triggeringMsg); err != nil {
				traceManager.RecordError(actSpan, err)
				return fmt.Errorf("failed to execute chat response: %w", err)
			}

		case "task.request":
			if err := c.executeTaskRequest(actCtx, traceManager, conversationState, action, triggeringMsg); err != nil {
				traceManager.RecordError(actSpan, err)
				return fmt.Errorf("failed to execute task request: %w", err)
			}

		default:
			err := fmt.Errorf("unknown action type: %s", action.Type)
			traceManager.RecordError(actSpan, err)
			return err
		}
	}

	traceManager.SetSpanSuccess(actSpan)
	return nil
}

// executeChatResponse sends a chat response to the user.
func (c *Cortex) executeChatResponse(ctx context.Context, traceManager *observability.TraceManager, conversationState *state.ConversationState, action llm.Action, triggeringMsg *pb.Message) error {
	// Start tracing for chat response execution
	respCtx, respSpan := traceManager.StartSpan(ctx, "cortex_send_chat_response",
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
	taskCtx, taskSpan := traceManager.StartSpan(ctx, "cortex_dispatch_task",
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
