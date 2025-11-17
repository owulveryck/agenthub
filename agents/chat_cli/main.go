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

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	cliAgentID = "agent_chat_cli"
)

// ANSI color codes for terminal output
const (
	colorCyan  = "\033[36m" // Cyan color for task results
	colorReset = "\033[0m"  // Reset to default color
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down CLI...")
		cancel()
	}()

	// Create gRPC configuration for CLI
	config := agenthub.NewGRPCConfig("chat_cli")
	config.HealthPort = "8087" // Unique port for CLI health

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

	// Generate session ID for this CLI session
	sessionID := fmt.Sprintf("cli_session_%d", time.Now().Unix())

	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║         Cortex Chat CLI - POC Demo                ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
	fmt.Printf("\nSession ID: %s\n", sessionID)
	fmt.Println("\nType your messages and press Enter.")
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println("Press Ctrl+C to shutdown.\n")

	// Subscribe to response messages
	responseChan := make(chan *pb.Message, 10)

	go func() {
		stream, err := client.Client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
			AgentId: cliAgentID,
		})

		if err != nil {
			client.Logger.ErrorContext(ctx, "Failed to subscribe to messages", "error", err)
			return
		}

		for {
			event, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				client.Logger.ErrorContext(ctx, "Error receiving message", "error", err)
				return
			}

			// Process message events
			if messageEvent := event.GetMessage(); messageEvent != nil {
				// Only show messages for our session (or task results from any session)
				isTaskResult := false
				if messageEvent.Metadata != nil && messageEvent.Metadata.Fields != nil {
					if taskType, exists := messageEvent.Metadata.Fields["task_type"]; exists {
						if taskType.GetStringValue() == "task_result" {
							isTaskResult = true
						}
					}
				}

				if (messageEvent.ContextId == sessionID || isTaskResult) && messageEvent.Role == pb.Role_ROLE_AGENT {
					// Check if this is a chat response or task result
					if messageEvent.Metadata != nil {
						if taskType, exists := messageEvent.Metadata.Fields["task_type"]; exists {
							taskTypeValue := taskType.GetStringValue()
							if taskTypeValue == "chat_response" || taskTypeValue == "task_result" {
								responseChan <- messageEvent
							}
						}
					}
				}
			}
		}
	}()

	// Handle incoming responses in a separate goroutine
	go func() {
		for {
			select {
			case msg := <-responseChan:
				if len(msg.Content) > 0 {
					responseText := msg.Content[0].GetText()

					// Check if this is a task result message
					isTaskResult := false
					if msg.Metadata != nil && msg.Metadata.Fields != nil {
						if taskType, exists := msg.Metadata.Fields["task_type"]; exists {
							if taskType.GetStringValue() == "task_result" {
								isTaskResult = true
							}
						}
					}

					// Display task results in cyan color
					if isTaskResult {
						fmt.Printf("\n%s🤖 [Task Result] %s%s\n\n> ", colorCyan, responseText, colorReset)
					} else {
						fmt.Printf("\n🤖 Cortex: %s\n\n> ", responseText)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Read user input from stdin
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		text := strings.TrimSpace(scanner.Text())

		if text == "" {
			fmt.Print("> ")
			continue
		}

		if text == "exit" || text == "quit" {
			fmt.Println("Goodbye!")
			cancel()
			return
		}

		// Create and send chat request with tracing
		message := &pb.Message{
			MessageId: fmt.Sprintf("cli_msg_%d", time.Now().UnixNano()),
			ContextId: sessionID,
			Role:      pb.Role_ROLE_USER,
			Content: []*pb.Part{
				{Part: &pb.Part_Text{Text: text}},
			},
			Metadata: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"task_type":  structpb.NewStringValue("chat_request"),
					"from_agent": structpb.NewStringValue(cliAgentID),
					"created_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
				},
			},
		}

		// Start tracing for user message publication
		pubCtx, pubSpan := client.TraceManager.StartA2AMessageSpan(
			ctx,
			"chat_cli.publish_message",
			message.MessageId,
			message.Role.String(),
		)
		defer pubSpan.End()

		// Add A2A message attributes
		client.TraceManager.AddA2AMessageAttributes(
			pubSpan,
			message.MessageId,
			message.ContextId,
			message.Role.String(),
			"chat_request",
			len(message.Content),
			message.Metadata != nil,
		)
		client.TraceManager.AddComponentAttribute(pubSpan, "chat_cli")

		_, err := client.Client.PublishMessage(pubCtx, &pb.PublishMessageRequest{
			Message: message,
			Routing: &pb.AgentEventMetadata{
				FromAgentId: cliAgentID,
				ToAgentId:   "cortex", // Direct to Cortex
				EventType:   "a2a.message.chat_request",
				Priority:    pb.Priority_PRIORITY_HIGH,
			},
		})

		if err != nil {
			client.TraceManager.RecordError(pubSpan, err)
			fmt.Printf("Error sending message: %v\n", err)
		} else {
			client.TraceManager.SetSpanSuccess(pubSpan)
		}

		// Wait a moment for response (basic approach for CLI)
		time.Sleep(100 * time.Millisecond)

		// Don't print prompt yet - will be printed after response
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
