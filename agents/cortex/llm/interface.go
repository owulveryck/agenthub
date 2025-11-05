package llm

import (
	"context"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

// Action represents a decision made by the LLM about what to do next.
type Action struct {
	Type string // "chat.response", "task.request", etc.

	// For chat.response
	ResponseText string

	// For task.request
	TaskType    string
	TaskPayload map[string]interface{}
	TargetAgent string // If empty, broadcast

	// Correlation
	CorrelationID string
}

// Decision represents the LLM's analysis and planned actions.
type Decision struct {
	Reasoning string   // Why the LLM decided to take these actions
	Actions   []Action // The actions to take
}

// Client is the interface for interacting with an LLM.
// The LLM is used by Cortex to decide what actions to take based on:
// - Conversation history
// - Available agents/tools
// - Incoming messages/events
type Client interface {
	// Decide analyzes the current state and decides what actions to take.
	// It takes:
	// - context: for cancellation and tracing
	// - conversationHistory: the full history of messages in this session
	// - availableAgents: the agent cards for all registered agents
	// - newEvent: the new message or event that triggered this decision
	Decide(
		ctx context.Context,
		conversationHistory []*pb.Message,
		availableAgents []*pb.AgentCard,
		newEvent *pb.Message,
	) (*Decision, error)
}
