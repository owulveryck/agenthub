---
title: "The Unified Abstraction Library"
weight: 70
description: >
  The AgentHub Unified Abstraction Library dramatically simplifies the development of agents and brokers while providing built-in observability, environment-based configuration, and automatic correlation tracking.
---

# The A2A-Compliant Unified Abstraction Library

## Overview

The AgentHub Unified Abstraction Library (`internal/agenthub/`) is a comprehensive set of A2A protocol-compliant abstractions that dramatically simplifies the development of A2A agents and brokers while providing built-in observability, environment-based configuration, and automatic correlation tracking.

## Key Benefits

### Before and After Comparison

**Before (Legacy approach):**
- `broker/main_observability.go`: 380+ lines of boilerplate
- Manual OpenTelemetry setup in every component
- Duplicate configuration handling across components
- Manual correlation ID management
- Separate observability and non-observability variants

**After (Unified abstractions):**
- `broker/main.go`: 29 lines using abstractions
- Automatic OpenTelemetry integration
- Environment-based configuration
- Automatic correlation ID generation and propagation
- Single implementation with built-in observability

## Core Components

### 1. gRPC Abstractions (`grpc.go`)

#### AgentHubServer
Provides a complete gRPC server abstraction with:
- Automatic OpenTelemetry instrumentation
- Environment-based configuration
- Built-in health checks
- Metrics collection
- Graceful shutdown

```go
// Create and start a broker in one line
func StartBroker(ctx context.Context) error {
    config := NewGRPCConfig("broker")
    server, err := NewAgentHubServer(config)
    if err != nil {
        return err
    }
    return server.Start(ctx)
}
```

#### AgentHubClient
Provides a complete gRPC client abstraction with:
- Automatic connection management
- Built-in observability
- Environment-based server discovery
- Health monitoring

```go
// Create a client with built-in observability
config := agenthub.NewGRPCConfig("publisher")
client, err := agenthub.NewAgentHubClient(config)
```

### 2. A2A Task Management Abstractions (`a2a.go`)

#### A2ATaskPublisher
Simplifies A2A task publishing with:
- Automatic A2A message generation
- Built-in observability tracing
- A2A context management
- Structured error handling
- A2A-compliant message formatting

```go
a2aPublisher := &agenthub.A2ATaskPublisher{
    Client:         client.Client,
    TraceManager:   client.TraceManager,
    MetricsManager: client.MetricsManager,
    Logger:         client.Logger,
    ComponentName:  "a2a_publisher",
}

// Create A2A task with structured message content
task := &a2a.Task{
    Id:        "task_greeting_" + uuid.New().String(),
    ContextId: "conversation_123",
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            MessageId: "msg_" + uuid.New().String(),
            Role:      a2a.Role_USER,
            Content: []*a2a.Part{
                {
                    Part: &a2a.Part_Text{
                        Text: "Please process greeting task",
                    },
                },
                {
                    Part: &a2a.Part_Data{
                        Data: &a2a.DataPart{
                            Data:        greetingParams,
                            Description: "Greeting parameters",
                        },
                    },
                },
            },
        },
        Timestamp: timestamppb.Now(),
    },
}

err := a2aPublisher.PublishA2ATask(ctx, task, &pb.AgentEventMetadata{
    FromAgentId: "publisher_id",
    ToAgentId:   "subscriber_id",
    EventType:   "task.submitted",
    Priority:    pb.Priority_PRIORITY_MEDIUM,
})
```

#### A2ATaskProcessor
Provides full observability for A2A task processing:
- Automatic A2A trace propagation
- Rich A2A span annotations with context and message details
- A2A message processing metrics
- A2A conversation context tracking
- Error tracking with A2A-compliant error messages

### 3. A2A Subscriber Abstractions (`a2a_subscriber.go`)

#### A2ATaskSubscriber
Complete A2A subscriber implementation with:
- A2A-compliant task handler system
- Built-in A2A message processors
- Automatic A2A artifact publishing
- Full A2A observability integration
- A2A conversation context awareness

```go
a2aSubscriber := agenthub.NewA2ATaskSubscriber(client, agentID)
a2aSubscriber.RegisterDefaultA2AHandlers()

// Custom A2A task handlers
a2aSubscriber.RegisterA2ATaskHandler("greeting", func(ctx context.Context, event *pb.AgentEvent) error {
    task := event.GetTask()
    if task == nil {
        return fmt.Errorf("no task in event")
    }

    // Process A2A task content
    requestMessage := task.Status.Update
    response := a2aSubscriber.ProcessA2AMessage(ctx, requestMessage)

    // Create completion artifact
    artifact := &a2a.Artifact{
        ArtifactId: "artifact_" + uuid.New().String(),
        Name:       "Greeting Response",
        Description: "Processed greeting task result",
        Parts: []*a2a.Part{
            {
                Part: &a2a.Part_Text{
                    Text: response,
                },
            },
        },
    }

    // Complete task with artifact
    return a2aSubscriber.CompleteA2ATaskWithArtifact(ctx, task, artifact)
})

go a2aSubscriber.SubscribeToA2ATasks(ctx)
go a2aSubscriber.SubscribeToA2AMessages(ctx)
```

### 4. A2A Broker Service (`a2a_broker.go`)

Complete A2A-compliant AgentHub service implementation that handles:
- A2A message routing and delivery
- A2A subscription management with context filtering
- A2A artifact distribution
- A2A task state management
- EDA+A2A hybrid routing
- Full A2A observability

```go
// A2A broker service with unified abstractions
type A2ABrokerService struct {
    // A2A-specific components
    MessageRouter    *A2AMessageRouter
    TaskManager      *A2ATaskManager
    ContextManager   *A2AContextManager
    ArtifactManager  *A2AArtifactManager

    // EDA integration
    EventBus         *EDAEventBus
    SubscriptionMgr  *A2ASubscriptionManager

    // Observability
    TraceManager     *TraceManager
    MetricsManager   *A2AMetricsManager
}
```

## A2A Environment-Based Configuration

The library uses environment variables for zero-configuration A2A setup:

```bash
# Core AgentHub A2A Settings
export AGENTHUB_BROKER_ADDR=localhost
export AGENTHUB_BROKER_PORT=50051

# A2A Protocol Configuration
export AGENTHUB_A2A_PROTOCOL_VERSION=1.0
export AGENTHUB_MESSAGE_BUFFER_SIZE=100
export AGENTHUB_CONTEXT_TIMEOUT=30s
export AGENTHUB_ARTIFACT_MAX_SIZE=10MB

# Observability Endpoints
export JAEGER_ENDPOINT=127.0.0.1:4317
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317

# A2A Health Check Ports
export AGENTHUB_HEALTH_PORT=8080
export A2A_PUBLISHER_HEALTH_PORT=8081
export A2A_SUBSCRIBER_HEALTH_PORT=8082
```

## A2A Automatic Observability

### A2A Distributed Tracing
- **Automatic A2A instrumentation**: OpenTelemetry gRPC interceptors handle A2A trace propagation
- **A2A service naming**: Unified "agenthub" service with A2A component differentiation
- **Rich A2A annotations**: Message content, conversation context, task state transitions, and artifact details
- **A2A context tracking**: Complete conversation thread visibility across multiple agents

### A2A Metrics Collection
- **A2A message metrics**: Message processing rates, A2A error rates, latencies by message type
- **A2A task metrics**: Task completion rates, state transition times, artifact production metrics
- **A2A context metrics**: Conversation context tracking, multi-agent coordination patterns
- **A2A system metrics**: Health checks, A2A connection status, protocol version compatibility
- **A2A component metrics**: Per-agent A2A performance, broker routing efficiency

### Health Monitoring
- **Automatic endpoints**: `/health`, `/ready`, `/metrics`
- **Component tracking**: Individual health per service
- **Graceful shutdown**: Proper cleanup and connection management

## A2A Correlation and Context Tracking

### Automatic A2A Correlation IDs
```go
// A2A task ID generation
taskID := fmt.Sprintf("task_%s_%s", taskDescription, uuid.New().String())

// A2A message ID generation
messageID := fmt.Sprintf("msg_%d_%s", time.Now().Unix(), uuid.New().String())

// A2A context ID for conversation threading
contextID := fmt.Sprintf("ctx_%s_%s", workflowType, uuid.New().String())
```

### A2A Context Propagation
- **A2A conversation threading**: Context IDs link related tasks across agents
- **A2A message history**: Complete audit trail of all messages in a conversation
- **A2A workflow tracking**: End-to-end visibility of multi-agent workflows

### Trace Propagation
- **W3C Trace Context**: Standard distributed tracing headers
- **Automatic propagation**: gRPC interceptors handle context passing
- **End-to-end visibility**: Publisher → Broker → Subscriber traces

## A2A Migration Guide

### From Legacy EventBus to A2A Abstractions

**Before (Legacy EventBus):**
```go
// 50+ lines of observability setup
obs, err := observability.New(ctx, observability.Config{...})
server := grpc.NewServer(grpc.UnaryInterceptor(...))
pb.RegisterEventBusServer(server, &eventBusService{...})

// Manual task message creation
task := &pb.TaskMessage{
    TaskId:   "task_123",
    TaskType: "greeting",
    // ... manual field population
}
```

**After (A2A Abstractions):**
```go
// One line A2A broker startup
err := agenthub.StartA2ABroker(ctx)

// A2A task creation with abstractions
task := a2aPublisher.CreateA2ATask("greeting", greetingContent, "conversation_123")
err := a2aPublisher.PublishA2ATask(ctx, task, routingMetadata)
```

## Best Practices

### 1. Use Environment Configuration
Let the library handle configuration automatically:
```bash
source .envrc  # Load all environment variables
go run broker/main.go
```

### 2. Register Custom A2A Handlers
Extend functionality with custom A2A task handlers:
```go
a2aSubscriber.RegisterA2ATaskHandler("my_task", myCustomA2AHandler)

// A2A handler signature with event and context
func myCustomA2AHandler(ctx context.Context, event *pb.AgentEvent) error {
    task := event.GetTask()
    // Process A2A message content
    return a2aSubscriber.CompleteA2ATaskWithArtifact(ctx, task, resultArtifact)
}
```

### 3. Leverage Built-in Observability
The library provides comprehensive observability by default - no additional setup required.

### 4. Use A2A Structured Logging
The library provides structured loggers with A2A trace correlation:
```go
// A2A-aware logging with context
client.Logger.InfoContext(ctx, "Processing A2A task",
    "task_id", task.GetId(),
    "context_id", task.GetContextId(),
    "message_count", len(task.GetHistory()),
    "current_state", task.GetStatus().GetState().String(),
)
```

## A2A Architecture Benefits

### Code Reduction with A2A Abstractions
- **A2A Broker**: 380+ lines → 29 lines (92% reduction)
- **A2A Publisher**: 150+ lines → 45 lines (70% reduction)
- **A2A Subscriber**: 200+ lines → 55 lines (72% reduction)
- **A2A Message Handling**: Complex manual parsing → automatic Part processing
- **A2A Context Management**: Manual tracking → automatic conversation threading

### A2A Maintainability
- **A2A protocol compliance**: Centralized A2A message handling ensures protocol adherence
- **Consistent A2A patterns**: Same abstractions across all A2A components
- **A2A-aware configuration**: Environment variables tuned for A2A performance
- **A2A context preservation**: Automatic conversation context management

### A2A Developer Experience
- **Zero A2A boilerplate**: Built-in A2A message parsing and artifact handling
- **A2A-native architecture**: Easy to extend with custom A2A message processors
- **Automatic A2A setup**: One-line A2A service creation with protocol compliance
- **A2A debugging**: Rich conversation context and message history for troubleshooting

## A2A Future Extensibility

The A2A abstraction library is designed for A2A protocol extension:
- **Custom A2A Part types**: Easy to add new content types (text, data, files, custom)
- **Custom A2A observability**: Extend A2A metrics and conversation tracing
- **A2A configuration**: Override A2A protocol defaults with environment variables
- **A2A transport options**: Extend beyond gRPC while maintaining A2A compliance
- **A2A protocol evolution**: Built-in version compatibility and migration support

### A2A Protocol Extension Points
```go
// Custom A2A Part type
type CustomPart struct {
    CustomData interface{} `json:"custom_data"`
    Format     string      `json:"format"`
}

// Custom A2A artifact processor
type CustomArtifactProcessor struct {
    SupportedTypes []string
    ProcessFunc    func(ctx context.Context, artifact *a2a.Artifact) error
}

// Custom A2A context manager
type CustomContextManager struct {
    ContextRules map[string]ContextRule
    RouteFunc    func(contextId string, message *a2a.Message) []string
}
```

This A2A-compliant unified approach provides a solid foundation for building complex multi-agent systems with full Agent2Agent protocol support while maintaining simplicity, comprehensive observability, and rich conversation capabilities.