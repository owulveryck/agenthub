# 🔍 Distributed Tracing & OpenTelemetry

**Understanding-oriented**: Deep dive into distributed tracing concepts, OpenTelemetry architecture, and how AgentHub implements comprehensive observability for event-driven systems.

## The Problem: Observing Distributed Systems

Traditional monolithic applications are relatively easy to debug—everything happens in one process, on one machine, with one log file. But modern event-driven architectures like AgentHub present unique challenges:

### The Complexity of Event-Driven Systems

```
Request Flow in AgentHub:
User → Publisher Agent → AgentHub Broker → Subscriber Agent → Result → Publisher Agent
```

Each step involves:
- **Different processes** (potentially on different machines)
- **Asynchronous communication** (events, not direct calls)
- **Multiple protocol layers** (gRPC, HTTP, network)
- **Independent failure modes** (network partitions, service crashes)
- **Varying performance characteristics** (CPU, memory, I/O)

### Traditional Debugging Challenges

**Without distributed tracing**:
```
Publisher logs:   "Published task task_123 at 10:00:01"
Broker logs:     "Received task from agent_pub at 10:00:01"
                 "Routed task to agent_sub at 10:00:01"
Subscriber logs: "Processing task task_456 at 10:00:02"
                 "Completed task task_789 at 10:00:03"
```

**Questions you can't answer**:
- Which subscriber processed task_123?
- How long did task_123 take end-to-end?
- Where did task_123 fail?
- What was the complete flow for a specific request?

## The Solution: Distributed Tracing

Distributed tracing solves these problems by creating a unified view of requests as they flow through multiple services.

### Core Concepts

#### **Trace**
A trace represents a complete request journey through the system. In AgentHub, a trace might represent:
- Publishing a task
- Processing the task
- Publishing the result
- Receiving the result

```
Trace ID: a1b2c3d4e5f67890
Duration: 150ms
Services: 3 (publisher, broker, subscriber)
Spans: 5
Status: Success
```

#### **Span**
A span represents a single operation within a trace. Each span has:
- **Name**: What operation it represents
- **Start/End time**: When it happened
- **Tags**: Metadata about the operation
- **Logs**: Events that occurred during the operation
- **Status**: Success, error, or timeout

```
Span: "publish_event"
  Service: agenthub-publisher
  Duration: 25ms
  Tags:
    event.type: "greeting"
    event.id: "task_123"
    responder.agent: "agent_demo_subscriber"
  Status: OK
```

#### **Span Context**
The glue that connects spans across service boundaries. Contains:
- **Trace ID**: Unique identifier for the entire request
- **Span ID**: Unique identifier for the current operation
- **Trace Flags**: Sampling decisions, debug mode, etc.

### How Tracing Works in AgentHub

#### 1. **Trace Initiation**
When a publisher creates a task, it starts a new trace:

```go
// Publisher starts a trace
ctx, span := tracer.Start(ctx, "publish_event")
defer span.End()

// Add metadata
span.SetAttributes(
    attribute.String("event.type", "greeting"),
    attribute.String("event.id", taskID),
)
```

#### 2. **Context Propagation**
The trace context is injected into the task metadata:

```go
// Inject trace context into task headers
headers := make(map[string]string)
otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))

// Embed headers in task metadata
task.Metadata = &structpb.Struct{
    Fields: map[string]*structpb.Value{
        "trace_headers": structpb.NewStructValue(&structpb.Struct{
            Fields: stringMapToStructFields(headers),
        }),
    },
}
```

#### 3. **Context Extraction**
The broker and subscriber extract the trace context:

```go
// Extract trace context from task metadata
if metadata := task.GetMetadata(); metadata != nil {
    if traceHeaders, ok := metadata.Fields["trace_headers"]; ok {
        headers := structFieldsToStringMap(traceHeaders.GetStructValue().Fields)
        ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
    }
}

// Continue the trace
ctx, span := tracer.Start(ctx, "process_event")
defer span.End()
```

#### 4. **Complete Request Flow**
The result is a complete trace showing the entire request journey:

```
Trace: a1b2c3d4e5f67890
├── publish_event (agenthub-publisher) [25ms]
│   ├── event.type: greeting
│   └── event.id: task_123
├── route_task (agenthub-broker) [2ms]
│   ├── source.agent: agent_demo_publisher
│   └── target.agent: agent_demo_subscriber
├── consume_event (agenthub-subscriber) [5ms]
│   └── messaging.operation: receive
├── process_task (agenthub-subscriber) [98ms]
│   ├── task.type: greeting
│   ├── task.parameter.name: Claude
│   └── processing.status: completed
└── publish_result (agenthub-subscriber) [20ms]
    └── result.status: success
```

## OpenTelemetry Architecture

OpenTelemetry is the observability framework that powers AgentHub's tracing implementation.

### The OpenTelemetry Stack

```
┌─────────────────────────────────────────────────────────┐
│                    Applications                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │  Publisher  │ │   Broker    │ │ Subscriber  │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
└─────────────────┬───────────────┬───────────────┬─────┘
                  │               │               │
┌─────────────────▼───────────────▼───────────────▼─────┐
│              OpenTelemetry SDK                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │   Tracer    │ │    Meter    │ │   Logger    │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
└─────────────────────────────────┬───────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────┐
│            OpenTelemetry Collector                     │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐      │
│  │  Receivers  │ │ Processors  │ │  Exporters  │      │
│  └─────────────┘ └─────────────┘ └─────────────┘      │
└─────────────────┬───────────────┬───────────────┬─────┘
                  │               │               │
┌─────────────────▼─────┐ ┌───────▼───────┐ ┌─────▼─────┐
│      Jaeger           │ │  Prometheus   │ │   Logs    │
│   (Tracing)           │ │  (Metrics)    │ │(Logging)  │
└───────────────────────┘ └───────────────┘ └───────────┘
```

### Core Components

#### **Tracer**
Creates and manages spans:
```go
tracer := otel.Tracer("agenthub-publisher")
ctx, span := tracer.Start(ctx, "publish_event")
defer span.End()
```

#### **Meter**
Creates and manages metrics:
```go
meter := otel.Meter("agenthub-publisher")
counter, _ := meter.Int64Counter("events_published_total")
counter.Add(ctx, 1)
```

#### **Propagators**
Handle context propagation across service boundaries:
```go
// Inject context
otel.GetTextMapPropagator().Inject(ctx, carrier)

// Extract context
ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
```

#### **Exporters**
Send telemetry data to backend systems:
- **OTLP Exporter**: Sends to OpenTelemetry Collector
- **Jaeger Exporter**: Sends directly to Jaeger
- **Prometheus Exporter**: Exposes metrics for Prometheus

### AgentHub's OpenTelemetry Implementation

#### **Configuration**
```go
func NewObservability(config Config) (*Observability, error) {
    // Create resource (service identification)
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(config.ServiceName),
            semconv.ServiceVersion(config.ServiceVersion),
        ),
    )

    // Setup tracing
    traceExporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(config.JaegerEndpoint),
        otlptracegrpc.WithInsecure(),
    )

    tracerProvider := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(traceExporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )

    otel.SetTracerProvider(tracerProvider)

    // Setup metrics
    meterProvider := sdkmetric.NewMeterProvider(
        sdkmetric.WithResource(res),
        sdkmetric.WithReader(promExporter),
    )

    otel.SetMeterProvider(meterProvider)
}
```

#### **Custom slog Handler Integration**
AgentHub's custom logging handler automatically correlates logs with traces:

```go
func (h *ObservabilityHandler) Handle(ctx context.Context, r slog.Record) error {
    // Extract trace context
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        spanCtx := span.SpanContext()
        attrs = append(attrs,
            slog.String("trace_id", spanCtx.TraceID().String()),
            slog.String("span_id", spanCtx.SpanID().String()),
        )
    }

    // Structured log output with trace correlation
    logData := map[string]interface{}{
        "time":     r.Time.Format(time.RFC3339),
        "level":    r.Level.String(),
        "msg":      r.Message,
        "trace_id": spanCtx.TraceID().String(),
        "span_id":  spanCtx.SpanID().String(),
        "service":  h.serviceName,
    }
}
```

## Observability Patterns in Event-Driven Systems

### Pattern 1: Event Correlation

**Challenge**: Correlating events across async boundaries
**Solution**: Inject trace context into event metadata

```go
// Publisher injects context
headers := make(map[string]string)
otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))
event.Metadata["trace_headers"] = headers

// Consumer extracts context
ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.Metadata["trace_headers"]))
```

### Pattern 2: Async Operation Tracking

**Challenge**: Tracking operations that complete asynchronously
**Solution**: Create child spans that can outlive their parents

```go
// Start async operation
ctx, span := tracer.Start(ctx, "async_operation")

go func() {
    defer span.End()
    // Long-running async work
    processTask()
    span.SetStatus(2, "") // Success
}()

// Parent can continue/return immediately
```

### Pattern 3: Error Propagation

**Challenge**: Understanding how errors flow through the system
**Solution**: Record errors at each span and propagate error status

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(1, err.Error()) // Error status

    // Optionally add error details
    span.SetAttributes(
        attribute.String("error.type", "validation_error"),
        attribute.String("error.message", err.Error()),
    )
}
```

### Pattern 4: Performance Attribution

**Challenge**: Understanding where time is spent in complex flows
**Solution**: Detailed span hierarchy with timing

```go
// High-level operation
ctx, span := tracer.Start(ctx, "process_task")
defer span.End()

// Sub-operations
ctx, validateSpan := tracer.Start(ctx, "validate_input")
// ... validation logic
validateSpan.End()

ctx, computeSpan := tracer.Start(ctx, "compute_result")
// ... computation logic
computeSpan.End()

ctx, persistSpan := tracer.Start(ctx, "persist_result")
// ... persistence logic
persistSpan.End()
```

## Benefits of AgentHub's Observability Implementation

### 1. **Complete Request Visibility**
- See every step of event processing
- Understand inter-service dependencies
- Track request flows across async boundaries

### 2. **Performance Analysis**
- Identify bottlenecks in event processing
- Understand where time is spent
- Optimize critical paths

### 3. **Error Diagnosis**
- Pinpoint exactly where failures occur
- Understand error propagation patterns
- Correlate errors with system state

### 4. **Capacity Planning**
- Understand system throughput characteristics
- Identify scaling bottlenecks
- Plan resource allocation

### 5. **Troubleshooting**
- Correlate logs, metrics, and traces
- Understand system behavior under load
- Debug complex distributed issues

## Advanced Tracing Concepts

### Sampling

Not every request needs to be traced. Sampling reduces overhead:

```go
// Probability sampling (trace 10% of requests)
sdktrace.WithSampler(sdktrace.ParentBased(
    sdktrace.TraceIDRatioBased(0.1),
))

// Rate limiting sampling (max 100 traces/second)
sdktrace.WithSampler(sdktrace.ParentBased(
    sdktrace.RateLimited(100),
))
```

### Custom Attributes

Add business context to spans:

```go
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.String("tenant.id", tenantID),
    attribute.Int("batch.size", len(items)),
    attribute.String("workflow.type", workflowType),
)
```

### Span Events

Add timestamped events within spans:

```go
span.AddEvent("validation.started")
// ... validation logic
span.AddEvent("validation.completed", trace.WithAttributes(
    attribute.Int("validation.rules.evaluated", ruleCount),
))
```

### Baggage

Propagate key-value pairs across the entire trace:

```go
// Set baggage
ctx = baggage.ContextWithValues(ctx,
    baggage.String("user.tier", "premium"),
    baggage.String("feature.flag", "new_algorithm"),
)

// Read baggage in any service
if member := baggage.FromContext(ctx).Member("user.tier"); member.Value() == "premium" {
    // Use premium algorithm
}
```

## Performance Considerations

### Overhead Analysis

AgentHub's observability adds:
- **CPU**: ~5% overhead for tracing
- **Memory**: ~50MB per service for buffers and metadata
- **Network**: Minimal (async batched export)
- **Latency**: ~10ms additional end-to-end latency

### Optimization Strategies

1. **Sampling**: Reduce trace volume for high-throughput systems
2. **Batching**: Export spans in batches to reduce network overhead
3. **Async Processing**: Never block business logic for observability
4. **Resource Limits**: Use memory limiters in the collector

### Production Recommendations

- **Enable sampling** for high-volume systems
- **Monitor collector performance** and scale horizontally if needed
- **Set retention policies** for traces and metrics
- **Use dedicated infrastructure** for observability stack

## Troubleshooting Common Issues

### Missing Traces

**Symptoms**: No traces appear in Jaeger
**Causes**:
- Context not propagated correctly
- Exporter configuration issues
- Collector connectivity problems

**Debugging**:
```bash
# Check if spans are being created
curl http://localhost:8080/metrics | grep trace

# Check collector logs
docker-compose logs otel-collector

# Verify Jaeger connectivity
curl http://localhost:16686/api/traces
```

### Broken Trace Chains

**Symptoms**: Spans appear disconnected
**Causes**:
- Context not extracted properly
- New context created instead of continuing existing

**Debugging**:
```go
// Always check if context contains active span
if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
    fmt.Printf("Active trace: %s\n", span.SpanContext().TraceID())
} else {
    fmt.Println("No active trace context")
}
```

### High Memory Usage

**Symptoms**: Observability causing OOM errors
**Causes**:
- Too many spans in memory
- Large span attributes
- Export failures causing backlog

**Solutions**:
```go
// Configure memory limits
config := sdktrace.NewTracerProvider(
    sdktrace.WithSpanLimits(sdktrace.SpanLimits{
        AttributeCountLimit: 128,
        EventCountLimit:     128,
        LinkCountLimit:      128,
    }),
)
```

## The Future of Observability

### Emerging Trends

1. **eBPF-based Observability**: Automatic instrumentation without code changes
2. **AI-Powered Analysis**: Automatic anomaly detection and root cause analysis
3. **Unified Observability**: Single pane of glass for metrics, traces, logs, and profiles
4. **Real-time Alerting**: Faster detection and response to issues

### OpenTelemetry Roadmap

- **Profiling**: Continuous profiling integration
- **Client-side Observability**: Browser and mobile app tracing
- **Database Instrumentation**: Automatic query tracing
- **Infrastructure Correlation**: Link application traces to infrastructure metrics

## Conclusion

Distributed tracing transforms debugging from guesswork into precise investigation. AgentHub's implementation with OpenTelemetry provides:

- **Complete visibility** into event-driven workflows
- **Performance insights** for optimization
- **Error correlation** for faster resolution
- **Business context** through custom attributes

The investment in observability pays dividends in:
- **Reduced MTTR** (Mean Time To Resolution)
- **Improved performance** through data-driven optimization
- **Better user experience** through proactive monitoring
- **Team productivity** through better tooling

---

**🎯 Ready to Implement?**

**Hands-on**: **[Observability Demo Tutorial](../tutorials/observability_demo.md)**

**Production**: **[Add Observability to Your Agent](../howto/add_observability.md)**

**Deep Dive**: **[Observability Architecture](observability_architecture.md)**