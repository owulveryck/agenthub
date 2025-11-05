package cortex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/owulveryck/agenthub/agents/cortex/llm"
	"github.com/owulveryck/agenthub/agents/cortex/state"
	pb "github.com/owulveryck/agenthub/events/a2a"
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
func (c *Cortex) HandleMessage(ctx context.Context, msg *pb.Message) error {
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
			return c.handleTaskResult(ctx, conversationState, msg)
		}

		// Otherwise, it's a new chat request
		return c.handleChatRequest(ctx, conversationState, msg)
	})
}

// handleChatRequest processes a new chat request from a user.
func (c *Cortex) handleChatRequest(ctx context.Context, conversationState *state.ConversationState, msg *pb.Message) error {
	// Get available agents
	availableAgents := c.GetAvailableAgents()

	// Call LLM to decide what to do
	decision, err := c.llmClient.Decide(ctx, conversationState.Messages, availableAgents, msg)
	if err != nil {
		return fmt.Errorf("LLM decision failed: %w", err)
	}

	// Execute the decided actions
	return c.executeActions(ctx, conversationState, decision.Actions, msg)
}

// handleTaskResult processes a task result from an agent.
func (c *Cortex) handleTaskResult(ctx context.Context, conversationState *state.ConversationState, msg *pb.Message) error {
	// Remove the task from pending tasks
	delete(conversationState.PendingTasks, msg.TaskId)

	// Get available agents
	availableAgents := c.GetAvailableAgents()

	// Call LLM to decide how to synthesize this result
	decision, err := c.llmClient.Decide(ctx, conversationState.Messages, availableAgents, msg)
	if err != nil {
		return fmt.Errorf("LLM decision failed: %w", err)
	}

	// Execute the decided actions
	return c.executeActions(ctx, conversationState, decision.Actions, msg)
}

// executeActions executes the actions decided by the LLM.
func (c *Cortex) executeActions(ctx context.Context, conversationState *state.ConversationState, actions []llm.Action, triggeringMsg *pb.Message) error {
	for _, action := range actions {
		switch action.Type {
		case "chat.response":
			if err := c.executeChatResponse(ctx, conversationState, action, triggeringMsg); err != nil {
				return fmt.Errorf("failed to execute chat response: %w", err)
			}

		case "task.request":
			if err := c.executeTaskRequest(ctx, conversationState, action, triggeringMsg); err != nil {
				return fmt.Errorf("failed to execute task request: %w", err)
			}

		default:
			return fmt.Errorf("unknown action type: %s", action.Type)
		}
	}

	return nil
}

// executeChatResponse sends a chat response to the user.
func (c *Cortex) executeChatResponse(ctx context.Context, conversationState *state.ConversationState, action llm.Action, triggeringMsg *pb.Message) error {
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

	// Add to conversation history
	conversationState.Messages = append(conversationState.Messages, responseMsg)

	// Publish the message
	routing := &pb.AgentEventMetadata{
		FromAgentId: CortexAgentID,
		EventType:   "a2a.message.chat_response",
		Priority:    pb.Priority_PRIORITY_MEDIUM,
	}

	return c.messagePublisher.PublishMessage(ctx, responseMsg, routing)
}

// executeTaskRequest dispatches a task request to an agent.
func (c *Cortex) executeTaskRequest(ctx context.Context, conversationState *state.ConversationState, action llm.Action, triggeringMsg *pb.Message) error {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

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

	// Track this task as pending
	conversationState.PendingTasks[taskID] = &state.TaskContext{
		TaskID:        taskID,
		TaskType:      action.TaskType,
		RequestedAt:   time.Now().Unix(),
		OriginalInput: triggeringMsg,
		UserNotified:  true, // We assume we've already sent an acknowledgment
	}

	// Publish the task request
	routing := &pb.AgentEventMetadata{
		FromAgentId: CortexAgentID,
		ToAgentId:   action.TargetAgent,
		EventType:   fmt.Sprintf("a2a.task.%s", action.TaskType),
		Priority:    pb.Priority_PRIORITY_MEDIUM,
	}

	return c.messagePublisher.PublishMessage(ctx, taskMsg, routing)
}
