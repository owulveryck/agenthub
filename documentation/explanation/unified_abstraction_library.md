# The Unified Abstraction Library

## Overview

The AgentHub Unified Abstraction Library (`internal/agenthub/`) is a comprehensive set of abstractions that dramatically simplifies the development of agents and brokers while providing built-in observability, environment-based configuration, and automatic correlation tracking.

## Key Benefits

### Before and After Comparison

**Before (Legacy approach):**
- Previous implementation: 380+ lines of boilerplate
- Manual OpenTelemetry setup in every component
- Duplicate configuration handling across components
- Manual correlation ID management
- Separate observability and non-observability variants

**After (Unified abstractions):**
- `broker/main.go`: 29 lines using unified abstractions
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

### 2. Task Management Abstractions (`metadata.go`)

#### TaskPublisher
Simplifies task publishing with:
- Automatic correlation ID generation
- Built-in observability tracing
- Structured error handling
- Metrics collection

```go
taskPublisher := &agenthub.TaskPublisher{
    Client:         client.Client,
    TraceManager:   client.TraceManager,
    MetricsManager: client.MetricsManager,
    Logger:         client.Logger,
    ComponentName:  "publisher",
}

err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
    TaskType: "greeting",
    Parameters: map[string]interface{}{"name": "Claude"},
    RequesterAgentID: "publisher_id",
    ResponderAgentID: "subscriber_id",
    Priority: pb.Priority_PRIORITY_MEDIUM,
})
```

#### TaskProcessor
Provides full observability for task processing:
- Automatic trace propagation
- Rich span annotations
- Performance metrics
- Error tracking

### 3. Subscriber Abstractions (`subscriber.go`)

#### TaskSubscriber
Complete subscriber implementation with:
- Pluggable task handler system
- Built-in default handlers
- Automatic result publishing
- Full observability integration

```go
taskSubscriber := agenthub.NewTaskSubscriber(client, agentID)
taskSubscriber.RegisterDefaultHandlers()

// Custom task handlers
taskSubscriber.RegisterTaskHandler("custom_task", func(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
    // Custom processing logic
    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
})

go taskSubscriber.SubscribeToTasks(ctx)
go taskSubscriber.SubscribeToTaskResults(ctx)
```

### 4. Broker Service (`broker.go`)

Complete EventBus service implementation that handles:
- Task routing and delivery
- Subscription management
- Result distribution
- Full observability

## Environment-Based Configuration

The library uses environment variables for zero-configuration setup:

```bash
# Core AgentHub Settings
export AGENTHUB_BROKER_ADDR=localhost
export AGENTHUB_BROKER_PORT=50051

# Observability Endpoints
export JAEGER_ENDPOINT=127.0.0.1:4317
export OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317

# Health Check Ports
export BROKER_HEALTH_PORT=8080
export PUBLISHER_HEALTH_PORT=8081
export SUBSCRIBER_HEALTH_PORT=8082
```

## Automatic Observability

### Distributed Tracing
- **Automatic instrumentation**: OpenTelemetry gRPC interceptors handle trace propagation
- **Service naming**: Unified "agenthub" service with component differentiation
- **Rich annotations**: Task parameters, results, and timing information

### Metrics Collection
- **Event metrics**: Processing rates, error rates, latencies
- **System metrics**: Health checks, connection status
- **Component metrics**: Per-agent and per-broker measurements

### Health Monitoring
- **Automatic endpoints**: `/health`, `/ready`, `/metrics`
- **Component tracking**: Individual health per service
- **Graceful shutdown**: Proper cleanup and connection management

## Correlation Tracking

### Automatic Correlation IDs
```go
// Automatic generation: task_greeting_1727598123
taskID := fmt.Sprintf("task_%s_%d", req.TaskType, time.Now().Unix())
```

### Trace Propagation
- **W3C Trace Context**: Standard distributed tracing headers
- **Automatic propagation**: gRPC interceptors handle context passing
- **End-to-end visibility**: Publisher → Broker → Subscriber traces

## Migration Guide

### From Legacy to Unified Abstractions

**Before:**
```go
// 50+ lines of observability setup
obs, err := observability.New(ctx, observability.Config{...})
server := grpc.NewServer(grpc.UnaryInterceptor(...))
pb.RegisterEventBusServer(server, &eventBusService{...})
// Manual health checks, metrics, etc.
```

**After:**
```go
// One line broker startup
err := agenthub.StartBroker(ctx)
```

## Best Practices

### 1. Use Environment Configuration
Let the library handle configuration automatically:
```bash
source .envrc  # Load all environment variables
go run broker/main.go
```

### 2. Register Custom Handlers
Extend functionality with custom task handlers:
```go
taskSubscriber.RegisterTaskHandler("my_task", myCustomHandler)
```

### 3. Leverage Built-in Observability
The library provides comprehensive observability by default - no additional setup required.

### 4. Use Structured Logging
The library provides structured loggers with trace correlation:
```go
client.Logger.InfoContext(ctx, "Processing task", "task_id", task.GetTaskId())
```

## Architecture Benefits

### Code Reduction
- **Broker**: 380+ lines → 29 lines (92% reduction)
- **Publisher**: 150+ lines → 50 lines (66% reduction)
- **Subscriber**: 200+ lines → 60 lines (70% reduction)

### Maintainability
- **Single source of truth**: All observability logic centralized
- **Consistent patterns**: Same abstractions across all components
- **Environment-driven**: Configuration externalized

### Developer Experience
- **Zero boilerplate**: Built-in observability and configuration
- **Pluggable architecture**: Easy to extend with custom handlers
- **Automatic setup**: One-line service creation

## Future Extensibility

The abstraction library is designed for extension:
- **Custom task handlers**: Easy to add new task types
- **Custom observability**: Extend metrics and tracing
- **Custom configuration**: Override defaults with environment variables
- **Custom transports**: Extend beyond gRPC if needed

This unified approach provides a solid foundation for building complex multi-agent systems while maintaining simplicity and comprehensive observability.