package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// TestServer wraps the eventBusServer with test utilities
type TestServer struct {
	*eventBusServer
	grpcServer *grpc.Server
	listener   *bufconn.Listener
	client     pb.EventBusClient
	conn       *grpc.ClientConn
}

// NewTestServer creates a new test server with gRPC client
func NewTestServer() *TestServer {
	const bufSize = 1024 * 1024

	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	eventBusService := NewEventBusServer()
	pb.RegisterEventBusServer(grpcServer, eventBusService)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			// Server stopped
		}
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to dial bufnet: %v", err))
	}

	client := pb.NewEventBusClient(conn)

	return &TestServer{
		eventBusServer: eventBusService,
		grpcServer:     grpcServer,
		listener:       lis,
		client:         client,
		conn:           conn,
	}
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	if ts.conn != nil {
		ts.conn.Close()
	}
	if ts.grpcServer != nil {
		ts.grpcServer.Stop()
	}
	if ts.listener != nil {
		ts.listener.Close()
	}
}

// Client returns the gRPC client
func (ts *TestServer) Client() pb.EventBusClient {
	return ts.client
}

// Server returns the underlying eventBusServer
func (ts *TestServer) Server() *eventBusServer {
	return ts.eventBusServer
}

// TaskBuilder helps build test tasks
type TaskBuilder struct {
	task *pb.TaskMessage
}

// NewTaskBuilder creates a new task builder with default values
func NewTaskBuilder() *TaskBuilder {
	return &TaskBuilder{
		task: &pb.TaskMessage{
			TaskId:           "test-task",
			TaskType:         "test",
			RequesterAgentId: "test-requester",
			Priority:         pb.Priority_PRIORITY_MEDIUM,
			CreatedAt:        timestamppb.Now(),
		},
	}
}

// WithTaskID sets the task ID
func (tb *TaskBuilder) WithTaskID(id string) *TaskBuilder {
	tb.task.TaskId = id
	return tb
}

// WithTaskType sets the task type
func (tb *TaskBuilder) WithTaskType(taskType string) *TaskBuilder {
	tb.task.TaskType = taskType
	return tb
}

// WithRequester sets the requester agent ID
func (tb *TaskBuilder) WithRequester(agentID string) *TaskBuilder {
	tb.task.RequesterAgentId = agentID
	return tb
}

// WithResponder sets the responder agent ID
func (tb *TaskBuilder) WithResponder(agentID string) *TaskBuilder {
	tb.task.ResponderAgentId = agentID
	return tb
}

// WithPriority sets the task priority
func (tb *TaskBuilder) WithPriority(priority pb.Priority) *TaskBuilder {
	tb.task.Priority = priority
	return tb
}

// WithParameters sets the task parameters
func (tb *TaskBuilder) WithParameters(params map[string]interface{}) *TaskBuilder {
	if paramsStruct, err := structpb.NewStruct(params); err == nil {
		tb.task.Parameters = paramsStruct
	}
	return tb
}

// WithMetadata sets the task metadata
func (tb *TaskBuilder) WithMetadata(metadata map[string]interface{}) *TaskBuilder {
	if metadataStruct, err := structpb.NewStruct(metadata); err == nil {
		tb.task.Metadata = metadataStruct
	}
	return tb
}

// WithDeadline sets the task deadline
func (tb *TaskBuilder) WithDeadline(deadline time.Time) *TaskBuilder {
	tb.task.Deadline = timestamppb.New(deadline)
	return tb
}

// Build returns the constructed task
func (tb *TaskBuilder) Build() *pb.TaskMessage {
	return tb.task
}

// ResultBuilder helps build test task results
type ResultBuilder struct {
	result *pb.TaskResult
}

// NewResultBuilder creates a new result builder with default values
func NewResultBuilder(taskID string) *ResultBuilder {
	return &ResultBuilder{
		result: &pb.TaskResult{
			TaskId:          taskID,
			Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
			ExecutorAgentId: "test-executor",
			CompletedAt:     timestamppb.Now(),
		},
	}
}

// WithStatus sets the result status
func (rb *ResultBuilder) WithStatus(status pb.TaskStatus) *ResultBuilder {
	rb.result.Status = status
	return rb
}

// WithExecutor sets the executor agent ID
func (rb *ResultBuilder) WithExecutor(agentID string) *ResultBuilder {
	rb.result.ExecutorAgentId = agentID
	return rb
}

// WithResult sets the result data
func (rb *ResultBuilder) WithResult(result map[string]interface{}) *ResultBuilder {
	if resultStruct, err := structpb.NewStruct(result); err == nil {
		rb.result.Result = resultStruct
	}
	return rb
}

// WithError sets the error message
func (rb *ResultBuilder) WithError(errorMsg string) *ResultBuilder {
	rb.result.ErrorMessage = errorMsg
	rb.result.Status = pb.TaskStatus_TASK_STATUS_FAILED
	return rb
}

// WithExecutionMetadata sets the execution metadata
func (rb *ResultBuilder) WithExecutionMetadata(metadata map[string]interface{}) *ResultBuilder {
	if metadataStruct, err := structpb.NewStruct(metadata); err == nil {
		rb.result.ExecutionMetadata = metadataStruct
	}
	return rb
}

// Build returns the constructed result
func (rb *ResultBuilder) Build() *pb.TaskResult {
	return rb.result
}

// ProgressBuilder helps build test task progress
type ProgressBuilder struct {
	progress *pb.TaskProgress
}

// NewProgressBuilder creates a new progress builder with default values
func NewProgressBuilder(taskID string) *ProgressBuilder {
	return &ProgressBuilder{
		progress: &pb.TaskProgress{
			TaskId:             taskID,
			Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
			ProgressPercentage: 50,
			ExecutorAgentId:    "test-executor",
			UpdatedAt:          timestamppb.Now(),
		},
	}
}

// WithStatus sets the progress status
func (pb *ProgressBuilder) WithStatus(status pb.TaskStatus) *ProgressBuilder {
	pb.progress.Status = status
	return pb
}

// WithPercentage sets the progress percentage
func (pb *ProgressBuilder) WithPercentage(percentage int32) *ProgressBuilder {
	pb.progress.ProgressPercentage = percentage
	return pb
}

// WithMessage sets the progress message
func (pb *ProgressBuilder) WithMessage(message string) *ProgressBuilder {
	pb.progress.ProgressMessage = message
	return pb
}

// WithExecutor sets the executor agent ID
func (pb *ProgressBuilder) WithExecutor(agentID string) *ProgressBuilder {
	pb.progress.ExecutorAgentId = agentID
	return pb
}

// WithProgressData sets the progress data
func (pb *ProgressBuilder) WithProgressData(data map[string]interface{}) *ProgressBuilder {
	if dataStruct, err := structpb.NewStruct(data); err == nil {
		pb.progress.ProgressData = dataStruct
	}
	return pb
}

// Build returns the constructed progress
func (pb *ProgressBuilder) Build() *pb.TaskProgress {
	return pb.progress
}

// MockSubscriber represents a test subscriber
type MockSubscriber struct {
	AgentID       string
	Channel       chan *pb.TaskMessage
	ReceivedTasks []*pb.TaskMessage
	mu            sync.RWMutex
	active        bool
}

// NewMockSubscriber creates a new mock subscriber
func NewMockSubscriber(agentID string, bufferSize int) *MockSubscriber {
	return &MockSubscriber{
		AgentID:       agentID,
		Channel:       make(chan *pb.TaskMessage, bufferSize),
		ReceivedTasks: make([]*pb.TaskMessage, 0),
		active:        true,
	}
}

// Start begins consuming tasks from the channel
func (ms *MockSubscriber) Start() {
	go func() {
		for task := range ms.Channel {
			ms.mu.Lock()
			if ms.active {
				ms.ReceivedTasks = append(ms.ReceivedTasks, task)
			}
			ms.mu.Unlock()
		}
	}()
}

// Stop stops the subscriber
func (ms *MockSubscriber) Stop() {
	ms.mu.Lock()
	ms.active = false
	ms.mu.Unlock()
	close(ms.Channel)
}

// GetReceivedTasks returns a copy of received tasks
func (ms *MockSubscriber) GetReceivedTasks() []*pb.TaskMessage {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	tasks := make([]*pb.TaskMessage, len(ms.ReceivedTasks))
	copy(tasks, ms.ReceivedTasks)
	return tasks
}

// GetReceivedTaskCount returns the number of received tasks
func (ms *MockSubscriber) GetReceivedTaskCount() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.ReceivedTasks)
}

// WaitForTasks waits for a specific number of tasks with timeout
func (ms *MockSubscriber) WaitForTasks(expectedCount int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if ms.GetReceivedTaskCount() >= expectedCount {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// TestSubscriberManager manages multiple test subscribers
type TestSubscriberManager struct {
	subscribers map[string]*MockSubscriber
	server      *eventBusServer
	mu          sync.RWMutex
}

// NewTestSubscriberManager creates a new subscriber manager
func NewTestSubscriberManager(server *eventBusServer) *TestSubscriberManager {
	return &TestSubscriberManager{
		subscribers: make(map[string]*MockSubscriber),
		server:      server,
	}
}

// AddSubscriber adds a new test subscriber
func (tsm *TestSubscriberManager) AddSubscriber(agentID string, bufferSize int) *MockSubscriber {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	subscriber := NewMockSubscriber(agentID, bufferSize)
	subscriber.Start()

	// Register with server
	tsm.server.taskMu.Lock()
	tsm.server.taskSubscribers[agentID] = append(tsm.server.taskSubscribers[agentID], subscriber.Channel)
	tsm.server.taskMu.Unlock()

	tsm.subscribers[agentID] = subscriber
	return subscriber
}

// RemoveSubscriber removes a test subscriber
func (tsm *TestSubscriberManager) RemoveSubscriber(agentID string) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	if subscriber, exists := tsm.subscribers[agentID]; exists {
		// Unregister from server
		tsm.server.taskMu.Lock()
		if subs, ok := tsm.server.taskSubscribers[agentID]; ok {
			newSubs := []chan *pb.TaskMessage{}
			for _, ch := range subs {
				if ch != subscriber.Channel {
					newSubs = append(newSubs, ch)
				}
			}
			tsm.server.taskSubscribers[agentID] = newSubs
			if len(tsm.server.taskSubscribers[agentID]) == 0 {
				delete(tsm.server.taskSubscribers, agentID)
			}
		}
		tsm.server.taskMu.Unlock()

		subscriber.Stop()
		delete(tsm.subscribers, agentID)
	}
}

// GetSubscriber returns a subscriber by agent ID
func (tsm *TestSubscriberManager) GetSubscriber(agentID string) *MockSubscriber {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	return tsm.subscribers[agentID]
}

// RemoveAll removes all subscribers
func (tsm *TestSubscriberManager) RemoveAll() {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	for agentID, subscriber := range tsm.subscribers {
		// Unregister from server
		tsm.server.taskMu.Lock()
		delete(tsm.server.taskSubscribers, agentID)
		tsm.server.taskMu.Unlock()

		subscriber.Stop()
	}

	tsm.subscribers = make(map[string]*MockSubscriber)
}

// AssertTaskReceived asserts that a task was received by a subscriber
func AssertTaskReceived(t *testing.T, subscriber *MockSubscriber, expectedTaskID string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tasks := subscriber.GetReceivedTasks()
		for _, task := range tasks {
			if task.GetTaskId() == expectedTaskID {
				return // Found the task
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("Task %s was not received by subscriber %s within timeout", expectedTaskID, subscriber.AgentID)
}

// AssertTaskNotReceived asserts that a task was not received by a subscriber
func AssertTaskNotReceived(t *testing.T, subscriber *MockSubscriber, unexpectedTaskID string, waitTime time.Duration) {
	t.Helper()

	time.Sleep(waitTime) // Wait to ensure task would have been received if it was going to be

	tasks := subscriber.GetReceivedTasks()
	for _, task := range tasks {
		if task.GetTaskId() == unexpectedTaskID {
			t.Errorf("Task %s was unexpectedly received by subscriber %s", unexpectedTaskID, subscriber.AgentID)
			return
		}
	}
}

// AssertSubscriberCount asserts the expected number of subscribers for an agent
func AssertSubscriberCount(t *testing.T, server *eventBusServer, agentID string, expectedCount int) {
	t.Helper()

	server.taskMu.RLock()
	actualCount := len(server.taskSubscribers[agentID])
	server.taskMu.RUnlock()

	if actualCount != expectedCount {
		t.Errorf("Expected %d subscribers for agent %s, got %d", expectedCount, agentID, actualCount)
	}
}

// AssertTotalAgentCount asserts the total number of agents with subscriptions
func AssertTotalAgentCount(t *testing.T, server *eventBusServer, expectedCount int) {
	t.Helper()

	server.taskMu.RLock()
	actualCount := len(server.taskSubscribers)
	server.taskMu.RUnlock()

	if actualCount != expectedCount {
		t.Errorf("Expected %d agents with subscriptions, got %d", expectedCount, actualCount)
	}
}

// PublishTestTask is a helper function to publish a test task
func PublishTestTask(t *testing.T, client pb.EventBusClient, task *pb.TaskMessage) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.PublishTaskRequest{Task: task}
	resp, err := client.PublishTask(ctx, req)

	if err != nil {
		t.Fatalf("Failed to publish task %s: %v", task.GetTaskId(), err)
	}

	if !resp.GetSuccess() {
		t.Fatalf("Task publication was not successful: %s", resp.GetError())
	}
}

// PublishTestResult is a helper function to publish a test result
func PublishTestResult(t *testing.T, client pb.EventBusClient, result *pb.TaskResult) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.PublishTaskResultRequest{Result: result}
	resp, err := client.PublishTaskResult(ctx, req)

	if err != nil {
		t.Fatalf("Failed to publish result for task %s: %v", result.GetTaskId(), err)
	}

	if !resp.GetSuccess() {
		t.Fatalf("Result publication was not successful: %s", resp.GetError())
	}
}

// PublishTestProgress is a helper function to publish test progress
func PublishTestProgress(t *testing.T, client pb.EventBusClient, progress *pb.TaskProgress) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.PublishTaskProgressRequest{Progress: progress}
	resp, err := client.PublishTaskProgress(ctx, req)

	if err != nil {
		t.Fatalf("Failed to publish progress for task %s: %v", progress.GetTaskId(), err)
	}

	if !resp.GetSuccess() {
		t.Fatalf("Progress publication was not successful: %s", resp.GetError())
	}
}
