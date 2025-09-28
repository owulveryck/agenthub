# 📈 How to Add Observability to Your Agent

**Goal-oriented guide**: Transform your existing AgentHub agent into a fully observable service with distributed tracing, metrics, and structured logging.

## Prerequisites

- Existing AgentHub agent (publisher or subscriber)
- Go 1.24+ installed
- Basic understanding of AgentHub concepts
- 15-20 minutes

## Overview: What You'll Add

✅ **Distributed Tracing** - Track events across service boundaries
✅ **Comprehensive Metrics** - Monitor performance and health
✅ **Structured Logging** - Correlate logs with traces
✅ **Health Endpoints** - Enable monitoring and alerting
✅ **Graceful Shutdown** - Clean resource management

## Step 1: Import Observability Package

Add the observability import to your agent:

```go
import (
    // ... your existing imports
    "log/slog"
    "github.com/owulveryck/agenthub/internal/observability"
)
```

## Step 2: Initialize Observability Components

Replace your basic logging setup with observable components:

### Before (Basic Agent):
```go
func main() {
    // Basic logging
    log.Printf("Starting agent...")

    // gRPC setup
    conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    client := pb.NewEventBusClient(conn)

    // Agent logic...
}
```

### After (Observable Agent):
```go
type ObservableAgent struct {
    client         pb.EventBusClient
    obs            *observability.Observability
    traceManager   *observability.TraceManager
    metricsManager *observability.MetricsManager
    healthServer   *observability.HealthServer
    logger         *slog.Logger
}

func NewObservableAgent() (*ObservableAgent, error) {
    // Initialize observability
    config := observability.DefaultConfig("your-agent-name")
    obs, err := observability.NewObservability(config)
    if err != nil {
        return nil, err
    }

    // Initialize metrics manager
    metricsManager, err := observability.NewMetricsManager(obs.Meter)
    if err != nil {
        return nil, err
    }

    // Initialize trace manager
    traceManager := observability.NewTraceManager(config.ServiceName)

    // Initialize health server (use unique port for each agent)
    healthServer := observability.NewHealthServer("8083", config.ServiceName, config.ServiceVersion)

    // Add health checks
    healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
        return nil // Your health check logic
    }))

    // Set up gRPC connection
    conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to agent hub: %w", err)
    }

    client := pb.NewEventBusClient(conn)

    // Add gRPC connection health check
    healthServer.AddChecker("agenthub_connection",
        observability.NewGRPCHealthChecker("agenthub_connection", agentHubAddr))

    return &ObservableAgent{
        client:         client,
        obs:            obs,
        traceManager:   traceManager,
        metricsManager: metricsManager,
        healthServer:   healthServer,
        logger:         obs.Logger,
    }, nil
}
```

## Step 3: Add Tracing to Event Publishing

### Before (Basic Publishing):
```go
func publishTask(ctx context.Context, client pb.EventBusClient, taskType string, params map[string]interface{}) {
    log.Printf("Publishing task type: %s", taskType)

    // Create task
    task := &pb.TaskMessage{
        TaskId:   generateTaskID(),
        TaskType: taskType,
        // ... other fields
    }

    // Publish
    res, err := client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
    if err != nil {
        log.Printf("Error: %v", err)
        return
    }

    log.Printf("Task published successfully")
}
```

### After (Observable Publishing):
```go
func (a *ObservableAgent) publishTask(ctx context.Context, taskType string, params map[string]interface{}, responderAgentID string) error {
    // Start tracing for task publishing
    ctx, span := a.traceManager.StartPublishSpan(ctx, responderAgentID, taskType)
    defer span.End()

    // Start timing
    timer := a.metricsManager.StartTimer()
    defer timer(ctx, taskType, "your-agent-name")

    // Generate task ID
    taskID := generateTaskID(taskType)

    a.logger.InfoContext(ctx, "Publishing task",
        slog.String("task_id", taskID),
        slog.String("task_type", taskType),
        slog.String("responder_agent_id", responderAgentID),
    )

    // Convert parameters to protobuf Struct
    parametersStruct, err := structpb.NewStruct(params)
    if err != nil {
        a.logger.ErrorContext(ctx, "Error creating parameters struct",
            slog.String("task_id", taskID),
            slog.Any("error", err),
        )
        a.traceManager.RecordError(span, err)
        a.metricsManager.IncrementEventErrors(ctx, taskType, "your-agent-name", "struct_conversion_error")
        return err
    }

    // Inject trace context into task metadata
    headers := make(map[string]string)
    a.traceManager.InjectTraceContext(ctx, headers)

    // Convert headers to protobuf Struct
    metadataStruct, err := structpb.NewStruct(map[string]interface{}{
        "trace_headers": headers,
        "publisher":     "your-agent-name",
        "published_at":  time.Now().Format(time.RFC3339),
    })
    if err != nil {
        a.logger.WarnContext(ctx, "Error creating metadata struct",
            slog.String("task_id", taskID),
            slog.Any("error", err),
        )
        metadataStruct = &structpb.Struct{}
    }

    // Create task message
    task := &pb.TaskMessage{
        TaskId:           taskID,
        TaskType:         taskType,
        Parameters:       parametersStruct,
        RequesterAgentId: "your-agent-name",
        ResponderAgentId: responderAgentID,
        Priority:         pb.Priority_PRIORITY_MEDIUM,
        CreatedAt:        timestamppb.Now(),
        Metadata:         metadataStruct,
    }

    // Publish the task
    res, err := a.client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
    if err != nil {
        a.logger.ErrorContext(ctx, "Error publishing task",
            slog.String("task_id", taskID),
            slog.Any("error", err),
        )
        a.traceManager.RecordError(span, err)
        a.metricsManager.IncrementEventErrors(ctx, taskType, "your-agent-name", "grpc_error")
        return err
    }

    if !res.GetSuccess() {
        err := fmt.Errorf("failed to publish task: %s", res.GetError())
        a.logger.ErrorContext(ctx, "Failed to publish task",
            slog.String("task_id", taskID),
            slog.String("error", res.GetError()),
        )
        a.traceManager.RecordError(span, err)
        a.metricsManager.IncrementEventErrors(ctx, taskType, "your-agent-name", "publish_failed")
        return err
    }

    a.logger.InfoContext(ctx, "Task published successfully",
        slog.String("task_id", taskID),
        slog.String("task_type", taskType),
    )

    // Record successful metrics
    a.metricsManager.IncrementEventsProcessed(ctx, taskType, "your-agent-name", true)
    a.metricsManager.IncrementEventsPublished(ctx, taskType, responderAgentID)
    a.traceManager.SetSpanSuccess(span)

    return nil
}
```

## Step 4: Add Tracing to Event Processing (For Subscribers)

### Before (Basic Processing):
```go
func processTask(task *pb.TaskMessage) {
    log.Printf("Processing task: %s", task.GetTaskId())

    // Process the task
    result := doSomeWork(task)

    log.Printf("Task completed: %s", task.GetTaskId())

    // Publish result...
}
```

### After (Observable Processing):
```go
func (a *ObservableAgent) processTask(ctx context.Context, task *pb.TaskMessage) {
    // Extract trace context from task metadata
    if metadata := task.GetMetadata(); metadata != nil {
        if traceHeaders, ok := metadata.Fields["trace_headers"]; ok {
            if headersStruct := traceHeaders.GetStructValue(); headersStruct != nil {
                headers := make(map[string]string)
                for k, v := range headersStruct.Fields {
                    headers[k] = v.GetStringValue()
                }
                ctx = a.traceManager.ExtractTraceContext(ctx, headers)
            }
        }
    }

    // Start processing span
    ctx, span := a.traceManager.StartEventProcessingSpan(ctx, task.GetTaskId(), task.GetTaskType(), task.GetRequesterAgentId(), "")
    defer span.End()

    // Start timing
    timer := a.metricsManager.StartTimer()
    defer timer(ctx, task.GetTaskType(), "your-agent-name")

    a.logger.InfoContext(ctx, "Processing task",
        slog.String("task_id", task.GetTaskId()),
        slog.String("task_type", task.GetTaskType()),
        slog.String("requester_agent_id", task.GetRequesterAgentId()),
    )

    // Process the task
    result, status, errorMessage := a.doTaskWork(ctx, task)

    // Create task result
    taskResult := &pb.TaskResult{
        TaskId:            task.GetTaskId(),
        Status:            status,
        Result:            result,
        ErrorMessage:      errorMessage,
        ExecutorAgentId:   "your-agent-name",
        CompletedAt:       timestamppb.Now(),
        ExecutionMetadata: &structpb.Struct{},
    }

    // Publish the result
    if err := a.publishTaskResult(ctx, taskResult); err != nil {
        a.logger.ErrorContext(ctx, "Failed to publish task result",
            slog.String("task_id", task.GetTaskId()),
            slog.Any("error", err),
        )
        a.traceManager.RecordError(span, err)
        a.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "your-agent-name", "result_publish_error")
    } else {
        a.logger.InfoContext(ctx, "Task completed and result published",
            slog.String("task_id", task.GetTaskId()),
            slog.String("status", status.String()),
        )
        a.metricsManager.IncrementEventsProcessed(ctx, task.GetTaskType(), "your-agent-name", status == pb.TaskStatus_TASK_STATUS_COMPLETED)
        a.traceManager.SetSpanSuccess(span)
    }
}
```

## Step 5: Add Background Services and Graceful Shutdown

```go
func (a *ObservableAgent) Run(ctx context.Context) error {
    a.logger.InfoContext(ctx, "Starting observable agent")

    // Start health server
    go func() {
        a.logger.Info("Starting health server on port 8083")
        if err := a.healthServer.Start(ctx); err != nil {
            a.logger.Error("Health server failed", slog.Any("error", err))
        }
    }()

    // Start metrics collection
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                a.metricsManager.UpdateSystemMetrics(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()

    // Your agent's main logic here...
    // (subscription loops, task processing, etc.)

    // Wait for context cancellation
    <-ctx.Done()
    return ctx.Err()
}

func (a *ObservableAgent) Shutdown(ctx context.Context) error {
    a.logger.InfoContext(ctx, "Shutting down observable agent")

    // Shutdown observability components
    if err := a.healthServer.Shutdown(ctx); err != nil {
        a.logger.ErrorContext(ctx, "Error shutting down health server", slog.Any("error", err))
    }

    if err := a.obs.Shutdown(ctx); err != nil {
        a.logger.ErrorContext(ctx, "Error shutting down observability", slog.Any("error", err))
        return err
    }

    return nil
}
```

## Step 6: Update Your Main Function

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    agent, err := NewObservableAgent()
    if err != nil {
        panic(fmt.Sprintf("Failed to create observable agent: %v", err))
    }

    defer func() {
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer shutdownCancel()
        if err := agent.Shutdown(shutdownCtx); err != nil {
            agent.logger.Error("Error during shutdown", slog.Any("error", err))
        }
    }()

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        agent.logger.Info("Received shutdown signal")
        cancel()
    }()

    if err := agent.Run(ctx); err != nil && err != context.Canceled {
        agent.logger.Error("Agent run failed", slog.Any("error", err))
        panic(err)
    }

    agent.logger.Info("Agent shutdown complete")
}
```

## Step 7: Add Build Tags

Add build tags to separate observable and basic versions:

```go
//go:build observability
// +build observability

package main

// ... rest of your observable agent code
```

## Step 8: Build and Test

```bash
# Build your observable agent
go build -tags observability -o bin/my-agent-obs your-agent-directory/

# Test health endpoint
curl http://localhost:8083/health

# Test metrics endpoint
curl http://localhost:8083/metrics
```

## Step 9: Configure Unique Ports

Make sure each agent uses a unique port for health endpoints:

| **Agent Type** | **Suggested Port** | **Health URL** |
|----------------|-------------------|----------------|
| Broker | 8080 | http://localhost:8080/health |
| Publisher | 8081 | http://localhost:8081/health |
| Subscriber | 8082 | http://localhost:8082/health |
| Your Agent | 8083+ | http://localhost:8083/health |

## Step 10: Update Prometheus Configuration

Add your agent to Prometheus scraping:

```yaml
# In observability/prometheus/prometheus.yml
scrape_configs:
  # ... existing configs
  - job_name: 'your-agent-name'
    static_configs:
      - targets: ['host.docker.internal:8083']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

## Verification Checklist

After implementing observability, verify:

✅ **Health endpoint responds**: `curl http://localhost:8083/health`
✅ **Metrics endpoint works**: `curl http://localhost:8083/metrics`
✅ **Traces appear in Jaeger**: Check http://localhost:16686
✅ **Metrics in Grafana**: Check dashboard for your service
✅ **Structured logs**: Look for trace_id in log output
✅ **Graceful shutdown**: Ctrl+C should shut down cleanly

## Common Issues and Solutions

### Issue: No traces in Jaeger
```bash
# Check if trace context is being propagated
grep "trace_id" your-agent.log

# Verify agent is using observability build tag
go build -tags observability -v your-agent-directory/
```

### Issue: No metrics in Prometheus
```bash
# Check if Prometheus can reach your agent
curl http://localhost:8083/metrics

# Verify Prometheus configuration
docker-compose logs prometheus
```

### Issue: Health checks failing
```bash
# Test health endpoint directly
curl -v http://localhost:8083/health

# Check if port is already in use
lsof -i :8083
```

## Advanced Customization

### Custom Metrics
```go
// Add custom business metrics
customCounter, err := a.metricsManager.meter.Int64Counter(
    "my_custom_metric_total",
    metric.WithDescription("My custom business metric"),
)

// Use in your code
customCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("business_unit", "sales"),
))
```

### Custom Health Checks
```go
// Add custom health check
a.healthServer.AddChecker("database", observability.NewBasicHealthChecker("database",
    func(ctx context.Context) error {
        // Your database health check logic
        return db.Ping()
    }))
```

### Custom Trace Attributes
```go
// Add custom span attributes
span.SetAttributes(
    attribute.String("user_id", userID),
    attribute.String("tenant_id", tenantID),
    attribute.Int("batch_size", len(items)),
)
```

## Next Steps

### **Production Readiness**:
- **[Configure Alerts](configure_alerts.md)** - Set up monitoring for your agent
- **[Use Grafana Dashboards](use_dashboards.md)** - Monitor your agent's performance

### **Debugging**:
- **[Debug with Distributed Tracing](debug_with_tracing.md)** - Troubleshoot issues effectively

### **Understanding**:
- **[Distributed Tracing Explained](../explanation/distributed_tracing.md)** - Deep dive into concepts

---

**🎯 Success!** Your agent is now fully observable with distributed tracing, metrics, and structured logging. Run it alongside the observability stack to see it in action!