package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
)

const (
	replAgentID = "agent_chat_repl"
	chatAgentID = "agent_chat_responder"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Create gRPC configuration for REPL agent
	config := agenthub.NewGRPCConfig("chat_repl")
	config.HealthPort = "8083" // Unique port for REPL agent health

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

	// Channel to receive chat responses
	responseChan := make(chan *pb.Message, 10)

	// Start subscribing to messages in a goroutine
	go func() {
		// Subscribe to messages for this agent
		stream, err := client.Client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
			AgentId: replAgentID,
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

			// Check if this is a message event from our chat responder
			if messageEvent := event.GetMessage(); messageEvent != nil {
				// Check if this is a response to our request
				if messageEvent.Role == pb.Role_ROLE_AGENT {
					select {
					case responseChan <- messageEvent:
					case <-ctx.Done():
						return
					default:
						// Channel full, skip this message
					}
				}
			}
		}
	}()

	client.Logger.InfoContext(ctx, "Chat REPL started")
	fmt.Println("=== A2A-Compliant Chat REPL ===")
	fmt.Println("Type your messages and press Enter. Type 'quit' to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Print("> ")

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					fmt.Printf("Error reading input: %v\n", err)
				}
				return
			}

			input := strings.TrimSpace(scanner.Text())

			if input == "quit" {
				return
			}

			if input == "" {
				continue
			}

			// Create A2A-compliant context ID
			contextID := fmt.Sprintf("chat_conversation_%d", time.Now().Unix())

			// Create A2A-compliant message
			message := &pb.Message{
				MessageId: fmt.Sprintf("msg_chat_request_%d", time.Now().Unix()),
				ContextId: contextID,
				Role:      pb.Role_ROLE_USER, // A2A spec: USER role for requests
				Content: []*pb.Part{
					{
						Part: &pb.Part_Text{
							Text: input,
						},
					},
				},
				Metadata: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"task_type":       structpb.NewStringValue("chat_request"),
						"requester_agent": structpb.NewStringValue(replAgentID),
						"created_at":      structpb.NewStringValue(time.Now().Format(time.RFC3339)),
					},
				},
			}

			// Validate A2A message
			if err := validateA2AMessage(message); err != nil {
				fmt.Printf("Error: Invalid A2A message: %v\n", err)
				continue
			}

			// Start tracing for message publishing
			pubCtx, pubSpan := client.TraceManager.StartA2AMessageSpan(
				ctx,
				"publish_chat_request",
				message.GetMessageId(),
				message.GetRole().String(),
			)
			defer pubSpan.End()

			// Add comprehensive A2A attributes to publishing span
			client.TraceManager.AddA2AMessageAttributes(
				pubSpan,
				message.GetMessageId(),
				message.GetContextId(),
				message.GetRole().String(),
				"chat_request",
				len(message.GetContent()),
				message.GetMetadata() != nil,
			)
			client.TraceManager.AddComponentAttribute(pubSpan, "chat_repl")

			// Publish A2A message with proper routing
			resp, err := client.Client.PublishMessage(pubCtx, &pb.PublishMessageRequest{
				Message: message,
				Routing: &pb.AgentEventMetadata{
					FromAgentId: replAgentID,
					ToAgentId:   chatAgentID,
					EventType:   "a2a.message.chat_request",
					Priority:    pb.Priority_PRIORITY_MEDIUM,
				},
			})

			if err != nil {
				client.TraceManager.RecordError(pubSpan, err)
				fmt.Printf("Error sending message: %v\n", err)
				continue
			}

			client.TraceManager.SetSpanSuccess(pubSpan)
			client.TraceManager.AddSpanEvent(pubSpan, "message_published",
				attribute.String("event_id", resp.GetEventId()),
			)

			client.Logger.InfoContext(ctx, "Published A2A chat message",
				"message_id", message.GetMessageId(),
				"context_id", contextID,
				"event_id", resp.GetEventId(),
				"trace_id", pubSpan.SpanContext().TraceID().String(),
			)

			// Wait for response with timeout
			fmt.Print("Waiting for response...")
			select {
			case response := <-responseChan:
				// Start tracing for response processing
				respCtx, respSpan := client.TraceManager.StartA2AMessageSpan(
					ctx,
					"process_chat_response",
					response.GetMessageId(),
					response.GetRole().String(),
				)
				defer respSpan.End()

				// Add A2A attributes for response processing
				client.TraceManager.AddA2AMessageAttributes(
					respSpan,
					response.GetMessageId(),
					response.GetContextId(),
					response.GetRole().String(),
					"chat_response",
					len(response.GetContent()),
					response.GetMetadata() != nil,
				)
				client.TraceManager.AddComponentAttribute(respSpan, "chat_repl")

				// Check if this response matches our context
				if response.ContextId == contextID {
					client.TraceManager.AddSpanEvent(respSpan, "context_matched",
						attribute.String("expected_context", contextID),
						attribute.String("received_context", response.ContextId),
					)
					fmt.Print("\r")
					if len(response.Content) > 0 && response.Content[0].GetText() != "" {
						fmt.Printf("< %s\n\n", response.Content[0].GetText())
						client.TraceManager.AddSpanEvent(respSpan, "response_displayed",
							attribute.String("response_text", response.Content[0].GetText()),
						)
					} else {
						fmt.Printf("< [Empty response]\n\n")
						client.TraceManager.AddSpanEvent(respSpan, "empty_response_received")
					}
					client.TraceManager.SetSpanSuccess(respSpan)
					client.Logger.InfoContext(respCtx, "Processed chat response",
						"response_message_id", response.GetMessageId(),
						"context_id", response.GetContextId(),
						"trace_id", respSpan.SpanContext().TraceID().String(),
					)
				} else {
					client.TraceManager.AddSpanEvent(respSpan, "context_mismatch",
						attribute.String("expected_context", contextID),
						attribute.String("received_context", response.ContextId),
					)
					// Put it back if it doesn't match (though this shouldn't happen with proper filtering)
					select {
					case responseChan <- response:
					default:
					}
					fmt.Print(".")
				}
			case <-time.After(30 * time.Second):
				fmt.Print("\r")
				fmt.Printf("< [Timeout - no response received]\n\n")
			case <-ctx.Done():
				return
			}
		}
	}
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
