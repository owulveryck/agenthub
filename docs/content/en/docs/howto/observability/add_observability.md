---
title: "How to Add Observability to Your Agent"
weight: 10
description: "Use AgentHub's unified abstractions to automatically get distributed tracing, metrics, and structured logging in your agents."
---

# How to Add Observability to Your Agent

**Goal-oriented guide**: Use AgentHub's unified abstractions to automatically get distributed tracing, metrics, and structured logging in your agents with minimal configuration.

## Prerequisites

- Go 1.24+ installed
- Basic understanding of AgentHub concepts
- 10-15 minutes

## Overview: What You Get Automatically

With AgentHub's unified abstractions, you automatically get:

✅ **Distributed Tracing** - OpenTelemetry traces with correlation IDs
✅ **Comprehensive Metrics** - Performance and health monitoring
✅ **Structured Logging** - JSON logs with trace correlation
✅ **Health Endpoints** - HTTP health checks and metrics endpoints
✅ **Graceful Shutdown** - Clean resource management

## Quick Start: Observable Agent in 5 Minutes

### Step 1: Create Your Agent Using Abstractions

```go
package main

import (
	"context"
	"time"

	"github.com/owulveryck/agenthub/internal/agenthub"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create configuration (observability included automatically)
	config := agenthub.NewGRPCConfig("my-agent")
	config.HealthPort = "8083" // Unique port for your agent

	// Create AgentHub client (observability built-in)
	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		panic("Failed to create AgentHub client: " + err.Error())
	}

	// Automatic graceful shutdown
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := client.Shutdown(shutdownCtx); err != nil {
			client.Logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
		}
	}()

	// Start the client (enables observability)
	if err := client.Start(ctx); err != nil {
		client.Logger.ErrorContext(ctx, "Failed to start client", "error", err)
		panic(err)
	}

	// Your agent logic here...
	client.Logger.Info("My observable agent is running!")

	// Keep running
	select {}
}
```

That's it! Your agent now has full observability.

### Step 2: Configure Environment Variables

Set observability configuration via environment:

```bash
# Tracing configuration
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
export OTEL_SERVICE_NAME="my-agent"
export OTEL_SERVICE_VERSION="1.0.0"

# Health server port
export BROKER_HEALTH_PORT="8083"

# Broker connection
export AGENTHUB_BROKER_ADDR="localhost"
export AGENTHUB_BROKER_PORT="50051"
```

### Step 3: Run Your Observable Agent

```bash
go run main.go
```

**Expected Output:**
```
time=2025-09-29T10:00:00.000Z level=INFO msg="Starting health server" port=8083
time=2025-09-29T10:00:00.000Z level=INFO msg="AgentHub client connected" broker_addr=localhost:50051
time=2025-09-29T10:00:00.000Z level=INFO msg="My observable agent is running!"
```

## Available Observability Features

### Automatic Health Endpoints

Your agent automatically exposes:

- **Health Check**: `http://localhost:8083/health`
- **Metrics**: `http://localhost:8083/metrics` (Prometheus format)
- **Readiness**: `http://localhost:8083/ready`

### Structured Logging

All logs are automatically structured with trace correlation:

```json
{
  "time": "2025-09-29T10:00:00.000Z",
  "level": "INFO",
  "msg": "Task published",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "task_type": "process_document",
  "correlation_id": "req_789"
}
```

### Distributed Tracing

Traces are automatically created for:
- gRPC calls to broker
- Task publishing and subscribing
- Custom operations (when you use the TraceManager)

### Metrics Collection

Automatic metrics include:
- Task processing duration
- Success/failure rates
- gRPC call metrics
- Health check status

## Advanced Usage

### Adding Custom Tracing

Use the built-in TraceManager for custom operations:

```go
// Custom operation with tracing
ctx, span := client.TraceManager.StartPublishSpan(ctx, "my_operation", "document")
defer span.End()

// Add custom attributes
client.TraceManager.AddComponentAttribute(span, "my-component")
span.SetAttributes(attribute.String("document.id", "doc-123"))

// Your operation logic
result, err := doCustomOperation(ctx)
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

### Adding Custom Metrics

Use the MetricsManager for custom metrics:

```go
// Start timing an operation
timer := client.MetricsManager.StartTimer()
defer timer(ctx, "my_operation", "my-component")

// Your operation
processDocument()
```

### Custom Log Fields

Use the structured logger with context:

```go
client.Logger.InfoContext(ctx, "Processing document",
    "document_id", "doc-123",
    "user_id", "user-456",
    "processing_type", "ocr",
)
```

## Publisher Example with Observability

```go
package main

import (
	"context"
	"time"

	"github.com/owulveryck/agenthub/internal/agenthub"
	pb "github.com/owulveryck/agenthub/events/a2a"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Observable client setup
	config := agenthub.NewGRPCConfig("publisher")
	config.HealthPort = "8081"

	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(context.Background())

	if err := client.Start(ctx); err != nil {
		panic(err)
	}

	// Create observable task publisher
	publisher := &agenthub.TaskPublisher{
		Client:         client.Client,
		TraceManager:   client.TraceManager,
		MetricsManager: client.MetricsManager,
		Logger:         client.Logger,
		ComponentName:  "publisher",
	}

	// Publish task with automatic tracing
	data, _ := structpb.NewStruct(map[string]interface{}{
		"message": "Hello, observable world!",
	})

	task := &pb.TaskMessage{
		TaskId:   "task-123",
		TaskType: "greeting",
		Data:     data,
		Priority: pb.Priority_MEDIUM,
	}

	// Automatically traced and metered
	if err := publisher.PublishTask(ctx, task); err != nil {
		client.Logger.ErrorContext(ctx, "Failed to publish task", "error", err)
	} else {
		client.Logger.InfoContext(ctx, "Task published successfully", "task_id", task.TaskId)
	}
}
```

## Subscriber Example with Observability

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/owulveryck/agenthub/internal/agenthub"
	pb "github.com/owulveryck/agenthub/events/a2a"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Observable client setup
	config := agenthub.NewGRPCConfig("subscriber")
	config.HealthPort = "8082"

	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		panic(err)
	}
	defer client.Shutdown(context.Background())

	if err := client.Start(ctx); err != nil {
		panic(err)
	}

	// Create observable task subscriber
	subscriber := agenthub.NewTaskSubscriber(client, "my-subscriber")

	// Register handler with automatic tracing
	subscriber.RegisterHandler("greeting", func(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
		// This is automatically traced and logged
		client.Logger.InfoContext(ctx, "Processing greeting task", "task_id", task.TaskId)

		// Your processing logic
		result, _ := structpb.NewStruct(map[string]interface{}{
			"response": "Hello back!",
		})

		return result, pb.TaskStatus_COMPLETED, ""
	})

	// Start processing with automatic observability
	go subscriber.StartProcessing(ctx)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
```

## Configuration Reference

> 📖 **Complete Reference**: For all environment variables and configuration options, see [Environment Variables Reference](../../reference/configuration/environment_variables/)

### Key Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `JAEGER_ENDPOINT` | Jaeger tracing endpoint | "" (tracing disabled) |
| `SERVICE_NAME` | Service name for tracing | "agenthub-service" |
| `SERVICE_VERSION` | Service version | "1.0.0" |
| `BROKER_HEALTH_PORT` | Health endpoint port | "8080" |
| `AGENTHUB_BROKER_ADDR` | Broker address | "localhost" |
| `AGENTHUB_BROKER_PORT` | Broker port | "50051" |

### Health Endpoints

Each agent exposes these endpoints:

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `/health` | Overall health status | JSON status |
| `/metrics` | Prometheus metrics | Metrics format |
| `/ready` | Readiness check | 200 OK or 503 |

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| No traces in Jaeger | Set `JAEGER_ENDPOINT` environment variable |
| Health endpoint not accessible | Check `BROKER_HEALTH_PORT` is unique |
| Logs not structured | Ensure using `client.Logger` not standard `log` |
| Missing correlation IDs | Use `context.Context` in all operations |

### Verification Steps

1. **Check health endpoint**:
   ```bash
   curl http://localhost:8083/health
   ```

2. **Verify metrics**:
   ```bash
   curl http://localhost:8083/metrics
   ```

3. **Check traces in Jaeger**:
   - Open http://localhost:16686
   - Search for your service name

## Migration from Manual Setup

If you have existing agents using manual observability setup:

### Old Approach (Manual)
```go
// 50+ lines of OpenTelemetry setup
obs, err := observability.NewObservability(config)
traceManager := observability.NewTraceManager(serviceName)
// Manual gRPC client setup
// Manual health server setup
```

### New Approach (Unified)
```go
// 3 lines - everything automatic
config := agenthub.NewGRPCConfig("my-agent")
client, err := agenthub.NewAgentHubClient(config)
client.Start(ctx)
```

The unified abstractions provide the same observability features with 90% less code and no manual setup required.

---

With AgentHub's unified abstractions, observability is no longer an add-on feature but a built-in capability that comes automatically with every agent. Focus on your business logic while the platform handles monitoring, tracing, and health checks for you.