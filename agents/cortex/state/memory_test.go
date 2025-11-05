package state

import (
	"sync"
	"testing"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

func TestInMemoryStateManager_GetSet(t *testing.T) {
	sm := NewInMemoryStateManager()

	// Test Get on non-existent session returns empty state
	state, err := sm.Get("non-existent")
	if err != nil {
		t.Fatalf("Get should not error on non-existent session: %v", err)
	}
	if state == nil {
		t.Fatal("Get should return a new state, not nil")
	}
	if state.SessionID != "non-existent" {
		t.Errorf("Expected SessionID to be 'non-existent', got %s", state.SessionID)
	}

	// Test Set and Get
	testState := &ConversationState{
		SessionID: "test-session",
		Messages: []*pb.Message{
			{
				MessageId: "msg-1",
				Role:      pb.Role_ROLE_USER,
				Content: []*pb.Part{
					{Part: &pb.Part_Text{Text: "Hello"}},
				},
			},
		},
		PendingTasks:     make(map[string]*TaskContext),
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}

	err = sm.Set("test-session", testState)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := sm.Get("test-session")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.SessionID != testState.SessionID {
		t.Errorf("Expected SessionID %s, got %s", testState.SessionID, retrieved.SessionID)
	}

	if len(retrieved.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(retrieved.Messages))
	}

	if retrieved.Messages[0].MessageId != "msg-1" {
		t.Errorf("Expected message ID 'msg-1', got %s", retrieved.Messages[0].MessageId)
	}
}

func TestInMemoryStateManager_Delete(t *testing.T) {
	sm := NewInMemoryStateManager()

	state := &ConversationState{
		SessionID:        "test-delete",
		Messages:         []*pb.Message{},
		PendingTasks:     make(map[string]*TaskContext),
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}

	err := sm.Set("test-delete", state)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify it exists
	retrieved, err := sm.Get("test-delete")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.SessionID != "test-delete" {
		t.Error("State should exist before delete")
	}

	// Delete it
	err = sm.Delete("test-delete")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it returns a new empty state after deletion
	retrieved, err = sm.Get("test-delete")
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if len(retrieved.Messages) != 0 {
		t.Error("State should be empty after deletion")
	}
}

func TestInMemoryStateManager_Concurrency(t *testing.T) {
	sm := NewInMemoryStateManager()
	sessionID := "concurrent-test"

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Launch multiple goroutines that all update the same session
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			err := sm.WithLock(sessionID, func(state *ConversationState) error {
				// Add a message
				state.Messages = append(state.Messages, &pb.Message{
					MessageId: string(rune('a' + index)),
					Role:      pb.Role_ROLE_USER,
					Content: []*pb.Part{
						{Part: &pb.Part_Text{Text: "Test message"}},
					},
				})
				return nil
			})

			if err != nil {
				t.Errorf("WithLock failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify we have exactly numGoroutines messages (no lost updates)
	state, err := sm.Get(sessionID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(state.Messages) != numGoroutines {
		t.Errorf("Expected %d messages, got %d (lost updates detected)", numGoroutines, len(state.Messages))
	}
}

func TestInMemoryStateManager_WithLock(t *testing.T) {
	sm := NewInMemoryStateManager()
	sessionID := "lock-test"

	// Use WithLock to safely update state
	err := sm.WithLock(sessionID, func(state *ConversationState) error {
		state.Messages = append(state.Messages, &pb.Message{
			MessageId: "msg-locked",
			Role:      pb.Role_ROLE_USER,
			Content: []*pb.Part{
				{Part: &pb.Part_Text{Text: "Locked message"}},
			},
		})
		return nil
	})

	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	// Verify the update was persisted
	state, err := sm.Get(sessionID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(state.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(state.Messages))
	}

	if state.Messages[0].MessageId != "msg-locked" {
		t.Errorf("Expected message ID 'msg-locked', got %s", state.Messages[0].MessageId)
	}
}

func TestInMemoryStateManager_WithLock_Error(t *testing.T) {
	sm := NewInMemoryStateManager()
	sessionID := "error-test"

	// Test that errors from the callback are propagated
	testErr := "test error"
	err := sm.WithLock(sessionID, func(state *ConversationState) error {
		return &StateError{Op: "test", Err: testErr}
	})

	if err == nil {
		t.Fatal("Expected error to be returned from WithLock")
	}

	if err.Error() != "state test: test error" {
		t.Errorf("Expected error message 'state test: test error', got '%s'", err.Error())
	}
}
