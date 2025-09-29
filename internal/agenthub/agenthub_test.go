package agenthub

import (
	"context"
	"testing"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// newTestEventBusService creates a new EventBusService for testing
func newTestEventBusService() *EventBusService {
	config := NewGRPCConfig("test")
	config.HealthPort = "0"
	config.ServerAddr = ":0"
	server, err := NewAgentHubServer(config)
	if err != nil {
		panic(err)
	}
	return NewEventBusService(server)
}

func TestEventBusService_Creation(t *testing.T) {
	service := newTestEventBusService()
	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}
	if service.Server == nil {
		t.Fatal("Expected server to be set, got nil")
	}
}

func TestEventBusService_PublishTask(t *testing.T) {
	service := newTestEventBusService()
	ctx := context.Background()

	// Test valid task
	task := &pb.TaskMessage{
		TaskId:           "test-task-1",
		TaskType:         "test_type",
		RequesterAgentId: "test-requester",
		ResponderAgentId: "test-responder",
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{
		Task: task,
	}

	resp, err := service.PublishTask(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !resp.Success {
		t.Fatal("Expected success to be true")
	}
}

func TestEventBusService_PublishTask_InvalidRequests(t *testing.T) {
	service := newTestEventBusService()
	ctx := context.Background()

	tests := []struct {
		name string
		req  *pb.PublishTaskRequest
	}{
		{
			name: "nil task",
			req:  &pb.PublishTaskRequest{Task: nil},
		},
		{
			name: "empty task_id",
			req: &pb.PublishTaskRequest{
				Task: &pb.TaskMessage{
					TaskId:           "",
					TaskType:         "test_type",
					RequesterAgentId: "test-requester",
				},
			},
		},
		{
			name: "empty task_type",
			req: &pb.PublishTaskRequest{
				Task: &pb.TaskMessage{
					TaskId:           "test-task-1",
					TaskType:         "",
					RequesterAgentId: "test-requester",
				},
			},
		},
		{
			name: "empty requester_agent_id",
			req: &pb.PublishTaskRequest{
				Task: &pb.TaskMessage{
					TaskId:           "test-task-1",
					TaskType:         "test_type",
					RequesterAgentId: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.PublishTask(ctx, tt.req)
			if err == nil {
				t.Fatal("Expected error for invalid request, got nil")
			}
		})
	}
}

func TestGRPCConfig_Creation(t *testing.T) {
	config := NewGRPCConfig("test")
	if config == nil {
		t.Fatal("Expected config to be created, got nil")
	}
	if config.ComponentName != "test" {
		t.Fatalf("Expected ComponentName to be 'test', got %s", config.ComponentName)
	}
	if config.ServerAddr == "" {
		t.Fatal("Expected ServerAddr to be set")
	}
	if config.BrokerAddr == "" {
		t.Fatal("Expected BrokerAddr to be set")
	}
	if config.HealthPort == "" {
		t.Fatal("Expected HealthPort to be set")
	}
}

func TestAgentHubServer_Creation(t *testing.T) {
	config := NewGRPCConfig("test")
	config.HealthPort = "0"
	config.ServerAddr = ":0"

	server, err := NewAgentHubServer(config)
	if err != nil {
		t.Fatalf("Expected no error creating server, got %v", err)
	}
	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}
	if server.Server == nil {
		t.Fatal("Expected gRPC server to be set")
	}
	if server.Logger == nil {
		t.Fatal("Expected logger to be set")
	}
	if server.TraceManager == nil {
		t.Fatal("Expected trace manager to be set")
	}
	if server.MetricsManager == nil {
		t.Fatal("Expected metrics manager to be set")
	}
}
