package main

import (
	"context"
	"testing"
	"time"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// Example test using the test helpers
func TestExampleUsingHelpers(t *testing.T) {
	// Create test server
	testServer := NewTestServer()
	defer testServer.Close()

	// Create subscriber manager
	subManager := NewTestSubscriberManager(testServer.Server())
	defer subManager.RemoveAll()

	// Add subscribers
	subscriber1 := subManager.AddSubscriber("example-agent-1", 10)
	subscriber2 := subManager.AddSubscriber("example-agent-2", 10)

	// Build and publish a task using helpers
	task := NewTaskBuilder().
		WithTaskID("example-task-1").
		WithTaskType("example").
		WithRequester("example-publisher").
		WithResponder("example-agent-1").
		WithParameters(map[string]interface{}{
			"message": "Hello, World!",
			"count":   42,
		}).
		Build()

	PublishTestTask(t, testServer.Client(), task)

	// Verify task was received by correct subscriber
	AssertTaskReceived(t, subscriber1, "example-task-1", 1*time.Second)
	AssertTaskNotReceived(t, subscriber2, "example-task-1", 100*time.Millisecond)

	// Publish broadcast task
	broadcastTask := NewTaskBuilder().
		WithTaskID("broadcast-task").
		WithTaskType("broadcast").
		WithRequester("broadcaster").
		// No responder = broadcast
		Build()

	PublishTestTask(t, testServer.Client(), broadcastTask)

	// Both subscribers should receive broadcast task
	AssertTaskReceived(t, subscriber1, "broadcast-task", 1*time.Second)
	AssertTaskReceived(t, subscriber2, "broadcast-task", 1*time.Second)

	// Publish result using helpers
	result := NewResultBuilder("example-task-1").
		WithStatus(pb.TaskStatus_TASK_STATUS_COMPLETED).
		WithExecutor("example-agent-1").
		WithResult(map[string]interface{}{
			"output": "Task completed successfully",
			"value":  123,
		}).
		Build()

	PublishTestResult(t, testServer.Client(), result)

	// Publish progress using helpers
	progress := NewProgressBuilder("example-task-1").
		WithPercentage(75).
		WithMessage("Almost done").
		WithExecutor("example-agent-1").
		Build()

	PublishTestProgress(t, testServer.Client(), progress)

	// Verify subscriber counts
	AssertSubscriberCount(t, testServer.Server(), "example-agent-1", 1)
	AssertSubscriberCount(t, testServer.Server(), "example-agent-2", 1)
	AssertTotalAgentCount(t, testServer.Server(), 2)
}

// Example of how to test with mock subscribers
func TestExampleWithMockSubscribers(t *testing.T) {
	server := NewEventBusServer()

	// Create mock subscribers directly
	mockSub1 := NewMockSubscriber("mock-agent-1", 10)
	mockSub1.Start()
	defer mockSub1.Stop()

	mockSub2 := NewMockSubscriber("mock-agent-2", 10)
	mockSub2.Start()
	defer mockSub2.Stop()

	// Register subscribers with server
	server.taskMu.Lock()
	server.taskSubscribers["mock-agent-1"] = []chan *pb.TaskMessage{mockSub1.Channel}
	server.taskSubscribers["mock-agent-2"] = []chan *pb.TaskMessage{mockSub2.Channel}
	server.taskMu.Unlock()

	// Create and publish tasks
	task := NewTaskBuilder().
		WithTaskID("mock-test-task").
		WithTaskType("mock-test").
		WithRequester("mock-publisher").
		// Broadcast task
		Build()

	ctx := context.Background()
	_, err := server.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
	if err != nil {
		t.Fatalf("Failed to publish task: %v", err)
	}

	// Wait for both subscribers to receive the task
	if !mockSub1.WaitForTasks(1, 2*time.Second) {
		t.Error("mockSub1 did not receive expected task")
	}

	if !mockSub2.WaitForTasks(1, 2*time.Second) {
		t.Error("mockSub2 did not receive expected task")
	}

	// Verify tasks were received
	tasks1 := mockSub1.GetReceivedTasks()
	if len(tasks1) != 1 {
		t.Errorf("Expected 1 task for mockSub1, got %d", len(tasks1))
	}

	tasks2 := mockSub2.GetReceivedTasks()
	if len(tasks2) != 1 {
		t.Errorf("Expected 1 task for mockSub2, got %d", len(tasks2))
	}

	if len(tasks1) > 0 && tasks1[0].GetTaskId() != "mock-test-task" {
		t.Errorf("Expected task ID 'mock-test-task', got %s", tasks1[0].GetTaskId())
	}
}
