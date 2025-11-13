package main

import (
	"context"
	"fmt"
	"io"
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
		Capabilities: &pb.AgentCapabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []*pb.AgentSkill{
			{
				Id:          "echo",
				Name:        "Echo",
				Description: "Echoes back the input message",
				Tags:        []string{"testing", "echo"},
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

	// Subscribe to messages
	go func() {
		stream, err := client.Client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
			AgentId: echoAgentID,
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
				// Extract parent trace context from the event for distributed tracing
				eventCtx := ctx
				if event.GetTraceId() != "" && event.GetSpanId() != "" {
					// Create W3C traceparent header format: version-trace_id-span_id-flags
					headers := map[string]string{
						"traceparent": fmt.Sprintf("00-%s-%s-01", event.GetTraceId(), event.GetSpanId()),
					}
					eventCtx = client.TraceManager.ExtractTraceContext(ctx, headers)
				}

				// Check if this is an echo request
				if messageEvent.Metadata != nil {
					if taskType, exists := messageEvent.Metadata.Fields["task_type"]; exists {
						if taskType.GetStringValue() == "echo_request" || taskType.GetStringValue() == "echo" {
							handleEchoRequest(eventCtx, client, messageEvent)
						}
					}
				}

				// Also handle messages with TaskId that are directed to us
				if messageEvent.TaskId != "" && messageEvent.Role == pb.Role_ROLE_AGENT {
					// Check if task type is echo
					if messageEvent.Metadata != nil {
						if taskType, exists := messageEvent.Metadata.Fields["task_type"]; exists {
							if taskType.GetStringValue() == "echo" {
								handleEchoRequest(eventCtx, client, messageEvent)
							}
						}
					}
				}
			}
		}
	}()

	client.Logger.InfoContext(ctx, "Starting Echo Agent")
	client.Logger.InfoContext(ctx, "Subscribing to echo_request messages")

	// Keep the service running
	select {
	case <-ctx.Done():
		// Context cancelled, exit gracefully
	}

	client.Logger.InfoContext(ctx, "Echo Agent shutting down")
}

func handleEchoRequest(ctx context.Context, client *agenthub.AgentHubClient, message *pb.Message) {
	// Start tracing for echo request processing
	reqCtx, reqSpan := client.TraceManager.StartA2AMessageSpan(
		ctx,
		"echo_agent.handle_request",
		message.GetMessageId(),
		message.GetRole().String(),
	)
	defer reqSpan.End()

	// Add comprehensive A2A attributes for request processing
	client.TraceManager.AddA2AMessageAttributes(
		reqSpan,
		message.GetMessageId(),
		message.GetContextId(),
		message.GetRole().String(),
		"echo_request",
		len(message.GetContent()),
		message.GetMetadata() != nil,
	)
	client.TraceManager.AddComponentAttribute(reqSpan, "echo_agent")

	client.Logger.InfoContext(reqCtx, "Received echo request",
		"message_id", message.GetMessageId(),
		"context_id", message.GetContextId(),
		"task_id", message.GetTaskId(),
		"trace_id", reqSpan.SpanContext().TraceID().String(),
	)

	// Extract the input message
	var inputText string
	if len(message.Content) > 0 {
		inputText = message.Content[0].GetText()
	}

	client.TraceManager.AddSpanEvent(reqSpan, "extracted_input_message",
		attribute.String("input_text", inputText),
		attribute.Int("content_parts", len(message.GetContent())),
	)

	// Create echo response
	echoText := fmt.Sprintf("Echo: %s", inputText)

	client.TraceManager.AddSpanEvent(reqSpan, "created_echo_response",
		attribute.String("echo_text", echoText),
		attribute.Int("response_length", len(echoText)),
	)

	responseMessage := &pb.Message{
		MessageId: fmt.Sprintf("msg_echo_response_%d", time.Now().Unix()),
		ContextId: message.GetContextId(),
		TaskId:    message.GetTaskId(), // Maintain task correlation
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: echoText,
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":           structpb.NewStringValue("echo_result"),
				"echo_agent":          structpb.NewStringValue(echoAgentID),
				"original_message_id": structpb.NewStringValue(message.GetMessageId()),
				"created_at":          structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	// Start tracing for response publishing
	pubCtx, pubSpan := client.TraceManager.StartA2AMessageSpan(
		reqCtx,
		"echo_agent.publish_response",
		responseMessage.GetMessageId(),
		responseMessage.GetRole().String(),
	)
	defer pubSpan.End()

	// Add A2A attributes for response publishing
	client.TraceManager.AddA2AMessageAttributes(
		pubSpan,
		responseMessage.GetMessageId(),
		responseMessage.GetContextId(),
		responseMessage.GetRole().String(),
		"echo_result",
		len(responseMessage.GetContent()),
		responseMessage.GetMetadata() != nil,
	)
	client.TraceManager.AddComponentAttribute(pubSpan, "echo_agent")

	// Publish the echo response
	resp, err := client.Client.PublishMessage(pubCtx, &pb.PublishMessageRequest{
		Message: responseMessage,
		Routing: &pb.AgentEventMetadata{
			FromAgentId: echoAgentID,
			ToAgentId:   "", // Broadcast for correlation matching
			EventType:   "a2a.message.echo_response",
			Priority:    pb.Priority_PRIORITY_MEDIUM,
		},
	})

	if err != nil {
		client.TraceManager.RecordError(reqSpan, err)
		client.TraceManager.RecordError(pubSpan, err)
		client.Logger.ErrorContext(pubCtx, "Failed to publish echo response",
			"error", err,
			"message_id", message.GetMessageId(),
			"trace_id", pubSpan.SpanContext().TraceID().String(),
		)
		return
	}

	client.TraceManager.SetSpanSuccess(reqSpan)
	client.TraceManager.SetSpanSuccess(pubSpan)
	client.TraceManager.AddSpanEvent(pubSpan, "response_published",
		attribute.String("event_id", resp.GetEventId()),
		attribute.String("echo_text", echoText),
	)

	client.Logger.InfoContext(pubCtx, "Published echo response",
		"message_id", message.GetMessageId(),
		"response_message_id", responseMessage.GetMessageId(),
		"context_id", message.GetContextId(),
		"task_id", message.GetTaskId(),
		"event_id", resp.GetEventId(),
		"echo_text", echoText,
		"trace_id", pubSpan.SpanContext().TraceID().String(),
	)
}
