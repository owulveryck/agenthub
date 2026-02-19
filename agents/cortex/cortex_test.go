package cortex

import (
	"context"
	"log/slog"
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
	PublishedRouting  []*pb.AgentEventMetadata
	PublishError      error
}

func (m *MockAgentHubClient) PublishMessage(ctx context.Context, msg *pb.Message, routing *pb.AgentEventMetadata) error {
	if m.PublishError != nil {
		return m.PublishError
	}
	m.PublishedMessages = append(m.PublishedMessages, msg)
	m.PublishedRouting = append(m.PublishedRouting, routing)
	return nil
}

func TestCortex_RegisterAgent(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	cortex := NewCortex(sm, llmClient, mockClient, slog.Default())

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
	llmClient := llm.NewMockClientWithFunc(func(ctx context.Context, history []*pb.Message, agents map[string]*pb.AgentCard, event *pb.Message) (*llm.Decision, error) {
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
	cortex := NewCortex(sm, llmClient, mockClient, slog.Default())

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
	llmClient := llm.NewMockClientWithFunc(func(ctx context.Context, history []*pb.Message, agents map[string]*pb.AgentCard, event *pb.Message) (*llm.Decision, error) {
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
	cortex := NewCortex(sm, llmClient, mockClient, slog.Default())

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

func TestCortex_UnregisterAgent(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	c := NewCortex(sm, llmClient, mockClient, slog.Default())

	// Register an agent
	agentCard := &pb.AgentCard{
		Name:        "test-agent",
		Description: "A test agent",
		Skills: []*pb.AgentSkill{
			{Id: "s1", Name: "Skill1", Description: "Does stuff"},
		},
	}
	c.RegisterAgent("test-agent", agentCard)

	// Verify it's registered
	agents := c.GetAvailableAgents()
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent after registration, got %d", len(agents))
	}

	// Unregister the agent
	c.UnregisterAgent("test-agent")

	// Verify it's gone
	agents = c.GetAvailableAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents after unregistration, got %d", len(agents))
	}

	// Unregistering a non-existent agent should not panic
	c.UnregisterAgent("nonexistent-agent")
}

func TestCortex_HandleTaskFailure(t *testing.T) {
	sm := state.NewInMemoryStateManager()

	// Set up a pending task in the state
	taskContext := &state.TaskContext{
		TaskID:      "task-fail-1",
		TaskType:    "mp3_analysis",
		RequestedAt: time.Now().Unix(),
		OriginalInput: &pb.Message{
			MessageId: "original-msg",
			Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "Analyze this audio"}}},
		},
		UserNotified: true,
	}

	initialState := &state.ConversationState{
		SessionID: "session-fail",
		Messages:  []*pb.Message{},
		PendingTasks: map[string]*state.TaskContext{
			"task-fail-1": taskContext,
		},
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}
	sm.Set("session-fail", initialState)

	// Mock LLM that relays the failure to the user
	llmClient := llm.NewMockClientWithFunc(func(ctx context.Context, history []*pb.Message, agents map[string]*pb.AgentCard, event *pb.Message) (*llm.Decision, error) {
		return &llm.Decision{
			Reasoning: "Task failed, informing user",
			Actions: []llm.Action{
				{
					Type:         "chat.response",
					ResponseText: "The audio analysis failed: No file path provided in message",
				},
			},
		}, nil
	})

	mockClient := &MockAgentHubClient{}
	c := NewCortex(sm, llmClient, mockClient, slog.Default())

	// Simulate a FAILED task status with an error message
	failedStatus := &pb.TaskStatus{
		State: pb.TaskState_TASK_STATE_FAILED,
		Update: &pb.Message{
			Role: pb.Role_ROLE_AGENT,
			Content: []*pb.Part{
				{Part: &pb.Part_Text{Text: "Task failed: No file path provided in message"}},
			},
		},
	}

	traceManager := observability.NewTraceManager("cortex_test")
	c.HandleTaskCompletion(context.Background(), traceManager, "task-fail-1", "session-fail", failedStatus)

	// Verify a response was published to the user
	if len(mockClient.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mockClient.PublishedMessages))
	}

	published := mockClient.PublishedMessages[0]
	if published.Role != pb.Role_ROLE_AGENT {
		t.Errorf("Expected published message role to be AGENT, got %v", published.Role)
	}
	if published.ContextId != "session-fail" {
		t.Errorf("Expected context ID 'session-fail', got '%s'", published.ContextId)
	}

	responseText := published.Content[0].GetText()
	if responseText != "The audio analysis failed: No file path provided in message" {
		t.Errorf("Unexpected response text: %s", responseText)
	}

	// Verify the pending task was removed from state
	sessionState, err := sm.Get("session-fail")
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}
	if len(sessionState.PendingTasks) != 0 {
		t.Errorf("Expected pending task to be removed, but %d tasks remain", len(sessionState.PendingTasks))
	}
}

func TestCortex_HandleAgentShutdown(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	c := NewCortex(sm, llmClient, mockClient, slog.Default())

	// Register an agent
	c.RegisterAgent("agent_mp3", &pb.AgentCard{
		Name:        "agent_mp3",
		Description: "Audio analyzer",
	})

	// Verify it's registered
	agents := c.GetAvailableAgents()
	if len(agents) != 1 {
		t.Fatalf("Expected 1 agent after registration, got %d", len(agents))
	}

	// Handle shutdown notification
	c.HandleAgentShutdown("agent_mp3")

	// Verify the agent is removed
	agents = c.GetAvailableAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents after shutdown, got %d", len(agents))
	}

	// Calling HandleAgentShutdown for a non-existent agent should not panic
	c.HandleAgentShutdown("nonexistent-agent")
}

func TestCortex_ExecuteStatusRequest(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	c := NewCortex(sm, llmClient, mockClient, slog.Default())

	traceManager := observability.NewTraceManager("cortex_test")

	convState := &state.ConversationState{
		SessionID:        "session-1",
		Messages:         []*pb.Message{},
		PendingTasks:     make(map[string]*state.TaskContext),
		RegisteredAgents: make(map[string]*pb.AgentCard),
	}

	action := llm.Action{
		Type:        "status.request",
		TargetAgent: "agent_mp3",
	}

	err := c.executeStatusRequest(context.Background(), traceManager, convState, action)
	if err != nil {
		t.Fatalf("executeStatusRequest failed: %v", err)
	}

	// Verify a message was published
	if len(mockClient.PublishedMessages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(mockClient.PublishedMessages))
	}

	published := mockClient.PublishedMessages[0]

	// Verify metadata type
	if published.GetMetadata() == nil || published.GetMetadata().GetFields() == nil {
		t.Fatal("Expected metadata to be set")
	}
	msgType := published.GetMetadata().GetFields()["type"].GetStringValue()
	if msgType != "status_request" {
		t.Errorf("Expected metadata type 'status_request', got '%s'", msgType)
	}
}

func TestCortex_GetAvailableAgents(t *testing.T) {
	sm := state.NewInMemoryStateManager()
	llmClient := llm.NewMockClient()
	mockClient := &MockAgentHubClient{}

	cortex := NewCortex(sm, llmClient, mockClient, slog.Default())

	// Register multiple agents
	cortex.RegisterAgent("agent-1", &pb.AgentCard{Name: "agent-1", Description: "First agent"})
	cortex.RegisterAgent("agent-2", &pb.AgentCard{Name: "agent-2", Description: "Second agent"})

	agents := cortex.GetAvailableAgents()
	if len(agents) != 2 {
		t.Errorf("Expected 2 available agents, got %d", len(agents))
	}
}
