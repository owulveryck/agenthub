---
title: "AgentHub Observability Demo Tutorial"
weight: 40
description: "Experience the complete observability stack with distributed tracing, real-time metrics, and intelligent alerting in under 10 minutes through hands-on learning."
---

# AgentHub Observability Demo Tutorial

**Learn by doing**: Experience the complete observability stack with distributed tracing, real-time metrics, and intelligent alerting in under 10 minutes.

## What You'll Learn

By the end of this tutorial, you'll have:
- ✅ **Seen distributed traces** flowing across multiple agents
- ✅ **Monitored real-time metrics** in beautiful Grafana dashboards
- ✅ **Understood event correlation** through trace IDs
- ✅ **Experienced intelligent alerting** when things go wrong
- ✅ **Explored the complete observability stack** components

## Prerequisites

- **Go 1.24+** installed
- **Docker and Docker Compose** installed
- **Environment variables configured** (see [Installation and Setup](../getting-started/installation_and_setup/))
- **10 minutes** of your time
- **Basic terminal** knowledge

> 💡 **Environment Note**: AgentHub agents automatically enable observability when `JAEGER_ENDPOINT` is configured. See [Environment Variables Reference](../../reference/configuration/environment_variables/) for all configuration options.

## Step 1: Clone and Setup (1 minute)

```bash
# Clone the repository
git clone https://github.com/owulveryck/agenthub.git
cd agenthub

# Verify you have the observability files
ls observability/
# You should see: docker-compose.yml, grafana/, prometheus/, etc.
```

## Step 2: Start the Observability Stack (2 minutes)

```bash
# Navigate to observability directory
cd observability

# Start all monitoring services
docker-compose up -d

# Verify services are running
docker-compose ps
```

**Expected Output:**
```
NAME                    COMMAND                  SERVICE             STATUS
agenthub-grafana        "/run.sh"                grafana             running
agenthub-jaeger         "/go/bin/all-in-one"     jaeger              running
agenthub-prometheus     "/bin/prometheus --c…"   prometheus          running
agenthub-otel-collector "/otelcol-contrib --…"   otel-collector      running
```

**🎯 Checkpoint 1**: All services should be "running". If not, check Docker logs: `docker-compose logs <service-name>`

## Step 3: Access the Dashboards (1 minute)

Open these URLs in your browser (keep them open in tabs):

| **Service** | **URL** | **Purpose** |
|-------------|---------|-------------|
| **Grafana** | http://localhost:3333 | Main observability dashboard |
| **Jaeger** | http://localhost:16686 | Distributed tracing |
| **Prometheus** | http://localhost:9090 | Raw metrics and alerts |

**Grafana Login**: admin / admin (skip password change for demo)

**🎯 Checkpoint 2**: You should see Grafana's welcome page and Jaeger's empty trace list.

## Step 4: Start the Observable Broker (1 minute)

Open a new terminal and navigate back to the project root:

```bash
# From agenthub root directory
go run broker/main.go
```

**Expected Output:**
```
time=2025-09-28T21:00:00.000Z level=INFO msg="Starting health server on port 8080"
time=2025-09-28T21:00:00.000Z level=INFO msg="AgentHub broker gRPC server with observability listening" address="[::]:50051" health_endpoint="http://localhost:8080/health" metrics_endpoint="http://localhost:8080/metrics"
```

**🎯 Checkpoint 3**:
- Broker is listening on port 50051
- Health endpoint available at http://localhost:8080/health
- Metrics endpoint available at http://localhost:8080/metrics

## Step 5: Start the Observable Subscriber (1 minute)

Open another terminal:

```bash
go run agents/subscriber/main.go
```

**Expected Output:**
```
time=2025-09-28T21:00:01.000Z level=INFO msg="Starting health server on port 8082"
time=2025-09-28T21:00:01.000Z level=INFO msg="Starting observable subscriber"
time=2025-09-28T21:00:01.000Z level=INFO msg="Agent started with observability. Listening for events and tasks."
```

**🎯 Checkpoint 4**:
- Subscriber is connected and listening
- Health available at http://localhost:8082/health

## Step 6: Generate Events with the Publisher (2 minutes)

Open a third terminal:

```bash
go run agents/publisher/main.go
```

**Expected Output:**
```
time=2025-09-28T21:00:02.000Z level=INFO msg="Starting health server on port 8081"
time=2025-09-28T21:00:02.000Z level=INFO msg="Starting observable publisher demo"
time=2025-09-28T21:00:02.000Z level=INFO msg="Publishing task" task_id=task_greeting_1727557202 task_type=greeting responder_agent_id=agent_demo_subscriber
time=2025-09-28T21:00:02.000Z level=INFO msg="Task published successfully" task_id=task_greeting_1727557202 task_type=greeting
```

**🎯 Checkpoint 5**: You should see:
- Publisher creating and sending tasks
- Subscriber receiving and processing tasks
- Broker routing messages between them

## Step 7: Explore Real-time Metrics in Grafana (2 minutes)

1. **Go to Grafana**: http://localhost:3333
2. **Navigate to Dashboards** → Browse → AgentHub → "AgentHub EDA System Observatory"
3. **Observe the real-time data**:

### What You'll See:

#### **Event Processing Rate** (Top Left)
- Lines showing events/second for each service
- Should show activity spikes when publisher runs

#### **Error Rate** (Top Right)
- Gauge showing error percentage
- Should be green (< 5% errors)

#### **Event Types Distribution** (Middle Left)
- Pie chart showing task types: greeting, math_calculation, random_number
- Different colors for each task type

#### **Processing Latency** (Middle Right)
- Three lines: p50, p95, p99 latencies
- Should show sub-second processing times

#### **System Health** (Bottom)
- CPU usage, memory usage, goroutines
- Service health status (all should be UP)

**🎯 Checkpoint 6**: Dashboard should show live metrics with recent activity.

## Step 8: Explore Distributed Traces in Jaeger (2 minutes)

1. **Go to Jaeger**: http://localhost:16686
2. **Select Service**: Choose "agenthub-broker" from dropdown
3. **Click "Find Traces"**
4. **Click on any trace** to see details

### What You'll See:

#### **Complete Event Journey**:
```
agenthub-publisher: publish_event (2ms)
  └── agenthub-broker: process_event (1ms)
      └── agenthub-subscriber: consume_event (5ms)
          └── agenthub-subscriber: process_task (15ms)
              └── agenthub-subscriber: publish_result (2ms)
```

#### **Trace Details**:
- **Span Tags**: event_id, event_type, service names
- **Timing Information**: Exact start/end times and durations
- **Log Correlation**: Each span linked to structured logs

#### **Error Detection**:
- Look for red spans indicating errors
- Trace the "unknown_task" type to see how errors propagate

**🎯 Checkpoint 7**: You should see complete traces showing the full event lifecycle.

## Step 9: Correlate Logs with Traces (1 minute)

1. **Copy a trace ID** from Jaeger (the long hex string)
2. **Check broker logs** for that trace ID:
   ```bash
   # In your broker terminal, look for lines like:
   time=2025-09-28T21:00:02.000Z level=INFO msg="Received task request" task_id=task_greeting_1727557202 trace_id=a1b2c3d4e5f6...
   ```

3. **Check subscriber logs** for the same trace ID

**🎯 Checkpoint 8**: You should find the same trace_id in logs across multiple services.

## Step 10: Experience Intelligent Alerting (Optional)

To see alerting in action:

1. **Simulate errors** by stopping the subscriber:
   ```bash
   # In subscriber terminal, press Ctrl+C
   ```

2. **Keep publisher running** (it will fail to process tasks)

3. **Check Prometheus alerts**:
   - Go to http://localhost:9090/alerts
   - After ~5 minutes, you should see "HighEventProcessingErrorRate" firing

4. **Restart subscriber** to clear the alert

## 🎉 Congratulations!

You've successfully experienced the complete AgentHub observability stack!

## Summary: What You Accomplished

✅ **Deployed a complete observability stack** with Docker Compose
✅ **Ran observable agents** with automatic instrumentation
✅ **Monitored real-time metrics** in Grafana dashboards
✅ **Traced event flows** across multiple services with Jaeger
✅ **Correlated logs with traces** using trace IDs
✅ **Experienced intelligent alerting** with Prometheus
✅ **Understood the complete event lifecycle** from publisher to subscriber

## Key Observability Concepts You Learned

### **Distributed Tracing**
- Events get unique trace IDs that follow them everywhere
- Each processing step creates a "span" with timing information
- Complete request flows are visible across service boundaries

### **Metrics Collection**
- 47+ different metrics automatically collected
- Real-time visualization of system health and performance
- Historical data for trend analysis

### **Structured Logging**
- All logs include trace context for correlation
- Consistent format across all services
- Easy debugging and troubleshooting

### **Intelligent Alerting**
- Proactive monitoring for error rates and performance
- Automatic notifications when thresholds are exceeded
- Helps prevent issues before they impact users

## Next Steps

### **For Development**:
- **[Add Observability to Your Agent](../howto/add_observability.md)** - Instrument your own agents
- **[Debug with Distributed Tracing](../howto/debug_with_tracing.md)** - Troubleshoot issues effectively

### **For Operations**:
- **[Use Grafana Dashboards](../howto/use_dashboards.md)** - Master the monitoring interface
- **[Configure Alerts](../howto/configure_alerts.md)** - Set up production alerting

### **For Understanding**:
- **[Distributed Tracing Explained](../explanation/distributed_tracing.md)** - Deep dive into concepts
- **[Observability Architecture](../explanation/observability_architecture.md)** - How it all works together

## Troubleshooting

| **Issue** | **Solution** |
|-----------|-------------|
| Services won't start | Run `docker-compose down && docker-compose up -d` |
| No metrics in Grafana | Check Prometheus targets: http://localhost:9090/targets |
| No traces in Jaeger | Verify JAEGER_ENDPOINT environment variable is set correctly |
| Permission errors | Ensure Docker has proper permissions |

## Clean Up

When you're done exploring:

```bash
# Stop the observability stack
cd observability
docker-compose down

# Stop the Go applications
# Press Ctrl+C in each terminal running the agents
```

---

**🎯 Ready for More?**

**Production Usage**: **[Add Observability to Your Agent](../howto/add_observability.md)**

**Deep Understanding**: **[Distributed Tracing Explained](../explanation/distributed_tracing.md)**