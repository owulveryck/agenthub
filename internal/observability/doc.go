// Package observability provides comprehensive observability infrastructure including
// distributed tracing, metrics collection, structured logging, and health checks.
//
// # Overview
//
// The observability package implements OpenTelemetry-based observability with:
//   - Distributed tracing (OpenTelemetry/Jaeger)
//   - Metrics collection (Prometheus)
//   - Structured logging (log/slog)
//   - Health check endpoints
//   - Automatic instrumentation for A2A protocol operations
//   - Graceful shutdown with trace flushing
//
// This package is the foundation for observability across AgentHub, providing
// consistent tracing, metrics, and logging for brokers, agents, and orchestrators.
//
// # Quick Start
//
// Initialize observability for your service:
//
//	config := observability.DefaultConfig("my_service")
//	obs, err := observability.NewObservability(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer obs.Shutdown(context.Background())
//
//	// Use the components
//	logger := obs.Logger
//	tracer := obs.Tracer
//	meter := obs.Meter
//
// This automatically sets up:
//   - OTLP trace exporter to Jaeger
//   - Prometheus metrics exporter
//   - Structured logger with trace context
//   - Proper resource attributes (service name, version, environment)
//
// # Architecture
//
// The package provides layered observability:
//
//	┌─────────────────────────────────────────────┐
//	│         Application Code                    │
//	│   (Agents, Broker, Orchestrators)           │
//	├─────────────────────────────────────────────┤
//	│         TraceManager                        │
//	│   - Span creation & management              │
//	│   - A2A-specific span attributes            │
//	│   - Context propagation                     │
//	├─────────────────────────────────────────────┤
//	│         MetricsManager                      │
//	│   - Counter metrics (events, errors)        │
//	│   - Histogram metrics (durations)           │
//	│   - Gauge metrics (goroutines, memory)      │
//	├─────────────────────────────────────────────┤
//	│         Logger (slog)                       │
//	│   - Structured logging                      │
//	│   - Trace context injection                 │
//	│   - Configurable log levels                 │
//	├─────────────────────────────────────────────┤
//	│         OpenTelemetry SDK                   │
//	│   - OTLP trace exporter → Jaeger            │
//	│   - Prometheus metrics exporter             │
//	│   - Resource detection                      │
//	└─────────────────────────────────────────────┘
//
// # Configuration
//
// **Config** specifies observability settings:
//
//	config := observability.Config{
//	    ServiceName:    "my_service",
//	    ServiceVersion: "1.0.0",
//	    JaegerEndpoint: "localhost:4317",    // OTLP gRPC endpoint
//	    PrometheusPort: "9090",
//	    Environment:    "production",
//	    LogLevel:       "INFO",              // DEBUG, INFO, WARN, ERROR
//	}
//
// **DefaultConfig** reads from environment:
//
//	config := observability.DefaultConfig("my_service")
//
// Environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT: Jaeger OTLP endpoint
//   - PROMETHEUS_PORT: Port for Prometheus metrics
//   - ENVIRONMENT: Deployment environment (dev, staging, prod)
//   - LOG_LEVEL: Logging level (DEBUG, INFO, WARN, ERROR)
//
// # Distributed Tracing
//
// Use TraceManager for creating and managing spans:
//
//	traceManager := observability.NewTraceManager("my_service")
//
//	// Start a span
//	ctx, span := traceManager.StartSpan(ctx, "process_request")
//	defer span.End()
//
//	// Add attributes
//	span.SetAttributes(
//	    attribute.String("user_id", "user123"),
//	    attribute.Int("items_count", 5),
//	)
//
//	// Record errors
//	if err != nil {
//	    traceManager.RecordError(span, err)
//	} else {
//	    traceManager.SetSpanSuccess(span)
//	}
//
// ## A2A-Specific Tracing
//
// TraceManager provides specialized methods for A2A protocol operations:
//
// **Task Publishing**:
//
//	ctx, span := traceManager.StartPublishSpan(ctx, "cortex", "agent_translator", "translation_task")
//	defer span.End()
//
// **Message Processing**:
//
//	ctx, span := traceManager.StartA2AMessageSpan(ctx, "agent.process", messageID, "ROLE_USER")
//	defer span.End()
//
// **Task Attributes**:
//
//	traceManager.AddA2ATaskAttributes(
//	    span,
//	    taskID,
//	    "language_translation",
//	    contextID,
//	    len(task.History),
//	    len(task.Artifacts),
//	)
//
// **Event Routing**:
//
//	ctx, span := traceManager.StartA2AEventRouteSpan(ctx, "broker", eventID, "task_message", subscriberCount)
//	defer span.End()
//
// ## Context Propagation
//
// Propagate trace context across service boundaries:
//
//	// Inject into headers (for HTTP/gRPC)
//	headers := make(map[string]string)
//	traceManager.InjectTraceContext(ctx, headers)
//
//	// Extract from headers
//	ctx = traceManager.ExtractTraceContext(ctx, headers)
//
// # Metrics Collection
//
// Use MetricsManager for recording metrics:
//
//	metricsManager, err := observability.NewMetricsManager(meter)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// ## Event Metrics
//
// **Processed Events**:
//
//	metricsManager.IncrementEventsProcessed(ctx, "task_message", "agent_echo", true)
//
// **Event Errors**:
//
//	metricsManager.IncrementEventErrors(ctx, "task_message", "agent_echo", "validation_error")
//
// **Published Events**:
//
//	metricsManager.IncrementEventsPublished(ctx, "task_completion", "broker")
//
// **Processing Duration**:
//
//	timer := metricsManager.StartTimer()
//	// ... do work ...
//	timer(ctx, "task_processing", "agent_translator")
//
// ## System Metrics
//
// **Runtime Metrics**:
//
//	metricsManager.UpdateSystemMetrics(ctx)
//
// This records:
//   - go_goroutines: Current goroutine count
//   - go_memstats_alloc_bytes: Allocated memory
//   - process_resident_memory_bytes: Resident memory size
//
// ## Available Metrics
//
// The package provides these standard metrics:
//
// **Event Metrics**:
//   - events_processed_total: Counter with labels (event_type, source, success)
//   - event_processing_duration_seconds: Histogram with labels (event_type, source)
//   - event_errors_total: Counter with labels (event_type, source, error)
//   - events_published_total: Counter with labels (event_type, destination)
//
// **System Metrics**:
//   - process_cpu_seconds_total: CPU time counter
//   - process_resident_memory_bytes: Memory gauge
//   - go_goroutines: Goroutine count gauge
//   - go_memstats_alloc_bytes: Allocated memory gauge
//
// **Broker Metrics**:
//   - message_broker_publish_duration_seconds: Publish duration histogram
//   - message_broker_consume_duration_seconds: Consume duration histogram
//   - message_broker_connection_errors_total: Connection error counter
//
// All metrics are exposed on the Prometheus endpoint (default: :9090/metrics).
//
// # Structured Logging
//
// The package provides slog-based structured logging with trace context:
//
//	logger := obs.Logger
//
//	// Context-aware logging (includes trace ID if present)
//	logger.InfoContext(ctx, "Processing task",
//	    "task_id", taskID,
//	    "agent_id", agentID,
//	)
//
//	logger.ErrorContext(ctx, "Task failed",
//	    "task_id", taskID,
//	    "error", err,
//	)
//
// ## Log Levels
//
// Configure via LogLevel in config:
//   - DEBUG: Verbose logging + stdout output
//   - INFO: Standard operation logging
//   - WARN: Warning conditions
//   - ERROR: Error conditions
//
// DEBUG mode enables dual output (observability handler + stdout).
//
// # Health Checks
//
// The package includes health check infrastructure (see healthcheck.go):
//
//	healthServer := observability.NewHealthServer(port, serviceName, version)
//
//	// Add health checkers
//	healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
//	    return nil  // Always healthy
//	}))
//
//	healthServer.AddChecker("broker", observability.NewGRPCHealthChecker("broker", "localhost:50051"))
//
//	// Start server (exposes /health and /metrics endpoints)
//	healthServer.Start(ctx)
//
// Health endpoints:
//   - GET /health: Overall health status
//   - GET /metrics: Prometheus metrics
//
// # Complete Example
//
// Here's a full example setting up observability for an agent:
//
//	func main() {
//	    // 1. Initialize observability
//	    config := observability.DefaultConfig("echo_agent")
//	    obs, err := observability.NewObservability(config)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer obs.Shutdown(context.Background())
//
//	    // 2. Create managers
//	    traceManager := observability.NewTraceManager(config.ServiceName)
//	    metricsManager, err := observability.NewMetricsManager(obs.Meter)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // 3. Setup health checks
//	    healthServer := observability.NewHealthServer("8085", config.ServiceName, config.ServiceVersion)
//	    healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
//	        return nil
//	    }))
//	    go healthServer.Start(context.Background())
//
//	    // 4. Use in application
//	    ctx := context.Background()
//	    ctx, span := traceManager.StartSpan(ctx, "process_task")
//	    defer span.End()
//
//	    timer := metricsManager.StartTimer()
//	    defer timer(ctx, "task_processing", "echo_agent")
//
//	    obs.Logger.InfoContext(ctx, "Processing task", "task_id", "task123")
//
//	    // ... do work ...
//
//	    metricsManager.IncrementEventsProcessed(ctx, "echo_task", "echo_agent", true)
//	    traceManager.SetSpanSuccess(span)
//	}
//
// # Graceful Shutdown
//
// Always shut down observability to flush traces and metrics:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	if err := obs.Shutdown(ctx); err != nil {
//	    log.Printf("Observability shutdown error: %v", err)
//	}
//
// Shutdown:
//  1. Flushes all pending traces to Jaeger
//  2. Exports final metrics to Prometheus
//  3. Closes all exporters
//  4. Releases resources
//
// Without shutdown, recent traces may be lost!
//
// # Integration with AgentHub
//
// The observability package is integrated throughout AgentHub:
//
// **In AgentHubServer** (internal/agenthub):
//
//	server, err := agenthub.NewAgentHubServer(config)
//	// Automatically includes:
//	// - server.TraceManager
//	// - server.MetricsManager
//	// - server.Logger
//	// - server.HealthServer
//
// **In AgentHubClient** (internal/agenthub):
//
//	client, err := agenthub.NewAgentHubClient(config)
//	// Automatically includes:
//	// - client.TraceManager
//	// - client.MetricsManager
//	// - client.Logger
//	// - client.HealthServer
//
// **In SubAgent** (internal/subagent):
//
//	agent, err := subagent.New(config)
//	agent.Run(ctx)
//	// Automatically wraps all handlers with:
//	// - Span creation and management
//	// - Metric recording
//	// - Structured logging
//
// # Trace Visualization
//
// View traces in Jaeger UI:
//
//	http://localhost:16686
//
// Search by:
//   - Service name (e.g., "echo_agent", "broker", "cortex")
//   - Operation name (e.g., "agent.echo_agent.handle_task")
//   - Tags (e.g., "a2a.task.id=task123")
//
// Trace structure for a typical task:
//
//	broker.publish_event (Cortex publishes task)
//	  └─ broker.route_event (Broker routes to agent)
//	      └─ agent.echo_agent.handle_task (Agent processes)
//	          └─ ... (Handler business logic)
//
// # Metrics Dashboard
//
// View metrics in Prometheus:
//
//	http://localhost:9090
//
// Example queries:
//
//	# Event processing rate
//	rate(events_processed_total[1m])
//
//	# Event error rate by type
//	rate(event_errors_total[1m])
//
//	# P95 processing duration
//	histogram_quantile(0.95, rate(event_processing_duration_seconds_bucket[5m]))
//
//	# Active goroutines
//	go_goroutines
//
// # Custom Span Attributes
//
// Add custom attributes to spans:
//
//	span.SetAttributes(
//	    attribute.String("custom.key", "value"),
//	    attribute.Int("custom.count", 42),
//	    attribute.Bool("custom.flag", true),
//	)
//
// Or use TraceManager helpers:
//
//	traceManager.AddComponentAttribute(span, "cortex_orchestrator")
//	traceManager.AddSpanEvent(span, "decision_made",
//	    attribute.String("agent", "translator"),
//	    attribute.String("reason", "best_match"),
//	)
//
// # Error Handling
//
// Observability initialization errors:
//   - OTLP endpoint unreachable: Logged but doesn't fail startup
//   - Invalid configuration: Returns error from NewObservability()
//   - Metrics creation failure: Returns error from NewMetricsManager()
//
// Runtime errors:
//   - Trace export failures: Logged via OpenTelemetry error handler
//   - Metric recording failures: Silently ignored (non-blocking)
//
// # Performance Considerations
//
// The observability package is designed for production:
//   - Asynchronous trace export (non-blocking)
//   - Efficient span attribute storage
//   - Metric aggregation before export
//   - Minimal overhead (<1ms per span)
//   - Batch trace export to reduce network calls
//   - Sampling support (currently AlwaysSample)
//
// # Thread Safety
//
// All components are thread-safe:
//   - TraceManager can be used from multiple goroutines
//   - MetricsManager is safe for concurrent use
//   - Logger is safe for concurrent use
//   - Shutdown can be called once safely
//
// # Best Practices
//
// **Always use context**:
//
//	ctx, span := traceManager.StartSpan(ctx, "operation")
//	defer span.End()
//	// Pass ctx to child operations
//
// **End spans with defer**:
//
//	ctx, span := traceManager.StartSpan(ctx, "operation")
//	defer span.End()  // Always ends, even on panic
//
// **Record errors**:
//
//	if err != nil {
//	    traceManager.RecordError(span, err)
//	    return err
//	}
//
// **Use structured logging**:
//
//	logger.InfoContext(ctx, "Message", "key", value)  // Not: fmt.Sprintf
//
// **Shutdown gracefully**:
//
//	defer obs.Shutdown(context.Background())
//
// **Name spans consistently**:
//
//	// Good: component.operation
//	"agent.translator.handle_task"
//	"broker.route_event"
//	"cortex.decide_delegation"
//
//	// Bad: Inconsistent naming
//	"handleTask"
//	"RouteEvent"
//	"decide"
//
// # Examples
//
// See the following for complete examples:
//   - broker/main.go: Broker with full observability
//   - agents/echo_agent/main.go: Agent with automatic observability
//   - agents/cortex/main.go: Orchestrator with custom tracing
//
// # Related Packages
//
//   - internal/agenthub: Uses observability for broker and client instrumentation
//   - internal/subagent: Wraps handlers with automatic observability
//   - internal/config: Provides configuration for observability settings
package observability
