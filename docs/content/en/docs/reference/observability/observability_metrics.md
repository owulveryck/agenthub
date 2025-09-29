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

### A2A Message Processing Metrics

#### `a2a_messages_processed_total`
**Type**: Counter
**Description**: Total number of A2A messages processed by service
**Labels**:
- `service` - Service name (agenthub, publisher, subscriber)
- `message_type` - Type of A2A message (task_update, message, artifact)
- `success` - Processing success (true/false)
- `context_id` - A2A conversation context (for workflow tracking)

**Usage**:
```promql
# A2A message processing rate per service
rate(a2a_messages_processed_total[5m])

# Success rate by A2A message type
rate(a2a_messages_processed_total{success="true"}[5m]) / rate(a2a_messages_processed_total[5m]) * 100

# Error rate across all A2A services
rate(a2a_messages_processed_total{success="false"}[5m]) / rate(a2a_messages_processed_total[5m]) * 100

# Workflow processing rate by context
rate(a2a_messages_processed_total[5m]) by (context_id)
```

#### `a2a_messages_published_total`
**Type**: Counter
**Description**: Total number of A2A messages published by agents
**Labels**:
- `message_type` - Type of A2A message published
- `from_agent_id` - Publishing agent identifier
- `to_agent_id` - Target agent identifier (empty for broadcast)

**Usage**:
```promql
# A2A publishing rate by message type
rate(a2a_messages_published_total[5m]) by (message_type)

# Most active A2A publishers
topk(5, rate(a2a_messages_published_total[5m]) by (from_agent_id))

# Broadcast vs direct messaging ratio
rate(a2a_messages_published_total{to_agent_id=""}[5m]) / rate(a2a_messages_published_total[5m])
```

#### `a2a_message_processing_duration_seconds`
**Type**: Histogram
**Description**: Time taken to process A2A messages
**Labels**:
- `service` - Service processing the message
- `message_type` - Type of A2A message being processed
- `task_state` - Current A2A task state (for task-related messages)

**Buckets**: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10

**Usage**:
```promql
# p95 A2A message processing latency
histogram_quantile(0.95, rate(a2a_message_processing_duration_seconds_bucket[5m]))

# p99 latency by service
histogram_quantile(0.99, rate(a2a_message_processing_duration_seconds_bucket[5m])) by (service)

# Average A2A processing time by task state
rate(a2a_message_processing_duration_seconds_sum[5m]) / rate(a2a_message_processing_duration_seconds_count[5m]) by (task_state)
```

#### `a2a_message_errors_total`
**Type**: Counter
**Description**: Total number of A2A message processing errors
**Labels**:
- `service` - Service where error occurred
- `message_type` - Type of A2A message that failed
- `error_type` - Category of error (grpc_error, validation_error, protocol_error, etc.)
- `a2a_version` - A2A protocol version for compatibility tracking

**Usage**:
```promql
# A2A error rate by error type
rate(a2a_message_errors_total[5m]) by (error_type)

# Services with highest A2A error rates
topk(3, rate(a2a_message_errors_total[5m]) by (service))

# A2A protocol version compatibility issues
rate(a2a_message_errors_total{error_type="protocol_error"}[5m]) by (a2a_version)
```

### AgentHub Broker Metrics

#### `agenthub_connections_total`
**Type**: Gauge
**Description**: Number of active agent connections to AgentHub broker
**Labels**:
- `connection_type` - Type of connection (a2a_publisher, a2a_subscriber, unified)
- `agent_type` - Classification of connected agent

**Usage**:
```promql
# Current AgentHub connection count
agenthub_connections_total

# A2A connection growth over time
increase(agenthub_connections_total[1h])

# Connection distribution by type
agenthub_connections_total by (connection_type)
```

#### `agenthub_subscriptions_total`
**Type**: Gauge
**Description**: Number of active A2A message subscriptions
**Labels**:
- `agent_id` - Subscriber agent identifier
- `subscription_type` - Type of A2A subscription (tasks, messages, agent_events)
- `filter_criteria` - Applied subscription filters (task_types, states, etc.)

**Usage**:
```promql
# Total active A2A subscriptions
sum(agenthub_subscriptions_total)

# A2A subscriptions by agent
sum(agenthub_subscriptions_total) by (agent_id)

# Most popular A2A subscription types
sum(agenthub_subscriptions_total) by (subscription_type)

# Filtered vs unfiltered subscriptions
sum(agenthub_subscriptions_total{filter_criteria!=""}) / sum(agenthub_subscriptions_total)
```

#### `agenthub_message_routing_duration_seconds`
**Type**: Histogram
**Description**: Time taken to route A2A messages through AgentHub broker
**Labels**:
- `routing_type` - Type of routing (direct, broadcast, filtered)
- `message_size_bucket` - Message size classification (small, medium, large)

**Buckets**: 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1

**Usage**:
```promql
# AgentHub A2A routing latency percentiles
histogram_quantile(0.95, rate(agenthub_message_routing_duration_seconds_bucket[5m]))

# A2A routing performance by type
rate(agenthub_message_routing_duration_seconds_sum[5m]) / rate(agenthub_message_routing_duration_seconds_count[5m]) by (routing_type)

# Message size impact on routing
histogram_quantile(0.95, rate(agenthub_message_routing_duration_seconds_bucket[5m])) by (message_size_bucket)
```

#### `agenthub_queue_size`
**Type**: Gauge
**Description**: Number of A2A messages queued awaiting routing
**Labels**:
- `queue_type` - Type of queue (incoming, outgoing, dead_letter, retry)
- `priority` - Message priority level
- `context_active` - Whether messages belong to active A2A contexts

**Usage**:
```promql
# Current A2A queue sizes
agenthub_queue_size by (queue_type)

# A2A queue growth rate
rate(agenthub_queue_size[5m])

# Priority queue distribution
agenthub_queue_size by (priority)

# Active context message backlog
agenthub_queue_size{context_active="true"}
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

### A2A Task-Specific Metrics

#### `a2a_tasks_created_total`
**Type**: Counter
**Description**: Total number of A2A tasks created
**Labels**:
- `task_type` - Type classification of the task
- `context_id` - A2A conversation context
- `priority` - Task priority level

**Usage**:
```promql
# A2A task creation rate
rate(a2a_tasks_created_total[5m])

# Task creation by type
rate(a2a_tasks_created_total[5m]) by (task_type)

# High priority task rate
rate(a2a_tasks_created_total{priority="PRIORITY_HIGH"}[5m])
```

#### `a2a_task_state_transitions_total`
**Type**: Counter
**Description**: Total number of A2A task state transitions
**Labels**:
- `from_state` - Previous task state
- `to_state` - New task state
- `task_type` - Type of task transitioning

**Usage**:
```promql
# Task completion rate
rate(a2a_task_state_transitions_total{to_state="TASK_STATE_COMPLETED"}[5m])

# Task failure rate
rate(a2a_task_state_transitions_total{to_state="TASK_STATE_FAILED"}[5m])

# Task state transition patterns
rate(a2a_task_state_transitions_total[5m]) by (from_state, to_state)
```

#### `a2a_task_duration_seconds`
**Type**: Histogram
**Description**: Duration of A2A task execution from submission to completion
**Labels**:
- `task_type` - Type of task
- `final_state` - Final task state (COMPLETED, FAILED, CANCELLED)

**Buckets**: 0.1, 0.5, 1, 5, 10, 30, 60, 300, 600, 1800

**Usage**:
```promql
# A2A task completion time percentiles
histogram_quantile(0.95, rate(a2a_task_duration_seconds_bucket{final_state="TASK_STATE_COMPLETED"}[5m]))

# Task duration by type
histogram_quantile(0.50, rate(a2a_task_duration_seconds_bucket[5m])) by (task_type)

# Failed vs successful task duration comparison
histogram_quantile(0.95, rate(a2a_task_duration_seconds_bucket[5m])) by (final_state)
```

#### `a2a_artifacts_produced_total`
**Type**: Counter
**Description**: Total number of A2A artifacts produced by completed tasks
**Labels**:
- `artifact_type` - Type of artifact (data, file, text)
- `task_type` - Type of task that produced the artifact
- `artifact_size_bucket` - Size classification of artifact

**Usage**:
```promql
# Artifact production rate
rate(a2a_artifacts_produced_total[5m])

# Artifacts by type
rate(a2a_artifacts_produced_total[5m]) by (artifact_type)

# Large artifact production rate
rate(a2a_artifacts_produced_total{artifact_size_bucket="large"}[5m])
```

### gRPC Metrics

#### `grpc_server_started_total`
**Type**: Counter
**Description**: Total number of RPCs started on the AgentHub server
**Labels**:
- `grpc_method` - gRPC method name (PublishMessage, SubscribeToTasks, etc.)
- `grpc_service` - gRPC service name (AgentHub)

**Usage**:
```promql
# AgentHub RPC request rate
rate(grpc_server_started_total[5m])

# Most called A2A methods
topk(5, rate(grpc_server_started_total[5m]) by (grpc_method))

# A2A vs EDA method usage
rate(grpc_server_started_total{grpc_method=~".*Message.*|.*Task.*"}[5m])
```

#### `grpc_server_handled_total`
**Type**: Counter
**Description**: Total number of RPCs completed on the AgentHub server
**Labels**:
- `grpc_method` - gRPC method name
- `grpc_service` - gRPC service name (AgentHub)
- `grpc_code` - gRPC status code
- `a2a_operation` - A2A operation type (publish, subscribe, get, cancel)

**Usage**:
```promql
# AgentHub RPC success rate
rate(grpc_server_handled_total{grpc_code="OK"}[5m]) / rate(grpc_server_handled_total[5m]) * 100

# A2A operation error rate
rate(grpc_server_handled_total{grpc_code!="OK"}[5m]) by (a2a_operation)

# A2A method-specific success rates
rate(grpc_server_handled_total{grpc_code="OK"}[5m]) / rate(grpc_server_handled_total[5m]) by (grpc_method)
```

#### `grpc_server_handling_seconds`
**Type**: Histogram
**Description**: Histogram of response latency of AgentHub RPCs
**Labels**:
- `grpc_method` - gRPC method name
- `grpc_service` - gRPC service name (AgentHub)
- `a2a_operation` - A2A operation type

**Usage**:
```promql
# AgentHub gRPC latency percentiles
histogram_quantile(0.95, rate(grpc_server_handling_seconds_bucket[5m]))

# Slow A2A operations
histogram_quantile(0.95, rate(grpc_server_handling_seconds_bucket[5m])) by (a2a_operation) > 0.1

# A2A method performance comparison
histogram_quantile(0.95, rate(grpc_server_handling_seconds_bucket[5m])) by (grpc_method)
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

### A2A Performance Analysis

```promql
# Top 5 slowest A2A message types
topk(5,
  histogram_quantile(0.95,
    rate(a2a_message_processing_duration_seconds_bucket[5m])
  ) by (message_type)
)

# A2A task completion time analysis
histogram_quantile(0.95,
  rate(a2a_task_duration_seconds_bucket{final_state="TASK_STATE_COMPLETED"}[5m])
) by (task_type)

# Services exceeding A2A latency SLA (>500ms p95)
histogram_quantile(0.95,
  rate(a2a_message_processing_duration_seconds_bucket[5m])
) by (service) > 0.5

# A2A throughput efficiency (messages per CPU percent)
rate(a2a_messages_processed_total[5m]) / system_cpu_usage_percent

# Task success rate by type
rate(a2a_task_state_transitions_total{to_state="TASK_STATE_COMPLETED"}[5m]) /
rate(a2a_tasks_created_total[5m]) by (task_type)
```

### A2A Error Analysis

```promql
# A2A message error rate by service over time
rate(a2a_message_errors_total[5m]) / rate(a2a_messages_processed_total[5m]) * 100

# A2A task failure rate
rate(a2a_task_state_transitions_total{to_state="TASK_STATE_FAILED"}[5m]) /
rate(a2a_tasks_created_total[5m]) * 100

# Most common A2A error types
topk(5, rate(a2a_message_errors_total[5m]) by (error_type))

# A2A protocol compatibility issues
rate(a2a_message_errors_total{error_type="protocol_error"}[5m]) by (a2a_version)

# Services with increasing A2A error rates
increase(a2a_message_errors_total[1h]) by (service) > 10
```

### A2A Capacity Planning

```promql
# Peak hourly A2A message throughput
max_over_time(
  rate(a2a_messages_processed_total[5m])[1h:]
) * 3600

# Peak A2A task creation rate
max_over_time(
  rate(a2a_tasks_created_total[5m])[1h:]
) * 3600

# Resource utilization during peak A2A load
(
  max_over_time(system_cpu_usage_percent[1h:]) +
  max_over_time(system_memory_usage_bytes[1h:] / 1024 / 1024 / 1024)
) by (service)

# AgentHub connection scaling needs
max_over_time(agenthub_connections_total[24h:])

# A2A queue depth trends
max_over_time(agenthub_queue_size[24h:]) by (queue_type)
```

### A2A System Health

```promql
# Overall A2A system health score (0-1)
avg(health_check_status)

# A2A services with degraded performance
(
  system_cpu_usage_percent > 70 or
  system_memory_usage_bytes > 1e9 or
  rate(a2a_message_errors_total[5m]) / rate(a2a_messages_processed_total[5m]) > 0.05
)

# A2A task backlog health
agenthub_queue_size{queue_type="incoming"} > 1000

# A2A protocol health indicators
rate(a2a_task_state_transitions_total{to_state="TASK_STATE_FAILED"}[5m]) /
rate(a2a_tasks_created_total[5m]) > 0.1

# Resource leak detection
increase(system_goroutines_total[1h]) > 1000 or
increase(system_file_descriptors_used[1h]) > 100
```

## Alert Rule Examples

### Critical A2A Alerts

```yaml
# High A2A message processing error rate alert
- alert: HighA2AMessageProcessingErrorRate
  expr: |
    (
      rate(a2a_message_errors_total[5m]) /
      rate(a2a_messages_processed_total[5m])
    ) * 100 > 10
  for: 2m
  annotations:
    summary: "High A2A message processing error rate"
    description: "{{ $labels.service }} has {{ $value }}% A2A error rate"

# High A2A task failure rate alert
- alert: HighA2ATaskFailureRate
  expr: |
    (
      rate(a2a_task_state_transitions_total{to_state="TASK_STATE_FAILED"}[5m]) /
      rate(a2a_tasks_created_total[5m])
    ) * 100 > 15
  for: 3m
  annotations:
    summary: "High A2A task failure rate"
    description: "{{ $value }}% of A2A tasks are failing for task type {{ $labels.task_type }}"

# AgentHub service down alert
- alert: AgentHubServiceDown
  expr: health_check_status == 0
  for: 1m
  annotations:
    summary: "AgentHub service health check failing"
    description: "{{ $labels.service }} health check {{ $labels.check_name }} is failing"

# A2A queue backlog alert
- alert: A2AQueueBacklog
  expr: agenthub_queue_size{queue_type="incoming"} > 1000
  for: 5m
  annotations:
    summary: "A2A message queue backlog"
    description: "AgentHub has {{ $value }} messages queued"
```

### A2A Warning Alerts

```yaml
# High A2A message processing latency warning
- alert: HighA2AMessageProcessingLatency
  expr: |
    histogram_quantile(0.95,
      rate(a2a_message_processing_duration_seconds_bucket[5m])
    ) > 0.5
  for: 5m
  annotations:
    summary: "High A2A message processing latency"
    description: "{{ $labels.service }} A2A p95 latency is {{ $value }}s"

# Slow A2A task completion warning
- alert: SlowA2ATaskCompletion
  expr: |
    histogram_quantile(0.95,
      rate(a2a_task_duration_seconds_bucket{final_state="TASK_STATE_COMPLETED"}[5m])
    ) > 300
  for: 10m
  annotations:
    summary: "Slow A2A task completion"
    description: "A2A tasks of type {{ $labels.task_type }} taking {{ $value }}s to complete"

# High CPU usage warning
- alert: HighCPUUsage
  expr: system_cpu_usage_percent > 80
  for: 5m
  annotations:
    summary: "High CPU usage"
    description: "{{ $labels.service }} CPU usage is {{ $value }}%"

# A2A protocol version compatibility warning
- alert: A2AProtocolVersionMismatch
  expr: |
    rate(a2a_message_errors_total{error_type="protocol_error"}[5m]) > 0.1
  for: 3m
  annotations:
    summary: "A2A protocol version compatibility issues"
    description: "A2A protocol errors detected for version {{ $labels.a2a_version }}"
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
    "query": "label_values(a2a_messages_processed_total, service)",
    "refresh": "on_time_range_changed"
  },
  "message_type": {
    "query": "label_values(a2a_messages_processed_total{service=\"$service\"}, message_type)",
    "refresh": "on_dashboard_load"
  },
  "task_type": {
    "query": "label_values(a2a_tasks_created_total, task_type)",
    "refresh": "on_dashboard_load"
  },
  "context_id": {
    "query": "label_values(a2a_messages_processed_total{service=\"$service\"}, context_id)",
    "refresh": "on_dashboard_load"
  }
}
```

### Custom A2A Application Metrics

```go
// Register custom A2A counter
a2aCustomCounter, err := meter.Int64Counter(
    "a2a_custom_business_metric_total",
    metric.WithDescription("Custom A2A business metric"),
)

// Increment with A2A context and labels
a2aCustomCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("task_type", "custom_analysis"),
    attribute.String("context_id", contextID),
    attribute.String("agent_type", "analytics_agent"),
    attribute.String("a2a_version", "1.0"),
))

// Register A2A task-specific histogram
a2aTaskHistogram, err := meter.Float64Histogram(
    "a2a_custom_task_processing_seconds",
    metric.WithDescription("Custom A2A task processing time"),
    metric.WithUnit("s"),
)

// Record A2A task timing
start := time.Now()
// ... process A2A task ...
duration := time.Since(start).Seconds()
a2aTaskHistogram.Record(ctx, duration, metric.WithAttributes(
    attribute.String("task_type", taskType),
    attribute.String("task_state", "TASK_STATE_COMPLETED"),
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