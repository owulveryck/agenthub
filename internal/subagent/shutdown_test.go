package subagent

import (
	"context"
	"testing"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"google.golang.org/grpc"
)

// mockPublishMessageClient captures PublishMessage calls for testing
type mockPublishMessageClient struct {
	pb.AgentHubClient
	publishedRequests []*pb.PublishMessageRequest
}

func (m *mockPublishMessageClient) PublishMessage(ctx context.Context, in *pb.PublishMessageRequest, opts ...grpc.CallOption) (*pb.PublishResponse, error) {
	m.publishedRequests = append(m.publishedRequests, in)
	return &pb.PublishResponse{Success: true}, nil
}

func TestSendShutdownNotification(t *testing.T) {
	mock := &mockPublishMessageClient{}

	s := &SubAgent{
		config: &Config{AgentID: "agent_test"},
		agentCard: &pb.AgentCard{
			Name:        "agent_test",
			Description: "A test agent",
		},
	}
	// Inject the mock gRPC client directly
	s.grpcClient = mock

	ctx := context.Background()
	s.sendShutdownNotification(ctx)

	if len(mock.publishedRequests) != 1 {
		t.Fatalf("Expected 1 published request, got %d", len(mock.publishedRequests))
	}

	req := mock.publishedRequests[0]
	msg := req.GetMessage()

	// Verify metadata.type = "agent_shutting_down"
	if msg.GetMetadata() == nil || msg.GetMetadata().GetFields() == nil {
		t.Fatal("Expected metadata to be set")
	}
	msgType := msg.GetMetadata().GetFields()["type"].GetStringValue()
	if msgType != "agent_shutting_down" {
		t.Errorf("Expected metadata type 'agent_shutting_down', got '%s'", msgType)
	}

	// Verify metadata.agent_id
	agentID := msg.GetMetadata().GetFields()["agent_id"].GetStringValue()
	if agentID != "agent_test" {
		t.Errorf("Expected metadata agent_id 'agent_test', got '%s'", agentID)
	}

	// Verify role is ROLE_AGENT
	if msg.GetRole() != pb.Role_ROLE_AGENT {
		t.Errorf("Expected role ROLE_AGENT, got %v", msg.GetRole())
	}

	// Verify broadcast routing (empty to_agent_id)
	routing := req.GetRouting()
	if routing == nil {
		t.Fatal("Expected routing to be set")
	}
	if routing.GetToAgentId() != "" {
		t.Errorf("Expected empty to_agent_id for broadcast, got '%s'", routing.GetToAgentId())
	}
	if routing.GetFromAgentId() != "agent_test" {
		t.Errorf("Expected from_agent_id 'agent_test', got '%s'", routing.GetFromAgentId())
	}

	// Verify priority is HIGH
	if routing.GetPriority() != pb.Priority_PRIORITY_HIGH {
		t.Errorf("Expected PRIORITY_HIGH, got %v", routing.GetPriority())
	}

	// Verify content mentions shutting down
	if len(msg.GetContent()) == 0 {
		t.Fatal("Expected content to be set")
	}
	text := msg.GetContent()[0].GetText()
	if text == "" {
		t.Error("Expected content text to mention shutting down")
	}
}
