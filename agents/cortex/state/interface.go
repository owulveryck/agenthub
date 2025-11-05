package state

import (
	pb "github.com/owulveryck/agenthub/events/a2a"
)

// ConversationState represents the history and context of a single conversation.
// This is the state that Cortex maintains for each session.
type ConversationState struct {
	SessionID        string        // Unique identifier for this conversation session
	Messages         []*pb.Message // Full conversation history (both USER and AGENT messages)
	PendingTasks     map[string]*TaskContext
	RegisteredAgents map[string]*pb.AgentCard // Agents available in this session
}

// TaskContext tracks the context of a pending task to maintain correlation
type TaskContext struct {
	TaskID        string
	TaskType      string
	RequestedAt   int64 // Unix timestamp
	OriginalInput *pb.Message
	UserNotified  bool // Did we send "I'm working on it" acknowledgment?
}

// StateManager defines the interface for persisting conversation state.
// This abstraction allows for different implementations (in-memory, Redis, PostgreSQL, etc.)
type StateManager interface {
	// Get retrieves the conversation state for a given session.
	// Returns a new empty state if the session doesn't exist.
	Get(sessionID string) (*ConversationState, error)

	// Set persists the conversation state for a given session.
	Set(sessionID string, state *ConversationState) error

	// Delete removes the conversation state for a given session.
	Delete(sessionID string) error

	// WithLock executes a function with exclusive access to a session's state.
	// This ensures thread-safe operations on the state.
	WithLock(sessionID string, fn func(*ConversationState) error) error
}
