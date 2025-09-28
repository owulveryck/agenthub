# How to Create a Task Subscriber (Agent)

This guide shows you how to create an agent that can receive, process, and respond to Agent2Agent protocol tasks through the AgentHub broker.

## Basic Agent Setup

Start by creating the basic structure for your agent:

```go
package main

import (
    "context"
    "io"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/timestamppb"

    pb "github.com/owulveryck/agenthub/broker/internal/grpc"
)

const (
    agentHubAddr = "localhost:50051"
    myAgentID    = "my_agent_processor"
)

func main() {
    // Connect to the AgentHub broker
    conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewEventBusClient(conn)
    ctx := context.Background()

    // Start task subscription
    go subscribeToTasks(ctx, client)

    // Keep the agent running
    log.Printf("Agent %s started. Press Ctrl+C to stop.", myAgentID)
    select {} // Block forever
}
```

## Subscribing to Tasks

Implement the task subscription mechanism:

```go
func subscribeToTasks(ctx context.Context, client pb.EventBusClient) {
    log.Printf("Agent %s subscribing to tasks...", myAgentID)

    req := &pb.SubscribeToTasksRequest{
        AgentId: myAgentID,
        // TaskTypes: []string{"math_calculation", "data_processing"}, // Optional: filter task types
    }

    stream, err := client.SubscribeToTasks(ctx, req)
    if err != nil {
        log.Printf("Error subscribing to tasks: %v", err)
        return
    }

    log.Printf("Successfully subscribed to tasks for agent %s", myAgentID)

    for {
        task, err := stream.Recv()
        if err == io.EOF {
            log.Printf("Task subscription stream closed by server")
            return
        }
        if err != nil {
            if ctx.Err() != nil {
                log.Printf("Task subscription context cancelled: %v", ctx.Err())
                return
            }
            log.Printf("Error receiving task: %v", err)
            return
        }

        log.Printf("Received task: %s (type: %s) from agent: %s",
            task.GetTaskId(), task.GetTaskType(), task.GetRequesterAgentId())

        // Process the task asynchronously
        go processTask(ctx, task, client)
    }
}
```

## Processing Tasks

Create a task processor that handles different task types:

```go
func processTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) {
    log.Printf("Processing task %s of type '%s'", task.GetTaskId(), task.GetTaskType())

    // Send initial progress update
    sendProgress(ctx, task, 10, "Task received and starting", client)

    // Process based on task type
    var result *structpb.Struct
    var status pb.TaskStatus
    var errorMsg string

    switch task.GetTaskType() {
    case "greeting":
        result, status, errorMsg = processGreetingTask(ctx, task, client)
    case "math_calculation":
        result, status, errorMsg = processMathTask(ctx, task, client)
    case "data_processing":
        result, status, errorMsg = processDataTask(ctx, task, client)
    case "file_conversion":
        result, status, errorMsg = processFileConversionTask(ctx, task, client)
    default:
        status = pb.TaskStatus_TASK_STATUS_FAILED
        errorMsg = fmt.Sprintf("Unknown task type: %s", task.GetTaskType())
    }

    // Send final progress update
    sendProgress(ctx, task, 100, "Task processing completed", client)

    // Send the result
    sendResult(ctx, task, result, status, errorMsg, client)
}
```

## Implementing Specific Task Handlers

### Math Calculation Handler

```go
func processMathTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) (*structpb.Struct, pb.TaskStatus, string) {
    sendProgress(ctx, task, 25, "Parsing math parameters", client)

    params := task.GetParameters().AsMap()
    operation, ok := params["operation"].(string)
    if !ok {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Missing operation parameter"
    }

    a, aOk := params["a"].(float64)
    b, bOk := params["b"].(float64)
    if !aOk || !bOk {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Invalid numeric parameters"
    }

    sendProgress(ctx, task, 50, "Performing calculation", client)

    var calcResult float64
    switch operation {
    case "add":
        calcResult = a + b
    case "subtract":
        calcResult = a - b
    case "multiply":
        calcResult = a * b
    case "divide":
        if b == 0 {
            return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Division by zero"
        }
        calcResult = a / b
    case "power":
        calcResult = math.Pow(a, b)
    default:
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Unknown operation: " + operation
    }

    sendProgress(ctx, task, 90, "Formatting result", client)

    result, _ := structpb.NewStruct(map[string]interface{}{
        "operation": operation,
        "operand_a": a,
        "operand_b": b,
        "result":    calcResult,
        "timestamp": time.Now().Format(time.RFC3339),
    })

    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
```

### Data Processing Handler

```go
func processDataTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) (*structpb.Struct, pb.TaskStatus, string) {
    sendProgress(ctx, task, 20, "Validating data parameters", client)

    params := task.GetParameters().AsMap()
    datasetPath, ok := params["dataset_path"].(string)
    if !ok {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Missing dataset_path parameter"
    }

    analysisType, ok := params["analysis_type"].(string)
    if !ok {
        analysisType = "basic_summary" // Default
    }

    sendProgress(ctx, task, 40, "Loading dataset", client)

    // Simulate data loading
    time.Sleep(1 * time.Second)

    sendProgress(ctx, task, 70, "Performing analysis", client)

    // Simulate data processing based on analysis type
    var analysisResult map[string]interface{}
    switch analysisType {
    case "summary_statistics":
        analysisResult = map[string]interface{}{
            "total_records": 1500,
            "mean_value":    42.7,
            "median_value":  41.2,
            "std_dev":       8.3,
        }
    case "correlation_analysis":
        analysisResult = map[string]interface{}{
            "correlation_matrix": [][]float64{{1.0, 0.75}, {0.75, 1.0}},
            "significant_pairs":  []string{"feature_a:feature_b"},
        }
    default:
        analysisResult = map[string]interface{}{
            "status": "basic analysis completed",
            "record_count": 1500,
        }
    }

    sendProgress(ctx, task, 90, "Formatting results", client)

    result, _ := structpb.NewStruct(map[string]interface{}{
        "dataset_path":   datasetPath,
        "analysis_type":  analysisType,
        "analysis_result": analysisResult,
        "processed_at":   time.Now().Format(time.RFC3339),
    })

    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
```

### File Conversion Handler

```go
func processFileConversionTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) (*structpb.Struct, pb.TaskStatus, string) {
    sendProgress(ctx, task, 15, "Validating file parameters", client)

    params := task.GetParameters().AsMap()
    inputPath, ok := params["input_path"].(string)
    if !ok {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Missing input_path parameter"
    }

    outputFormat, ok := params["output_format"].(string)
    if !ok {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Missing output_format parameter"
    }

    sendProgress(ctx, task, 30, "Reading input file", client)
    time.Sleep(500 * time.Millisecond) // Simulate file reading

    sendProgress(ctx, task, 60, "Converting file format", client)
    time.Sleep(1 * time.Second) // Simulate conversion

    sendProgress(ctx, task, 85, "Writing output file", client)
    time.Sleep(300 * time.Millisecond) // Simulate file writing

    outputPath := strings.Replace(inputPath, filepath.Ext(inputPath), "."+outputFormat, 1)

    result, _ := structpb.NewStruct(map[string]interface{}{
        "input_path":     inputPath,
        "output_path":    outputPath,
        "output_format":  outputFormat,
        "file_size":      "2.5MB",
        "conversion_time": "1.8s",
        "converted_at":   time.Now().Format(time.RFC3339),
    })

    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
```

## Sending Progress Updates

Keep requesters informed about task progress:

```go
func sendProgress(ctx context.Context, task *pb.TaskMessage, percentage int32, message string, client pb.EventBusClient) {
    progress := &pb.TaskProgress{
        TaskId:             task.GetTaskId(),
        Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
        ProgressMessage:    message,
        ProgressPercentage: percentage,
        ExecutorAgentId:    myAgentID,
        UpdatedAt:          timestamppb.Now(),
    }

    req := &pb.PublishTaskProgressRequest{Progress: progress}

    if _, err := client.PublishTaskProgress(ctx, req); err != nil {
        log.Printf("Error publishing progress for task %s: %v", task.GetTaskId(), err)
    } else {
        log.Printf("Progress for task %s: %d%% - %s", task.GetTaskId(), percentage, message)
    }
}
```

## Sending Task Results

Send the final result back to the requester:

```go
func sendResult(ctx context.Context, task *pb.TaskMessage, result *structpb.Struct, status pb.TaskStatus, errorMsg string, client pb.EventBusClient) {
    taskResult := &pb.TaskResult{
        TaskId:          task.GetTaskId(),
        Status:          status,
        Result:          result,
        ErrorMessage:    errorMsg,
        ExecutorAgentId: myAgentID,
        CompletedAt:     timestamppb.Now(),
        ExecutionMetadata: createExecutionMetadata(task),
    }

    req := &pb.PublishTaskResultRequest{Result: taskResult}

    if _, err := client.PublishTaskResult(ctx, req); err != nil {
        log.Printf("Error publishing result for task %s: %v", task.GetTaskId(), err)
    } else {
        log.Printf("Published result for task %s with status %s", task.GetTaskId(), status.String())
    }
}

func createExecutionMetadata(task *pb.TaskMessage) *structpb.Struct {
    metadata, _ := structpb.NewStruct(map[string]interface{}{
        "agent_id":      myAgentID,
        "agent_version": "1.0.0",
        "execution_time": time.Since(task.GetCreatedAt().AsTime()).String(),
        "task_priority": task.GetPriority().String(),
    })
    return metadata
}
```

## Advanced Features

### Task Validation

Add validation to ensure tasks can be processed:

```go
func validateTask(task *pb.TaskMessage) error {
    if task.GetTaskId() == "" {
        return fmt.Errorf("task ID cannot be empty")
    }

    if task.GetTaskType() == "" {
        return fmt.Errorf("task type cannot be empty")
    }

    // Check if deadline has passed
    if deadline := task.GetDeadline(); deadline != nil {
        if time.Now().After(deadline.AsTime()) {
            return fmt.Errorf("task deadline has passed")
        }
    }

    return nil
}
```

### Graceful Shutdown

Handle shutdown gracefully:

```go
func main() {
    // ... setup code ...

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    ctx, cancel := context.WithCancel(context.Background())

    // Start agent
    go subscribeToTasks(ctx, client)

    // Wait for shutdown signal
    <-sigChan
    log.Println("Shutdown signal received, stopping agent...")

    cancel() // Cancel context to stop subscriptions
    time.Sleep(2 * time.Second) // Allow graceful shutdown

    log.Println("Agent stopped")
}
```

### Task Capacity Management

Limit concurrent task processing:

```go
type Agent struct {
    client      pb.EventBusClient
    agentID     string
    taskSemaphore chan struct{} // Limit concurrent tasks
}

func NewAgent(client pb.EventBusClient, agentID string, maxConcurrentTasks int) *Agent {
    return &Agent{
        client:      client,
        agentID:     agentID,
        taskSemaphore: make(chan struct{}, maxConcurrentTasks),
    }
}

func (a *Agent) processTask(ctx context.Context, task *pb.TaskMessage) {
    // Acquire semaphore slot
    select {
    case a.taskSemaphore <- struct{}{}:
        defer func() { <-a.taskSemaphore }() // Release when done
    case <-ctx.Done():
        return
    }

    // Process the task...
}
```

## Best Practices

1. **Always validate task parameters** before processing
2. **Send regular progress updates** for long-running tasks
3. **Handle errors gracefully** and provide meaningful error messages
4. **Use timeouts** for external operations to prevent hanging
5. **Log extensively** for debugging and monitoring
6. **Implement health checks** to report agent status
7. **Support graceful shutdown** to finish current tasks before stopping
8. **Limit concurrent tasks** to prevent resource exhaustion

Your agent is now ready to receive and process tasks from other agents in the system!