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

// BenchmarkTaskPublishing benchmarks task publishing performance
func BenchmarkTaskPublishing(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Create a task template
	task := &pb.TaskMessage{
		TaskId:           "benchmark-task",
		TaskType:         "benchmark",
		RequesterAgentId: "benchmark-agent",
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task.TaskId = fmt.Sprintf("benchmark-task-%d", i)
		_, err := server.PublishTask(ctx, req)
		if err != nil {
			b.Fatalf("PublishTask failed: %v", err)
		}
	}
}

// BenchmarkTaskPublishingWithSubscribers benchmarks task publishing with active subscribers
func BenchmarkTaskPublishingWithSubscribers(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	numSubscribers := 10
	agentChannels := make([]chan *pb.TaskMessage, numSubscribers)

	// Set up subscribers
	for i := 0; i < numSubscribers; i++ {
		agentID := fmt.Sprintf("benchmark-agent-%d", i)
		subChan := make(chan *pb.TaskMessage, 100)
		agentChannels[i] = subChan

		server.taskMu.Lock()
		server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
		server.taskMu.Unlock()
	}

	// Start consumers to prevent channel blocking
	var wg sync.WaitGroup
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(ch chan *pb.TaskMessage) {
			defer wg.Done()
			for range ch {
				// Consume messages
			}
		}(agentChannels[i])
	}

	task := &pb.TaskMessage{
		TaskId:           "benchmark-task",
		TaskType:         "benchmark",
		RequesterAgentId: "benchmark-publisher",
		// Broadcast to all subscribers
		CreatedAt: timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task.TaskId = fmt.Sprintf("benchmark-task-%d", i)
		_, err := server.PublishTask(ctx, req)
		if err != nil {
			b.Fatalf("PublishTask failed: %v", err)
		}
	}

	b.StopTimer()

	// Cleanup
	for i := 0; i < numSubscribers; i++ {
		close(agentChannels[i])
	}
	wg.Wait()
}

// BenchmarkDirectTaskRouting benchmarks direct (targeted) task routing
func BenchmarkDirectTaskRouting(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	agentID := "target-agent"
	subChan := make(chan *pb.TaskMessage, 1000)

	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
	server.taskMu.Unlock()

	// Start consumer
	go func() {
		for range subChan {
			// Consume messages
		}
	}()

	task := &pb.TaskMessage{
		TaskId:           "benchmark-task",
		TaskType:         "benchmark",
		RequesterAgentId: "benchmark-publisher",
		ResponderAgentId: agentID, // Direct routing
		CreatedAt:        timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task.TaskId = fmt.Sprintf("benchmark-task-%d", i)
		_, err := server.PublishTask(ctx, req)
		if err != nil {
			b.Fatalf("PublishTask failed: %v", err)
		}
	}

	b.StopTimer()
	close(subChan)
}

// BenchmarkBroadcastRouting benchmarks broadcast routing performance
func BenchmarkBroadcastRouting(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	numAgents := 50
	agentChannels := make([]chan *pb.TaskMessage, numAgents)

	// Set up many subscribers
	for i := 0; i < numAgents; i++ {
		agentID := fmt.Sprintf("broadcast-agent-%d", i)
		subChan := make(chan *pb.TaskMessage, 100)
		agentChannels[i] = subChan

		server.taskMu.Lock()
		server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
		server.taskMu.Unlock()

		// Start consumer
		go func(ch chan *pb.TaskMessage) {
			for range ch {
				// Consume messages
			}
		}(subChan)
	}

	task := &pb.TaskMessage{
		TaskId:           "broadcast-task",
		TaskType:         "broadcast",
		RequesterAgentId: "broadcast-publisher",
		// No ResponderAgentId = broadcast
		CreatedAt: timestamppb.Now(),
	}

	req := &pb.PublishTaskRequest{Task: task}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task.TaskId = fmt.Sprintf("broadcast-task-%d", i)
		_, err := server.PublishTask(ctx, req)
		if err != nil {
			b.Fatalf("PublishTask failed: %v", err)
		}
	}

	b.StopTimer()

	// Cleanup
	for i := 0; i < numAgents; i++ {
		close(agentChannels[i])
	}
}

// BenchmarkResultPublishing benchmarks task result publishing
func BenchmarkResultPublishing(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	result := &pb.TaskResult{
		TaskId:          "benchmark-result",
		Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
		ExecutorAgentId: "benchmark-executor",
		CompletedAt:     timestamppb.Now(),
	}

	req := &pb.PublishTaskResultRequest{Result: result}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result.TaskId = fmt.Sprintf("benchmark-result-%d", i)
		_, err := server.PublishTaskResult(ctx, req)
		if err != nil {
			b.Fatalf("PublishTaskResult failed: %v", err)
		}
	}
}

// BenchmarkProgressPublishing benchmarks task progress publishing
func BenchmarkProgressPublishing(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	progress := &pb.TaskProgress{
		TaskId:             "benchmark-progress",
		Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		ProgressPercentage: 50,
		ExecutorAgentId:    "benchmark-executor",
		UpdatedAt:          timestamppb.Now(),
	}

	req := &pb.PublishTaskProgressRequest{Progress: progress}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		progress.TaskId = fmt.Sprintf("benchmark-progress-%d", i)
		_, err := server.PublishTaskProgress(ctx, req)
		if err != nil {
			b.Fatalf("PublishTaskProgress failed: %v", err)
		}
	}
}

// BenchmarkConcurrentPublishing benchmarks concurrent task publishing
func BenchmarkConcurrentPublishing(b *testing.B) {
	server := NewEventBusServer()
	ctx := context.Background()

	// Set up subscriber
	agentID := "concurrent-agent"
	subChan := make(chan *pb.TaskMessage, 10000)

	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
	server.taskMu.Unlock()

	// Start consumer
	go func() {
		for range subChan {
			// Consume messages
		}
	}()

	numGoroutines := runtime.NumCPU()
	tasksPerGoroutine := b.N / numGoroutines

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < tasksPerGoroutine; j++ {
				task := &pb.TaskMessage{
					TaskId:           fmt.Sprintf("concurrent-task-%d-%d", goroutineID, j),
					TaskType:         "concurrent",
					RequesterAgentId: fmt.Sprintf("publisher-%d", goroutineID),
					ResponderAgentId: agentID,
					CreatedAt:        timestamppb.Now(),
				}

				req := &pb.PublishTaskRequest{Task: task}
				_, err := server.PublishTask(ctx, req)
				if err != nil {
					b.Errorf("PublishTask failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	b.StopTimer()
	close(subChan)
}

// TestThroughputMeasurement measures actual throughput under load
func TestThroughputMeasurement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	server := NewEventBusServer()
	ctx := context.Background()

	// Set up multiple subscribers
	numSubscribers := 10
	for i := 0; i < numSubscribers; i++ {
		agentID := fmt.Sprintf("throughput-agent-%d", i)
		subChan := make(chan *pb.TaskMessage, 1000)

		server.taskMu.Lock()
		server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
		server.taskMu.Unlock()

		// Start consumer
		go func(ch chan *pb.TaskMessage) {
			for range ch {
				// Consume messages
			}
		}(subChan)
	}

	duration := 5 * time.Second
	var taskCount int64

	start := time.Now()
	deadline := start.Add(duration)

	var wg sync.WaitGroup
	numPublishers := runtime.NumCPU()

	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()

			taskNum := 0
			for time.Now().Before(deadline) {
				task := &pb.TaskMessage{
					TaskId:           fmt.Sprintf("throughput-task-%d-%d", publisherID, taskNum),
					TaskType:         "throughput-test",
					RequesterAgentId: fmt.Sprintf("throughput-publisher-%d", publisherID),
					// Broadcast
					CreatedAt: timestamppb.Now(),
				}

				req := &pb.PublishTaskRequest{Task: task}
				_, err := server.PublishTask(ctx, req)
				if err != nil {
					t.Errorf("PublishTask failed: %v", err)
					return
				}

				atomic.AddInt64(&taskCount, 1)
				taskNum++
			}
		}(i)
	}

	wg.Wait()
	actualDuration := time.Since(start)

	throughput := float64(taskCount) / actualDuration.Seconds()
	t.Logf("Throughput: %.2f tasks/second (%d tasks in %v)", throughput, taskCount, actualDuration)

	// Verify reasonable throughput (adjust threshold based on expected performance)
	if throughput < 1000 {
		t.Logf("Warning: Low throughput detected: %.2f tasks/second", throughput)
	}
}

// TestLatencyMeasurement measures task routing latency
func TestLatencyMeasurement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	server := NewEventBusServer()
	ctx := context.Background()

	agentID := "latency-agent"
	subChan := make(chan *pb.TaskMessage, 1000)

	server.taskMu.Lock()
	server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
	server.taskMu.Unlock()

	numSamples := 1000
	latencies := make([]time.Duration, numSamples)

	// Measure latencies
	for i := 0; i < numSamples; i++ {
		task := &pb.TaskMessage{
			TaskId:           fmt.Sprintf("latency-task-%d", i),
			TaskType:         "latency-test",
			RequesterAgentId: "latency-publisher",
			ResponderAgentId: agentID,
			CreatedAt:        timestamppb.Now(),
		}

		req := &pb.PublishTaskRequest{Task: task}

		start := time.Now()
		_, err := server.PublishTask(ctx, req)
		if err != nil {
			t.Fatalf("PublishTask failed: %v", err)
		}

		// Wait for task to be received
		select {
		case <-subChan:
			latencies[i] = time.Since(start)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Task %d not received within timeout", i)
		}
	}

	// Calculate statistics
	var total time.Duration
	min := latencies[0]
	max := latencies[0]

	for _, latency := range latencies {
		total += latency
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
	}

	avg := total / time.Duration(numSamples)

	t.Logf("Latency statistics:")
	t.Logf("  Average: %v", avg)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)

	// Verify reasonable latency (adjust threshold based on expected performance)
	if avg > 10*time.Millisecond {
		t.Logf("Warning: High average latency detected: %v", avg)
	}
}

// TestMemoryUsageUnderLoad tests memory usage during sustained load
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	server := NewEventBusServer()
	ctx := context.Background()

	// Set up subscribers
	numSubscribers := 100
	subscribers := make([]chan *pb.TaskMessage, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		agentID := fmt.Sprintf("memory-agent-%d", i)
		subChan := make(chan *pb.TaskMessage, 100)
		subscribers[i] = subChan

		server.taskMu.Lock()
		server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
		server.taskMu.Unlock()

		// Start consumer
		go func(ch chan *pb.TaskMessage) {
			for range ch {
				// Consume messages
			}
		}(subChan)
	}

	// Generate sustained load
	numTasks := 10000
	for i := 0; i < numTasks; i++ {
		task := &pb.TaskMessage{
			TaskId:           fmt.Sprintf("memory-task-%d", i),
			TaskType:         "memory-test",
			RequesterAgentId: "memory-publisher",
			// Broadcast to all
			CreatedAt: timestamppb.Now(),
		}

		req := &pb.PublishTaskRequest{Task: task}
		_, err := server.PublishTask(ctx, req)
		if err != nil {
			t.Fatalf("PublishTask failed: %v", err)
		}

		// Periodic GC to get accurate measurements
		if i%1000 == 0 {
			runtime.GC()
		}
	}

	// Final measurement
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Cleanup
	for i := 0; i < numSubscribers; i++ {
		close(subscribers[i])
	}

	memoryIncrease := m2.Alloc - m1.Alloc
	t.Logf("Memory usage increase: %d bytes (%.2f MB)", memoryIncrease, float64(memoryIncrease)/1024/1024)

	// Memory increase should be reasonable for the workload
	expectedMaxIncrease := uint64(50 * 1024 * 1024) // 50MB threshold
	if memoryIncrease > expectedMaxIncrease {
		t.Errorf("Memory increase too high: %d bytes (expected < %d)", memoryIncrease, expectedMaxIncrease)
	}
}

// TestResourceCleanup tests that resources are properly cleaned up
func TestResourceCleanup(t *testing.T) {
	var initialGoroutines int
	var initialMemory uint64

	// Measure initial state
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialGoroutines = runtime.NumGoroutine()
	initialMemory = m1.Alloc

	// Create and destroy many subscriber connections
	for cycle := 0; cycle < 10; cycle++ {
		server := NewEventBusServer()
		ctx := context.Background()

		// Create many subscribers
		var channels []chan *pb.TaskMessage
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			agentID := fmt.Sprintf("cleanup-agent-%d-%d", cycle, i)
			subChan := make(chan *pb.TaskMessage, 100) // Larger buffer to prevent blocking
			channels = append(channels, subChan)

			server.taskMu.Lock()
			server.taskSubscribers[agentID] = []chan *pb.TaskMessage{subChan}
			server.taskMu.Unlock()

			// Publish some tasks
			for j := 0; j < 10; j++ {
				wg.Add(1)
				go func(taskID string, responder string) {
					defer wg.Done()
					task := &pb.TaskMessage{
						TaskId:           taskID,
						TaskType:         "cleanup-test",
						RequesterAgentId: "cleanup-publisher",
						ResponderAgentId: responder,
						CreatedAt:        timestamppb.Now(),
					}

					server.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
				}(fmt.Sprintf("cleanup-task-%d-%d-%d", cycle, i, j), agentID)
			}
		}

		// Wait for all publishing to complete
		wg.Wait()

		// Allow time for all goroutines to finish sending
		time.Sleep(100 * time.Millisecond)

		// Clean up all subscribers
		for i, subChan := range channels {
			agentID := fmt.Sprintf("cleanup-agent-%d-%d", cycle, i)
			server.taskMu.Lock()
			delete(server.taskSubscribers, agentID)
			server.taskMu.Unlock()

			// Drain channel before closing
			for len(subChan) > 0 {
				<-subChan
			}
			close(subChan)
		}

		// Force cleanup
		runtime.GC()
	}

	// Final measurement
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow cleanup to complete

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalGoroutines := runtime.NumGoroutine()
	finalMemory := m2.Alloc

	t.Logf("Goroutines: %d -> %d (diff: %d)", initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
	t.Logf("Memory: %d -> %d (diff: %d bytes)", initialMemory, finalMemory, finalMemory-initialMemory)

	// Verify no significant resource leaks
	goroutineDiff := finalGoroutines - initialGoroutines
	if goroutineDiff > 10 {
		t.Errorf("Potential goroutine leak: %d extra goroutines", goroutineDiff)
	}

	// Memory might legitimately increase due to runtime optimizations
	// Handle potential underflow when GC reduces memory usage
	var memoryDiff int64
	if finalMemory > initialMemory {
		memoryDiff = int64(finalMemory - initialMemory)
	} else {
		memoryDiff = -int64(initialMemory - finalMemory)
	}

	// Only check for significant increases (memory decreases are good)
	if memoryDiff > 10*1024*1024 { // 10MB threshold
		t.Errorf("Potential memory leak: %d extra bytes", memoryDiff)
	}
}
