package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/owulveryck/agenthub/agents/cortex"
	"github.com/owulveryck/agenthub/agents/cortex/llm"
	"github.com/owulveryck/agenthub/agents/cortex/state"
	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
)

const (
	cortexAgentID = "cortex"
)

// AgentHubMessagePublisher adapts the AgentHub client to the MessagePublisher interface
type AgentHubMessagePublisher struct {
	client *agenthub.AgentHubClient
}

func (a *AgentHubMessagePublisher) PublishMessage(ctx context.Context, msg *pb.Message, routing *pb.AgentEventMetadata) error {
	_, err := a.client.Client.PublishMessage(ctx, &pb.PublishMessageRequest{
		Message: msg,
		Routing: routing,
	})
	return err
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Shutting down Cortex...")
		cancel()
	}()

	// Create gRPC configuration for Cortex
	config := agenthub.NewGRPCConfig("cortex")
	config.HealthPort = "8086" // Unique port for Cortex health

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

	// Create state manager (in-memory for POC)
	stateManager := state.NewInMemoryStateManager()

	// Create LLM client (using mock for POC - replace with real LLM client)
	// For production, use a real LLM client (Vertex AI, OpenAI, etc.)
	llmClient := createLLMClient()

	// Create message publisher adapter
	messagePublisher := &AgentHubMessagePublisher{client: client}

	// Create Cortex instance
	cortexInstance := cortex.NewCortex(stateManager, llmClient, messagePublisher)

	client.Logger.InfoContext(ctx, "Cortex initialized",
		"agent_id", cortexAgentID,
		"llm_client", "mock",
		"state_manager", "in-memory",
	)

	// Subscribe to all messages to orchestrate
	go func() {
		stream, err := client.Client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
			AgentId: cortexAgentID,
		})

		if err != nil {
			client.Logger.ErrorContext(ctx, "Failed to subscribe to messages", "error", err)
			return
		}

		for {
			event, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					client.Logger.InfoContext(ctx, "Message stream ended")
					break
				}
				client.Logger.ErrorContext(ctx, "Error receiving message", "error", err)
				break
			}

			// Process message events
			if messageEvent := event.GetMessage(); messageEvent != nil {
				handleMessage(ctx, client, cortexInstance, messageEvent)
			}
		}
	}()

	// Subscribe to agent registrations
	// Note: In a full implementation, we'd have a separate stream for agent cards
	// For now, we'll manually register known agents or use a discovery mechanism

	client.Logger.InfoContext(ctx, "Starting Cortex Orchestrator")
	client.Logger.InfoContext(ctx, "Cortex is ready to orchestrate conversations and tasks")

	// Keep the service running
	select {
	case <-ctx.Done():
		// Context cancelled, exit gracefully
	}

	client.Logger.InfoContext(ctx, "Cortex shutting down")
}

// handleMessage processes incoming messages through Cortex
func handleMessage(ctx context.Context, client *agenthub.AgentHubClient, cortexInstance *cortex.Cortex, message *pb.Message) {
	client.Logger.InfoContext(ctx, "Cortex received message",
		"message_id", message.GetMessageId(),
		"context_id", message.GetContextId(),
		"role", message.GetRole().String(),
		"task_id", message.GetTaskId(),
	)

	// Check if this is an agent card registration
	if message.Metadata != nil {
		if msgType, exists := message.Metadata.Fields["message_type"]; exists {
			if msgType.GetStringValue() == "agent_card" {
				// TODO: Extract agent card from message and register
				client.Logger.InfoContext(ctx, "Agent card registration received (not yet implemented)")
				return
			}
		}
	}

	// Process the message through Cortex
	err := cortexInstance.HandleMessage(ctx, message)
	if err != nil {
		client.Logger.ErrorContext(ctx, "Cortex failed to handle message",
			"error", err,
			"message_id", message.GetMessageId(),
		)
		return
	}

	client.Logger.InfoContext(ctx, "Cortex successfully processed message",
		"message_id", message.GetMessageId(),
		"context_id", message.GetContextId(),
	)
}

// createLLMClient creates the LLM client based on configuration
// For POC, we use a mock that provides simple echo behavior
// In production, replace with real LLM client (Vertex AI, OpenAI, etc.)
func createLLMClient() llm.Client {
	// Check if we should use a real LLM
	if os.Getenv("CORTEX_LLM_MODEL") != "" {
		// TODO: Create real LLM client based on CORTEX_LLM_MODEL
		// For now, fall back to mock
		fmt.Println("Warning: CORTEX_LLM_MODEL set but real LLM client not yet implemented, using mock")
	}

	// Use mock with simple echo behavior for POC
	return llm.NewMockClientWithFunc(llm.SimpleEchoDecider())
}
