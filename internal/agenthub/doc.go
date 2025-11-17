// Package agenthub provides the core infrastructure for building AgentHub brokers
// and clients with automatic observability, event routing, and A2A protocol support.
//
// # Overview
//
// The agenthub package implements the foundational components for Agent-to-Agent (A2A)
// communication through an event-driven broker architecture. It provides:
//   - gRPC server and client implementations with OpenTelemetry instrumentation
//   - A2A task publishing and subscription mechanisms
//   - Event routing and broker services
//   - Automatic distributed tracing, structured logging, and metrics
//   - Health check endpoints and graceful shutdown
//   - Agent registration and discovery
//
// This package is the infrastructure layer that powers the AgentHub system. Most
// application developers should use the higher-level subagent package instead,
// which wraps this infrastructure in a simpler API.
//
// # Architecture
//
// The package implements a three-tier architecture:
//
//	┌─────────────────────────────────────────────┐
//	│         AgentHub Broker                     │
//	│   (AgentHubService + AgentHubServer)        │
//	│   - Event routing                           │
//	│   - Agent registry                          │
//	│   - Task storage                            │
//	│   - Pub/Sub coordination                    │
//	├─────────────────────────────────────────────┤
//	│         AgentHub Clients                    │
//	│   (AgentHubClient + Task Pub/Sub)           │
//	│   - Connect to broker                       │
//	│   - Publish tasks/messages                  │
//	│   - Subscribe to events                     │
//	│   - Process tasks                           │
//	├─────────────────────────────────────────────┤
//	│         Observability Layer                 │
//	│   - OpenTelemetry tracing                   │
//	│   - Structured logging (slog)               │
//	│   - Prometheus metrics                      │
//	│   - Health checks                           │
//	└─────────────────────────────────────────────┘
//
// # Key Components
//
// ## Broker Components
//
// **AgentHubServer**: The gRPC server infrastructure with observability.
//
//	server, err := agenthub.NewAgentHubServer(config)
//	service := agenthub.NewAgentHubService(server)
//	pb.RegisterAgentHubServer(server.Server, service)
//	server.Start(ctx)
//
// **AgentHubService**: Implements the A2A broker logic including event routing,
// task management, and agent registry.
//
// ## Client Components
//
// **AgentHubClient**: The gRPC client for connecting to the broker with full
// observability integration.
//
//	client, err := agenthub.NewAgentHubClient(config)
//	client.Start(ctx)
//
// **A2ATaskPublisher**: High-level abstraction for publishing A2A-compliant tasks.
//
//	publisher := &agenthub.A2ATaskPublisher{
//	    Client:         client.Client,
//	    TraceManager:   client.TraceManager,
//	    MetricsManager: client.MetricsManager,
//	    Logger:         client.Logger,
//	    ComponentName:  "my_agent",
//	    AgentID:        "agent_123",
//	}
//	task, err := publisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
//	    TaskType:         "translate",
//	    Content:          parts,
//	    RequesterAgentID: "agent_123",
//	    ResponderAgentID: "agent_translator",
//	})
//
// **A2ATaskSubscriber**: High-level abstraction for subscribing to and processing tasks.
//
//	subscriber := agenthub.NewA2ATaskSubscriber(client, "my_agent")
//	subscriber.RegisterTaskHandler("task_type", handlerFunc)
//	subscriber.SubscribeToTasks(ctx)
//
// # Configuration
//
// **GRPCConfig** specifies connection parameters:
//
//	config := agenthub.NewGRPCConfig("my_component")
//
// This reads from environment variables:
//   - AGENTHUB_BROKER_ADDR: Broker hostname (default: "localhost")
//   - AGENTHUB_BROKER_PORT: Broker port (default: "50051")
//   - AGENTHUB_GRPC_PORT: Server listen address (default: ":50051")
//   - BROKER_HEALTH_PORT: Health endpoint port (default: "8080")
//
// Custom configuration:
//
//	config := &agenthub.GRPCConfig{
//	    ComponentName: "my_agent",
//	    BrokerAddr:    "broker.example.com:50051",
//	    HealthPort:    "9000",
//	}
//
// # Creating a Broker
//
// The broker is the central coordination point for all agent communication:
//
//	func main() {
//	    ctx := context.Background()
//	    if err := agenthub.StartBroker(ctx); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// The broker provides:
//   - Event routing between agents
//   - Task storage and retrieval
//   - Agent registration and discovery
//   - Message pub/sub
//   - Full observability (traces, logs, metrics)
//
// # Creating a Client
//
// Clients connect to the broker to publish and receive events:
//
//	config := agenthub.NewGRPCConfig("my_agent")
//	client, err := agenthub.NewAgentHubClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := client.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Shutdown(ctx)
//
// The client provides:
//   - gRPC connection to broker
//   - OpenTelemetry tracing
//   - Structured logging
//   - Metrics collection
//   - Health checks
//
// # Publishing Tasks
//
// Use A2ATaskPublisher for high-level task publishing:
//
//	publisher := &agenthub.A2ATaskPublisher{
//	    Client:         client.Client,
//	    TraceManager:   client.TraceManager,
//	    MetricsManager: client.MetricsManager,
//	    Logger:         client.Logger,
//	    ComponentName:  "cortex",
//	    AgentID:        "cortex_orchestrator",
//	}
//
//	task, err := publisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
//	    TaskType:         "language_translation",
//	    Content: []*pb.Part{
//	        {Part: &pb.Part_Text{Text: "Hello, world!"}},
//	    },
//	    RequesterAgentID: "cortex_orchestrator",
//	    ResponderAgentID: "agent_translator",
//	    Priority:         pb.Priority_PRIORITY_MEDIUM,
//	    ContextID:        "conversation_123",
//	})
//
// This automatically:
//   - Generates unique IDs (task ID, message ID)
//   - Creates A2A-compliant task and message structures
//   - Adds observability (tracing, logging, metrics)
//   - Publishes through the broker
//   - Returns the created task
//
// # Subscribing to Tasks
//
// Use A2ATaskSubscriber for high-level task subscription:
//
//	subscriber := agenthub.NewA2ATaskSubscriber(client, "agent_translator")
//
//	// Register handler for specific task type
//	subscriber.RegisterTaskHandler("language_translation", func(
//	    ctx context.Context,
//	    task *pb.Task,
//	    message *pb.Message,
//	) (*pb.Artifact, pb.TaskState, string) {
//	    // Process the task
//	    result := translateText(message.Content[0].GetText())
//
//	    // Create artifact
//	    artifact := &pb.Artifact{
//	        ArtifactId: fmt.Sprintf("trans_%s_%d", task.Id, time.Now().Unix()),
//	        Name:       "translation",
//	        Parts:      []*pb.Part{{Part: &pb.Part_Text{Text: result}}},
//	    }
//
//	    return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
//	})
//
//	// Start subscription (blocks until context cancelled)
//	if err := subscriber.SubscribeToTasks(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// The subscriber automatically:
//   - Subscribes to task events from broker
//   - Routes tasks to registered handlers
//   - Publishes task completion status
//   - Publishes task artifacts
//   - Handles errors gracefully
//
// # Observability
//
// All components include automatic observability:
//
// **Tracing (OpenTelemetry)**:
//   - Spans for all operations (publish, subscribe, route)
//   - A2A-specific attributes (task ID, context ID, agent IDs)
//   - Parent-child span relationships
//   - Error recording
//
// **Logging (structured slog)**:
//   - All events logged with context
//   - Debug, info, and error levels
//   - Correlation IDs included
//
// **Metrics (Prometheus)**:
//   - Events processed counter
//   - Events published counter
//   - Event errors counter
//   - Processing duration histogram
//   - Active connections gauge
//
// **Health Checks**:
//   - Self health check
//   - gRPC connection health check
//   - Exposed on /health endpoint
//   - Metrics on /metrics endpoint
//
// # Event Routing
//
// The broker routes events based on AgentEventMetadata:
//
//	routing := &pb.AgentEventMetadata{
//	    FromAgentId: "agent_a",
//	    ToAgentId:   "agent_b",    // Specific agent
//	    EventType:   "task_message",
//	    Priority:    pb.Priority_PRIORITY_MEDIUM,
//	}
//
// Routing modes:
//   - Direct: ToAgentId specified → routes to that agent
//   - Broadcast: ToAgentId empty → routes to all subscribers
//   - By event type: Different subscriptions receive different event types
//
// # Agent Registration
//
// Agents register with the broker to enable discovery:
//
//	agentCard := &pb.AgentCard{
//	    ProtocolVersion: "0.2.9",
//	    Name:            "agent_translator",
//	    Description:     "Translates text between languages",
//	    Skills: []*pb.AgentSkill{
//	        {
//	            Name:        "Language Translation",
//	            Description: "Translates text between languages",
//	            Tags:        []string{"translation", "language"},
//	        },
//	    },
//	}
//
//	resp, err := client.Client.RegisterAgent(ctx, &pb.RegisterAgentRequest{
//	    AgentCard: agentCard,
//	})
//
// Registration triggers:
//   - AgentCard stored in broker's registry
//   - AgentCardEvent published to all event subscribers
//   - Orchestrators (like Cortex) discover the new agent
//
// # Graceful Shutdown
//
// All components support graceful shutdown:
//
//	// Set up signal handling
//	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
//	defer cancel()
//
//	// Start client
//	client.Start(ctx)
//
//	// Wait for shutdown signal
//	<-ctx.Done()
//
//	// Graceful shutdown with timeout
//	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer shutdownCancel()
//	client.Shutdown(shutdownCtx)
//
// Shutdown process:
//  1. Stop accepting new requests
//  2. Wait for in-flight operations to complete (up to timeout)
//  3. Close gRPC connections
//  4. Shutdown health server
//  5. Flush observability data (traces, metrics)
//  6. Cleanup resources
//
// # Package vs SubAgent Library
//
// **Use this package (agenthub) when**:
//   - Building the broker itself
//   - Need direct control over gRPC layer
//   - Building custom orchestrators
//   - Implementing non-standard agent patterns
//   - Creating infrastructure components
//
// **Use subagent library when**:
//   - Building task-processing agents
//   - Want minimal boilerplate
//   - Need automatic AgentCard creation
//   - Want consistent observability
//   - Building production agents quickly
//
// Most application developers should use the subagent package, which provides
// a simpler API built on top of this infrastructure layer.
//
// # Advanced Usage
//
// ## Custom Event Types
//
// Subscribe to specific event types:
//
//	// Subscribe to agent card events for discovery
//	stream, err := client.Client.SubscribeToAgentEvents(ctx, &pb.SubscribeToAgentEventsRequest{
//	    AgentId: "cortex",
//	})
//
//	for {
//	    event, err := stream.Recv()
//	    if err != nil {
//	        break
//	    }
//	    if agentCard := event.GetAgentCard(); agentCard != nil {
//	        // Process agent registration
//	    }
//	}
//
// ## Task Management
//
// Retrieve and manage tasks:
//
//	// Get task by ID
//	task, err := client.Client.GetTask(ctx, &pb.GetTaskRequest{
//	    TaskId: "task_123",
//	})
//
//	// List tasks by criteria
//	tasks, err := client.Client.ListTasks(ctx, &pb.ListTasksRequest{
//	    AgentId:   "agent_translator",
//	    ContextId: "conversation_123",
//	    States:    []pb.TaskState{pb.TaskState_TASK_STATE_SUBMITTED},
//	})
//
//	// Cancel a task
//	task, err := client.Client.CancelTask(ctx, &pb.CancelTaskRequest{
//	    TaskId: "task_123",
//	    Reason: "User requested cancellation",
//	})
//
// ## Direct Message Publishing
//
// For low-level control, publish messages directly:
//
//	message := &pb.Message{
//	    MessageId: "msg_123",
//	    ContextId: "ctx_123",
//	    TaskId:    "task_123",
//	    Role:      pb.Role_ROLE_USER,
//	    Content: []*pb.Part{
//	        {Part: &pb.Part_Text{Text: "Hello"}},
//	    },
//	}
//
//	resp, err := client.Client.PublishMessage(ctx, &pb.PublishMessageRequest{
//	    Message: message,
//	    Routing: &pb.AgentEventMetadata{
//	        FromAgentId: "user",
//	        ToAgentId:   "agent_translator",
//	        EventType:   "task_message",
//	    },
//	})
//
// # Performance Considerations
//
// The agenthub package is designed for production use:
//   - Asynchronous event delivery (non-blocking)
//   - Buffered channels for event routing (configurable)
//   - Efficient concurrent map access with RWMutex
//   - gRPC connection pooling
//   - Minimal overhead (<1ms per event)
//   - Graceful degradation under load
//
// # Error Handling
//
// The package uses multiple error handling strategies:
//
// **gRPC Status Codes**:
//   - InvalidArgument: Missing required fields
//   - NotFound: Task or agent not found
//   - FailedPrecondition: Operation not allowed in current state
//
// **Logged Errors**:
//   - Routing failures logged but don't fail the request
//   - Subscription errors logged and returned
//
// **Metrics**:
//   - Error counters track failure rates by type
//
// # Examples
//
// See the following for complete examples:
//   - broker/main.go: Full broker implementation
//   - agents/echo_agent/main.go: Agent using subagent library (built on agenthub)
//   - agents/cortex/main.go: Orchestrator using agenthub directly
//
// # Thread Safety
//
// All components are thread-safe:
//   - AgentHubService uses sync.RWMutex for all shared state
//   - Event channels are safely managed
//   - Client can be used from multiple goroutines
//   - Task handlers are called in separate goroutines
package agenthub
