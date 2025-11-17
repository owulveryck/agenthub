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
	"github.com/owulveryck/agenthub/agents/cortex/llm/vertexai"
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
	// Publish message - broker will automatically extract trace context from ctx
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

	// Create LLM client (VertexAI or mock)
	llmClient, err := createLLMClient(ctx)
	if err != nil {
		client.Logger.ErrorContext(ctx, "Failed to create LLM client", "error", err)
		panic(fmt.Sprintf("Failed to create LLM client: %v", err))
	}

	// Create message publisher adapter
	messagePublisher := &AgentHubMessagePublisher{client: client}

	// Create Cortex instance
	cortexInstance := cortex.NewCortex(stateManager, llmClient, messagePublisher)

	llmType := "mock"
	if os.Getenv("GCP_PROJECT") != "" && os.Getenv("GCP_PROJECT") != "your-project" {
		llmType = "vertexai"
	}

	client.Logger.InfoContext(ctx, "Cortex initialized",
		"agent_id", cortexAgentID,
		"llm_client", llmType,
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
				// Skip messages from Cortex itself to prevent infinite loops
				// Cortex should only process USER messages and AGENT task results
				if messageEvent.Metadata != nil && messageEvent.Metadata.Fields != nil {
					if fromAgent, exists := messageEvent.Metadata.Fields["from_agent"]; exists {
						if fromAgent.GetStringValue() == cortexAgentID {
							// This is a message Cortex published, ignore it
							continue
						}
					}
				}

				// Extract parent trace context from the event for distributed tracing
				eventCtx := ctx
				if event.GetTraceId() != "" && event.GetSpanId() != "" {
					// Create W3C traceparent header format: version-trace_id-span_id-flags
					headers := map[string]string{
						"traceparent": fmt.Sprintf("00-%s-%s-01", event.GetTraceId(), event.GetSpanId()),
					}
					eventCtx = client.TraceManager.ExtractTraceContext(ctx, headers)
				}

				handleMessage(eventCtx, client, cortexInstance, messageEvent)
			}
		}
	}()

	// Subscribe to agent events (including agent card registrations)
	go func() {
		stream, err := client.Client.SubscribeToAgentEvents(ctx, &pb.SubscribeToAgentEventsRequest{
			AgentId:    cortexAgentID,
			EventTypes: []string{"agent.registered", "agent.updated"},
		})

		if err != nil {
			client.Logger.ErrorContext(ctx, "Failed to subscribe to agent events", "error", err)
			return
		}

		client.Logger.InfoContext(ctx, "Subscribed to agent registration events")

		for {
			event, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					client.Logger.InfoContext(ctx, "Agent event stream ended")
					break
				}
				client.Logger.ErrorContext(ctx, "Error receiving agent event", "error", err)
				break
			}

			// Process agent card events
			if agentCardEvent := event.GetAgentCard(); agentCardEvent != nil {
				handleAgentCardEvent(ctx, client, cortexInstance, agentCardEvent)
			}
		}
	}()

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
	// Start tracing for Cortex message handling
	handlerCtx, handlerSpan := client.TraceManager.StartA2AMessageSpan(
		ctx,
		"cortex.handle_message",
		message.GetMessageId(),
		message.GetRole().String(),
	)
	defer handlerSpan.End()

	// Add comprehensive A2A attributes for message handling
	taskType := ""
	if message.Metadata != nil && message.Metadata.Fields != nil {
		if taskTypeValue, exists := message.Metadata.Fields["task_type"]; exists {
			taskType = taskTypeValue.GetStringValue()
		}
	}
	client.TraceManager.AddA2AMessageAttributes(
		handlerSpan,
		message.GetMessageId(),
		message.GetContextId(),
		message.GetRole().String(),
		taskType,
		len(message.GetContent()),
		message.GetMetadata() != nil,
	)
	client.TraceManager.AddComponentAttribute(handlerSpan, "cortex")

	client.Logger.InfoContext(handlerCtx, "Cortex received message",
		"message_id", message.GetMessageId(),
		"context_id", message.GetContextId(),
		"role", message.GetRole().String(),
		"task_id", message.GetTaskId(),
		"trace_id", handlerSpan.SpanContext().TraceID().String(),
	)

	// Check if this is an agent card registration
	if message.Metadata != nil {
		if msgType, exists := message.Metadata.Fields["message_type"]; exists {
			if msgType.GetStringValue() == "agent_card" {
				// TODO: Extract agent card from message and register
				client.Logger.InfoContext(handlerCtx, "Agent card registration received (not yet implemented)")
				client.TraceManager.AddSpanEvent(handlerSpan, "agent_card_registration_skipped")
				client.TraceManager.SetSpanSuccess(handlerSpan)
				return
			}
		}
	}

	// Process the message through Cortex
	err := cortexInstance.HandleMessage(handlerCtx, client.TraceManager, message)
	if err != nil {
		client.TraceManager.RecordError(handlerSpan, err)
		client.Logger.ErrorContext(handlerCtx, "Cortex failed to handle message",
			"error", err,
			"message_id", message.GetMessageId(),
			"trace_id", handlerSpan.SpanContext().TraceID().String(),
		)
		return
	}

	client.TraceManager.SetSpanSuccess(handlerSpan)
	client.Logger.InfoContext(handlerCtx, "Cortex successfully processed message",
		"message_id", message.GetMessageId(),
		"context_id", message.GetContextId(),
		"trace_id", handlerSpan.SpanContext().TraceID().String(),
	)
}

// handleAgentCardEvent processes agent registration/update events
func handleAgentCardEvent(ctx context.Context, client *agenthub.AgentHubClient, cortexInstance *cortex.Cortex, cardEvent *pb.AgentCardEvent) {
	agentID := cardEvent.GetAgentId()
	agentCard := cardEvent.GetAgentCard()
	eventType := cardEvent.GetEventType()

	client.Logger.InfoContext(ctx, "Received agent card event",
		"agent_id", agentID,
		"event_type", eventType,
		"agent_name", agentCard.GetName(),
		"agent_description", agentCard.GetDescription(),
		"skills_count", len(agentCard.GetSkills()),
	)

	// Register the agent with Cortex
	cortexInstance.RegisterAgent(agentID, agentCard)

	// Log the skills for visibility
	if len(agentCard.GetSkills()) > 0 {
		client.Logger.InfoContext(ctx, "Agent skills registered",
			"agent_id", agentID,
			"skills", func() []string {
				skills := make([]string, 0, len(agentCard.GetSkills()))
				for _, skill := range agentCard.GetSkills() {
					skills = append(skills, fmt.Sprintf("%s: %s", skill.GetName(), skill.GetDescription()))
				}
				return skills
			}(),
		)
	}

	client.Logger.InfoContext(ctx, "Agent registered with Cortex orchestrator",
		"agent_id", agentID,
		"total_agents", len(cortexInstance.GetAvailableAgents()),
	)
}

// createLLMClient creates the LLM client based on configuration
// Uses VertexAI when GCP_PROJECT is set, otherwise falls back to mock
func createLLMClient(ctx context.Context) (llm.Client, error) {
	// Check if VertexAI configuration is available
	gcpProject := os.Getenv("GCP_PROJECT")
	if gcpProject != "" && gcpProject != "your-project" {
		// Create VertexAI client
		config := vertexai.NewConfigFromEnv()
		fmt.Printf("Initializing VertexAI client (project: %s, location: %s, model: %s)\n",
			config.Project, config.Location, config.Model)

		client, err := vertexai.NewClient(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create VertexAI client: %w", err)
		}
		fmt.Println("VertexAI client initialized successfully")
		return client, nil
	}

	// Fall back to mock for development
	fmt.Println("Using mock LLM client (set GCP_PROJECT to use VertexAI)")
	return llm.NewMockClientWithFunc(llm.IntelligentDecider()), nil
}
