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
	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
)

const (
	responderAgentID = "agent_chat_responder"
)

var (
	// Environment variables for Vertex AI configuration
	gcpProject  = getEnvOrDefault("GCP_PROJECT", "your-project")
	gcpLocation = getEnvOrDefault("GCP_LOCATION", "us-central1")
	modelName   = getEnvOrDefault("VERTEX_AI_MODEL", "gemini-2.0-flash")
)

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createVertexAIClient creates and returns a Vertex AI client
func createVertexAIClient(ctx context.Context) (*genai.Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  gcpProject,
		Location: gcpLocation,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}
	return client, nil
}

// queryVertexAI sends a message to Vertex AI and returns the response
func queryVertexAI(ctx context.Context, userMessage string) (string, error) {
	client, err := createVertexAIClient(ctx)
	if err != nil {
		return "", err
	}

	chat, err := client.Chats.Create(ctx, modelName, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create chat: %w", err)
	}

	result, err := chat.SendMessage(ctx, genai.Part{Text: userMessage})
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	// Extract the text response from the result
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		part := result.Candidates[0].Content.Parts[0]
		if part.Text != "" {
			return part.Text, nil
		}
	}

	return "I'm sorry, I couldn't generate a response.", nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Shutting down chat responder...")
		cancel()
	}()

	// Create gRPC configuration for responder
	config := agenthub.NewGRPCConfig("chat_responder")
	config.HealthPort = "8084" // Unique port for responder agent health

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

	// Subscribe to messages for ChatCompletionRequest
	go func() {
		stream, err := client.Client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
			AgentId: responderAgentID,
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

			// Check if this is a message event from USER role (A2A spec)
			if messageEvent := event.GetMessage(); messageEvent != nil {
				// A2A compliance: Only process messages with USER role
				if messageEvent.Role == pb.Role_ROLE_USER {
					if messageEvent.Metadata != nil {
						if taskType, exists := messageEvent.Metadata.Fields["task_type"]; exists {
							if taskType.GetStringValue() == "chat_request" {
								// Validate A2A message before processing
								if err := validateA2AMessage(messageEvent); err != nil {
									client.Logger.ErrorContext(ctx, "Invalid A2A message", "error", err)
									continue
								}
								handleChatRequest(ctx, client, messageEvent)
							}
						}
					}
				}
			}
		}
	}()

	client.Logger.InfoContext(ctx, "Starting Chat Responder")
	client.Logger.InfoContext(ctx, "Subscribing to A2A chat_request messages from USER role")

	// Keep the service running
	select {
	case <-ctx.Done():
		// Context cancelled, exit gracefully
	}

	client.Logger.InfoContext(ctx, "Chat Responder shutting down")
}

func handleChatRequest(ctx context.Context, client *agenthub.AgentHubClient, message *pb.Message) {
	// Start tracing for chat request processing
	reqCtx, reqSpan := client.TraceManager.StartA2AMessageSpan(
		ctx,
		"handle_chat_request",
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
		"chat_request",
		len(message.GetContent()),
		message.GetMetadata() != nil,
	)
	client.TraceManager.AddComponentAttribute(reqSpan, "chat_responder")

	client.Logger.InfoContext(reqCtx, "Received A2A chat request",
		"message_id", message.GetMessageId(),
		"context_id", message.GetContextId(),
		"trace_id", reqSpan.SpanContext().TraceID().String(),
	)

	// Extract the user message from the message
	var userMessage string
	if len(message.Content) > 0 {
		userMessage = message.Content[0].GetText()
	}

	client.TraceManager.AddSpanEvent(reqSpan, "extracted_user_message",
		attribute.String("user_message", userMessage),
		attribute.Int("content_parts", len(message.GetContent())),
	)

	client.Logger.InfoContext(reqCtx, "Processing user message",
		"message", userMessage,
		"message_id", message.GetMessageId(),
	)

	// Query Vertex AI for response
	client.TraceManager.AddSpanEvent(reqSpan, "querying_vertex_ai",
		attribute.String("model", modelName),
		attribute.String("project", gcpProject),
		attribute.String("location", gcpLocation),
	)

	aiResponse, err := queryVertexAI(reqCtx, userMessage)
	if err != nil {
		client.TraceManager.RecordError(reqSpan, err)
		client.Logger.ErrorContext(reqCtx, "Failed to query Vertex AI",
			"error", err,
			"message_id", message.GetMessageId(),
		)
		aiResponse = "I'm sorry, I'm having trouble processing your request at the moment."
	}

	client.TraceManager.AddSpanEvent(reqSpan, "vertex_ai_response_received",
		attribute.String("response_length", fmt.Sprintf("%d", len(aiResponse))),
	)

	client.Logger.InfoContext(reqCtx, "Generated AI response",
		"response_length", len(aiResponse),
		"message_id", message.GetMessageId(),
	)

	// Create A2A-compliant response message
	responseMessage := &pb.Message{
		MessageId: fmt.Sprintf("msg_chat_response_%d", time.Now().Unix()),
		ContextId: message.GetContextId(), // A2A spec: Same context for correlation
		Role:      pb.Role_ROLE_AGENT,     // A2A spec: AGENT role for responses
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: aiResponse,
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":           structpb.NewStringValue("chat_response"),
				"responder_agent":     structpb.NewStringValue(responderAgentID),
				"original_message_id": structpb.NewStringValue(message.GetMessageId()),
				"created_at":          structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	// Validate A2A response message
	if err := validateA2AMessage(responseMessage); err != nil {
		client.Logger.ErrorContext(ctx, "Invalid A2A response message", "error", err)
		return
	}

	// Start tracing for response publishing
	pubCtx, pubSpan := client.TraceManager.StartA2AMessageSpan(
		reqCtx,
		"publish_chat_response",
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
		"chat_response",
		len(responseMessage.GetContent()),
		responseMessage.GetMetadata() != nil,
	)
	client.TraceManager.AddComponentAttribute(pubSpan, "chat_responder")

	// Publish A2A response with proper routing
	resp, err := client.Client.PublishMessage(pubCtx, &pb.PublishMessageRequest{
		Message: responseMessage,
		Routing: &pb.AgentEventMetadata{
			FromAgentId: responderAgentID,
			ToAgentId:   "", // Broadcast response for correlation matching
			EventType:   "a2a.message.chat_response",
			Priority:    pb.Priority_PRIORITY_MEDIUM,
		},
	})

	if err != nil {
		client.TraceManager.RecordError(reqSpan, err)
		client.TraceManager.RecordError(pubSpan, err)
		client.Logger.ErrorContext(pubCtx, "Failed to publish A2A chat response",
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
		attribute.String("response_content", aiResponse),
	)

	client.Logger.InfoContext(pubCtx, "Published A2A chat response",
		"message_id", message.GetMessageId(),
		"response_message_id", responseMessage.GetMessageId(),
		"context_id", message.GetContextId(),
		"event_id", resp.GetEventId(),
		"response_length", len(aiResponse),
		"trace_id", pubSpan.SpanContext().TraceID().String(),
	)
}

// validateA2AMessage validates message against A2A protocol requirements
func validateA2AMessage(message *pb.Message) error {
	if message.GetMessageId() == "" {
		return fmt.Errorf("message_id is required")
	}

	if message.GetRole() == pb.Role_ROLE_UNSPECIFIED {
		return fmt.Errorf("role must be specified (USER or AGENT)")
	}

	if len(message.GetContent()) == 0 {
		return fmt.Errorf("message must have at least one content part")
	}

	// Validate each content part
	for i, part := range message.GetContent() {
		if part.GetPart() == nil {
			return fmt.Errorf("content part %d is empty", i)
		}
	}

	return nil
}
