package subagent

import (
	"context"
	"testing"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

func TestHandleStatusRequest_DefaultProvider(t *testing.T) {
	mock := &mockPublishMessageClient{}

	s := &SubAgent{
		config: &Config{AgentID: "agent_test"},
		agentCard: &pb.AgentCard{
			Name: "agent_test",
		},
	}
	s.grpcClient = mock

	request := &pb.Message{
		MessageId: "status_req_1",
		ContextId: "session-42",
	}

	s.handleStatusRequest(context.Background(), request)

	if len(mock.publishedRequests) != 1 {
		t.Fatalf("Expected 1 published request, got %d", len(mock.publishedRequests))
	}

	req := mock.publishedRequests[0]
	msg := req.GetMessage()

	// Default status should be "idle"
	if len(msg.GetContent()) == 0 {
		t.Fatal("Expected content to be set")
	}
	text := msg.GetContent()[0].GetText()
	if text != "idle" {
		t.Errorf("Expected status 'idle', got '%s'", text)
	}
}

func TestHandleStatusRequest_CustomProvider(t *testing.T) {
	mock := &mockPublishMessageClient{}

	s := &SubAgent{
		config: &Config{AgentID: "agent_test"},
		agentCard: &pb.AgentCard{
			Name: "agent_test",
		},
		statusProvider: func() string {
			return "processing task X"
		},
	}
	s.grpcClient = mock

	request := &pb.Message{
		MessageId: "status_req_2",
		ContextId: "session-42",
	}

	s.handleStatusRequest(context.Background(), request)

	if len(mock.publishedRequests) != 1 {
		t.Fatalf("Expected 1 published request, got %d", len(mock.publishedRequests))
	}

	req := mock.publishedRequests[0]
	msg := req.GetMessage()

	text := msg.GetContent()[0].GetText()
	if text != "processing task X" {
		t.Errorf("Expected status 'processing task X', got '%s'", text)
	}
}

func TestHandleStatusRequest_ReplyRouting(t *testing.T) {
	mock := &mockPublishMessageClient{}

	s := &SubAgent{
		config: &Config{AgentID: "agent_mp3"},
		agentCard: &pb.AgentCard{
			Name: "agent_mp3",
		},
	}
	s.grpcClient = mock

	request := &pb.Message{
		MessageId: "status_req_3",
		ContextId: "session-99",
	}

	s.handleStatusRequest(context.Background(), request)

	if len(mock.publishedRequests) != 1 {
		t.Fatalf("Expected 1 published request, got %d", len(mock.publishedRequests))
	}

	req := mock.publishedRequests[0]
	msg := req.GetMessage()
	routing := req.GetRouting()

	// Verify metadata type
	msgType := msg.GetMetadata().GetFields()["type"].GetStringValue()
	if msgType != "status_reply" {
		t.Errorf("Expected metadata type 'status_reply', got '%s'", msgType)
	}

	// Verify metadata agent_id
	agentID := msg.GetMetadata().GetFields()["agent_id"].GetStringValue()
	if agentID != "agent_mp3" {
		t.Errorf("Expected metadata agent_id 'agent_mp3', got '%s'", agentID)
	}

	// Verify targeted to cortex
	if routing.GetToAgentId() != "cortex" {
		t.Errorf("Expected to_agent_id 'cortex', got '%s'", routing.GetToAgentId())
	}

	// Verify context_id is preserved
	if msg.GetContextId() != "session-99" {
		t.Errorf("Expected context_id 'session-99', got '%s'", msg.GetContextId())
	}
}
