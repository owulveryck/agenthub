package llm

import (
	"context"
	"strings"
	"testing"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"google.golang.org/protobuf/types/known/structpb"
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
	customFunc := func(ctx context.Context, history []*pb.Message, agents map[string]*pb.AgentCard, event *pb.Message) (*Decision, error) {
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

func TestIntelligentDecider_AudioRequestWithoutAgent(t *testing.T) {
	decider := IntelligentDecider()

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "analyze this mp3 file"}}},
	}

	// No agents available
	agents := map[string]*pb.AgentCard{}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should NOT dispatch a task — should respond directly
	for _, action := range decision.Actions {
		if action.Type == "task.request" {
			t.Error("Should not dispatch task.request when agent_mp3 is not available")
		}
	}

	// Should have a chat response mentioning unavailability
	if len(decision.Actions) == 0 {
		t.Fatal("Expected at least one action")
	}
	found := false
	for _, action := range decision.Actions {
		if action.Type == "chat.response" {
			found = true
		}
	}
	if !found {
		t.Error("Expected a chat.response action")
	}
}

func TestIntelligentDecider_AudioRequestWithAgent(t *testing.T) {
	decider := IntelligentDecider()

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "analyze this mp3 file"}}},
	}

	// agent_mp3 is available
	agents := map[string]*pb.AgentCard{
		"agent_mp3": {Name: "agent_mp3", Description: "Audio analyzer"},
	}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should dispatch to agent_mp3
	hasTaskRequest := false
	for _, action := range decision.Actions {
		if action.Type == "task.request" && action.TargetAgent == "agent_mp3" {
			hasTaskRequest = true
		}
	}
	if !hasTaskRequest {
		t.Error("Expected a task.request action dispatching to agent_mp3")
	}
}

func TestIntelligentDecider_EchoRequestWithoutAgent(t *testing.T) {
	decider := IntelligentDecider()

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "echo this message"}}},
	}

	// No agents available
	agents := map[string]*pb.AgentCard{}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should NOT dispatch a task
	for _, action := range decision.Actions {
		if action.Type == "task.request" {
			t.Error("Should not dispatch task.request when agent_echo is not available")
		}
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

func TestIntelligentDecider_StatusRequestWithAgent(t *testing.T) {
	decider := IntelligentDecider()

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "what is agent_mp3 doing?"}}},
	}

	agents := map[string]*pb.AgentCard{
		"agent_mp3": {Name: "agent_mp3", Description: "Audio analyzer"},
	}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should have a status.request action targeting agent_mp3
	hasStatusRequest := false
	for _, action := range decision.Actions {
		if action.Type == "status.request" && action.TargetAgent == "agent_mp3" {
			hasStatusRequest = true
		}
	}
	if !hasStatusRequest {
		t.Error("Expected a status.request action targeting agent_mp3")
	}
}

func TestIntelligentDecider_StatusRequestWithoutAgent(t *testing.T) {
	decider := IntelligentDecider()

	event := &pb.Message{
		MessageId: "test",
		Role:      pb.Role_ROLE_USER,
		Content:   []*pb.Part{{Part: &pb.Part_Text{Text: "what is the status?"}}},
	}

	// No agents available
	agents := map[string]*pb.AgentCard{}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should NOT have a status.request action — should respond directly
	for _, action := range decision.Actions {
		if action.Type == "status.request" {
			t.Error("Should not dispatch status.request when no matching agent found")
		}
	}

	// Should have a chat.response
	hasChatResponse := false
	for _, action := range decision.Actions {
		if action.Type == "chat.response" {
			hasChatResponse = true
		}
	}
	if !hasChatResponse {
		t.Error("Expected a chat.response action")
	}
}

func TestIntelligentDecider_TranscriptionWithoutSummaryAgent(t *testing.T) {
	decider := IntelligentDecider()

	meta, err := structpb.NewStruct(map[string]interface{}{
		"content_type": "transcription",
	})
	if err != nil {
		t.Fatalf("Failed to create metadata: %v", err)
	}

	event := &pb.Message{
		MessageId: "test-transcription",
		Role:      pb.Role_ROLE_AGENT,
		TaskId:    "task-123",
		Metadata:  meta,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: "Hello, this is the transcribed text from the audio file."}},
		},
	}

	// No agents available
	agents := map[string]*pb.AgentCard{}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should NOT dispatch a task.request
	for _, action := range decision.Actions {
		if action.Type == "task.request" {
			t.Error("Should not dispatch task.request when no summary agent is available")
		}
	}

	// Should have a single chat.response with the raw transcription text
	if len(decision.Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(decision.Actions))
	}

	action := decision.Actions[0]
	if action.Type != "chat.response" {
		t.Errorf("Expected action type 'chat.response', got '%s'", action.Type)
	}
	if action.ResponseText != "Hello, this is the transcribed text from the audio file." {
		t.Errorf("Expected raw transcription text, got '%s'", action.ResponseText)
	}

	// Reasoning should mention transcription
	if !strings.Contains(strings.ToLower(decision.Reasoning), "transcription") {
		t.Errorf("Expected reasoning to mention 'transcription', got '%s'", decision.Reasoning)
	}
}

func TestIntelligentDecider_TranscriptionWithSummaryAgent(t *testing.T) {
	decider := IntelligentDecider()

	meta, err := structpb.NewStruct(map[string]interface{}{
		"content_type": "transcription",
	})
	if err != nil {
		t.Fatalf("Failed to create metadata: %v", err)
	}

	event := &pb.Message{
		MessageId: "test-transcription",
		Role:      pb.Role_ROLE_AGENT,
		TaskId:    "task-123",
		Metadata:  meta,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: "Hello, this is the transcribed text from the audio file."}},
		},
	}

	// agent_summary is available
	agents := map[string]*pb.AgentCard{
		"agent_summary": {Name: "Summary Agent", Description: "Summarizes text content"},
	}

	decision, err := decider(context.Background(), nil, agents, event)
	if err != nil {
		t.Fatalf("Decider failed: %v", err)
	}

	// Should contain a task.request targeting agent_summary
	hasTaskRequest := false
	for _, action := range decision.Actions {
		if action.Type == "task.request" && action.TargetAgent == "agent_summary" {
			hasTaskRequest = true
		}
	}
	if !hasTaskRequest {
		t.Error("Expected a task.request action dispatching to agent_summary")
	}
}
