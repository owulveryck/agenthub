---
title: "Unified Abstraction Library API Reference"
weight: 70
description: >
  The AgentHub unified abstraction library provides simplified APIs for building gRPC-based agent communication systems with built-in observability, automatic configuration, and correlation tracking.
---

# Unified Abstraction Library API Reference

The AgentHub unified abstraction library provides simplified APIs for building gRPC-based agent communication systems with built-in observability, automatic configuration, and correlation tracking.

## Package: internal/agenthub

The `internal/agenthub` package contains the core unified abstraction components that dramatically simplify AgentHub development by providing high-level APIs with automatic observability integration.

### Overview

The unified abstraction library reduces agent implementation complexity from 380+ lines to ~29 lines by providing:

- **Automatic gRPC Setup**: One-line server and client creation
- **Built-in Observability**: Integrated OpenTelemetry tracing and metrics
- **Environment-Based Configuration**: Automatic configuration from environment variables
- **Correlation Tracking**: Automatic correlation ID generation and propagation
- **Pluggable Architecture**: Simple task handler registration

## Core Components

### GRPCConfig

Configuration structure for gRPC servers and clients with environment-based initialization.

```go
type GRPCConfig struct {
    ServerAddr    string // gRPC server listen address (e.g., ":50051")
    BrokerAddr    string // Broker connection address (e.g., "localhost:50051")
    HealthPort    string // Health check endpoint port
    ComponentName string // Component identifier for observability
}
```

#### Constructor

```go
func NewGRPCConfig(componentName string) *GRPCConfig
```

Creates a new gRPC configuration with environment variable defaults:

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `AGENTHUB_BROKER_ADDR` | `localhost` | Broker server host |
| `AGENTHUB_BROKER_PORT` | `50051` | Broker gRPC port |
| `AGENTHUB_GRPC_PORT` | `:50051` | Server listen port |
| `BROKER_HEALTH_PORT` | `8080` | Health endpoint port |

**Example:**
```go
config := agenthub.NewGRPCConfig("my-agent")
// Results in BrokerAddr: "localhost:50051" (automatically combined)
```

### AgentHubServer

High-level gRPC server wrapper with integrated observability.

```go
type AgentHubServer struct {
    Server         *grpc.Server                    // Underlying gRPC server
    Listener       net.Listener                    // Network listener
    Observability  *observability.Observability    // OpenTelemetry integration
    TraceManager   *observability.TraceManager     // Distributed tracing
    MetricsManager *observability.MetricsManager   // Metrics collection
    HealthServer   *observability.HealthServer     // Health monitoring
    Logger         *slog.Logger                    // Structured logging
    Config         *GRPCConfig                     // Configuration
}
```

#### Constructor

```go
func NewAgentHubServer(config *GRPCConfig) (*AgentHubServer, error)
```

Creates a complete gRPC server with:
- OpenTelemetry instrumentation
- Health check endpoints
- Metrics collection
- Structured logging with trace correlation

#### Methods

```go
func (s *AgentHubServer) Start(ctx context.Context) error
```
Starts the server with automatic:
- Health endpoint setup (`/health`, `/ready`, `/metrics`)
- Metrics collection goroutine
- gRPC server with observability

```go
func (s *AgentHubServer) Shutdown(ctx context.Context) error
```
Gracefully shuts down all components:
- gRPC server graceful stop
- Health server shutdown
- Observability cleanup

**Example:**
```go
config := agenthub.NewGRPCConfig("broker")
server, err := agenthub.NewAgentHubServer(config)
if err != nil {
    log.Fatal(err)
}

// Register services
eventBusService := agenthub.NewEventBusService(server)
pb.RegisterEventBusServer(server.Server, eventBusService)

// Start server
if err := server.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### AgentHubClient

High-level gRPC client wrapper with integrated observability.

```go
type AgentHubClient struct {
    Client         pb.EventBusClient               // gRPC client
    Connection     *grpc.ClientConn                // Connection
    Observability  *observability.Observability    // OpenTelemetry integration
    TraceManager   *observability.TraceManager     // Distributed tracing
    MetricsManager *observability.MetricsManager   // Metrics collection
    HealthServer   *observability.HealthServer     // Health monitoring
    Logger         *slog.Logger                    // Structured logging
    Config         *GRPCConfig                     // Configuration
}
```

#### Constructor

```go
func NewAgentHubClient(config *GRPCConfig) (*AgentHubClient, error)
```

Creates a complete gRPC client with:
- OpenTelemetry instrumentation
- Connection health monitoring
- Metrics collection
- Automatic retry and timeout handling

#### Methods

```go
func (c *AgentHubClient) Start(ctx context.Context) error
```
Initializes client with health monitoring and metrics collection.

```go
func (c *AgentHubClient) Shutdown(ctx context.Context) error
```
Gracefully closes connection and cleans up resources.

**Example:**
```go
config := agenthub.NewGRPCConfig("publisher")
client, err := agenthub.NewAgentHubClient(config)
if err != nil {
    log.Fatal(err)
}

err = client.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Use client.Client for gRPC calls
```

## Service Abstractions

### EventBusService

Broker service implementation with built-in observability and correlation tracking.

```go
type EventBusService struct {
    Server          *AgentHubServer
    subscriptions   map[string][]Subscription
    resultSubs      map[string][]ResultSubscription
    progressSubs    map[string][]ProgressSubscription
    mu              sync.RWMutex
}
```

#### Constructor

```go
func NewEventBusService(server *AgentHubServer) *EventBusService
```

Creates an EventBus service with automatic:
- Subscription management
- Task routing and correlation
- Observability integration

#### Key Methods

```go
func (s *EventBusService) PublishTask(ctx context.Context, req *pb.PublishTaskRequest) (*pb.PublishResponse, error)
```

Publishes tasks with automatic:
- Input validation
- Correlation ID generation
- Distributed tracing
- Metrics collection

```go
func (s *EventBusService) SubscribeToTasks(req *pb.SubscribeToTasksRequest, stream pb.EventBus_SubscribeToTasksServer) error
```

Manages task subscriptions with:
- Automatic subscription lifecycle
- Context cancellation handling
- Error recovery

### SubscriberAgent

High-level subscriber implementation with pluggable task handlers.

```go
type SubscriberAgent struct {
    client      *AgentHubClient
    agentID     string
    handlers    map[string]TaskHandler
    ctx         context.Context
    cancel      context.CancelFunc
}
```

#### Constructor

```go
func NewSubscriberAgent(client *AgentHubClient, agentID string) *SubscriberAgent
```

#### Task Handler Interface

```go
type TaskHandler interface {
    Handle(ctx context.Context, task *pb.TaskMessage) (*pb.TaskResult, error)
}
```

#### Methods

```go
func (s *SubscriberAgent) RegisterHandler(taskType string, handler TaskHandler)
```

Registers handlers for specific task types with automatic:
- Task routing
- Error handling
- Result publishing

```go
func (s *SubscriberAgent) Start(ctx context.Context) error
```

Starts the subscriber with automatic:
- Task subscription
- Handler dispatch
- Observability integration

**Example:**
```go
type GreetingHandler struct{}

func (h *GreetingHandler) Handle(ctx context.Context, task *pb.TaskMessage) (*pb.TaskResult, error) {
    // Process greeting task
    return result, nil
}

// Register handler
subscriber.RegisterHandler("greeting", &GreetingHandler{})
```

## Utility Functions

### Metadata Operations

```go
func ExtractCorrelationID(ctx context.Context) string
func InjectCorrelationID(ctx context.Context, correlationID string) context.Context
func GenerateCorrelationID() string
```

Automatic correlation ID management for distributed tracing.

### Metrics Helpers

```go
func NewMetricsTicker(ctx context.Context, manager *observability.MetricsManager) *MetricsTicker
```

Automatic metrics collection with configurable intervals.

## Configuration Reference

### Environment Variables

The unified abstraction library uses environment-based configuration:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AGENTHUB_BROKER_ADDR` | string | `localhost` | Broker server hostname |
| `AGENTHUB_BROKER_PORT` | string | `50051` | Broker gRPC port |
| `AGENTHUB_GRPC_PORT` | string | `:50051` | Server listen address |
| `BROKER_HEALTH_PORT` | string | `8080` | Health endpoint port |
| `SERVICE_VERSION` | string | `1.0.0` | Service version for observability |
| `ENVIRONMENT` | string | `development` | Deployment environment |

### Observability Integration

The unified abstraction automatically configures:

- **OpenTelemetry Tracing**: Automatic span creation and context propagation
- **Prometheus Metrics**: 47+ built-in metrics for performance monitoring
- **Health Checks**: Comprehensive health endpoints for service monitoring
- **Structured Logging**: Correlated logging with trace context

## Performance Characteristics

| Metric | Standard gRPC | Unified Abstraction | Overhead |
|--------|---------------|-------------------|----------|
| **Setup Complexity** | 380+ lines | ~29 lines | -92% code |
| **Throughput** | 10,000+ tasks/sec | 9,500+ tasks/sec | -5% |
| **Latency** | Baseline | +10ms for tracing | +10ms |
| **Memory** | Baseline | +50MB per agent | +50MB |
| **CPU** | Baseline | +5% for observability | +5% |

## Migration Guide

### From Standard gRPC

**Before (Standard gRPC):**
```go
// 380+ lines of boilerplate code
lis, err := net.Listen("tcp", ":50051")
server := grpc.NewServer()
// ... extensive setup code
```

**After (Unified Abstraction):**
```go
// 29 lines total
config := agenthub.NewGRPCConfig("my-service")
server, err := agenthub.NewAgentHubServer(config)
service := agenthub.NewEventBusService(server)
pb.RegisterEventBusServer(server.Server, service)
server.Start(ctx)
```

### Observability Benefits

The unified abstraction provides automatic:

1. **Distributed Tracing**: Every request automatically traced
2. **Metrics Collection**: 47+ metrics without configuration
3. **Health Monitoring**: Built-in health and readiness endpoints
4. **Error Correlation**: Automatic error tracking across services
5. **Performance Monitoring**: Latency, throughput, and error rates

## Error Handling

The unified abstraction provides comprehensive error handling:

- **Automatic Retries**: Built-in retry logic for transient failures
- **Circuit Breaking**: Protection against cascading failures
- **Graceful Degradation**: Service continues operating during partial failures
- **Error Correlation**: Distributed error tracking across service boundaries

## Best Practices

### 1. Configuration Management
```go
// Use environment-based configuration
config := agenthub.NewGRPCConfig("my-service")

// Override specific values if needed
config.HealthPort = "8083"
```

### 2. Handler Registration
```go
// Register handlers before starting
subscriber.RegisterHandler("task-type", handler)
subscriber.Start(ctx)
```

### 3. Graceful Shutdown
```go
// Always implement proper shutdown
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}()
```

### 4. Error Handling
```go
// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

result, err := client.Client.PublishTask(ctx, request)
if err != nil {
    // Error is automatically traced and logged
    return fmt.Errorf("failed to publish task: %w", err)
}
```

## See Also

- [Observability Metrics Reference](observability_metrics.md)
- [Health Endpoints Reference](health_endpoints.md)
- [Tracing API Reference](tracing_api.md)
- [Configuration Reference](configuration_reference.md)