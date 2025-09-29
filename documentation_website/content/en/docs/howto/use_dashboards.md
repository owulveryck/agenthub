---
title: "How to Use Grafana Dashboards"
weight: 50
description: "Master the AgentHub observability dashboards to monitor, analyze, and troubleshoot your event-driven system effectively."
---

# How to Use Grafana Dashboards

**Goal-oriented guide**: Master the AgentHub observability dashboards to monitor, analyze, and troubleshoot your event-driven system effectively.

## Prerequisites

- AgentHub observability stack running (`docker-compose up -d`)
- Observable agents running (with `-tags observability`)
- Basic understanding of metrics concepts
- 10-15 minutes

## Quick Access

- **Grafana Dashboard**: http://localhost:3333 (admin/admin)
- **Direct Dashboard**: http://localhost:3333/d/agenthub-eda-dashboard

## Dashboard Overview

The **AgentHub EDA System Observatory** provides comprehensive monitoring across three main areas:

1. **Event Metrics** (Top Row) - Event processing performance
2. **Distributed Tracing** (Middle) - Request flow visualization
3. **System Health** (Bottom Row) - Infrastructure monitoring

## Panel-by-Panel Guide

### 🚀 Event Processing Rate (Top Left)

**What it shows**: Events processed per second by each service

**How to use**:
- **Monitor throughput**: See how many events your system processes
- **Identify bottlenecks**: Low rates may indicate performance issues
- **Compare services**: See which agents are busiest

**Reading the chart**:
```
Green line: agenthub-broker (150 events/sec)
Blue line:  agenthub-publisher (50 events/sec)
Red line:   agenthub-subscriber (145 events/sec)
```

**Troubleshooting**:
- **Flat lines**: No activity - check if agents are running
- **Dropping rates**: Performance degradation - check CPU/memory
- **Spiky patterns**: Bursty workloads - consider load balancing

### 🚨 Event Processing Error Rate (Top Right)

**What it shows**: Percentage of events that failed processing

**How to use**:
- **Monitor reliability**: Should stay below 5% (green zone)
- **Alert threshold**: Yellow above 5%, red above 10%
- **Quick health check**: Single glance system reliability

**Color coding**:
- **Green (0-5%)**: Healthy system
- **Yellow (5-10%)**: Moderate issues
- **Red (>10%)**: Critical problems

**Troubleshooting**:
- **High error rates**: Check Jaeger for failing traces
- **Sudden spikes**: Look for recent deployments or config changes
- **Persistent errors**: Check logs for recurring issues

### 📈 Event Types Distribution (Middle Left)

**What it shows**: Breakdown of event types by volume

**How to use**:
- **Understand workload**: See what types of tasks dominate
- **Capacity planning**: Identify which task types need scaling
- **Anomaly detection**: Unusual distributions may indicate issues

**Example interpretation**:
```
greeting: 40% (blue) - Most common task type
math_calculation: 35% (green) - Heavy computational tasks
random_number: 20% (yellow) - Quick tasks
unknown_task: 5% (red) - Error-generating tasks
```

**Troubleshooting**:
- **Missing task types**: Check if specific agents are down
- **Unexpected distributions**: May indicate upstream issues
- **Dominant error types**: Focus optimization efforts

### ⏱️ Event Processing Latency (Middle Right)

**What it shows**: Processing time percentiles (p50, p95, p99)

**How to use**:
- **Performance monitoring**: Track how fast events are processed
- **SLA compliance**: Ensure latencies meet requirements
- **Outlier detection**: p99 shows worst-case scenarios

**Understanding percentiles**:
- **p50 (median)**: 50% of events process faster than this
- **p95**: 95% of events process faster than this
- **p99**: 99% of events process faster than this

**Healthy ranges**:
- **p50**: < 50ms (very responsive)
- **p95**: < 200ms (good performance)
- **p99**: < 500ms (acceptable outliers)

**Troubleshooting**:
- **Rising latencies**: Check CPU/memory usage
- **High p99**: Look for resource contention or long-running tasks
- **Flatlined metrics**: May indicate measurement issues

### 🔍 Distributed Traces (Middle Section)

**What it shows**: Integration with Jaeger for trace visualization

**How to use**:
1. **Click "Explore"** to open Jaeger
2. **Select service** from dropdown
3. **Find specific traces** to debug issues
4. **Analyze request flows** across services

**When to use**:
- **Debugging errors**: Find root cause of failures
- **Performance analysis**: Identify slow operations
- **Understanding flows**: See complete request journeys

### 🖥️ Service CPU Usage (Bottom Left)

**What it shows**: CPU utilization by service

**How to use**:
- **Capacity monitoring**: Ensure services aren't overloaded
- **Resource planning**: Identify when to scale
- **Performance correlation**: High CPU often explains high latency

**Healthy ranges**:
- **< 50%**: Comfortable utilization
- **50-70%**: Moderate load
- **> 70%**: Consider scaling

### 💾 Service Memory Usage (Bottom Center)

**What it shows**: Memory consumption by service

**How to use**:
- **Memory leak detection**: Watch for continuously growing usage
- **Capacity planning**: Ensure sufficient memory allocation
- **Garbage collection**: High usage may impact performance

**Monitoring tips**:
- **Steady growth**: May indicate memory leaks
- **Sawtooth pattern**: Normal GC behavior
- **Sudden spikes**: Check for large event batches

### 🧵 Go Goroutines (Bottom Right)

**What it shows**: Number of concurrent goroutines per service

**How to use**:
- **Concurrency monitoring**: Track parallel processing
- **Resource leak detection**: Continuously growing numbers indicate leaks
- **Performance tuning**: Optimize concurrency levels

**Normal patterns**:
- **Stable baseline**: Normal operation
- **Activity spikes**: During high load
- **Continuous growth**: Potential goroutine leaks

### 🏥 Service Health Status (Bottom Far Right)

**What it shows**: Up/down status of each service

**How to use**:
- **Quick status check**: See if all services are running
- **Outage detection**: Immediately identify down services
- **Health monitoring**: Green = UP, Red = DOWN

## Dashboard Variables and Filters

### Service Filter
**Location**: Top of dashboard
**Purpose**: Filter metrics by specific services
**Usage**:
- Select "All" to see everything
- Choose specific services to focus analysis
- Useful for isolating problems to specific components

### Event Type Filter
**Location**: Top of dashboard
**Purpose**: Filter by event/task types
**Usage**:
- Analyze specific workflow types
- Debug particular task categories
- Compare performance across task types

### Time Range Selector
**Location**: Top right of dashboard
**Purpose**: Control time window for analysis
**Common ranges**:
- **5 minutes**: Real-time monitoring
- **1 hour**: Recent trend analysis
- **24 hours**: Daily pattern analysis
- **7 days**: Weekly trend and capacity planning

## Advanced Usage Patterns

### Performance Investigation Workflow

1. **Start with Overview**:
   - Check error rates (should be < 5%)
   - Verify processing rates look normal
   - Scan for any red/yellow indicators

2. **Drill Down on Issues**:
   - If high error rates → check distributed traces
   - If high latency → examine CPU/memory usage
   - If low throughput → check service health

3. **Root Cause Analysis**:
   - Use time range selector to find when problems started
   - Filter by specific services to isolate issues
   - Correlate metrics across different panels

### Capacity Planning Workflow

1. **Analyze Peak Patterns**:
   - Set time range to 7 days
   - Identify peak usage periods
   - Note maximum throughput achieved

2. **Resource Utilization**:
   - Check CPU usage during peaks
   - Monitor memory consumption trends
   - Verify goroutine scaling behavior

3. **Plan Scaling**:
   - If CPU > 70% during peaks, scale up
   - If memory continuously growing, investigate leaks
   - If error rates spike during load, optimize before scaling

### Troubleshooting Workflow

1. **Identify Symptoms**:
   - High error rates: Focus on traces and logs
   - High latency: Check resource utilization
   - Low throughput: Verify service health

2. **Time Correlation**:
   - Use time range to find when issues started
   - Look for correlated changes across metrics
   - Check for deployment or configuration changes

3. **Service Isolation**:
   - Use service filter to identify problematic components
   - Compare healthy vs unhealthy services
   - Check inter-service dependencies

## Dashboard Customization

### Adding New Panels

1. **Click "+ Add panel"** in top menu
2. **Choose visualization type**:
   - Time series for trends
   - Stat for current values
   - Gauge for thresholds
3. **Configure query**:
   ```promql
   # Example: Custom error rate
   rate(my_custom_errors_total[5m]) / rate(my_custom_requests_total[5m]) * 100
   ```

### Creating Alerts

1. **Edit existing panel** or create new one
2. **Click "Alert" tab**
3. **Configure conditions**:
   ```
   Query: rate(event_errors_total[5m]) / rate(events_processed_total[5m]) * 100
   Condition: IS ABOVE 5
   Evaluation: Every 1m for 2m
   ```
4. **Set notification channels**

### Custom Time Ranges

1. **Click time picker** (top right)
2. **Select "Custom range"**
3. **Set specific dates/times** for historical analysis
4. **Use "Refresh" settings** for auto-updating

## Troubleshooting Dashboard Issues

### Dashboard Not Loading
```bash
# Check Grafana status
docker-compose ps grafana

# Check Grafana logs
docker-compose logs grafana

# Restart if needed
docker-compose restart grafana
```

### No Data in Panels
```bash
# Check Prometheus connection
curl http://localhost:9090/api/v1/targets

# Verify agents are exposing metrics
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics
curl http://localhost:8082/metrics

# Check Prometheus configuration
docker-compose logs prometheus
```

### Slow Dashboard Performance
1. **Reduce time range**: Use shorter windows for better performance
2. **Limit service selection**: Filter to specific services
3. **Optimize queries**: Use appropriate rate intervals
4. **Check resource usage**: Ensure Prometheus has enough memory

### Authentication Issues
- **Default credentials**: admin/admin
- **Reset password**: Through Grafana UI after first login
- **Lost access**: Restart Grafana container to reset

## Best Practices

### Regular Monitoring
- **Check dashboard daily**: Quick health overview
- **Weekly reviews**: Trend analysis and capacity planning
- **Set up alerts**: Proactive monitoring for critical metrics

### Performance Optimization
- **Use appropriate time ranges**: Don't query more data than needed
- **Filter effectively**: Use service and event type filters
- **Refresh intervals**: Balance real-time needs with performance

### Team Usage
- **Share dashboard URLs**: Bookmark specific views
- **Create annotations**: Mark deployments and incidents
- **Export snapshots**: Share findings with team members

## Integration with Other Tools

### Jaeger Integration
- Click **Explore** in traces panel
- Auto-links to Jaeger with service context
- Correlate traces with metrics timeframes

### Prometheus Integration
- Click **Explore** on any panel
- Edit queries in Prometheus query language
- Access raw metrics for custom analysis

### Log Correlation
- Use trace IDs from Jaeger
- Search logs for matching trace IDs
- Correlate log events with metric spikes

---

**🎯 Next Steps**:

**Deep Debugging**: **[Debug with Distributed Tracing](debug_with_tracing.md)**

**Production Setup**: **[Configure Alerts](configure_alerts.md)**

**Understanding**: **[Observability Architecture Explained](../explanation/observability_architecture.md)**