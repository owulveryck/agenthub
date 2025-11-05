package llm

import (
	"context"
	"testing"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

func TestMockClient_DefaultBehavior(t *testing.T) {
	client := NewMockClient()

	event := &pb.Message{
		MessageId: "test-msg",
		Role:      pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: "Hello"}},
		},
	}

	decision, err := client.Decide(context.Background(), nil, nil, event)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}

	if decision.Reasoning == "" {
		t.Error("Expected reasoning to be set")
	}

	if len(decision.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(decision.Actions))
	}

	action := decision.Actions[0]
	if action.Type != "chat.response" {
		t.Errorf("Expected action type 'chat.response', got '%s'", action.Type)
	}

	if action.ResponseText == "" {
		t.Error("Expected response text to be set")
	}

	if client.CallCount != 1 {
		t.Errorf("Expected CallCount to be 1, got %d", client.CallCount)
	}

	if client.LastEvent != event {
		t.Error("Expected LastEvent to be set to the event")
	}
}

func TestMockClient_CustomDecideFunc(t *testing.T) {
	called := false
	customFunc := func(ctx context.Context, history []*pb.Message, agents []*pb.AgentCard, event *pb.Message) (*Decision, error) {
		called = true
		return &Decision{
			Reasoning: "Custom logic",
			Actions: []Action{
				{Type: "custom.action"},
			},
		}, nil
	}

	client := NewMockClientWithFunc(customFunc)

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "Test"}}},
	}

	decision, err := client.Decide(context.Background(), nil, nil, event)
	if err != nil {
		t.Fatalf("Decide failed: %v", err)
	}

	if !called {
		t.Error("Expected custom function to be called")
	}

	if decision.Reasoning != "Custom logic" {
		t.Errorf("Expected reasoning 'Custom logic', got '%s'", decision.Reasoning)
	}

	if len(decision.Actions) != 1 || decision.Actions[0].Type != "custom.action" {
		t.Error("Expected custom action")
	}
}

func TestSimpleEchoDecider(t *testing.T) {
	decider := SimpleEchoDecider()

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "Hello world"}}},
	}

	decision, err := decider(context.Background(), nil, nil, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	if len(decision.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(decision.Actions))
	}

	if decision.Actions[0].ResponseText != "Echo: Hello world" {
		t.Errorf("Expected 'Echo: Hello world', got '%s'", decision.Actions[0].ResponseText)
	}
}

func TestTaskDispatcherDecider(t *testing.T) {
	decider := TaskDispatcherDecider("transcription", "transcriber-agent")

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "Transcribe this"}}},
	}

	decision, err := decider(context.Background(), nil, nil, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should have 2 actions: acknowledgment + task dispatch
	if len(decision.Actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(decision.Actions))
	}

	// First action: chat response
	if decision.Actions[0].Type != "chat.response" {
		t.Errorf("Expected first action to be chat.response, got %s", decision.Actions[0].Type)
	}

	// Second action: task request
	taskAction := decision.Actions[1]
	if taskAction.Type != "task.request" {
		t.Errorf("Expected second action to be task.request, got %s", taskAction.Type)
	}

	if taskAction.TaskType != "transcription" {
		t.Errorf("Expected task type 'transcription', got '%s'", taskAction.TaskType)
	}

	if taskAction.TargetAgent != "transcriber-agent" {
		t.Errorf("Expected target agent 'transcriber-agent', got '%s'", taskAction.TargetAgent)
	}
}
