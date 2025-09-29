package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/owulveryck/agenthub/internal/agenthub"
)

const (
	subscriberAgentID = "agent_demo_subscriber"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Create gRPC configuration for subscriber
	config := agenthub.NewGRPCConfig("subscriber")
	config.HealthPort = "8082" // Different port for subscriber health

	// Create AgentHub client
	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		panic("Failed to create AgentHub client: " + err.Error())
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

	// Create A2A task subscriber
	taskSubscriber := agenthub.NewA2ATaskSubscriber(client, subscriberAgentID)

	// Register default handlers for A2A tasks
	taskSubscriber.RegisterDefaultHandlers()

	client.Logger.InfoContext(ctx, "Starting A2A subscriber demo")
	client.Logger.InfoContext(ctx, "Subscribing to A2A tasks and processing them")

	// Start subscribing to A2A tasks
	if err := taskSubscriber.SubscribeToTasks(ctx); err != nil {
		client.Logger.ErrorContext(ctx, "Error in task subscription", "error", err)
	}

	client.Logger.InfoContext(ctx, "A2A subscriber shutting down")
}
