---
title: "How to Create an A2A Task Subscriber (Agent)"
weight: 30
description: "Learn how to create an agent that can receive, process, and respond to Agent2Agent (A2A) protocol tasks through the AgentHub EDA broker using A2A-compliant abstractions."
---

# How to Create an A2A Task Subscriber (Agent)

This guide shows you how to create an agent that can receive, process, and respond to Agent2Agent (A2A) protocol tasks through the AgentHub Event-Driven Architecture (EDA) broker using AgentHub's A2A-compliant abstractions.

## Basic Agent Setup

Start by creating the basic structure for your agent using the unified abstraction:

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/owulveryck/agenthub/internal/agenthub"
    pb "github.com/owulveryck/agenthub/events/a2a"
    "google.golang.org/protobuf/types/known/structpb"
)

const (
    agentID = "my_agent_processor"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create configuration with automatic observability
    config := agenthub.NewGRPCConfig("subscriber")
    config.HealthPort = "8082" // Unique port for this agent

    // Create AgentHub client with built-in observability
    client, err := agenthub.NewAgentHubClient(config)
    if err != nil {
        panic("Failed to create AgentHub client: " + err.Error())
    }

    // Automatic graceful shutdown
    defer func() {
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer shutdownCancel()
        if err := client.Shutdown(shutdownCtx); err != nil {
            client.Logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
        }
    }()

    // Start the client (enables observability)
    if err := client.Start(ctx); err != nil {
        client.Logger.ErrorContext(ctx, "Failed to start client", "error", err)
        panic(err)
    }

    // Create A2A task subscriber with automatic observability
    taskSubscriber := agenthub.NewA2ATaskSubscriber(client, agentID)

    // Register A2A task handlers (see below for examples)
    taskSubscriber.RegisterDefaultHandlers()

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        client.Logger.Info("Received shutdown signal")
        cancel()
    }()

    client.Logger.InfoContext(ctx, "Starting subscriber agent")

    // Start task subscription (with automatic observability)
    go func() {
        if err := taskSubscriber.SubscribeToTasks(ctx); err != nil {
            client.Logger.ErrorContext(ctx, "Task subscription failed", "error", err)
        }
    }()

    // Optional: Subscribe to task results if this agent also publishes tasks
    go func() {
        if err := taskSubscriber.SubscribeToTaskResults(ctx); err != nil {
            client.Logger.ErrorContext(ctx, "Task result subscription failed", "error", err)
        }
    }()

    client.Logger.InfoContext(ctx, "Agent started with observability. Listening for tasks.")

    // Wait for context cancellation
    <-ctx.Done()
    client.Logger.Info("Agent shutdown complete")
}
```

## Default Task Handlers

The `RegisterDefaultHandlers()` method provides built-in handlers for common task types:

- **`greeting`**: Simple greeting with name parameter
- **`math_calculation`**: Basic arithmetic operations (add, subtract, multiply, divide)
- **`random_number`**: Random number generation with seed

## Custom Task Handlers

### Simple Custom Handler

Add your own task handlers using `RegisterTaskHandler()`:

```go
func setupCustomHandlers(taskSubscriber *agenthub.TaskSubscriber) {
    // Register a custom data processing handler
    taskSubscriber.RegisterTaskHandler("data_processing", handleDataProcessing)

    // Register a file conversion handler
    taskSubscriber.RegisterTaskHandler("file_conversion", handleFileConversion)

    // Register a status check handler
    taskSubscriber.RegisterTaskHandler("status_check", handleStatusCheck)
}

func handleDataProcessing(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
    params := task.GetParameters()
    datasetPath := params.Fields["dataset_path"].GetStringValue()
    analysisType := params.Fields["analysis_type"].GetStringValue()

    if datasetPath == "" {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "dataset_path parameter is required"
    }

    // Simulate data processing
    time.Sleep(2 * time.Second)

    result, err := structpb.NewStruct(map[string]interface{}{
        "dataset_path":    datasetPath,
        "analysis_type":   analysisType,
        "records_processed": 1500,
        "processing_time": "2.1s",
        "summary": map[string]interface{}{
            "mean":   42.7,
            "median": 41.2,
            "stddev": 8.3,
        },
        "processed_at": time.Now().Format(time.RFC3339),
    })

    if err != nil {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Failed to create result structure"
    }

    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
```

### Advanced Handler with Validation

```go
func handleFileConversion(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
    params := task.GetParameters()

    // Extract and validate parameters
    inputPath := params.Fields["input_path"].GetStringValue()
    outputFormat := params.Fields["output_format"].GetStringValue()

    if inputPath == "" {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "input_path parameter is required"
    }

    if outputFormat == "" {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "output_format parameter is required"
    }

    // Validate output format
    validFormats := []string{"pdf", "docx", "txt", "html"}
    isValidFormat := false
    for _, format := range validFormats {
        if outputFormat == format {
            isValidFormat = true
            break
        }
    }

    if !isValidFormat {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("unsupported output format: %s", outputFormat)
    }

    // Simulate file conversion process
    time.Sleep(1 * time.Second)

    outputPath := strings.Replace(inputPath, filepath.Ext(inputPath), "."+outputFormat, 1)

    result, err := structpb.NewStruct(map[string]interface{}{
        "input_path":      inputPath,
        "output_path":     outputPath,
        "output_format":   outputFormat,
        "file_size":       "2.5MB",
        "conversion_time": "1.2s",
        "status":          "success",
        "converted_at":    time.Now().Format(time.RFC3339),
    })

    if err != nil {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Failed to create result structure"
    }

    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
```

## Handler with External Service Integration

```go
func handleStatusCheck(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
    params := task.GetParameters()
    serviceURL := params.Fields["service_url"].GetStringValue()

    if serviceURL == "" {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "service_url parameter is required"
    }

    // Create HTTP client with timeout
    client := &http.Client{
        Timeout: 10 * time.Second,
    }

    // Perform health check
    resp, err := client.Get(serviceURL + "/health")
    if err != nil {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to reach service: %v", err)
    }
    defer resp.Body.Close()

    // Determine status
    isHealthy := resp.StatusCode >= 200 && resp.StatusCode < 300
    status := "unhealthy"
    if isHealthy {
        status = "healthy"
    }

    result, err := structpb.NewStruct(map[string]interface{}{
        "service_url":     serviceURL,
        "status":          status,
        "status_code":     resp.StatusCode,
        "response_time":   "150ms",
        "checked_at":      time.Now().Format(time.RFC3339),
    })

    if err != nil {
        return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Failed to create result structure"
    }

    return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
```

## Complete Agent Example

Here's a complete agent that handles multiple task types:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "syscall"
    "time"

    "github.com/owulveryck/agenthub/internal/agenthub"
    pb "github.com/owulveryck/agenthub/events/a2a"
    "google.golang.org/protobuf/types/known/structpb"
)

const agentID = "multi_task_agent"

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create AgentHub client with observability
    config := agenthub.NewGRPCConfig("subscriber")
    config.HealthPort = "8082"

    client, err := agenthub.NewAgentHubClient(config)
    if err != nil {
        panic("Failed to create AgentHub client: " + err.Error())
    }

    defer func() {
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer shutdownCancel()
        if err := client.Shutdown(shutdownCtx); err != nil {
            client.Logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
        }
    }()

    if err := client.Start(ctx); err != nil {
        panic(err)
    }

    // Create and configure task subscriber
    taskSubscriber := agenthub.NewTaskSubscriber(client, agentID)

    // Register both default and custom handlers
    taskSubscriber.RegisterDefaultHandlers()
    setupCustomHandlers(taskSubscriber)

    // Graceful shutdown handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        client.Logger.Info("Received shutdown signal")
        cancel()
    }()

    client.Logger.InfoContext(ctx, "Starting multi-task agent")

    // Start subscriptions
    go func() {
        if err := taskSubscriber.SubscribeToTasks(ctx); err != nil {
            client.Logger.ErrorContext(ctx, "Task subscription failed", "error", err)
        }
    }()

    go func() {
        if err := taskSubscriber.SubscribeToTaskResults(ctx); err != nil {
            client.Logger.ErrorContext(ctx, "Task result subscription failed", "error", err)
        }
    }()

    client.Logger.InfoContext(ctx, "Agent ready to process tasks",
        "supported_tasks", []string{"greeting", "math_calculation", "random_number", "data_processing", "file_conversion", "status_check"})

    <-ctx.Done()
    client.Logger.Info("Agent shutdown complete")
}

func setupCustomHandlers(taskSubscriber *agenthub.TaskSubscriber) {
    taskSubscriber.RegisterTaskHandler("data_processing", handleDataProcessing)
    taskSubscriber.RegisterTaskHandler("file_conversion", handleFileConversion)
    taskSubscriber.RegisterTaskHandler("status_check", handleStatusCheck)
}

// ... (include the handler functions from above)
```

## Automatic Features

The unified abstraction provides automatic features:

### Observability
- **Distributed tracing** for each task processing
- **Metrics collection** for processing times and success rates
- **Structured logging** with correlation IDs

### Task Management
- **Automatic result publishing** back to the broker
- **Error handling** and status reporting
- **Progress tracking** capabilities

### Resource Management
- **Graceful shutdown** handling
- **Connection management** to the broker
- **Health endpoints** for monitoring

## Best Practices

1. **Parameter Validation**: Always validate task parameters before processing
   ```go
   if requiredParam == "" {
       return nil, pb.TaskStatus_TASK_STATUS_FAILED, "required_param is missing"
   }
   ```

2. **Error Handling**: Provide meaningful error messages
   ```go
   if err != nil {
       return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Processing failed: %v", err)
   }
   ```

3. **Timeouts**: Use context with timeouts for external operations
   ```go
   client := &http.Client{Timeout: 10 * time.Second}
   ```

4. **Resource Cleanup**: Always clean up resources in handlers
   ```go
   defer file.Close()
   defer resp.Body.Close()
   ```

5. **Structured Results**: Return well-structured result data
   ```go
   result, _ := structpb.NewStruct(map[string]interface{}{
       "status": "completed",
       "timestamp": time.Now().Format(time.RFC3339),
       "data": processedData,
   })
   ```

## Handler Function Signature

All task handlers must implement the `TaskHandler` interface:

```go
type TaskHandler func(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string)
```

**Return values:**
- `*structpb.Struct`: The result data (can be `nil` on failure)
- `pb.TaskStatus`: One of:
  - `pb.TaskStatus_TASK_STATUS_COMPLETED`
  - `pb.TaskStatus_TASK_STATUS_FAILED`
  - `pb.TaskStatus_TASK_STATUS_CANCELLED`
- `string`: Error message (empty string on success)

Your agent is now ready to receive and process tasks from other agents in the system with full observability and automatic result publishing!