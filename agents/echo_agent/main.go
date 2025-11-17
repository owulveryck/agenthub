package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	echoAgentID = "agent_echo"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Shutting down echo agent...")
		cancel()
	}()

	// Create gRPC configuration for echo agent
	config := agenthub.NewGRPCConfig("echo_agent")
	config.HealthPort = "8085" // Unique port for echo agent health

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

	// Register agent with broker
	agentCard := &pb.AgentCard{
		ProtocolVersion: "0.2.9",
		Name:            echoAgentID,
		Description:     "A simple echo agent that repeats back messages for testing purposes",
		Version:         "1.0.0",
		Capabilities: &pb.AgentCapabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []*pb.AgentSkill{
			{
				Id:          "echo",
				Name:        "Echo Messages",
				Description: "Echoes back any text message with an 'Echo: ' prefix for testing and verification",
				Tags:        []string{"testing", "echo", "debug", "verification"},
				Examples: []string{
					"Echo this message",
					"Repeat what I say",
					"Can you echo hello world?",
					"Test the echo functionality",
				},
				InputModes:  []string{"text/plain"},
				OutputModes: []string{"text/plain"},
			},
		},
	}

	_, err = client.Client.RegisterAgent(ctx, &pb.RegisterAgentRequest{
		AgentCard:     agentCard,
		Subscriptions: []string{"echo_request"},
	})

	if err != nil {
		client.Logger.ErrorContext(ctx, "Failed to register agent", "error", err)
		panic(err)
	}

	client.Logger.InfoContext(ctx, "Echo agent registered successfully",
		"agent_id", echoAgentID,
	)

	// Create A2A task subscriber
	taskSubscriber := agenthub.NewA2ATaskSubscriber(client, echoAgentID)

	// Register handler for Echo Messages tasks
	taskSubscriber.RegisterTaskHandler("Echo Messages", func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
		// Start tracing for task processing
		taskCtx, taskSpan := client.TraceManager.StartA2ATaskSpan(
			ctx,
			"echo_agent.process_task",
			task.GetId(),
			"Echo Messages",
		)
		defer taskSpan.End()

		// Add comprehensive A2A task attributes
		client.TraceManager.AddA2ATaskAttributes(
			taskSpan,
			task.GetId(),
			task.GetContextId(),
			"Echo Messages",
			task.GetState().String(),
		)
		client.TraceManager.AddComponentAttribute(taskSpan, "echo_agent")

		client.Logger.InfoContext(taskCtx, "Received Echo Messages task",
			"task_id", task.GetId(),
			"context_id", task.GetContextId(),
			"trace_id", taskSpan.SpanContext().TraceID().String(),
		)

		// Extract the input text from the message
		var inputText string
		if message != nil && len(message.Content) > 0 {
			inputText = message.Content[0].GetText()
		}

		client.TraceManager.AddSpanEvent(taskSpan, "extracted_input",
			attribute.String("input_text", inputText),
			attribute.Int("content_parts", len(message.GetContent())),
		)

		// Create echo response
		echoText := fmt.Sprintf("Echo: %s", inputText)

		client.TraceManager.AddSpanEvent(taskSpan, "created_echo_response",
			attribute.String("echo_text", echoText),
			attribute.Int("response_length", len(echoText)),
		)

		client.Logger.InfoContext(taskCtx, "Processing echo request",
			"task_id", task.GetId(),
			"input", inputText,
			"output", echoText,
			"trace_id", taskSpan.SpanContext().TraceID().String(),
		)

		// Create artifact with the echo response
		artifact := &pb.Artifact{
			Type: pb.ArtifactType_ARTIFACT_TYPE_TEXT,
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"text":       structpb.NewStringValue(echoText),
					"input":      structpb.NewStringValue(inputText),
					"agent_id":   structpb.NewStringValue(echoAgentID),
					"created_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
				},
			},
		}

		// Mark span as successful
		client.TraceManager.SetSpanSuccess(taskSpan)
		client.TraceManager.AddSpanEvent(taskSpan, "task_completed",
			attribute.String("artifact_type", artifact.GetType().String()),
			attribute.String("echo_text", echoText),
		)

		return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
	})

	client.Logger.InfoContext(ctx, "Starting Echo Agent")
	client.Logger.InfoContext(ctx, "Subscribing to Echo Messages tasks")

	// Start task subscription
	if err := taskSubscriber.SubscribeToTasks(ctx); err != nil {
		client.Logger.ErrorContext(ctx, "Failed to subscribe to tasks", "error", err)
		panic(err)
	}

	// Keep the service running
	select {
	case <-ctx.Done():
		// Context cancelled, exit gracefully
	}

	client.Logger.InfoContext(ctx, "Echo Agent shutting down")
}
