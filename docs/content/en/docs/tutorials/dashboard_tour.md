---
title: "Interactive Dashboard Tour"
weight: 30
description: "Take a guided tour through AgentHub's Grafana dashboards while the system is running, learning to interpret metrics, identify issues, and understand system behavior in real-time."
---

# Interactive Dashboard Tour

**Learn by doing**: Take a guided tour through AgentHub's Grafana dashboards while the system is running, learning to interpret metrics, identify issues, and understand system behavior in real-time.

## Prerequisites

- **Observability stack running** (from the [Observability Demo](observability_demo.md))
- **Observable agents running** (broker, publisher, subscriber)
- **Grafana open** at http://localhost:3333
- **10-15 minutes** for the complete tour

## Quick Setup Reminder

If you haven't completed the observability demo yet:

```bash
# Start observability stack
cd agenthub/observability
docker-compose up -d

# Run observable agents (3 terminals)
go run -tags observability broker/main_observability.go
go run -tags observability agents/subscriber/main_observability.go
go run -tags observability agents/publisher/main_observability.go
```

## Dashboard Navigation

### Accessing the Main Dashboard

1. **Open Grafana**: http://localhost:3333
2. **Login**: admin / admin (skip password change for demo)
3. **Navigate**: Dashboards → Browse → AgentHub → "AgentHub EDA System Observatory"
4. **Bookmark**: Save this URL for quick access: http://localhost:3333/d/agenthub-eda-dashboard

### Dashboard Layout Overview

The dashboard is organized in **4 main rows**:

```
🎯 Row 1: Event Processing Overview
├── Event Processing Rate (events/sec)
└── Event Processing Error Rate (%)

📊 Row 2: Event Analysis
├── Event Types Distribution (pie chart)
└── Event Processing Latency (p50, p95, p99)

🔍 Row 3: Distributed Tracing
└── Jaeger Integration Panel

💻 Row 4: System Health
├── Service CPU Usage (%)
├── Service Memory Usage (MB)
├── Go Goroutines Count
└── Service Health Status
```

## Interactive Tour

### Tour 1: Understanding Event Flow (3 minutes)

#### Step 1: Watch the Event Processing Rate

**Location**: Top-left panel
**What to observe**: Real-time lines showing events per second

1. **Identify the services**:
   - **Green line**: `agenthub-broker` (should be highest - processes all events)
   - **Blue line**: `agenthub-publisher` (events being created)
   - **Orange line**: `agenthub-subscriber` (events being processed)

2. **Watch the pattern**:
   - Publisher creates bursts of events
   - Broker immediately processes them (routing)
   - Subscriber processes them shortly after

3. **Understand the flow**:
   ```
   Publisher (creates) → Broker (routes) → Subscriber (processes)
        50/sec      →      150/sec     →      145/sec
   ```

**💡 Tour Insight**: The broker rate is higher because it processes both incoming tasks AND outgoing results.

#### Step 2: Monitor Error Rates

**Location**: Top-right panel (gauge)
**What to observe**: Error percentage gauge

1. **Healthy system**: Should show 0-2% (green zone)
2. **If you see higher errors**:
   - Check if all services are running
   - Look for red traces in Jaeger (we'll do this next)

3. **Error rate calculation**:
   ```
   Error Rate = (Failed Events / Total Events) × 100
   ```

**🎯 Action**: Note your current error rate - we'll compare it later.

### Tour 2: Event Analysis Deep Dive (3 minutes)

#### Step 3: Explore Event Types

**Location**: Middle-left panel (pie chart)
**What to observe**: Distribution of different event types

1. **Identify event types**:
   - **greeting**: Most common (usually 40-50%)
   - **math_calculation**: Compute-heavy tasks (30-40%)
   - **random_number**: Quick tasks (15-25%)
   - **unknown_task**: Error-generating tasks (2-5%)

2. **Business insights**:
   - Larger slices = more frequent tasks
   - Small red slice = intentional error tasks for testing

**💡 Tour Insight**: The publisher randomly generates different task types to simulate real-world workload diversity.

#### Step 4: Analyze Processing Latency

**Location**: Middle-right panel
**What to observe**: Three latency lines (p50, p95, p99)

1. **Understand percentiles**:
   - **p50 (blue)**: 50% of events process faster than this
   - **p95 (green)**: 95% of events process faster than this
   - **p99 (red)**: 99% of events process faster than this

2. **Healthy ranges**:
   - **p50**: < 50ms (very responsive)
   - **p95**: < 200ms (good performance)
   - **p99**: < 500ms (acceptable outliers)

3. **Pattern recognition**:
   - Spiky p99 = occasional slow tasks (normal)
   - Rising p50 = systemic slowdown (investigate)
   - Flat lines = no activity or measurement issues

**🎯 Action**: Hover over the lines to see exact values at different times.

### Tour 3: Distributed Tracing Exploration (4 minutes)

#### Step 5: Jump into Jaeger

**Location**: Middle section - "Distributed Traces" panel
**Action**: Click the **"Explore"** button

This opens Jaeger in a new tab. Let's explore:

1. **In Jaeger UI**:
   - **Service dropdown**: Select "agenthub-broker"
   - **Operation**: Leave as "All"
   - **Click "Find Traces"**

2. **Pick a trace to examine**:
   - Look for traces that show multiple spans
   - Click on any trace line to open details

3. **Understand the trace structure**:
   ```
   Timeline View:
   agenthub-publisher: publish_event [2ms]
     └── agenthub-broker: process_event [1ms]
         └── agenthub-subscriber: consume_event [3ms]
             └── agenthub-subscriber: process_task [15ms]
                 └── agenthub-subscriber: publish_result [2ms]
   ```

4. **Explore span details**:
   - Click individual spans to see:
     - **Tags**: event_type, event_id, agent names
     - **Process**: Which service handled the span
     - **Duration**: Exact timing information

**💡 Tour Insight**: Each event creates a complete "trace" showing its journey from creation to completion.

#### Step 6: Find and Analyze an Error

1. **Search for error traces**:
   - In Jaeger, add tag filter: `error=true`
   - Or look for traces with red spans

2. **Examine the error trace**:
   - **Red spans** indicate errors
   - **Error tags** show the error type and message
   - **Stack traces** help with debugging

3. **Follow the error propagation**:
   - See how errors affect child spans
   - Notice error context in span attributes

**🎯 Action**: Find a trace with "unknown_task" event type - these are designed to fail for demonstration.

### Tour 4: System Health Monitoring (3 minutes)

#### Step 7: Monitor Resource Usage

**Location**: Bottom row panels
**What to observe**: System resource consumption

1. **CPU Usage Panel (Bottom-left)**:
   - **Normal range**: 10-50% for demo workload
   - **Watch for**: Sustained high CPU (>70%)
   - **Services comparison**: See which service uses most CPU

2. **Memory Usage Panel (Bottom-center-left)**:
   - **Normal range**: 30-80MB per service for demo
   - **Watch for**: Continuously growing memory (memory leaks)
   - **Pattern**: Sawtooth = normal GC, steady growth = potential leak

3. **Goroutines Panel (Bottom-center-right)**:
   - **Normal range**: 10-50 goroutines per service
   - **Watch for**: Continuously growing count (goroutine leaks)
   - **Pattern**: Stable baseline with activity spikes

#### Step 8: Verify Service Health

**Location**: Bottom-right panel
**What to observe**: Service up/down status

1. **Health indicators**:
   - **Green**: Service healthy and responding
   - **Red**: Service down or health check failing
   - **Yellow**: Service degraded but operational

2. **Health check details**:
   - Each service exposes `/health` endpoint
   - Prometheus monitors these endpoints
   - Dashboard shows aggregated status

**🎯 Action**: Open http://localhost:8080/health in a new tab to see raw health data.

### Tour 5: Time-based Analysis (2 minutes)

#### Step 9: Change Time Ranges

**Location**: Top-right of dashboard (time picker)
**Current**: Likely showing "Last 5 minutes"

1. **Try different ranges**:
   - **Last 15 minutes**: See longer trends
   - **Last 1 hour**: See full demo session
   - **Custom range**: Pick specific time period

2. **Observe pattern changes**:
   - **Longer ranges**: Show trends and patterns
   - **Shorter ranges**: Show real-time detail
   - **Custom ranges**: Zoom into specific incidents

#### Step 10: Use Dashboard Filters

**Location**: Top of dashboard - variable dropdowns

1. **Service Filter**:
   - Select "All" to see everything
   - Pick specific service to focus analysis
   - Useful for isolating service-specific issues

2. **Event Type Filter**:
   - Filter to specific event types
   - Compare performance across task types
   - Identify problematic event categories

**💡 Tour Insight**: Filtering helps you drill down from system-wide view to specific components or workloads.

## Hands-on Experiments

### Experiment 1: Create a Service Outage

**Goal**: See how the dashboard shows service failures

1. **Stop the subscriber**:
   ```bash
   # In subscriber terminal, press Ctrl+C
   ```

2. **Watch the dashboard changes**:
   - Error rate increases (top-right gauge turns red)
   - Subscriber metrics disappear from bottom panels
   - Service health shows subscriber as down

3. **Check Jaeger for failed traces**:
   - Look for traces that don't complete
   - See where the chain breaks

4. **Restart subscriber**:
   ```bash
   go run -tags observability agents/subscriber/main_observability.go
   ```

**🎯 Learning**: Dashboard immediately shows impact of service failures.

### Experiment 2: Generate High Load

**Goal**: See system behavior under stress

1. **Modify publisher** to generate more events:
   ```bash
   # Edit agents/publisher/main_observability.go
   # Change: time.Sleep(5 * time.Second)
   # To:     time.Sleep(1 * time.Second)
   ```

2. **Watch dashboard changes**:
   - Processing rate increases
   - Latency may increase
   - CPU/memory usage grows

3. **Observe scaling behavior**:
   - How does the system handle increased load?
   - Do error rates increase?
   - Where are the bottlenecks?

**🎯 Learning**: Dashboard shows system performance characteristics under load.

## Dashboard Interpretation Guide

### What Good Looks Like

✅ **Event Processing Rate**: Steady activity matching workload
✅ **Error Rate**: < 5% (green zone)
✅ **Event Types**: Expected distribution
✅ **Latency**: p95 < 200ms, p99 < 500ms
✅ **CPU Usage**: < 50% sustained
✅ **Memory**: Stable or slow growth with GC cycles
✅ **Goroutines**: Stable baseline with activity spikes
✅ **Service Health**: All services green/up

### Warning Signs

⚠️ **Error Rate**: 5-10% (yellow zone)
⚠️ **Latency**: p95 > 200ms or rising trend
⚠️ **CPU**: Sustained > 70%
⚠️ **Memory**: Continuous growth without GC
⚠️ **Missing data**: Gaps in metrics (service issues)

### Critical Issues

🚨 **Error Rate**: > 10% (red zone)
🚨 **Latency**: p95 > 500ms
🚨 **CPU**: Sustained > 90%
🚨 **Memory**: Rapid growth or OOM
🚨 **Service Health**: Any service showing red/down
🚨 **Traces**: Missing or broken trace chains

## Next Steps After the Tour

### **For Daily Operations**:
- **Bookmark**: Save dashboard URL for quick access
- **Set up alerts**: Configure notifications for critical metrics
- **Create views**: Use filters to create focused views for your team

### **For Development**:
- **[Add Observability to Your Agent](../howto/add_observability.md)** - Instrument your own agents
- **[Debug with Distributed Tracing](../howto/debug_with_tracing.md)** - Use Jaeger for troubleshooting

### **For Deep Understanding**:
- **[Distributed Tracing Explained](../explanation/distributed_tracing.md)** - Learn the concepts
- **[Observability Metrics Reference](../reference/observability_metrics.md)** - Complete metrics catalog

## Troubleshooting Tour Issues

| **Issue** | **Solution** |
|-----------|-------------|
| Dashboard shows no data | Verify agents running with `-tags observability` |
| Grafana won't load | Check `docker-compose ps` in observability/ |
| Metrics missing | Verify Prometheus targets at http://localhost:9090/targets |
| Jaeger empty | Ensure trace context propagation is working |

---

**🎉 Congratulations!** You've completed the interactive dashboard tour and learned to read AgentHub's observability signals like a pro!

**🎯 Ready for More?**

**Master the Tools**: **[Use Grafana Dashboards](../howto/use_dashboards.md)** - Advanced dashboard usage

**Troubleshoot Issues**: **[Debug with Distributed Tracing](../howto/debug_with_tracing.md)** - Use Jaeger effectively