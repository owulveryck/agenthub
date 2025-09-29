---
title: "How to Debug Agent Issues"
weight: 40
description: "Practical steps for troubleshooting common issues when developing and deploying agents with AgentHub."
---

# How to Debug Agent Issues

This guide provides practical steps for troubleshooting common issues when developing and deploying agents with AgentHub.

## Common Connection Issues

### Problem: Agent Can't Connect to Broker

**Symptoms:**
```
Failed to connect: connection refused
```

**Solutions:**

1. **Check if broker is running:**
   ```bash
   # Check if broker process is running
   ps aux | grep eventbus-server

   # Check if port 50051 is listening
   netstat -tlnp | grep 50051
   # or
   lsof -i :50051
   ```

2. **Verify broker address:**
   ```go
   const agentHubAddr = "localhost:50051"  // Correct for local development
   // const agentHubAddr = "broker.example.com:50051"  // For remote broker
   ```

3. **Check firewall settings:**
   ```bash
   # On Linux, check if port is blocked
   sudo ufw status

   # Allow port if needed
   sudo ufw allow 50051
   ```

### Problem: TLS/SSL Errors

**Symptoms:**
```
transport: authentication handshake failed
```

**Solution:**
Ensure you're using insecure credentials for development:
```go
conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
```

## Task Processing Issues

### Problem: Agent Not Receiving Tasks

**Debug Steps:**

1. **Check subscription logs:**
   ```go
   log.Printf("Agent %s subscribing to tasks...", agentID)
   // Should see: "Successfully subscribed to tasks for agent {agentID}"
   ```

2. **Verify agent ID matching:**
   ```go
   // In publisher
   ResponderAgentId: "my_processing_agent"

   // In subscriber (must match exactly)
   const agentID = "my_processing_agent"
   ```

3. **Check task type filtering:**
   ```go
   req := &pb.SubscribeToTasksRequest{
       AgentId: agentID,
       TaskTypes: []string{"math_calculation"}, // Remove to receive all types
   }
   ```

4. **Monitor broker logs:**
   ```
   # Broker should show:
   Received task request: task_xyz (type: math) from agent: publisher_agent
   # And either:
   No subscribers for task from agent 'publisher_agent'  # Bad - no matching agents
   # Or task routing to subscribers  # Good - task delivered
   ```

### Problem: Tasks Timing Out

**Debug Steps:**

1. **Check task processing time:**
   ```go
   func processTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) {
       start := time.Now()
       defer func() {
           log.Printf("Task %s took %v to process", task.GetTaskId(), time.Since(start))
       }()

       // Your processing logic
   }
   ```

2. **Add timeout handling:**
   ```go
   func processTaskWithTimeout(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) {
       // Create timeout context
       taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
       defer cancel()

       // Process with timeout
       select {
       case <-taskCtx.Done():
           if taskCtx.Err() == context.DeadlineExceeded {
               sendResult(ctx, task, nil, pb.TaskStatus_TASK_STATUS_FAILED, "Task timeout", client)
           }
           return
       default:
           // Process normally
       }
   }
   ```

3. **Monitor progress updates:**
   ```go
   // Send progress every few seconds
   ticker := time.NewTicker(5 * time.Second)
   defer ticker.Stop()

   go func() {
       progress := 0
       for range ticker.C {
           progress += 10
           if progress > 100 {
               return
           }
           sendProgress(ctx, task, int32(progress), "Still processing...", client)
       }
   }()
   ```

## Message Serialization Issues

### Problem: Parameter Marshaling Errors

**Symptoms:**
```
Error creating parameters struct: proto: invalid value type
```

**Solution:**
Ensure all parameter values are compatible with `structpb`:

```go
// Bad - channels, functions, complex types not supported
params := map[string]interface{}{
    "callback": func() {},  // Not supported
    "channel": make(chan int),  // Not supported
}

// Good - basic types only
params := map[string]interface{}{
    "name": "value",           // string
    "count": 42,               // number
    "enabled": true,           // boolean
    "items": []string{"a", "b"}, // array
    "config": map[string]interface{}{ // nested object
        "timeout": 30,
    },
}
```

### Problem: Result Unmarshaling Issues

**Debug Steps:**

1. **Check result structure:**
   ```go
   func handleTaskResult(result *pb.TaskResult) {
       log.Printf("Raw result: %+v", result.GetResult())

       resultMap := result.GetResult().AsMap()
       log.Printf("Result as map: %+v", resultMap)

       // Type assert carefully
       if value, ok := resultMap["count"].(float64); ok {
           log.Printf("Count: %f", value)
       } else {
           log.Printf("Count field missing or wrong type: %T", resultMap["count"])
       }
   }
   ```

2. **Handle type conversion safely:**
   ```go
   func getStringField(m map[string]interface{}, key string) (string, error) {
       if val, ok := m[key]; ok {
           if str, ok := val.(string); ok {
               return str, nil
           }
           return "", fmt.Errorf("field %s is not a string: %T", key, val)
       }
       return "", fmt.Errorf("field %s not found", key)
   }

   func getNumberField(m map[string]interface{}, key string) (float64, error) {
       if val, ok := m[key]; ok {
           if num, ok := val.(float64); ok {
               return num, nil
           }
           return 0, fmt.Errorf("field %s is not a number: %T", key, val)
       }
       return 0, fmt.Errorf("field %s not found", key)
   }
   ```

## Stream and Connection Issues

### Problem: Stream Disconnections

**Symptoms:**
```
Error receiving task: rpc error: code = Unavailable desc = connection error
```

**Solutions:**

1. **Implement retry logic:**
   ```go
   func subscribeToTasksWithRetry(ctx context.Context, client pb.EventBusClient) {
       for {
           err := subscribeToTasks(ctx, client)
           if err != nil {
               log.Printf("Subscription error: %v, retrying in 5 seconds...", err)
               time.Sleep(5 * time.Second)
               continue
           }
           break
       }
   }
   ```

2. **Handle context cancellation:**
   ```go
   for {
       task, err := stream.Recv()
       if err == io.EOF {
           log.Printf("Stream closed by server")
           return
       }
       if err != nil {
           if ctx.Err() != nil {
               log.Printf("Context cancelled: %v", ctx.Err())
               return
           }
           log.Printf("Stream error: %v", err)
           return
       }
       // Process task
   }
   ```

### Problem: Memory Leaks in Long-Running Agents

**Debug Steps:**

1. **Monitor memory usage:**
   ```bash
   # Check memory usage
   ps -o pid,ppid,cmd,%mem,%cpu -p $(pgrep -f "your-agent")

   # Continuous monitoring
   watch -n 5 'ps -o pid,ppid,cmd,%mem,%cpu -p $(pgrep -f "your-agent")'
   ```

2. **Profile memory usage:**
   ```go
   import _ "net/http/pprof"
   import "net/http"

   func main() {
       // Start pprof server
       go func() {
           log.Println(http.ListenAndServe("localhost:6060", nil))
       }()

       // Your agent code
   }
   ```

   Access profiles at `http://localhost:6060/debug/pprof/`

3. **Check for goroutine leaks:**
   ```go
   import "runtime"

   func logGoroutines() {
       ticker := time.NewTicker(30 * time.Second)
       go func() {
           for range ticker.C {
               log.Printf("Goroutines: %d", runtime.NumGoroutine())
           }
       }()
   }
   ```

## Performance Issues

### Problem: Slow Task Processing

**Debug Steps:**

1. **Add timing measurements:**
   ```go
   func processTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) {
       timings := make(map[string]time.Duration)

       start := time.Now()

       // Phase 1: Parameter validation
       timings["validation"] = time.Since(start)
       last := time.Now()

       // Phase 2: Business logic
       // ... your logic here ...
       timings["processing"] = time.Since(last)
       last = time.Now()

       // Phase 3: Result formatting
       // ... result creation ...
       timings["formatting"] = time.Since(last)

       log.Printf("Task %s timings: %+v", task.GetTaskId(), timings)
   }
   ```

2. **Profile CPU usage:**
   ```go
   import "runtime/pprof"
   import "os"

   func startCPUProfile() func() {
       f, err := os.Create("cpu.prof")
       if err != nil {
           log.Fatal(err)
       }
       pprof.StartCPUProfile(f)

       return func() {
           pprof.StopCPUProfile()
           f.Close()
       }
   }

   func main() {
       stop := startCPUProfile()
       defer stop()

       // Your agent code
   }
   ```

3. **Monitor queue sizes:**
   ```go
   type Agent struct {
       taskQueue chan *pb.TaskMessage
   }

   func (a *Agent) logQueueSize() {
       ticker := time.NewTicker(10 * time.Second)
       go func() {
           for range ticker.C {
               log.Printf("Task queue size: %d/%d", len(a.taskQueue), cap(a.taskQueue))
           }
       }()
   }
   ```

## Debugging Tools and Techniques

### 1. Enable Verbose Logging

```go
import "log"
import "os"

func init() {
    // Enable verbose logging
    log.SetFlags(log.LstdFlags | log.Lshortfile)

    // Set log level from environment
    if os.Getenv("DEBUG") == "true" {
        log.SetOutput(os.Stdout)
    }
}
```

### 2. Add Structured Logging

```go
import "encoding/json"
import "time"

type LogEntry struct {
    Timestamp string                 `json:"timestamp"`
    Level     string                 `json:"level"`
    AgentID   string                 `json:"agent_id"`
    TaskID    string                 `json:"task_id,omitempty"`
    Message   string                 `json:"message"`
    Data      map[string]interface{} `json:"data,omitempty"`
}

func logInfo(agentID, taskID, message string, data map[string]interface{}) {
    entry := LogEntry{
        Timestamp: time.Now().Format(time.RFC3339),
        Level:     "INFO",
        AgentID:   agentID,
        TaskID:    taskID,
        Message:   message,
        Data:      data,
    }

    if jsonData, err := json.Marshal(entry); err == nil {
        log.Println(string(jsonData))
    }
}
```

### 3. Health Check Endpoint

```go
import "net/http"
import "encoding/json"

type HealthStatus struct {
    Status       string    `json:"status"`
    AgentID      string    `json:"agent_id"`
    Uptime       string    `json:"uptime"`
    TasksProcessed int64   `json:"tasks_processed"`
    LastTaskTime  time.Time `json:"last_task_time"`
}

func startHealthServer(agent *Agent) {
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        status := HealthStatus{
            Status:         "healthy",
            AgentID:        agent.ID,
            Uptime:         time.Since(agent.StartTime).String(),
            TasksProcessed: agent.TasksProcessed,
            LastTaskTime:   agent.LastTaskTime,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(status)
    })

    log.Printf("Health server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### 4. Task Tracing

```go
import "context"

type TraceID string

func withTraceID(ctx context.Context) context.Context {
    traceID := TraceID(fmt.Sprintf("trace-%d", time.Now().UnixNano()))
    return context.WithValue(ctx, "trace_id", traceID)
}

func getTraceID(ctx context.Context) TraceID {
    if traceID, ok := ctx.Value("trace_id").(TraceID); ok {
        return traceID
    }
    return ""
}

func processTaskWithTracing(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) {
    ctx = withTraceID(ctx)
    traceID := getTraceID(ctx)

    log.Printf("[%s] Starting task %s", traceID, task.GetTaskId())
    defer log.Printf("[%s] Finished task %s", traceID, task.GetTaskId())

    // Your processing logic with trace ID logging
}
```

## Common Error Patterns

### 1. Resource Exhaustion

**Signs:**
- Tasks start failing after running for a while
- Memory usage continuously increases
- File descriptor limits reached

**Solutions:**
- Implement proper resource cleanup
- Add connection pooling
- Set task processing limits

### 2. Deadlocks

**Signs:**
- Agent stops processing tasks
- Health checks show agent as "stuck"

**Solutions:**
- Avoid blocking operations in main goroutines
- Use timeouts for all operations
- Implement deadlock detection

### 3. Race Conditions

**Signs:**
- Intermittent task failures
- Inconsistent behavior
- Data corruption

**Solutions:**
- Use proper synchronization primitives
- Run race detector: `go run -race your-agent.go`
- Add mutex protection for shared state

With these debugging techniques, you should be able to identify and resolve most agent-related issues efficiently.