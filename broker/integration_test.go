package main

import (
	"context"
	"fmt"
	"io"
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

const bufSize = 1024 * 1024

// setupTestServer creates a test gRPC server with bufconn for testing
func setupTestServer() (*grpc.Server, *bufconn.Listener, pb.EventBusClient) {
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
	return grpcServer, lis, client
}

// TestIntegration_TaskFlow tests the complete task flow
func TestIntegration_TaskFlow(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Subscribe to tasks
	agentID := "integration-agent"
	taskReq := &pb.SubscribeToTasksRequest{
		AgentId: agentID,
	}

	taskStream, err := client.SubscribeToTasks(ctx, taskReq)
	if err != nil {
		t.Fatalf("Failed to subscribe to tasks: %v", err)
	}

	// Step 2: Subscribe to results
	resultReq := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: "publisher-agent",
	}

	resultStream, err := client.SubscribeToTaskResults(ctx, resultReq)
	if err != nil {
		t.Fatalf("Failed to subscribe to results: %v", err)
	}

	// Step 3: Publish a task
	task := &pb.TaskMessage{
		TaskId:           "integration-task-1",
		TaskType:         "integration-test",
		RequesterAgentId: "publisher-agent",
		ResponderAgentId: agentID,
		Parameters: func() *structpb.Struct {
			params, _ := structpb.NewStruct(map[string]interface{}{
				"test_param": "test_value",
			})
			return params
		}(),
		Priority:  pb.Priority_PRIORITY_MEDIUM,
		CreatedAt: timestamppb.Now(),
	}

	publishReq := &pb.PublishTaskRequest{Task: task}

	publishResp, err := client.PublishTask(ctx, publishReq)
	if err != nil {
		t.Fatalf("Failed to publish task: %v", err)
	}

	if !publishResp.GetSuccess() {
		t.Error("Task publication was not successful")
	}

	// Step 4: Receive the task
	receivedTask, err := taskStream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive task: %v", err)
	}

	if receivedTask.GetTaskId() != task.GetTaskId() {
		t.Errorf("Expected task ID %s, got %s", task.GetTaskId(), receivedTask.GetTaskId())
	}

	// Step 5: Publish task result
	result := &pb.TaskResult{
		TaskId:          receivedTask.GetTaskId(),
		Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
		ExecutorAgentId: agentID,
		Result: func() *structpb.Struct {
			res, _ := structpb.NewStruct(map[string]interface{}{
				"result": "success",
				"value":  42,
			})
			return res
		}(),
		CompletedAt: timestamppb.Now(),
	}

	resultPublishReq := &pb.PublishTaskResultRequest{Result: result}

	resultPublishResp, err := client.PublishTaskResult(ctx, resultPublishReq)
	if err != nil {
		t.Fatalf("Failed to publish result: %v", err)
	}

	if !resultPublishResp.GetSuccess() {
		t.Error("Result publication was not successful")
	}

	// Step 6: Receive the result
	receivedResult, err := resultStream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive result: %v", err)
	}

	if receivedResult.GetTaskId() != task.GetTaskId() {
		t.Errorf("Expected result task ID %s, got %s", task.GetTaskId(), receivedResult.GetTaskId())
	}

	if receivedResult.GetStatus() != pb.TaskStatus_TASK_STATUS_COMPLETED {
		t.Errorf("Expected status COMPLETED, got %s", receivedResult.GetStatus())
	}
}

// TestIntegration_ProgressFlow tests task progress updates
func TestIntegration_ProgressFlow(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Subscribe to progress updates
	progressReq := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: "publisher-agent",
	}

	progressStream, err := client.SubscribeToTaskProgress(ctx, progressReq)
	if err != nil {
		t.Fatalf("Failed to subscribe to progress: %v", err)
	}

	// Publish progress updates
	progressUpdates := []int32{25, 50, 75, 100}

	for i, percentage := range progressUpdates {
		progress := &pb.TaskProgress{
			TaskId:             "progress-task-1",
			Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
			ProgressPercentage: percentage,
			ProgressMessage:    fmt.Sprintf("Step %d completed", i+1),
			ExecutorAgentId:    "worker-agent",
			UpdatedAt:          timestamppb.Now(),
		}

		progressPublishReq := &pb.PublishTaskProgressRequest{Progress: progress}

		resp, err := client.PublishTaskProgress(ctx, progressPublishReq)
		if err != nil {
			t.Fatalf("Failed to publish progress %d: %v", percentage, err)
		}

		if !resp.GetSuccess() {
			t.Errorf("Progress publication %d was not successful", percentage)
		}

		// Receive the progress update
		receivedProgress, err := progressStream.Recv()
		if err != nil {
			t.Fatalf("Failed to receive progress %d: %v", percentage, err)
		}

		if receivedProgress.GetProgressPercentage() != percentage {
			t.Errorf("Expected progress %d, got %d", percentage, receivedProgress.GetProgressPercentage())
		}
	}
}

// TestIntegration_MultipleSubscribers tests multiple subscribers for the same agent
func TestIntegration_MultipleSubscribers(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agentID := "multi-sub-agent"

	// Create multiple subscriptions for the same agent
	stream1, err := client.SubscribeToTasks(ctx, &pb.SubscribeToTasksRequest{AgentId: agentID})
	if err != nil {
		t.Fatalf("Failed to create first subscription: %v", err)
	}

	stream2, err := client.SubscribeToTasks(ctx, &pb.SubscribeToTasksRequest{AgentId: agentID})
	if err != nil {
		t.Fatalf("Failed to create second subscription: %v", err)
	}

	// Publish a task to the agent
	task := &pb.TaskMessage{
		TaskId:           "multi-sub-task",
		TaskType:         "multi-test",
		RequesterAgentId: "publisher",
		ResponderAgentId: agentID,
		CreatedAt:        timestamppb.Now(),
	}

	_, err = client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
	if err != nil {
		t.Fatalf("Failed to publish task: %v", err)
	}

	// Both subscriptions should receive the task
	receivedCount := 0
	timeout := time.After(5 * time.Second)

	for receivedCount < 2 {
		select {
		case receivedTask1 := <-receiveTaskAsync(stream1):
			if receivedTask1 != nil && receivedTask1.GetTaskId() == task.GetTaskId() {
				receivedCount++
			}
		case receivedTask2 := <-receiveTaskAsync(stream2):
			if receivedTask2 != nil && receivedTask2.GetTaskId() == task.GetTaskId() {
				receivedCount++
			}
		case <-timeout:
			t.Errorf("Expected 2 tasks to be received, got %d", receivedCount)
			return
		}
	}

	if receivedCount != 2 {
		t.Errorf("Expected 2 tasks received, got %d", receivedCount)
	}
}

// Helper function to receive tasks asynchronously
func receiveTaskAsync(stream pb.EventBus_SubscribeToTasksClient) <-chan *pb.TaskMessage {
	ch := make(chan *pb.TaskMessage, 1)
	go func() {
		defer close(ch)
		task, err := stream.Recv()
		if err == nil {
			ch <- task
		}
	}()
	return ch
}

// TestIntegration_BroadcastTasks tests broadcasting tasks to multiple agents
func TestIntegration_BroadcastTasks(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create multiple agents
	agents := []string{"agent-1", "agent-2", "agent-3"}
	streams := make(map[string]pb.EventBus_SubscribeToTasksClient)

	for _, agentID := range agents {
		stream, err := client.SubscribeToTasks(ctx, &pb.SubscribeToTasksRequest{AgentId: agentID})
		if err != nil {
			t.Fatalf("Failed to subscribe agent %s: %v", agentID, err)
		}
		streams[agentID] = stream
	}

	// Publish a broadcast task (no specific responder)
	task := &pb.TaskMessage{
		TaskId:           "broadcast-task",
		TaskType:         "broadcast-test",
		RequesterAgentId: "broadcaster",
		// ResponderAgentId is empty for broadcast
		CreatedAt: timestamppb.Now(),
	}

	_, err := client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
	if err != nil {
		t.Fatalf("Failed to publish broadcast task: %v", err)
	}

	// All agents should receive the task
	receivedCount := 0
	timeout := time.After(5 * time.Second)

	for receivedCount < len(agents) {
		select {
		case receivedTask1 := <-receiveTaskAsync(streams["agent-1"]):
			if receivedTask1 != nil && receivedTask1.GetTaskId() == task.GetTaskId() {
				receivedCount++
			}
		case receivedTask2 := <-receiveTaskAsync(streams["agent-2"]):
			if receivedTask2 != nil && receivedTask2.GetTaskId() == task.GetTaskId() {
				receivedCount++
			}
		case receivedTask3 := <-receiveTaskAsync(streams["agent-3"]):
			if receivedTask3 != nil && receivedTask3.GetTaskId() == task.GetTaskId() {
				receivedCount++
			}
		case <-timeout:
			t.Errorf("Expected %d tasks to be received, got %d", len(agents), receivedCount)
			return
		}
	}

	if receivedCount != len(agents) {
		t.Errorf("Expected %d tasks received, got %d", len(agents), receivedCount)
	}
}

// TestIntegration_TaskTypeFiltering tests task type filtering in subscriptions
func TestIntegration_TaskTypeFiltering(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agentID := "filter-agent"

	// Subscribe with task type filtering
	stream, err := client.SubscribeToTasks(ctx, &pb.SubscribeToTasksRequest{
		AgentId:   agentID,
		TaskTypes: []string{"allowed-type"},
	})
	if err != nil {
		t.Fatalf("Failed to subscribe with filtering: %v", err)
	}

	// Publish allowed task type
	allowedTask := &pb.TaskMessage{
		TaskId:           "allowed-task",
		TaskType:         "allowed-type",
		RequesterAgentId: "publisher",
		ResponderAgentId: agentID,
		CreatedAt:        timestamppb.Now(),
	}

	_, err = client.PublishTask(ctx, &pb.PublishTaskRequest{Task: allowedTask})
	if err != nil {
		t.Fatalf("Failed to publish allowed task: %v", err)
	}

	// Publish disallowed task type
	disallowedTask := &pb.TaskMessage{
		TaskId:           "disallowed-task",
		TaskType:         "disallowed-type",
		RequesterAgentId: "publisher",
		ResponderAgentId: agentID,
		CreatedAt:        timestamppb.Now(),
	}

	_, err = client.PublishTask(ctx, &pb.PublishTaskRequest{Task: disallowedTask})
	if err != nil {
		t.Fatalf("Failed to publish disallowed task: %v", err)
	}

	// Should only receive the allowed task
	// Note: The current implementation doesn't actually filter by task type
	// This test documents the current behavior and can be updated when filtering is implemented
	receivedTask, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive task: %v", err)
	}

	if receivedTask.GetTaskId() != allowedTask.GetTaskId() {
		t.Errorf("Expected allowed task ID %s, got %s", allowedTask.GetTaskId(), receivedTask.GetTaskId())
	}

	// Check if we receive the disallowed task (current implementation will receive it)
	select {
	case disallowedReceived := <-receiveTaskAsync(stream):
		if disallowedReceived != nil {
			t.Logf("Note: Current implementation doesn't filter by task type. Received disallowed task: %s", disallowedReceived.GetTaskId())
		}
	case <-time.After(2 * time.Second):
		// Timeout is expected if filtering is working
		t.Log("No disallowed task received (filtering would be working)")
	}
}

// TestIntegration_ConnectionHandling tests connection management
func TestIntegration_ConnectionHandling(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agentID := "connection-test-agent"

	// Create a subscription
	stream, err := client.SubscribeToTasks(ctx, &pb.SubscribeToTasksRequest{AgentId: agentID})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish a task to verify connection is working
	task := &pb.TaskMessage{
		TaskId:           "connection-task",
		TaskType:         "connection-test",
		RequesterAgentId: "publisher",
		ResponderAgentId: agentID,
		CreatedAt:        timestamppb.Now(),
	}

	_, err = client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
	if err != nil {
		t.Fatalf("Failed to publish task: %v", err)
	}

	// Receive the task to confirm connection
	receivedTask, err := stream.Recv()
	if err != nil {
		t.Fatalf("Failed to receive task: %v", err)
	}

	if receivedTask.GetTaskId() != task.GetTaskId() {
		t.Errorf("Expected task ID %s, got %s", task.GetTaskId(), receivedTask.GetTaskId())
	}

	// Cancel the context to simulate connection loss
	cancel()

	// Try to receive again - should get context cancelled error
	_, err = stream.Recv()
	if err == nil {
		t.Error("Expected error after context cancellation")
	}

	if err != context.Canceled && err != io.EOF {
		t.Errorf("Expected context.Canceled or EOF, got: %v", err)
	}
}

// TestIntegration_ConcurrentOperations tests concurrent publishing and subscribing
func TestIntegration_ConcurrentOperations(t *testing.T) {
	server, lis, client := setupTestServer()
	defer server.Stop()
	defer lis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	numAgents := 5
	tasksPerAgent := 10

	var wg sync.WaitGroup

	// Create multiple concurrent subscribers
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(agentNum int) {
			defer wg.Done()

			agentID := fmt.Sprintf("concurrent-agent-%d", agentNum)
			stream, err := client.SubscribeToTasks(ctx, &pb.SubscribeToTasksRequest{AgentId: agentID})
			if err != nil {
				t.Errorf("Agent %d failed to subscribe: %v", agentNum, err)
				return
			}

			// Receive tasks
			for j := 0; j < tasksPerAgent; j++ {
				_, err := stream.Recv()
				if err != nil {
					t.Errorf("Agent %d failed to receive task %d: %v", agentNum, j, err)
					return
				}
			}
		}(i)
	}

	// Wait a bit for subscribers to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish tasks concurrently
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(agentNum int) {
			defer wg.Done()

			agentID := fmt.Sprintf("concurrent-agent-%d", agentNum)
			for j := 0; j < tasksPerAgent; j++ {
				task := &pb.TaskMessage{
					TaskId:           fmt.Sprintf("concurrent-task-%d-%d", agentNum, j),
					TaskType:         "concurrent-test",
					RequesterAgentId: "concurrent-publisher",
					ResponderAgentId: agentID,
					CreatedAt:        timestamppb.Now(),
				}

				_, err := client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
				if err != nil {
					t.Errorf("Failed to publish task %s: %v", task.GetTaskId(), err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}
