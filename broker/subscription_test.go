package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// MockTaskStream simulates a gRPC stream for tasks
type MockTaskStream struct {
	ctx       context.Context
	sentTasks []*pb.TaskMessage
	sendError error
	mu        sync.Mutex
}

func (m *MockTaskStream) Send(task *pb.TaskMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}

	m.sentTasks = append(m.sentTasks, task)
	return nil
}

func (m *MockTaskStream) Context() context.Context {
	return m.ctx
}

func (m *MockTaskStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *MockTaskStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *MockTaskStream) SetHeader(metadata.MD) error {
	return nil
}

func (m *MockTaskStream) SendHeader(metadata.MD) error {
	return nil
}

func (m *MockTaskStream) SetTrailer(metadata.MD) {}

func (m *MockTaskStream) SetRPCStatus(st *status.Status) {}

func (m *MockTaskStream) GetSentTasks() []*pb.TaskMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks := make([]*pb.TaskMessage, len(m.sentTasks))
	copy(tasks, m.sentTasks)
	return tasks
}

// MockResultStream simulates a gRPC stream for results
type MockResultStream struct {
	ctx         context.Context
	sentResults []*pb.TaskResult
	sendError   error
	mu          sync.Mutex
}

func (m *MockResultStream) Send(result *pb.TaskResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}

	m.sentResults = append(m.sentResults, result)
	return nil
}

func (m *MockResultStream) Context() context.Context {
	return m.ctx
}

func (m *MockResultStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *MockResultStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *MockResultStream) SetHeader(metadata.MD) error {
	return nil
}

func (m *MockResultStream) SendHeader(metadata.MD) error {
	return nil
}

func (m *MockResultStream) SetTrailer(metadata.MD) {}

func (m *MockResultStream) SetRPCStatus(st *status.Status) {}

func (m *MockResultStream) GetSentResults() []*pb.TaskResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	results := make([]*pb.TaskResult, len(m.sentResults))
	copy(results, m.sentResults)
	return results
}

// MockProgressStream simulates a gRPC stream for progress
type MockProgressStream struct {
	ctx          context.Context
	sentProgress []*pb.TaskProgress
	sendError    error
	mu           sync.Mutex
}

func (m *MockProgressStream) Send(progress *pb.TaskProgress) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendError != nil {
		return m.sendError
	}

	m.sentProgress = append(m.sentProgress, progress)
	return nil
}

func (m *MockProgressStream) Context() context.Context {
	return m.ctx
}

func (m *MockProgressStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *MockProgressStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *MockProgressStream) SetHeader(metadata.MD) error {
	return nil
}

func (m *MockProgressStream) SendHeader(metadata.MD) error {
	return nil
}

func (m *MockProgressStream) SetTrailer(metadata.MD) {}

func (m *MockProgressStream) SetRPCStatus(st *status.Status) {}

func (m *MockProgressStream) GetSentProgress() []*pb.TaskProgress {
	m.mu.Lock()
	defer m.mu.Unlock()

	progress := make([]*pb.TaskProgress, len(m.sentProgress))
	copy(progress, m.sentProgress)
	return progress
}

// TestSubscribeToTasks_ValidSubscription tests task subscription
func TestSubscribeToTasks_ValidSubscription(t *testing.T) {
	server := NewEventBusServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &MockTaskStream{ctx: ctx}

	req := &pb.SubscribeToTasksRequest{
		AgentId: "test-agent",
	}

	// Start subscription in goroutine
	done := make(chan error, 1)
	go func() {
		err := server.SubscribeToTasks(req, stream)
		done <- err
	}()

	// Wait for subscription to be set up
	time.Sleep(10 * time.Millisecond)

	// Publish a task targeted to this agent
	task := &pb.TaskMessage{
		TaskId:           "test-task-1",
		TaskType:         "test-type",
		RequesterAgentId: "requester",
		ResponderAgentId: "test-agent",
		CreatedAt:        timestamppb.Now(),
	}

	publishReq := &pb.PublishTaskRequest{Task: task}
	_, err := server.PublishTask(context.Background(), publishReq)
	if err != nil {
		t.Fatalf("Failed to publish task: %v", err)
	}

	// Wait for task to be sent
	time.Sleep(10 * time.Millisecond)

	// Cancel context to stop subscription
	cancel()

	// Wait for subscription to end
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Subscription did not end within timeout")
	}

	// Verify task was sent
	sentTasks := stream.GetSentTasks()
	if len(sentTasks) != 1 {
		t.Errorf("Expected 1 task to be sent, got %d", len(sentTasks))
	} else if sentTasks[0].GetTaskId() != "test-task-1" {
		t.Errorf("Expected task ID 'test-task-1', got '%s'", sentTasks[0].GetTaskId())
	}
}

// TestSubscribeToTasks_InvalidArguments tests subscription validation
func TestSubscribeToTasks_InvalidArguments(t *testing.T) {
	server := NewEventBusServer()

	ctx := context.Background()
	stream := &MockTaskStream{ctx: ctx}

	req := &pb.SubscribeToTasksRequest{
		AgentId: "", // Empty agent ID
	}

	err := server.SubscribeToTasks(req, stream)

	if err == nil {
		t.Fatal("Expected error for empty agent ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got: %v", st.Code())
	}

	if st.Message() != "agent_id cannot be empty" {
		t.Errorf("Expected 'agent_id cannot be empty', got: '%s'", st.Message())
	}
}

// TestSubscribeToTaskResults_ValidSubscription tests result subscription
func TestSubscribeToTaskResults_ValidSubscription(t *testing.T) {
	server := NewEventBusServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &MockResultStream{ctx: ctx}

	req := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: "requester-agent",
	}

	// Start subscription in goroutine
	done := make(chan error, 1)
	go func() {
		err := server.SubscribeToTaskResults(req, stream)
		done <- err
	}()

	// Wait for subscription to be set up
	time.Sleep(10 * time.Millisecond)

	// Publish a task result
	result := &pb.TaskResult{
		TaskId:          "test-task-1",
		Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
		ExecutorAgentId: "executor-agent",
		CompletedAt:     timestamppb.Now(),
	}

	publishReq := &pb.PublishTaskResultRequest{Result: result}
	_, err := server.PublishTaskResult(context.Background(), publishReq)
	if err != nil {
		t.Fatalf("Failed to publish result: %v", err)
	}

	// Wait for result to be sent
	time.Sleep(10 * time.Millisecond)

	// Cancel context to stop subscription
	cancel()

	// Wait for subscription to end
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Subscription did not end within timeout")
	}

	// Verify result was sent
	sentResults := stream.GetSentResults()
	if len(sentResults) != 1 {
		t.Errorf("Expected 1 result to be sent, got %d", len(sentResults))
	} else if sentResults[0].GetTaskId() != "test-task-1" {
		t.Errorf("Expected task ID 'test-task-1', got '%s'", sentResults[0].GetTaskId())
	}
}

// TestSubscribeToTaskResults_InvalidArguments tests result subscription validation
func TestSubscribeToTaskResults_InvalidArguments(t *testing.T) {
	server := NewEventBusServer()

	ctx := context.Background()
	stream := &MockResultStream{ctx: ctx}

	req := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: "", // Empty requester ID
	}

	err := server.SubscribeToTaskResults(req, stream)

	if err == nil {
		t.Fatal("Expected error for empty requester agent ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got: %v", st.Code())
	}

	if st.Message() != "requester_agent_id cannot be empty" {
		t.Errorf("Expected 'requester_agent_id cannot be empty', got: '%s'", st.Message())
	}
}

// TestSubscribeToTaskProgress_ValidSubscription tests progress subscription
func TestSubscribeToTaskProgress_ValidSubscription(t *testing.T) {
	server := NewEventBusServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &MockProgressStream{ctx: ctx}

	req := &pb.SubscribeToTaskResultsRequest{ // Note: using same request type
		RequesterAgentId: "requester-agent",
	}

	// Start subscription in goroutine
	done := make(chan error, 1)
	go func() {
		err := server.SubscribeToTaskProgress(req, stream)
		done <- err
	}()

	// Wait for subscription to be set up
	time.Sleep(10 * time.Millisecond)

	// Publish task progress
	progress := &pb.TaskProgress{
		TaskId:             "test-task-1",
		Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		ProgressPercentage: 50,
		ExecutorAgentId:    "executor-agent",
		UpdatedAt:          timestamppb.Now(),
	}

	publishReq := &pb.PublishTaskProgressRequest{Progress: progress}
	_, err := server.PublishTaskProgress(context.Background(), publishReq)
	if err != nil {
		t.Fatalf("Failed to publish progress: %v", err)
	}

	// Wait for progress to be sent
	time.Sleep(10 * time.Millisecond)

	// Cancel context to stop subscription
	cancel()

	// Wait for subscription to end
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Subscription did not end within timeout")
	}

	// Verify progress was sent
	sentProgress := stream.GetSentProgress()
	if len(sentProgress) != 1 {
		t.Errorf("Expected 1 progress to be sent, got %d", len(sentProgress))
	} else if sentProgress[0].GetTaskId() != "test-task-1" {
		t.Errorf("Expected task ID 'test-task-1', got '%s'", sentProgress[0].GetTaskId())
	}
}

// TestSubscribeToTaskProgress_InvalidArguments tests progress subscription validation
func TestSubscribeToTaskProgress_InvalidArguments(t *testing.T) {
	server := NewEventBusServer()

	ctx := context.Background()
	stream := &MockProgressStream{ctx: ctx}

	req := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: "", // Empty requester ID
	}

	err := server.SubscribeToTaskProgress(req, stream)

	if err == nil {
		t.Fatal("Expected error for empty requester agent ID")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got: %v", err)
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument, got: %v", st.Code())
	}

	if st.Message() != "requester_agent_id cannot be empty" {
		t.Errorf("Expected 'requester_agent_id cannot be empty', got: '%s'", st.Message())
	}
}
