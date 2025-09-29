package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
)

const (
	debugResponderAgentID = "agent_chat_responder"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Shutting down debug responder...")
		cancel()
	}()

	// Create gRPC configuration for responder
	config := agenthub.NewGRPCConfig("debug_responder")
	config.HealthPort = "8085" // Different port for debug responder

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

	fmt.Printf("=== Debug Chat Responder ===\n")
	fmt.Printf("Agent ID: %s\n", debugResponderAgentID)
	fmt.Printf("Broker Address: %s\n", config.BrokerAddr)
	fmt.Printf("Health Port: %s\n", config.HealthPort)
	fmt.Printf("Registering task handler for: ChatCompletionRequest\n")
	fmt.Printf("Starting subscription...\n\n")

	// Create A2A task subscriber
	taskSubscriber := agenthub.NewA2ATaskSubscriber(client, debugResponderAgentID)

	// Register handler for ChatCompletionRequest tasks
	taskSubscriber.RegisterTaskHandler("ChatCompletionRequest", func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
		fmt.Printf("🎉 RECEIVED ChatCompletionRequest!\n")
		fmt.Printf("  Task ID: %s\n", task.GetId())
		fmt.Printf("  Context ID: %s\n", task.GetContextId())

		// Extract the user message from the task
		var userMessage string
		if message != nil && len(message.Content) > 0 {
			userMessage = message.Content[0].GetText()
		} else if task.Status != nil && task.Status.Update != nil && len(task.Status.Update.Content) > 0 {
			userMessage = task.Status.Update.Content[0].GetText()
		}

		fmt.Printf("  User Message: '%s'\n", userMessage)

		client.Logger.InfoContext(ctx, "Received ChatCompletionRequest",
			"task_id", task.GetId(),
			"context_id", task.GetContextId(),
			"message", userMessage,
		)

		// Create correlation ID for response - use the same context ID as the request
		correlationID := task.GetContextId()

		// Create ChatResponse message
		responseMessage := &pb.Message{
			MessageId: fmt.Sprintf("response_%s_%d", task.GetId(), time.Now().Unix()),
			ContextId: correlationID,
			TaskId:    task.GetId(),
			Role:      pb.Role_ROLE_AGENT,
			Content: []*pb.Part{
				{
					Part: &pb.Part_Text{
						Text: "hello",
					},
				},
			},
			Metadata: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"message_type":       structpb.NewStringValue("ChatResponse"),
					"responder_agent_id": structpb.NewStringValue(debugResponderAgentID),
					"original_task_id":   structpb.NewStringValue(task.GetId()),
					"created_at":         structpb.NewStringValue(time.Now().Format(time.RFC3339)),
				},
			},
		}

		// Publish the response message
		_, err := client.Client.PublishMessage(ctx, &pb.PublishMessageRequest{
			Message: responseMessage,
		})

		if err != nil {
			fmt.Printf("❌ Failed to publish ChatResponse: %v\n", err)
			client.Logger.ErrorContext(ctx, "Failed to publish ChatResponse",
				"error", err,
				"task_id", task.GetId(),
			)
			return nil, pb.TaskState_TASK_STATE_FAILED, fmt.Sprintf("failed to publish ChatResponse: %v", err)
		}

		fmt.Printf("✅ Published ChatResponse with 'hello'\n")
		fmt.Printf("  Correlation ID: %s\n\n", correlationID)

		client.Logger.InfoContext(ctx, "Published ChatResponse",
			"task_id", task.GetId(),
			"correlation_id", correlationID,
			"response", "hello",
		)

		// Return success status
		return nil, pb.TaskState_TASK_STATE_COMPLETED, ""
	})

	client.Logger.InfoContext(ctx, "Starting Debug Chat Responder")
	client.Logger.InfoContext(ctx, "Subscribing to ChatCompletionRequest tasks")
	fmt.Printf("✅ Task handler registered\n")
	fmt.Printf("⏳ Waiting for ChatCompletionRequest tasks...\n\n")

	// Start subscribing to tasks
	if err := taskSubscriber.SubscribeToTasks(ctx); err != nil {
		client.Logger.ErrorContext(ctx, "Error in task subscription", "error", err)
	}

	client.Logger.InfoContext(ctx, "Debug Chat Responder shutting down")
	fmt.Printf("👋 Debug Chat Responder shutting down\n")
}