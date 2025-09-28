package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// TestConcurrentSubscriptions tests concurrent subscription operations
func TestConcurrentSubscriptions(t *testing.T) {
	server := NewEventBusServer()

	numGoroutines := 50
	subscriptionsPerGoroutine := 10
	var wg sync.WaitGroup

	// Track successful subscriptions
	var successfulSubs int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < subscriptionsPerGoroutine; j++ {
				agentID := fmt.Sprintf("agent-%d-%d", routineID, j)
				subChan := make(chan *pb.TaskMessage, 10)

				// Simulate subscription
				server.taskMu.Lock()
				server.taskSubscribers[agentID] = append(server.taskSubscribers[agentID], subChan)
				server.taskMu.Unlock()

				atomic.AddInt32(&successfulSubs, 1)

				// Simulate cleanup
				server.taskMu.Lock()
				if subs, ok := server.taskSubscribers[agentID]; ok {
					newSubs := []chan *pb.TaskMessage{}
					for _, ch := range subs {
						if ch != subChan {
							newSubs = append(newSubs, ch)
						}
					}
					server.taskSubscribers[agentID] = newSubs
					if len(server.taskSubscribers[agentID]) == 0 {
						delete(server.taskSubscribers, agentID)
					}
				}
				close(subChan)
				server.taskMu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	expectedSubs := int32(numGoroutines * subscriptionsPerGoroutine)
	if successfulSubs != expectedSubs {
		t.Errorf("Expected %d successful subscriptions, got %d", expectedSubs, successfulSubs)
	}

	// Verify all subscriptions were cleaned up
	server.taskMu.RLock()
	remainingAgents := len(server.taskSubscribers)
	server.taskMu.RUnlock()

	if remainingAgents != 0 {
		t.Errorf("Expected 0 remaining agents, got %d", remainingAgents)
	}
}

// TestConcurrentTaskPublishing tests concurrent task publishing
func TestConcurrentTaskPublishing(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Add a subscriber
	agentID := "stress-test-agent"
	subChan := make(chan *pb.TaskMessage, 1000)

	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
	server.taskMu.Unlock()

	numPublishers := 20
	tasksPerPublisher := 25
	var wg sync.WaitGroup
	var publishedCount int32
	var publishErrors int32

	// Concurrent publishing
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()

			for j := 0; j < tasksPerPublisher; j++ {
				task := &pb.TaskMessage{
					TaskId:           fmt.Sprintf("stress-task-%d-%d", publisherID, j),
					TaskType:         "stress-test",
					RequesterAgentId: fmt.Sprintf("publisher-%d", publisherID),
					ResponderAgentId: agentID,
					CreatedAt:        timestamppb.Now(),
				}

				req := &pb.PublishTaskRequest{Task: task}
				_, err := server.PublishTask(ctx, req)
				if err != nil {
					atomic.AddInt32(&publishErrors, 1)
					t.Errorf("Publisher %d task %d failed: %v", publisherID, j, err)
				} else {
					atomic.AddInt32(&publishedCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedTasks := int32(numPublishers * tasksPerPublisher)
	if publishedCount != expectedTasks {
		t.Errorf("Expected %d published tasks, got %d", expectedTasks, publishedCount)
	}

	if publishErrors > 0 {
		t.Errorf("Expected 0 publish errors, got %d", publishErrors)
	}

	// Count received tasks with timeout
	receivedCount := 0
	timeout := time.After(10 * time.Second)

	for receivedCount < int(expectedTasks) {
		select {
		case <-subChan:
			receivedCount++
		case <-timeout:
			t.Errorf("Timeout: Expected %d tasks, received %d", expectedTasks, receivedCount)
			return
		}
	}

	if receivedCount != int(expectedTasks) {
		t.Errorf("Expected %d received tasks, got %d", expectedTasks, receivedCount)
	}
}

// TestRaceConditionTaskRouting tests for race conditions in task routing
func TestRaceConditionTaskRouting(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	numAgents := 10
	agentChannels := make(map[string]chan *pb.TaskMessage)

	// Create agents
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("race-agent-%d", i)
		subChan := make(chan *pb.TaskMessage, 100)
		agentChannels[agentID] = subChan

		server.taskMu.Lock()
		server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
		server.taskMu.Unlock()
	}

	var wg sync.WaitGroup
	numOperations := 100

	// Concurrent operations: add/remove subscribers while publishing tasks
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			// Add new subscriber
			newAgentID := fmt.Sprintf("dynamic-agent-%d", i)
			newSubChan := make(chan *pb.TaskMessage, 10)

			server.taskMu.Lock()
			server.taskSubscribers[newAgentID] = []chan *pb.TaskMessage{newSubChan}
			server.taskMu.Unlock()

			// Brief pause
			time.Sleep(time.Microsecond)

			// Remove subscriber
			server.taskMu.Lock()
			delete(server.taskSubscribers, newAgentID)
			server.taskMu.Unlock()
			close(newSubChan)
		}
	}()

	// Concurrent task publishing
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			task := &pb.TaskMessage{
				TaskId:           fmt.Sprintf("race-task-%d", i),
				TaskType:         "race-test",
				RequesterAgentId: "race-publisher",
				// Broadcast (no specific responder)
				CreatedAt: timestamppb.Now(),
			}

			req := &pb.PublishTaskRequest{Task: task}
			_, err := server.PublishTask(ctx, req)
			if err != nil {
				t.Errorf("Task %d publication failed: %v", i, err)
			}

			time.Sleep(time.Microsecond)
		}
	}()

	// Concurrent targeted task publishing
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			targetAgent := fmt.Sprintf("race-agent-%d", i%numAgents)
			task := &pb.TaskMessage{
				TaskId:           fmt.Sprintf("targeted-race-task-%d", i),
				TaskType:         "targeted-race-test",
				RequesterAgentId: "targeted-race-publisher",
				ResponderAgentId: targetAgent,
				CreatedAt:        timestamppb.Now(),
			}

			req := &pb.PublishTaskRequest{Task: task}
			_, err := server.PublishTask(ctx, req)
			if err != nil {
				t.Errorf("Targeted task %d publication failed: %v", i, err)
			}

			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()

	// Cleanup
	for agentID, subChan := range agentChannels {
		server.taskMu.Lock()
		delete(server.taskSubscribers, agentID)
		server.taskMu.Unlock()
		close(subChan)
	}
}

// TestMemoryLeakPrevention tests that channels and goroutines are properly cleaned up
func TestMemoryLeakPrevention(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	initialGoroutines := runtime.NumGoroutine()

	numCycles := 100
	agentsPerCycle := 10

	for cycle := 0; cycle < numCycles; cycle++ {
		var cycleChannels []chan *pb.TaskMessage

		// Create agents
		for i := 0; i < agentsPerCycle; i++ {
			agentID := fmt.Sprintf("leak-test-agent-%d-%d", cycle, i)
			subChan := make(chan *pb.TaskMessage, 10)
			cycleChannels = append(cycleChannels, subChan)

			server.taskMu.Lock()
			server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
			server.taskMu.Unlock()

			// Publish some tasks
			for j := 0; j < 5; j++ {
				task := &pb.TaskMessage{
					TaskId:           fmt.Sprintf("leak-task-%d-%d-%d", cycle, i, j),
					TaskType:         "leak-test",
					RequesterAgentId: "leak-publisher",
					ResponderAgentId: agentID,
					CreatedAt:        timestamppb.Now(),
				}

				server.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
			}
		}

		// Clean up all agents from this cycle
		for i := 0; i < agentsPerCycle; i++ {
			agentID := fmt.Sprintf("leak-test-agent-%d-%d", cycle, i)
			subChan := cycleChannels[i]

			server.taskMu.Lock()
			delete(server.taskSubscribers, agentID)
			server.taskMu.Unlock()
			close(subChan)

			// Drain channel
			for len(subChan) > 0 {
				<-subChan
			}
		}

		// Force garbage collection
		if cycle%10 == 0 {
			runtime.GC()
		}
	}

	// Final cleanup
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow goroutines to finish

	finalGoroutines := runtime.NumGoroutine()

	// Allow some tolerance for test framework goroutines
	goroutineDiff := finalGoroutines - initialGoroutines
	if goroutineDiff > 5 {
		t.Errorf("Potential goroutine leak: started with %d, ended with %d (diff: %d)",
			initialGoroutines, finalGoroutines, goroutineDiff)
	}

	// Verify no subscribers remain
	server.taskMu.RLock()
	remainingSubscribers := len(server.taskSubscribers)
	server.taskMu.RUnlock()

	if remainingSubscribers != 0 {
		t.Errorf("Memory leak: %d subscribers not cleaned up", remainingSubscribers)
	}
}

// TestConcurrentResultAndProgressPublishing tests concurrent result and progress publishing
func TestConcurrentResultAndProgressPublishing(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Add result and progress subscribers
	requesterID := "result-progress-requester"
	resultChan := make(chan *pb.TaskResult, 1000)
	progressChan := make(chan *pb.TaskProgress, 1000)

	server.taskMu.Lock()
	server.taskResultSubscribers[requesterID] = []chan *pb.TaskResult{resultChan}
	server.taskProgressSubscribers[requesterID] = []chan *pb.TaskProgress{progressChan}
	server.taskMu.Unlock()

	numWorkers := 10
	resultsPerWorker := 20
	progressUpdatesPerResult := 5

	var wg sync.WaitGroup
	var resultCount int32
	var progressCount int32

	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for result := 0; result < resultsPerWorker; result++ {
				taskID := fmt.Sprintf("task-%d-%d", workerID, result)

				// Publish progress updates
				for prog := 0; prog < progressUpdatesPerResult; prog++ {
					progress := &pb.TaskProgress{
						TaskId:             taskID,
						Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
						ProgressPercentage: int32((prog + 1) * 20),
						ExecutorAgentId:    fmt.Sprintf("worker-%d", workerID),
						UpdatedAt:          timestamppb.Now(),
					}

					_, err := server.PublishTaskProgress(ctx, &pb.PublishTaskProgressRequest{Progress: progress})
					if err != nil {
						t.Errorf("Worker %d progress %d failed: %v", workerID, prog, err)
					} else {
						atomic.AddInt32(&progressCount, 1)
					}
				}

				// Publish final result
				taskResult := &pb.TaskResult{
					TaskId:          taskID,
					Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
					ExecutorAgentId: fmt.Sprintf("worker-%d", workerID),
					CompletedAt:     timestamppb.Now(),
				}

				_, err := server.PublishTaskResult(ctx, &pb.PublishTaskResultRequest{Result: taskResult})
				if err != nil {
					t.Errorf("Worker %d result %d failed: %v", workerID, result, err)
				} else {
					atomic.AddInt32(&resultCount, 1)
				}
			}
		}(worker)
	}

	wg.Wait()

	expectedResults := int32(numWorkers * resultsPerWorker)
	expectedProgress := int32(numWorkers * resultsPerWorker * progressUpdatesPerResult)

	if resultCount != expectedResults {
		t.Errorf("Expected %d results, got %d", expectedResults, resultCount)
	}

	if progressCount != expectedProgress {
		t.Errorf("Expected %d progress updates, got %d", expectedProgress, progressCount)
	}

	// Verify messages were received
	receivedResults := 0
	receivedProgress := 0
	timeout := time.After(5 * time.Second)

	done := false
	for !done {
		select {
		case <-resultChan:
			receivedResults++
		case <-progressChan:
			receivedProgress++
		case <-timeout:
			done = true
		default:
			if receivedResults >= int(expectedResults) && receivedProgress >= int(expectedProgress) {
				done = true
			}
			time.Sleep(time.Millisecond)
		}
	}

	if receivedResults != int(expectedResults) {
		t.Errorf("Expected to receive %d results, got %d", expectedResults, receivedResults)
	}

	if receivedProgress != int(expectedProgress) {
		t.Errorf("Expected to receive %d progress updates, got %d", expectedProgress, receivedProgress)
	}
}

// TestDeadlockPrevention tests that the system doesn't deadlock under stress
func TestDeadlockPrevention(t *testing.T) {
	server := NewEventBusServer()
	ctx := context.Background()

	numAgents := 5
	numOperations := 50

	// Create agents with small buffers to increase chance of blocking
	agentChannels := make(map[string]chan *pb.TaskMessage)
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("deadlock-agent-%d", i)
		subChan := make(chan *pb.TaskMessage, 2) // Small buffer
		agentChannels[agentID] = subChan

		server.taskMu.Lock()
		server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
		server.taskMu.Unlock()
	}

	var wg sync.WaitGroup

	// Rapid publishing to potentially fill buffers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			task := &pb.TaskMessage{
				TaskId:           fmt.Sprintf("deadlock-task-%d", i),
				TaskType:         "deadlock-test",
				RequesterAgentId: "deadlock-publisher",
				// Broadcast to all agents
				CreatedAt: timestamppb.Now(),
			}

			server.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
		}
	}()

	// Slow consumers to create backpressure
	for agentID, subChan := range agentChannels {
		wg.Add(1)
		go func(id string, ch chan *pb.TaskMessage) {
			defer wg.Done()
			consumed := 0
			for consumed < numOperations {
				select {
				case <-ch:
					consumed++
					// Simulate slow processing
					time.Sleep(time.Millisecond)
				case <-time.After(10 * time.Second):
					t.Errorf("Agent %s timed out after consuming %d tasks", id, consumed)
					return
				}
			}
		}(agentID, subChan)
	}

	// Use a timeout to detect deadlocks
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(30 * time.Second):
		t.Fatal("Potential deadlock detected - operations did not complete within timeout")
	}

	// Cleanup
	for agentID, subChan := range agentChannels {
		server.taskMu.Lock()
		delete(server.taskSubscribers, agentID)
		server.taskMu.Unlock()
		close(subChan)
	}
}
