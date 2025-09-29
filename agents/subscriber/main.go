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
	agentID = "agent_demo_subscriber"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Create task subscriber
	taskSubscriber := agenthub.NewTaskSubscriber(client, agentID)

	// Register default task handlers
	taskSubscriber.RegisterDefaultHandlers()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		client.Logger.Info("Received shutdown signal")
		cancel()
	}()

	client.Logger.InfoContext(ctx, "Starting subscriber")

	// Start task subscribers in goroutines
	go func() {
		if err := taskSubscriber.SubscribeToTasks(ctx); err != nil {
			client.Logger.ErrorContext(ctx, "Task subscription failed", "error", err)
		}
	}()

	go func() {
		if err := taskSubscriber.SubscribeToTaskResults(ctx); err != nil {
			client.Logger.ErrorContext(ctx, "Task result subscription failed", "error", err)
		}
	}()

	client.Logger.InfoContext(ctx, "Agent started with observability. Listening for events and tasks.")

	// Wait for context cancellation
	<-ctx.Done()
	client.Logger.Info("Subscriber shutdown complete")
}
