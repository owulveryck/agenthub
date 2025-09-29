---
title: "AgentHub Tracing API Reference"
weight: 60
description: >
  Complete API documentation for AgentHub's OpenTelemetry tracing integration, span management, context propagation, and instrumentation patterns.
---

# 🔍 AgentHub Tracing API Reference

**Technical reference**: Complete API documentation for AgentHub's OpenTelemetry tracing integration, span management, context propagation, and instrumentation patterns.

## Core Components

### TraceManager

The `TraceManager` provides high-level tracing operations for AgentHub events.

#### Constructor

```go
func NewTraceManager(serviceName string) *TraceManager
```

**Parameters**:
- `serviceName` - Name of the service creating spans

**Returns**: Configured TraceManager instance

#### Methods

##### StartPublishSpan
```go
func (tm *TraceManager) StartPublishSpan(ctx context.Context, responderAgentID, eventType string) (context.Context, trace.Span)
```

**Purpose**: Creates a span for event publishing operations

**Parameters**:
- `ctx` - Parent context (may contain existing trace)
- `responderAgentID` - Target agent for the event
- `eventType` - Type of event being published

**Returns**:
- `context.Context` - New context with active span
- `trace.Span` - The created span

**Attributes Set**:
- `event.type` - Event type being published
- `responder.agent` - Target agent ID
- `operation.type` - "publish"

**Usage**:
```go
ctx, span := tm.StartPublishSpan(ctx, "agent_subscriber", "greeting")
defer span.End()
// ... publishing logic
```

##### StartEventProcessingSpan
```go
func (tm *TraceManager) StartEventProcessingSpan(ctx context.Context, eventID, eventType, requesterAgentID, responderAgentID string) (context.Context, trace.Span)
```

**Purpose**: Creates a span for event processing operations

**Parameters**:
- `ctx` - Context with extracted trace information
- `eventID` - Unique identifier for the event
- `eventType` - Type of event being processed
- `requesterAgentID` - Agent that requested processing
- `responderAgentID` - Agent performing processing

**Returns**:
- `context.Context` - Context with processing span
- `trace.Span` - The processing span

**Attributes Set**:
- `event.id` - Event identifier
- `event.type` - Event type
- `requester.agent` - Requesting agent ID
- `responder.agent` - Processing agent ID
- `operation.type` - "process"

##### StartBrokerSpan
```go
func (tm *TraceManager) StartBrokerSpan(ctx context.Context, operation, eventType string) (context.Context, trace.Span)
```

**Purpose**: Creates spans for broker operations

**Parameters**:
- `ctx` - Request context
- `operation` - Broker operation (route, subscribe, unsubscribe)
- `eventType` - Event type being handled

**Returns**:
- `context.Context` - Context with broker span
- `trace.Span` - The broker span

**Attributes Set**:
- `operation.type` - Broker operation type
- `event.type` - Event type being handled
- `component` - "broker"

##### InjectTraceContext
```go
func (tm *TraceManager) InjectTraceContext(ctx context.Context, headers map[string]string)
```

**Purpose**: Injects trace context into headers for propagation

**Parameters**:
- `ctx` - Context containing trace information
- `headers` - Map to inject headers into

**Headers Injected**:
- `traceparent` - W3C trace context header
- `tracestate` - W3C trace state header (if present)

**Usage**:
```go
headers := make(map[string]string)
tm.InjectTraceContext(ctx, headers)
// headers now contain trace context for propagation
```

##### ExtractTraceContext
```go
func (tm *TraceManager) ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context
```

**Purpose**: Extracts trace context from headers

**Parameters**:
- `ctx` - Base context
- `headers` - Headers containing trace context

**Returns**: Context with extracted trace information

**Usage**:
```go
// Extract from event metadata
if metadata := event.GetMetadata(); metadata != nil {
    if traceHeaders, ok := metadata.Fields["trace_headers"]; ok {
        headers := structFieldsToStringMap(traceHeaders.GetStructValue().Fields)
        ctx = tm.ExtractTraceContext(ctx, headers)
    }
}
```

##### RecordError
```go
func (tm *TraceManager) RecordError(span trace.Span, err error)
```

**Purpose**: Records an error on a span with proper formatting

**Parameters**:
- `span` - Span to record error on
- `err` - Error to record

**Effects**:
- Sets span status to error
- Records error as span event
- Adds error type attribute

##### SetSpanSuccess
```go
func (tm *TraceManager) SetSpanSuccess(span trace.Span)
```

**Purpose**: Marks a span as successful

**Parameters**:
- `span` - Span to mark as successful

**Effects**:
- Sets span status to OK
- Ensures span is properly completed

## Context Propagation

### W3C Trace Context Standards

AgentHub uses the W3C Trace Context specification for interoperability.

#### Trace Context Headers

##### traceparent
**Format**: `00-{trace-id}-{span-id}-{trace-flags}`
- `00` - Version (currently always 00)
- `trace-id` - 32-character hex string
- `span-id` - 16-character hex string
- `trace-flags` - 2-character hex flags

**Example**: `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`

##### tracestate
**Format**: Vendor-specific key-value pairs
**Example**: `rojo=00f067aa0ba902b7,congo=t61rcWkgMzE`

### Propagation Implementation

#### Manual Injection
```go
// Create headers map
headers := make(map[string]string)

// Inject trace context
otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))

// Headers now contain trace context
// Convert to protobuf metadata if needed
metadataStruct, err := structpb.NewStruct(map[string]interface{}{
    "trace_headers": headers,
    "timestamp": time.Now().Format(time.RFC3339),
})
```

#### Manual Extraction
```go
// Extract from protobuf metadata
if metadata := task.GetMetadata(); metadata != nil {
    if traceHeaders, ok := metadata.Fields["trace_headers"]; ok {
        headers := make(map[string]string)
        for k, v := range traceHeaders.GetStructValue().Fields {
            headers[k] = v.GetStringValue()
        }
        ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
    }
}
```

## Span Lifecycle Management

### Creating Spans

#### Basic Span Creation
```go
tracer := otel.Tracer("my-service")
ctx, span := tracer.Start(ctx, "operation_name")
defer span.End()
```

#### Span with Attributes
```go
ctx, span := tracer.Start(ctx, "operation_name", trace.WithAttributes(
    attribute.String("operation.type", "publish"),
    attribute.String("event.type", "greeting"),
    attribute.Int("event.priority", 1),
))
defer span.End()
```

#### Child Span Creation
```go
// Parent span
ctx, parentSpan := tracer.Start(ctx, "parent_operation")
defer parentSpan.End()

// Child span (automatically linked)
ctx, childSpan := tracer.Start(ctx, "child_operation")
defer childSpan.End()
```

### Span Attributes

#### Standard Attributes
AgentHub uses consistent attribute naming:

```go
// Event attributes
attribute.String("event.id", taskID)
attribute.String("event.type", taskType)
attribute.Int("event.priority", priority)

// Agent attributes
attribute.String("agent.id", agentID)
attribute.String("agent.type", agentType)
attribute.String("requester.agent", requesterID)
attribute.String("responder.agent", responderID)

// Operation attributes
attribute.String("operation.type", "publish|process|route")
attribute.String("component", "broker|publisher|subscriber")

// Result attributes
attribute.Bool("success", true)
attribute.String("error.type", "validation|timeout|network")
```

#### Custom Attributes
```go
span.SetAttributes(
    attribute.String("business.unit", "sales"),
    attribute.String("user.tenant", "acme-corp"),
    attribute.Int("batch.size", len(items)),
    attribute.Duration("timeout", 30*time.Second),
)
```

### Span Events

#### Adding Events
```go
// Simple event
span.AddEvent("validation.started")

// Event with attributes
span.AddEvent("cache.miss", trace.WithAttributes(
    attribute.String("cache.key", key),
    attribute.String("cache.type", "redis"),
))

// Event with timestamp
span.AddEvent("external.api.call", trace.WithAttributes(
    attribute.String("api.endpoint", "/v1/users"),
    attribute.Int("api.status_code", 200),
), trace.WithTimestamp(time.Now()))
```

#### Common Event Patterns
```go
// Processing milestones
span.AddEvent("processing.started")
span.AddEvent("validation.completed")
span.AddEvent("business.logic.completed")
span.AddEvent("result.published")

// Error events
span.AddEvent("error.occurred", trace.WithAttributes(
    attribute.String("error.message", err.Error()),
    attribute.String("error.stack", debug.Stack()),
))
```

### Span Status

#### Setting Status
```go
// Success
span.SetStatus(codes.Ok, "")

// Error with message
span.SetStatus(codes.Error, "validation failed")

// Error without message
span.SetStatus(codes.Error, "")
```

#### Status Code Mapping
```go
// gRPC codes to OpenTelemetry codes
statusCode := codes.Ok
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        statusCode = codes.DeadlineExceeded
    case errors.Is(err, context.Canceled):
        statusCode = codes.Cancelled
    default:
        statusCode = codes.Error
    }
}
span.SetStatus(statusCode, err.Error())
```

## Advanced Instrumentation

### Baggage Propagation

#### Setting Baggage
```go
// Add baggage to context
ctx = baggage.ContextWithValues(ctx,
    baggage.String("user.id", userID),
    baggage.String("tenant.id", tenantID),
    baggage.String("request.id", requestID),
)
```

#### Reading Baggage
```go
// Read baggage anywhere in the trace
if member := baggage.FromContext(ctx).Member("user.id"); member.Value() != "" {
    userID := member.Value()
    // Use user ID for business logic
}
```

### Span Links

#### Creating Links
```go
// Link to related span
linkedSpanContext := trace.SpanContextFromContext(relatedCtx)
ctx, span := tracer.Start(ctx, "operation", trace.WithLinks(
    trace.Link{
        SpanContext: linkedSpanContext,
        Attributes: []attribute.KeyValue{
            attribute.String("link.type", "related_operation"),
        },
    },
))
```

### Sampling Control

#### Conditional Sampling
```go
// Force sampling for important operations
ctx, span := tracer.Start(ctx, "critical_operation",
    trace.WithNewRoot(), // Start new trace
    trace.WithSpanKind(trace.SpanKindServer),
)

// Add sampling priority
span.SetAttributes(
    attribute.String("sampling.priority", "high"),
)
```

## Integration Patterns

### gRPC Integration

#### Server Interceptor
```go
func TracingUnaryInterceptor(tracer trace.Tracer) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        ctx, span := tracer.Start(ctx, info.FullMethod)
        defer span.End()

        resp, err := handler(ctx, req)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }
        return resp, err
    }
}
```

#### Client Interceptor
```go
func TracingUnaryClientInterceptor(tracer trace.Tracer) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        ctx, span := tracer.Start(ctx, method)
        defer span.End()

        err := invoker(ctx, method, req, reply, cc, opts...)
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        }
        return err
    }
}
```

### HTTP Integration

#### HTTP Handler Wrapper
```go
func TracingHandler(tracer trace.Tracer, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
        ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
        defer span.End()

        span.SetAttributes(
            attribute.String("http.method", r.Method),
            attribute.String("http.url", r.URL.String()),
            attribute.String("http.user_agent", r.UserAgent()),
        )

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Error Handling

### Error Recording Best Practices

#### Complete Error Recording
```go
if err != nil {
    // Record error on span
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())

    // Add error context
    span.SetAttributes(
        attribute.String("error.type", classifyError(err)),
        attribute.Bool("error.retryable", isRetryable(err)),
    )

    // Log with context
    logger.ErrorContext(ctx, "Operation failed",
        slog.Any("error", err),
        slog.String("operation", "event_processing"),
    )

    return err
}
```

#### Error Classification
```go
func classifyError(err error) string {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return "timeout"
    case errors.Is(err, context.Canceled):
        return "cancelled"
    case strings.Contains(err.Error(), "connection"):
        return "network"
    case strings.Contains(err.Error(), "validation"):
        return "validation"
    default:
        return "unknown"
    }
}
```

## Performance Considerations

### Span Creation Overhead
- **Span creation**: ~1-2μs per span
- **Attribute setting**: ~100ns per attribute
- **Event recording**: ~200ns per event
- **Context propagation**: ~500ns per injection/extraction

### Memory Usage
- **Active span**: ~500 bytes
- **Completed span buffer**: ~1KB per span
- **Context overhead**: ~100 bytes per context

### Best Practices
1. **Limit span attributes** to essential information
2. **Use batch exporters** to reduce network overhead
3. **Sample appropriately** for high-throughput services
4. **Pool span contexts** where possible
5. **Avoid deep span nesting** (>10 levels)

## Troubleshooting

### Missing Spans Checklist
1. ✅ OpenTelemetry properly initialized
2. ✅ Tracer retrieved from global provider
3. ✅ Context propagated correctly
4. ✅ Spans properly ended
5. ✅ Exporter configured and accessible

### Common Issues

#### Broken Trace Chains
```go
// ❌ Wrong - creates new root trace
ctx, span := tracer.Start(context.Background(), "operation")

// ✅ Correct - continues existing trace
ctx, span := tracer.Start(ctx, "operation")
```

#### Missing Context Propagation
```go
// ❌ Wrong - context not propagated
go func() {
    ctx, span := tracer.Start(context.Background(), "async_work")
    // work...
}()

// ✅ Correct - context properly propagated
go func(ctx context.Context) {
    ctx, span := tracer.Start(ctx, "async_work")
    // work...
}(ctx)
```

---

**🎯 Next Steps**:

**Implementation**: **[Add Observability to Your Agent](../howto/add_observability.md)**

**Debugging**: **[Debug with Distributed Tracing](../howto/debug_with_tracing.md)**

**Metrics**: **[Observability Metrics Reference](observability_metrics.md)**