package state

import (
	"fmt"
	"sync"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

// InMemoryStateManager is a simple in-memory implementation of StateManager.
// This is suitable for the POC but should be replaced with persistent storage
// (Redis, PostgreSQL, etc.) for production use.
type InMemoryStateManager struct {
	mu       sync.RWMutex
	sessions map[string]*ConversationState
	locks    sync.Map // sessionID → *sync.Mutex for per-session locking
}

// NewInMemoryStateManager creates a new in-memory state manager.
func NewInMemoryStateManager() *InMemoryStateManager {
	return &InMemoryStateManager{
		sessions: make(map[string]*ConversationState),
	}
}

// Get retrieves the conversation state for a given session.
// If the session doesn't exist, it returns a new empty state.
func (sm *InMemoryStateManager) Get(sessionID string) (*ConversationState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, exists := sm.sessions[sessionID]; exists {
		// Return a copy to prevent external modifications
		return copyState(state), nil
	}

	// Return a new empty state
	return &ConversationState{
		SessionID:        sessionID,
		Messages:         []*pb.Message{},
		PendingTasks:     make(map[string]*TaskContext),
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}, nil
}

// Set persists the conversation state for a given session.
func (sm *InMemoryStateManager) Set(sessionID string, state *ConversationState) error {
	if state == nil {
		return &StateError{Op: "set", Err: "state cannot be nil"}
	}

	if sessionID == "" {
		return &StateError{Op: "set", Err: "sessionID cannot be empty"}
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Store a copy to prevent external modifications
	sm.sessions[sessionID] = copyState(state)
	return nil
}

// Delete removes the conversation state for a given session.
func (sm *InMemoryStateManager) Delete(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)
	return nil
}

// WithLock executes a function with exclusive access to a session's state.
// This ensures thread-safe operations on the state.
func (sm *InMemoryStateManager) WithLock(sessionID string, fn func(*ConversationState) error) error {
	if sessionID == "" {
		return &StateError{Op: "withlock", Err: "sessionID cannot be empty"}
	}

	// Get or create a lock for this session
	lockInterface, _ := sm.locks.LoadOrStore(sessionID, &sync.Mutex{})
	sessionLock := lockInterface.(*sync.Mutex)

	// Lock the session
	sessionLock.Lock()
	defer sessionLock.Unlock()

	// Get the current state (or create new one)
	state, err := sm.Get(sessionID)
	if err != nil {
		return err
	}

	// Execute the function with the state
	if err := fn(state); err != nil {
		return err
	}

	// Persist the updated state
	return sm.Set(sessionID, state)
}

// StateError represents an error from a state operation.
type StateError struct {
	Op  string // Operation that failed (e.g., "get", "set")
	Err string // Error message
}

func (e *StateError) Error() string {
	return fmt.Sprintf("state %s: %s", e.Op, e.Err)
}

// copyState creates a deep copy of a ConversationState.
// This prevents external modifications to stored states.
func copyState(state *ConversationState) *ConversationState {
	if state == nil {
		return nil
	}

	// Create new state
	newState := &ConversationState{
		SessionID:        state.SessionID,
		Messages:         make([]*pb.Message, len(state.Messages)),
		PendingTasks:     make(map[string]*TaskContext),
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}

	// Copy messages (proto messages are immutable in Go, so we can share pointers)
	copy(newState.Messages, state.Messages)

	// Copy pending tasks
	for k, v := range state.PendingTasks {
		newState.PendingTasks[k] = &TaskContext{
			TaskID:        v.TaskID,
			TaskType:      v.TaskType,
			RequestedAt:   v.RequestedAt,
			OriginalInput: v.OriginalInput,
			UserNotified:  v.UserNotified,
		}
	}

	// Copy registered agents (proto messages are immutable)
	for k, v := range state.RegisteredAgents {
		newState.RegisteredAgents[k] = v
	}

	return newState
}
