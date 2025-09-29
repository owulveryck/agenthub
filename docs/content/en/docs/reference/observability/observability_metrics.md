---
title: "AgentHub Observability Metrics Reference"
weight: 40
description: "Complete catalog of all metrics exposed by AgentHub's observability system, their meanings, usage patterns, and query examples."
---

# AgentHub Observability Metrics Reference

**Technical reference**: Complete catalog of all metrics exposed by AgentHub's observability system, their meanings, usage patterns, and query examples.

## Overview

AgentHub automatically collects **47+ distinct metrics** across all observable services, providing comprehensive visibility into event processing, system health, and performance characteristics.

## Metric Categories

### Event Processing Metrics

#### `events_processed_total`
**Type**: Counter
**Description**: Total number of events processed by service
**Labels**:
- `service` - Service name (broker, publisher, subscriber)
- `event_type` - Type of event (greeting, math_calculation, etc.)
- `success` - Processing success (true/false)

**Usage**:
```promql
# Processing rate per service
rate(events_processed_total[5m])

# Success rate by event type
rate(events_processed_total{success="true"}[5m]) / rate(events_processed_total[5m]) * 100

# Error rate across all services
rate(events_processed_total{success="false"}[5m]) / rate(events_processed_total[5m]) * 100
```

#### `events_published_total`
**Type**: Counter
**Description**: Total number of events published by publisher agents
**Labels**:
- `event_type` - Type of event published
- `target_agent` - Target subscriber agent ID

**Usage**:
```promql
# Publishing rate by event type
rate(events_published_total[5m]) by (event_type)

# Most active publishers
topk(5, rate(events_published_total[5m]) by (target_agent))
```

#### `event_processing_duration_seconds`
**Type**: Histogram
**Description**: Time taken to process events
**Labels**:
- `service` - Service processing the event
- `event_type` - Type of event being processed

**Buckets**: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10

**Usage**:
```promql
# p95 processing latency
histogram_quantile(0.95, rate(event_processing_duration_seconds_bucket[5m]))

# p99 latency by service
histogram_quantile(0.99, rate(event_processing_duration_seconds_bucket[5m])) by (service)

# Average processing time
rate(event_processing_duration_seconds_sum[5m]) / rate(event_processing_duration_seconds_count[5m])
```

#### `event_errors_total`
**Type**: Counter
**Description**: Total number of event processing errors
**Labels**:
- `service` - Service where error occurred
- `event_type` - Type of event that failed
- `error_type` - Category of error (grpc_error, validation_error, etc.)

**Usage**:
```promql
# Error rate by error type
rate(event_errors_total[5m]) by (error_type)

# Services with highest error rates
topk(3, rate(event_errors_total[5m]) by (service))
```

### Broker-Specific Metrics

#### `broker_connections_total`
**Type**: Gauge
**Description**: Number of active agent connections to broker
**Labels**: None

**Usage**:
```promql
# Current connection count
broker_connections_total

# Connection growth over time
increase(broker_connections_total[1h])
```

#### `broker_subscriptions_total`
**Type**: Gauge
**Description**: Number of active event subscriptions
**Labels**:
- `agent_id` - Subscriber agent identifier
- `event_type` - Event type being subscribed to

**Usage**:
```promql
# Total active subscriptions
sum(broker_subscriptions_total)

# Subscriptions by agent
sum(broker_subscriptions_total) by (agent_id)

# Most popular event types
sum(broker_subscriptions_total) by (event_type)
```

#### `broker_message_routing_duration_seconds`
**Type**: Histogram
**Description**: Time taken to route messages through broker
**Labels**: None

**Buckets**: 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1

**Usage**:
```promql
# Broker routing latency percentiles
histogram_quantile(0.95, rate(broker_message_routing_duration_seconds_bucket[5m]))

# Routing performance over time
rate(broker_message_routing_duration_seconds_sum[5m]) / rate(broker_message_routing_duration_seconds_count[5m])
```

#### `broker_queue_size`
**Type**: Gauge
**Description**: Number of queued messages awaiting routing
**Labels**:
- `queue_type` - Type of queue (incoming, outgoing, dead_letter)

**Usage**:
```promql
# Current queue sizes
broker_queue_size by (queue_type)

# Queue growth rate
rate(broker_queue_size[5m])
```

### System Health Metrics

#### `system_cpu_usage_percent`
**Type**: Gauge
**Description**: CPU utilization percentage
**Labels**:
- `service` - Service name

**Usage**:
```promql
# Current CPU usage
system_cpu_usage_percent

# High CPU services
system_cpu_usage_percent > 80

# Average CPU over time
avg_over_time(system_cpu_usage_percent[1h])
```

#### `system_memory_usage_bytes`
**Type**: Gauge
**Description**: Memory usage in bytes
**Labels**:
- `service` - Service name
- `type` - Memory type (heap, stack, total)

**Usage**:
```promql
# Memory usage in MB
system_memory_usage_bytes / 1024 / 1024

# Memory growth rate
rate(system_memory_usage_bytes[10m])

# Memory usage by type
system_memory_usage_bytes by (type)
```

#### `system_goroutines_total`
**Type**: Gauge
**Description**: Number of active goroutines
**Labels**:
- `service` - Service name

**Usage**:
```promql
# Current goroutine count
system_goroutines_total

# Goroutine leaks detection
increase(system_goroutines_total[1h]) > 1000

# Goroutine efficiency
system_goroutines_total / system_cpu_usage_percent
```

#### `system_file_descriptors_used`
**Type**: Gauge
**Description**: Number of open file descriptors
**Labels**:
- `service` - Service name

**Usage**:
```promql
# Current FD usage
system_file_descriptors_used

# FD growth rate
rate(system_file_descriptors_used[5m])
```

### gRPC Metrics

#### `grpc_server_started_total`
**Type**: Counter
**Description**: Total number of RPCs started on the server
**Labels**:
- `grpc_method` - gRPC method name
- `grpc_service` - gRPC service name

**Usage**:
```promql
# RPC request rate
rate(grpc_server_started_total[5m])

# Most called methods
topk(5, rate(grpc_server_started_total[5m]) by (grpc_method))
```

#### `grpc_server_handled_total`
**Type**: Counter
**Description**: Total number of RPCs completed on the server
**Labels**:
- `grpc_method` - gRPC method name
- `grpc_service` - gRPC service name
- `grpc_code` - gRPC status code

**Usage**:
```promql
# RPC success rate
rate(grpc_server_handled_total{grpc_code="OK"}[5m]) / rate(grpc_server_handled_total[5m]) * 100

# Error rate by method
rate(grpc_server_handled_total{grpc_code!="OK"}[5m]) by (grpc_method)
```

#### `grpc_server_handling_seconds`
**Type**: Histogram
**Description**: Histogram of response latency of RPCs
**Labels**:
- `grpc_method` - gRPC method name
- `grpc_service` - gRPC service name

**Usage**:
```promql
# gRPC latency percentiles
histogram_quantile(0.95, rate(grpc_server_handling_seconds_bucket[5m]))

# Slow methods
histogram_quantile(0.95, rate(grpc_server_handling_seconds_bucket[5m])) by (grpc_method) > 0.1
```

### Health Check Metrics

#### `health_check_status`
**Type**: Gauge
**Description**: Health check status (1=healthy, 0=unhealthy)
**Labels**:
- `service` - Service name
- `check_name` - Name of the health check
- `endpoint` - Health check endpoint

**Usage**:
```promql
# Unhealthy services
health_check_status == 0

# Health check success rate
avg_over_time(health_check_status[5m])
```

#### `health_check_duration_seconds`
**Type**: Histogram
**Description**: Time taken to execute health checks
**Labels**:
- `service` - Service name
- `check_name` - Name of the health check

**Usage**:
```promql
# Health check latency
histogram_quantile(0.95, rate(health_check_duration_seconds_bucket[5m]))

# Slow health checks
histogram_quantile(0.95, rate(health_check_duration_seconds_bucket[5m])) by (check_name) > 0.5
```

### OpenTelemetry Metrics

#### `otelcol_processor_batch_batch_send_size_count`
**Type**: Counter
**Description**: Number of batches sent by OTEL collector
**Labels**: None

#### `otelcol_exporter_sent_spans`
**Type**: Counter
**Description**: Number of spans sent to tracing backend
**Labels**:
- `exporter` - Exporter name (jaeger, otlp)

**Usage**:
```promql
# Span export rate
rate(otelcol_exporter_sent_spans[5m])

# Export success by backend
rate(otelcol_exporter_sent_spans[5m]) by (exporter)
```

## Common Query Patterns

### Performance Analysis

```promql
# Top 5 slowest event types
topk(5,
  histogram_quantile(0.95,
    rate(event_processing_duration_seconds_bucket[5m])
  ) by (event_type)
)

# Services exceeding latency SLA (>500ms p95)
histogram_quantile(0.95,
  rate(event_processing_duration_seconds_bucket[5m])
) by (service) > 0.5

# Throughput efficiency (events per CPU percent)
rate(events_processed_total[5m]) / system_cpu_usage_percent
```

### Error Analysis

```promql
# Error rate by service over time
rate(event_errors_total[5m]) / rate(events_processed_total[5m]) * 100

# Most common error types
topk(5, rate(event_errors_total[5m]) by (error_type))

# Services with increasing error rates
increase(event_errors_total[1h]) by (service) > 10
```

### Capacity Planning

```promql
# Peak hourly throughput
max_over_time(
  rate(events_processed_total[5m])[1h:]
) * 3600

# Resource utilization during peak load
(
  max_over_time(system_cpu_usage_percent[1h:]) +
  max_over_time(system_memory_usage_bytes[1h:] / 1024 / 1024 / 1024)
) by (service)

# Connection scaling needs
max_over_time(broker_connections_total[24h:])
```

### System Health

```promql
# Overall system health score (0-1)
avg(health_check_status)

# Services with degraded performance
(
  system_cpu_usage_percent > 70 or
  system_memory_usage_bytes > 1e9 or
  rate(event_errors_total[5m]) / rate(events_processed_total[5m]) > 0.05
)

# Resource leak detection
increase(system_goroutines_total[1h]) > 1000 or
increase(system_file_descriptors_used[1h]) > 100
```

## Alert Rule Examples

### Critical Alerts

```yaml
# High error rate alert
- alert: HighEventProcessingErrorRate
  expr: |
    (
      rate(event_errors_total[5m]) /
      rate(events_processed_total[5m])
    ) * 100 > 10
  for: 2m
  annotations:
    summary: "High event processing error rate"
    description: "{{ $labels.service }} has {{ $value }}% error rate"

# Service down alert
- alert: ServiceDown
  expr: health_check_status == 0
  for: 1m
  annotations:
    summary: "Service health check failing"
    description: "{{ $labels.service }} health check {{ $labels.check_name }} is failing"
```

### Warning Alerts

```yaml
# High latency warning
- alert: HighEventProcessingLatency
  expr: |
    histogram_quantile(0.95,
      rate(event_processing_duration_seconds_bucket[5m])
    ) > 0.5
  for: 5m
  annotations:
    summary: "High event processing latency"
    description: "{{ $labels.service }} p95 latency is {{ $value }}s"

# High CPU usage warning
- alert: HighCPUUsage
  expr: system_cpu_usage_percent > 80
  for: 5m
  annotations:
    summary: "High CPU usage"
    description: "{{ $labels.service }} CPU usage is {{ $value }}%"
```

## Metric Retention and Storage

### Retention Policies
- **Raw metrics**: 15 days at 15-second resolution
- **5m averages**: 60 days
- **1h averages**: 1 year
- **1d averages**: 5 years

### Storage Requirements
- **Per service**: ~2MB/day for all metrics
- **Complete system**: ~10MB/day for 5 services
- **1 year retention**: ~3.6GB total

### Performance Considerations
- **Scrape interval**: 10 seconds (configurable)
- **Evaluation interval**: 15 seconds for alerts
- **Query timeout**: 30 seconds
- **Max samples**: 50M per query

## Integration Examples

### Grafana Dashboard Variables

```json
{
  "service": {
    "query": "label_values(events_processed_total, service)",
    "refresh": "on_time_range_changed"
  },
  "event_type": {
    "query": "label_values(events_processed_total{service=\"$service\"}, event_type)",
    "refresh": "on_dashboard_load"
  }
}
```

### Custom Application Metrics

```go
// Register custom counter
customCounter, err := meter.Int64Counter(
    "my_business_metric_total",
    metric.WithDescription("Custom business metric"),
)

// Increment with context and labels
customCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("department", "sales"),
    attribute.String("region", "us-west"),
))
```

## Troubleshooting Metrics

### Missing Metrics Checklist
1. ✅ Service built with `-tags observability`
2. ✅ Prometheus can reach metrics endpoint
3. ✅ Correct port in Prometheus config
4. ✅ Service is actually processing events
5. ✅ OpenTelemetry exporter configured correctly

### High Cardinality Warning
Avoid metrics with unbounded label values:
- ❌ User IDs as labels (millions of values)
- ❌ Timestamps as labels
- ❌ Request IDs as labels
- ✅ Event types (limited set)
- ✅ Service names (limited set)
- ✅ Status codes (limited set)

---

**🎯 Next Steps**:

**Implementation**: **[Add Observability to Your Agent](../howto/add_observability.md)**

**Monitoring**: **[Use Grafana Dashboards](../howto/use_dashboards.md)**

**Understanding**: **[Distributed Tracing Explained](../explanation/distributed_tracing.md)**