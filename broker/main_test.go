package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// TestNewEventBusServer tests the creation of a new event bus server
func TestNewEventBusServer(t *testing.T) {
	server := NewEventBusServer()

	if server == nil {
		t.Fatal("NewEventBusServer() returned nil")
	}

	if server.taskSubscribers == nil {
		t.Error("taskSubscribers map not initialized")
	}

	if server.taskResultSubscribers == nil {
		t.Error("taskResultSubscribers map not initialized")
	}

	if server.taskProgressSubscribers == nil {
		t.Error("taskProgressSubscribers map not initialized")
	}

	if len(server.taskSubscribers) != 0 {
		t.Error("taskSubscribers should be empty initially")
	}

	if len(server.taskResultSubscribers) != 0 {
		t.Error("taskResultSubscribers should be empty initially")
	}

	if len(server.taskProgressSubscribers) != 0 {
		t.Error("taskProgressSubscribers should be empty initially")
	}
}

// TestPublishTask_ValidTask tests publishing a valid task
func TestPublishTask_ValidTask(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Create a valid task
	task := &pb.TaskMessage{
		TaskId:           "test-task-1",
		TaskType:         "test-type",
		RequesterAgentId: "agent-1",
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{
		Task: task,
	}

	resp, err := server.PublishTask(ctx, req)

	if err != nil {
		t.Fatalf("PublishTask failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true")
	}

	if resp.GetError() != "" {
		t.Errorf("Expected no error, got: %s", resp.GetError())
	}
}

// TestPublishTask_InvalidArguments tests various invalid argument scenarios
func TestPublishTask_InvalidArguments(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	testCases := []struct {
		name        string
		request     *pb.PublishTaskRequest
		expectedErr codes.Code
		expectedMsg string
	}{
		{
			name:        "nil request task",
			request:     &pb.PublishTaskRequest{Task: nil},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "task cannot be nil",
		},
		{
			name: "empty task_id",
			request: &pb.PublishTaskRequest{
				Task: &pb.TaskMessage{
					TaskId:           "",
					TaskType:         "test-type",
					RequesterAgentId: "agent-1",
				},
			},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "task_id cannot be empty",
		},
		{
			name: "empty task_type",
			request: &pb.PublishTaskRequest{
				Task: &pb.TaskMessage{
					TaskId:           "test-task-1",
					TaskType:         "",
					RequesterAgentId: "agent-1",
				},
			},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "task_type cannot be empty",
		},
		{
			name: "empty requester_agent_id",
			request: &pb.PublishTaskRequest{
				Task: &pb.TaskMessage{
					TaskId:           "test-task-1",
					TaskType:         "test-type",
					RequesterAgentId: "",
				},
			},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "requester_agent_id cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.PublishTask(ctx, tc.request)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Expected gRPC status error, got: %v", err)
			}

			if st.Code() != tc.expectedErr {
				t.Errorf("Expected error code %v, got %v", tc.expectedErr, st.Code())
			}

			if st.Message() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, st.Message())
			}
		})
	}
}

// TestPublishTaskResult_ValidResult tests publishing a valid task result
func TestPublishTaskResult_ValidResult(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	result := &pb.TaskResult{
		TaskId:          "test-task-1",
		Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
		ExecutorAgentId: "agent-1",
		CompletedAt:     timestamppb.Now(),
	}

	req := &pb.PublishTaskResultRequest{
		Result: result,
	}

	resp, err := server.PublishTaskResult(ctx, req)

	if err != nil {
		t.Fatalf("PublishTaskResult failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true")
	}
}

// TestPublishTaskResult_InvalidArguments tests invalid argument scenarios for task results
func TestPublishTaskResult_InvalidArguments(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	testCases := []struct {
		name        string
		request     *pb.PublishTaskResultRequest
		expectedErr codes.Code
		expectedMsg string
	}{
		{
			name:        "nil result",
			request:     &pb.PublishTaskResultRequest{Result: nil},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "result cannot be nil",
		},
		{
			name: "empty task_id",
			request: &pb.PublishTaskResultRequest{
				Result: &pb.TaskResult{
					TaskId:          "",
					Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
					ExecutorAgentId: "agent-1",
				},
			},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "task_id cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.PublishTaskResult(ctx, tc.request)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Expected gRPC status error, got: %v", err)
			}

			if st.Code() != tc.expectedErr {
				t.Errorf("Expected error code %v, got %v", tc.expectedErr, st.Code())
			}

			if st.Message() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, st.Message())
			}
		})
	}
}

// TestPublishTaskProgress_ValidProgress tests publishing valid task progress
func TestPublishTaskProgress_ValidProgress(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	progress := &pb.TaskProgress{
		TaskId:             "test-task-1",
		Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		ProgressPercentage: 50,
		ExecutorAgentId:    "agent-1",
		UpdatedAt:          timestamppb.Now(),
	}

	req := &pb.PublishTaskProgressRequest{
		Progress: progress,
	}

	resp, err := server.PublishTaskProgress(ctx, req)

	if err != nil {
		t.Fatalf("PublishTaskProgress failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true")
	}
}

// TestPublishTaskProgress_InvalidArguments tests invalid argument scenarios for task progress
func TestPublishTaskProgress_InvalidArguments(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	testCases := []struct {
		name        string
		request     *pb.PublishTaskProgressRequest
		expectedErr codes.Code
		expectedMsg string
	}{
		{
			name:        "nil progress",
			request:     &pb.PublishTaskProgressRequest{Progress: nil},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "progress cannot be nil",
		},
		{
			name: "empty task_id",
			request: &pb.PublishTaskProgressRequest{
				Progress: &pb.TaskProgress{
					TaskId:             "",
					Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
					ExecutorAgentId:    "agent-1",
					ProgressPercentage: 50,
				},
			},
			expectedErr: codes.InvalidArgument,
			expectedMsg: "task_id cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.PublishTaskProgress(ctx, tc.request)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("Expected gRPC status error, got: %v", err)
			}

			if st.Code() != tc.expectedErr {
				t.Errorf("Expected error code %v, got %v", tc.expectedErr, st.Code())
			}

			if st.Message() != tc.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", tc.expectedMsg, st.Message())
			}
		})
	}
}

// TestTaskRouting tests task routing logic
func TestTaskRouting(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Create a mock subscriber channel
	agentID := "test-agent"
	subChan := make(chan *pb.TaskMessage, 10)

	// Add subscriber
	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
	server.taskMu.Unlock()

	// Create a task targeted to the specific agent
	task := &pb.TaskMessage{
		TaskId:           "test-task-1",
		TaskType:         "test-type",
		RequesterAgentId: "requester-agent",
		ResponderAgentId: agentID, // Target specific agent
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	// Publish the task
	resp, err := server.PublishTask(ctx, req)
	if err != nil {
		t.Fatalf("PublishTask failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true")
	}

	// Verify task was received
	select {
	case receivedTask := <-subChan:
		if receivedTask.GetTaskId() != task.GetTaskId() {
			t.Errorf("Expected task ID %s, got %s", task.GetTaskId(), receivedTask.GetTaskId())
		}
		if receivedTask.GetTaskType() != task.GetTaskType() {
			t.Errorf("Expected task type %s, got %s", task.GetTaskType(), receivedTask.GetTaskType())
		}
	case <-time.After(1 * time.Second):
		t.Error("Task was not received within timeout")
	}
}

// TestBroadcastRouting tests broadcasting to all subscribers
func TestBroadcastRouting(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Create multiple mock subscribers
	agent1 := "agent-1"
	agent2 := "agent-2"
	subChan1 := make(chan *pb.TaskMessage, 10)
	subChan2 := make(chan *pb.TaskMessage, 10)

	// Add subscribers
	server.taskMu.Lock()
	server.taskSubscribers[agent1] = []chan *pb.TaskMessage{subChan1}
	server.taskSubscribers[agent2] = []chan *pb.TaskMessage{subChan2}
	server.taskMu.Unlock()

	// Create a broadcast task (no specific responder)
	task := &pb.TaskMessage{
		TaskId:           "broadcast-task-1",
		TaskType:         "broadcast-type",
		RequesterAgentId: "requester-agent",
		// ResponderAgentId is empty for broadcast
		CreatedAt: timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	// Publish the task
	resp, err := server.PublishTask(ctx, req)
	if err != nil {
		t.Fatalf("PublishTask failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true")
	}

	// Verify both agents received the task
	receivedCount := 0
	timeout := time.After(2 * time.Second)

	for receivedCount < 2 {
		select {
		case <-subChan1:
			receivedCount++
		case <-subChan2:
			receivedCount++
		case <-timeout:
			t.Errorf("Expected 2 tasks to be received, got %d", receivedCount)
			return
		}
	}
}

// TestNoSubscribers tests behavior when no subscribers exist
func TestNoSubscribers(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	task := &pb.TaskMessage{
		TaskId:           "test-task-1",
		TaskType:         "test-type",
		RequesterAgentId: "requester-agent",
		ResponderAgentId: "non-existent-agent",
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	// Should succeed even with no subscribers
	resp, err := server.PublishTask(ctx, req)
	if err != nil {
		t.Fatalf("PublishTask failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true even with no subscribers")
	}
}

// TestConcurrentTaskPublishing tests concurrent task publishing
func TestConcurrentTaskPublishing(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Add a subscriber
	agentID := "test-agent"
	subChan := make(chan *pb.TaskMessage, 100)

	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
	server.taskMu.Unlock()

	numTasks := 50
	var wg sync.WaitGroup

	// Publish tasks concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(taskNum int) {
			defer wg.Done()

			task := &pb.TaskMessage{
				TaskId:           fmt.Sprintf("concurrent-task-%d", taskNum),
				TaskType:         "concurrent-type",
				RequesterAgentId: "requester-agent",
				ResponderAgentId: agentID,
				CreatedAt:        timestamppb.Now(),
			}

			req := &pb.PublishTaskRequest{Task: task}
			_, err := server.PublishTask(ctx, req)
			if err != nil {
				t.Errorf("Failed to publish task %d: %v", taskNum, err)
			}
		}(i)
	}

	wg.Wait()

	// Count received tasks
	receivedCount := 0
	timeout := time.After(5 * time.Second)

	for receivedCount < numTasks {
		select {
		case <-subChan:
			receivedCount++
		case <-timeout:
			t.Errorf("Expected %d tasks, received %d", numTasks, receivedCount)
			return
		}
	}

	if receivedCount != numTasks {
		t.Errorf("Expected %d tasks, received %d", numTasks, receivedCount)
	}
}

// TestTaskParametersAndMetadata tests task with parameters and metadata
func TestTaskParametersAndMetadata(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Create parameters and metadata
	params, err := structpb.NewStruct(map[string]interface{}{
		"param1": "value1",
		"param2": 42,
		"param3": true,
	})
	if err != nil {
		t.Fatalf("Failed to create parameters: %v", err)
	}

	metadata, err := structpb.NewStruct(map[string]interface{}{
		"workflow_id": "workflow-123",
		"priority":    "high",
	})
	if err != nil {
		t.Fatalf("Failed to create metadata: %v", err)
	}

	task := &pb.TaskMessage{
		TaskId:           "param-task-1",
		TaskType:         "param-type",
		Parameters:       params,
		RequesterAgentId: "requester-agent",
		Priority:         pb.Priority_PRIORITY_HIGH,
		Metadata:         metadata,
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	resp, err := server.PublishTask(ctx, req)
	if err != nil {
		t.Fatalf("PublishTask failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("Expected success to be true")
	}
}

// TestChannelCleanup tests proper cleanup of subscriber channels
func TestChannelCleanup(t *testing.T) {
	server := NewEventBusServer()

	agentID := "cleanup-agent"

	// Simulate adding and removing subscribers
	subChan1 := make(chan *pb.TaskMessage, 10)
	subChan2 := make(chan *pb.TaskMessage, 10)

	// Add subscribers
	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan1, subChan2}
	server.taskMu.Unlock()

	// Verify subscribers exist
	server.taskMu.RLock()
	if len(server.taskSubscribers[agentID]) != 2 {
		t.Errorf("Expected 2 subscribers, got %d", len(server.taskSubscribers[agentID]))
	}
	server.taskMu.RUnlock()

	// Simulate cleanup for one subscriber
	server.taskMu.Lock()
	if subs, ok := server.taskSubscribers[agentID]; ok {
		newSubs := []chan *pb.TaskMessage{}
		for _, ch := range subs {
			if ch != subChan1 {
				newSubs = append(newSubs, ch)
			}
		}
		server.taskSubscribers[agentID] = newSubs
	}
	close(subChan1)
	server.taskMu.Unlock()

	// Verify one subscriber removed
	server.taskMu.RLock()
	if len(server.taskSubscribers[agentID]) != 1 {
		t.Errorf("Expected 1 subscriber after cleanup, got %d", len(server.taskSubscribers[agentID]))
	}
	server.taskMu.RUnlock()

	// Simulate cleanup for last subscriber
	server.taskMu.Lock()
	if subs, ok := server.taskSubscribers[agentID]; ok {
		newSubs := []chan *pb.TaskMessage{}
		for _, ch := range subs {
			if ch != subChan2 {
				newSubs = append(newSubs, ch)
			}
		}
		server.taskSubscribers[agentID] = newSubs
		if len(server.taskSubscribers[agentID]) == 0 {
			delete(server.taskSubscribers, agentID)
		}
	}
	close(subChan2)
	server.taskMu.Unlock()

	// Verify agent completely removed
	server.taskMu.RLock()
	if _, exists := server.taskSubscribers[agentID]; exists {
		t.Error("Expected agent to be completely removed from subscribers")
	}
	server.taskMu.RUnlock()
}