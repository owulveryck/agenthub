package main

import (
	"context"
	"fmt"
	"time"

	"github.com/owulveryck/agenthub/internal/agenthub"
	pb "github.com/owulveryck/agenthub/internal/grpc"
)

const (
	publisherAgentID = "agent_demo_publisher"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create gRPC configuration for publisher
	config := agenthub.NewGRPCConfig("publisher")
	config.HealthPort = "8081" // Different port for publisher health

	// Create AgentHub client
	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create AgentHub client: %v", err))
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := client.Shutdown(shutdownCtx); err != nil {
			client.Logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
		}
	}()

	// Start the client
	if err := client.Start(ctx); err != nil {
		client.Logger.ErrorContext(ctx, "Failed to start client", "error", err)
		panic(err)
	}

	// Create task publisher
	taskPublisher := &agenthub.TaskPublisher{
		Client:         client.Client,
		TraceManager:   client.TraceManager,
		MetricsManager: client.MetricsManager,
		Logger:         client.Logger,
		ComponentName:  "publisher",
	}

	client.Logger.InfoContext(ctx, "Starting publisher demo")
	client.Logger.InfoContext(ctx, "Testing Agent2Agent Task Publishing via AgentHub with observability")

	// Demo Task 1: Greeting task
	if err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
		TaskType: "greeting",
		Parameters: map[string]interface{}{
			"name": "Claude",
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	}); err != nil {
		panic(fmt.Sprintf("Failed to publish greeting task: %v", err))
	}

	time.Sleep(3 * time.Second)

	// Demo Task 2: Math calculation
	if err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
		TaskType: "math_calculation",
		Parameters: map[string]interface{}{
			"operation": "add",
			"a":         42.0,
			"b":         58.0,
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	}); err != nil {
		panic(fmt.Sprintf("Failed to publish math calculation task: %v", err))
	}

	time.Sleep(2 * time.Second)

	// Demo Task 3: Random number generation
	if err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
		TaskType: "random_number",
		Parameters: map[string]interface{}{
			"seed": 12345,
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	}); err != nil {
		panic(fmt.Sprintf("Failed to publish random number task: %v", err))
	}

	time.Sleep(2 * time.Second)

	// Demo Task 4: Unknown task type (should fail)
	if err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
		TaskType: "unknown_task",
		Parameters: map[string]interface{}{
			"data": "test",
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	}); err != nil {
		client.Logger.InfoContext(ctx, "Expected failure for unknown task type", "error", err)
	}

	client.Logger.InfoContext(ctx, "All tasks published! Check subscriber logs for results")
}
