# Complete Event-Driven Architecture (EDA) System Observability Specification

## 1. Architecture Overview

This specification defines a comprehensive observability stack for a distributed Event-Driven Architecture (EDA) system built in Go, providing end-to-end visibility across all agents and event flows.

### 1.1 System Components
- **Agents**: Distributed Go services that produce and consume events
- **Message Broker**: Event transport layer (Kafka, RabbitMQ, etc.)
- **Observability Stack**: Monitoring, tracing, and logging infrastructure
- **Dashboard Layer**: Visualization and alerting interface

### 1.2 Observability Pillars
1. **Metrics**: Quantitative measurements (Prometheus + OpenTelemetry)
2. **Traces**: Request flow tracking (Jaeger + OpenTelemetry)
3. **Logs**: Structured event logging (slog with trace correlation)

## 2. Technical Requirements

### 2.1 Go Dependencies
```go
// Core observability
"go.opentelemetry.io/otel" v1.21.0+
"go.opentelemetry.io/otel/trace" v1.21.0+
"go.opentelemetry.io/otel/metric" v1.21.0+
"go.opentelemetry.io/otel/propagation" v1.21.0+

// Exporters
"go.opentelemetry.io/otel/exporters/jaeger" v1.21.0+
"go.opentelemetry.io/otel/exporters/prometheus" v0.44.0+

// Standard library
"log/slog" (Go 1.21+)
"context"

// Metrics
"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promhttp"

// Message brokers (choose one)
"github.com/IBM/sarama" // Kafka
"github.com/rabbitmq/amqp091-go" // RabbitMQ
```

### 2.2 Infrastructure Components
```yaml
# Required Services
- Prometheus: v2.40.0+
- Grafana: v9.0.0+
- Jaeger: all-in-one latest
- OpenTelemetry Collector: contrib latest

# Optional Services
- Elasticsearch: v8.11.0+ (for log aggregation)
- Kibana: v8.11.0+ (for log visualization)
- AlertManager: v0.25.0+ (for alerting)
```

## 3. Implementation Specification

### 3.1 Event Data Structure
See the @proto definition

### 3.2 Custom Logging Handler
**Directive**: All agents MUST implement a custom `slog.Handler` that:
- Automatically injects trace context (trace_id, span_id) into every log entry
- Supports structured logging with event correlation
- Provides fallback mechanisms for broker failures
- Implements non-blocking buffered event posting

**Implementation Requirements**:
```go
type ObservabilityHandler struct {
    logger       *slog.Logger
    tracer       oteltrace.Tracer
    meter        metric.Meter
    eventCounter    metric.Int64Counter
    eventDuration   metric.Float64Histogram
    eventErrors     metric.Int64Counter
    postEvent func(event EventData) error
}
```

### 3.3 Required Metrics
All agents MUST expose the following Prometheus metrics:

#### Event Metrics
- `events_processed_total{event_type, source, success}` - Counter
- `event_processing_duration_seconds{event_type, source}` - Histogram
- `event_errors_total{event_type, source, error}` - Counter
- `events_published_total{event_type, destination}` - Counter

#### System Metrics
- `process_cpu_seconds_total` - Counter
- `process_resident_memory_bytes` - Gauge
- `go_goroutines` - Gauge
- `go_memstats_alloc_bytes` - Gauge

#### Message Broker Metrics
- `message_broker_publish_duration_seconds{topic}` - Histogram
- `message_broker_consume_duration_seconds{topic}` - Histogram
- `message_broker_connection_errors_total` - Counter

### 3.4 Distributed Tracing Requirements
**Directive**: All agents MUST implement distributed tracing with:
- OpenTelemetry SDK integration
- Context propagation through event headers
- Span creation for all event processing operations
- Error recording and status setting

#### Required Span Attributes
```go
// Event processing spans
attribute.String("event.id", event.ID)
attribute.String("event.type", event.Type)
attribute.String("event.source", event.Source)
attribute.String("event.subject", event.Subject)

// Message broker spans
attribute.String("messaging.system", "kafka")
attribute.String("messaging.destination", topic)
attribute.String("messaging.operation", "publish|receive")
```

### 3.5 Log Structure Requirements
**Directive**: All logs MUST include:
```json
{
  "time": "2025-09-28T10:00:00Z",
  "level": "INFO|WARN|ERROR|DEBUG",
  "msg": "Human readable message",
  "trace_id": "hex-encoded-trace-id",
  "span_id": "hex-encoded-span-id",
  "service": "service-name",
  "event_id": "event-identifier",
  "event_type": "event.type",
  "duration_ms": 150,
  "source": "file:line"
}
```

## 4. Dashboard Specification

### 4.1 Grafana Dashboard Requirements
**Directive**: The observability dashboard MUST include the following panels:

#### Primary Event Metrics (Top Row)
1. **Event Processing Rate**: Time series showing events/sec by type and service
2. **Event Processing Errors**: Stat panel showing error rate with thresholds
3. **Event Types Distribution**: Pie chart of event volume by type
4. **Event Processing Latency**: Time series with 50th, 95th, 99th percentiles

#### Distributed Tracing (Middle Section)
5. **Trace Timeline**: Jaeger panel showing end-to-end traces
6. **Service Map**: Visual representation of service dependencies
7. **Trace Duration Distribution**: Histogram of trace durations

#### System Health (Bottom Row)
8. **Service CPU Usage**: Time series per service
9. **Service Memory Usage**: Time series per service
10. **Go Goroutines**: Time series per service
11. **Message Broker Health**: Connection status and throughput

### 4.2 Dashboard Variables
```yaml
Variables:
  - service: Multi-select from label_values(events_processed_total, service)
  - event_type: Multi-select from label_values(events_processed_total, event_type)
  - time_range: 5m, 15m, 1h, 6h, 24h, 7d
```

### 4.3 Alert Rules
**Directive**: The following alerts MUST be configured:

#### Critical Alerts
- Event processing error rate > 5% for 5 minutes
- Event processing latency p95 > 5 seconds for 10 minutes
- Service memory usage > 80% for 15 minutes
- Message broker connection failures > 0 for 2 minutes

#### Warning Alerts
- Event processing rate drops by 50% for 10 minutes
- Service CPU usage > 70% for 20 minutes
- Trace error rate > 1% for 15 minutes

## 5. Deployment Specification

### 5.1 Docker Compose Configuration
**Directive**: Use the provided docker-compose.yml with these mandatory services:
- Jaeger (ports: 16686, 14268, 4317, 4318)
- Prometheus (port: 9090)
- Grafana (port: 3000)
- OpenTelemetry Collector (ports: 4317, 4318, 8888, 8889)

### 5.2 Configuration Files Structure
```
observability/
├── docker-compose.yml
├── prometheus/
│   ├── prometheus.yml
│   └── alert_rules.yml
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   └── datasources.yml
│   │   └── dashboards/
│   │       └── dashboards.yml
│   └── dashboards/
│       └── eda-system-dashboard.json
└── otel-collector/
    └── otel-collector.yml
```

### 5.3 Agent Configuration
**Directive**: Each agent MUST:
1. Expose metrics on `/metrics` endpoint
2. Configure OpenTelemetry with service name and version
3. Use consistent port allocation (8080, 8081, 8082, etc.)
4. Implement graceful shutdown for observability components

## 6. Event Flow Tracing Specification

### 6.1 Trace Propagation
**Directive**: When agent1 publishes an event consumed by agent2:

```go
// Agent1 (Publisher)
ctx, span := tracer.Start(ctx, "publish_event")
defer span.End()

// Inject trace context into event headers
otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(event.Headers))

// Publish event with headers
broker.Publish(ctx, topic, event)

// Agent2 (Consumer)  
// Extract trace context from event headers
ctx := otel.GetTextMapPropagator().Extract(context.Background(), 
    propagation.MapCarrier(event.Headers))

// Continue trace
ctx, span := tracer.Start(ctx, "process_event")
defer span.End()
```

### 6.2 Event Chain Visualization
**Directive**: The dashboard MUST show:
- Complete event lineage (agent1 → agent2 → agent3)
- Processing time at each hop
- Error correlation across the chain
- Parallel event processing visualization

## 7. Performance Requirements

### 7.1 Observability Overhead
**Directive**: The observability stack MUST NOT:
- Add more than 5% CPU overhead to agents
- Add more than 50MB memory overhead per agent
- Block event processing (use async/buffered approaches)
- Impact event processing latency by more than 10ms

### 7.2 Data Retention
- **Metrics**: 30 days (configurable)
- **Traces**: 7 days (configurable)
- **Logs**: 14 days (configurable)

### 7.3 Scalability Requirements
- Support up to 100 agents
- Handle 10,000+ events/second per agent
- Process 1M+ traces per day
- Support 50+ concurrent Grafana users

## 8. Security Considerations

### 8.1 Access Control
**Directive**: Implement:
- Grafana authentication (LDAP/OAuth preferred)
- Prometheus query restrictions
- Jaeger UI access controls
- Network segmentation for observability components

### 8.2 Data Privacy
- No sensitive data in event logs or traces
- PII masking in structured logs
- Secure communication between components (TLS recommended)

## 9. Operational Procedures

### 9.1 Health Checks
**Directive**: Each component MUST expose:
- `/health` endpoint for application health
- `/ready` endpoint for readiness checks
- `/metrics` endpoint for Prometheus scraping

### 9.2 Backup and Recovery
- Prometheus data backup strategy
- Grafana dashboard backup/restore
- Configuration version control

### 9.3 Troubleshooting Runbook
1. **High Error Rate**: Check Jaeger traces for root cause
2. **High Latency**: Analyze processing duration histograms
3. **Missing Events**: Verify broker connectivity and traces
4. **Dashboard Issues**: Check data source connectivity

## 10. Testing and Validation

### 10.1 Integration Tests
**Directive**: Implement tests for:
- Trace context propagation across agents
- Metric accuracy and labeling
- Log correlation with traces
- Dashboard query functionality

### 10.2 Load Testing
- Event processing under load
- Observability stack performance
- Dashboard responsiveness under concurrent users

## 11. Maintenance and Updates

### 11.1 Monitoring the Monitors
- Prometheus self-monitoring
- Grafana availability checks  
- Jaeger query performance
- OpenTelemetry Collector health

### 11.2 Regular Tasks
- Clean up old traces and metrics
- Update dashboard queries
- Review and tune alert thresholds
- Security patching of observability stack

---

## Implementation Checklist

- [ ] Deploy observability infrastructure (docker-compose)
- [ ] Implement custom slog handler with tracing
- [ ] Add OpenTelemetry to all agents
- [ ] Configure Prometheus metric collection
- [ ] Set up Grafana dashboards and alerts
- [ ] Implement event trace propagation
- [ ] Test end-to-end event flow tracing
- [ ] Configure backup and monitoring procedures
- [ ] Validate performance requirements
- [ ] Document troubleshooting procedures

**Success Criteria**: When agent1 generates an event processed by agent2, you can:
1. See the complete trace in Jaeger with timing
2. View correlated logs with matching trace_id
3. Monitor metrics showing event flow rates
4. Receive alerts for processing errors
5. Analyze performance bottlenecks in Grafana
