package cortex

import (
	"context"
	"testing"
	"time"

	"github.com/owulveryck/agenthub/agents/cortex/llm"
	"github.com/owulveryck/agenthub/agents/cortex/state"
	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/observability"
)

// MockAgentHubClient is a mock of the AgentHub client for testing
type MockAgentHubClient struct {
	PublishedMessages []*pb.Message
	PublishError      error
}

func (m *MockAgentHubClient) PublishMessage(ctx context.Context, msg *pb.Message, routing *pb.AgentEventMetadata) error {
	if m.PublishError != nil {
		return m.PublishError
	}
	m.PublishedMessages = append(m.PublishedMessages, msg)
	return nil
}

func TestCortex_RegisterAgent(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	cortex := NewCortex(sm, llmClient, mockClient)

	// Register an agent
	agentCard := &pb.AgentCard{
		Name:        "test-agent",
		Description: "A test agent that does testing",
		Skills: []*pb.AgentSkill{
			{
				Id:          "test-skill",
				Name:        "Testing",
				Description: "Performs testing tasks",
			},
		},
	}

	cortex.RegisterAgent("test-agent", agentCard)

	// Verify agent was registered
	if len(cortex.registeredAgents) != 1 {
		t.Errorf("Expected 1 registered agent, got %d", len(cortex.registeredAgents))
	}

	retrieved, exists := cortex.registeredAgents["test-agent"]
	if !exists {
		t.Fatal("Agent should be registered")
	}

	if retrieved.Name != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", retrieved.Name)
	}
}

func TestCortex_HandleChatRequest(t *testing.T) {
	sm := state.NewInMemoryStateManager()

	// Mock LLM that returns a simple acknowledgment
	llmClient := llm.NewMockClientWithFunc(func(ctx context.Context, history []*pb.Message, agents []*pb.AgentCard, event *pb.Message) (*llm.Decision, error) {
		return &llm.Decision{
			Reasoning: "User said hello, responding",
			Actions: []llm.Action{
				{
					Type:         "chat.response",
					ResponseText: "Hello! How can I help you?",
				},
			},
		}, nil
	})

	mockClient := &MockAgentHubClient{}
	cortex := NewCortex(sm, llmClient, mockClient)

	// Create a chat request
	chatRequest := &pb.Message{
		MessageId: "msg-1",
		ContextId: "session-1",
		Role:      pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: "Hello"}},
		},
	}

	// Handle the chat request
	traceManager := observability.NewTraceManager("cortex_test")
	err := cortex.HandleMessage(context.Background(), traceManager, chatRequest)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify state was updated
	sessionState, err := sm.Get("session-1")
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	// Should have 2 messages: user request + cortex response
	if len(sessionState.Messages) != 2 {
		t.Errorf("Expected 2 messages in state, got %d", len(sessionState.Messages))
	}

	// Verify a message was published
	if len(mockClient.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mockClient.PublishedMessages))
	}

	published := mockClient.PublishedMessages[0]
	if published.Role != pb.Role_ROLE_AGENT {
		t.Errorf("Expected published message role to be AGENT, got %v", published.Role)
	}

	if published.ContextId != "session-1" {
		t.Errorf("Expected context ID 'session-1', got '%s'", published.ContextId)
	}

	responseText := published.Content[0].GetText()
	if responseText != "Hello! How can I help you?" {
		t.Errorf("Unexpected response text: %s", responseText)
	}
}

func TestCortex_HandleTaskResult(t *testing.T) {
	sm := state.NewInMemoryStateManager()

	// First, set up a pending task in the state
	taskContext := &state.TaskContext{
		TaskID:      "task-123",
		TaskType:    "echo",
		RequestedAt: time.Now().Unix(),
		OriginalInput: &pb.Message{
			MessageId: "original-msg",
			Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "Echo this"}}},
		},
		UserNotified: true,
	}

	initialState := &state.ConversationState{
		SessionID: "session-1",
		Messages:  []*pb.Message{},
		PendingTasks: map[string]*state.TaskContext{
			"task-123": taskContext,
		},
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}

	sm.Set("session-1", initialState)

	// Mock LLM that synthesizes the result
	llmClient := llm.NewMockClientWithFunc(func(ctx context.Context, history []*pb.Message, agents []*pb.AgentCard, event *pb.Message) (*llm.Decision, error) {
		return &llm.Decision{
			Reasoning: "Task completed, informing user",
			Actions: []llm.Action{
				{
					Type:         "chat.response",
					ResponseText: "The echo task is complete: Echo this",
				},
			},
		}, nil
	})

	mockClient := &MockAgentHubClient{}
	cortex := NewCortex(sm, llmClient, mockClient)

	// Create a task result message
	taskResult := &pb.Message{
		MessageId: "result-msg",
		ContextId: "session-1",
		TaskId:    "task-123",
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: "Echo this"}},
		},
	}

	// Handle the task result
	traceManager := observability.NewTraceManager("cortex_test")
	err := cortex.HandleMessage(context.Background(), traceManager, taskResult)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify the pending task was removed
	sessionState, err := sm.Get("session-1")
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	if len(sessionState.PendingTasks) != 0 {
		t.Errorf("Expected pending task to be removed, but %d tasks remain", len(sessionState.PendingTasks))
	}

	// Verify response was published
	if len(mockClient.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mockClient.PublishedMessages))
	}
}

func TestCortex_GetAvailableAgents(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	cortex := NewCortex(sm, llmClient, mockClient)

	// Register multiple agents
	cortex.RegisterAgent("agent-1", &pb.AgentCard{Name: "agent-1", Description: "First agent"})
	cortex.RegisterAgent("agent-2", &pb.AgentCard{Name: "agent-2", Description: "Second agent"})

	agents := cortex.GetAvailableAgents()
	if len(agents) != 2 {
		t.Errorf("Expected 2 available agents, got %d", len(agents))
	}
}
