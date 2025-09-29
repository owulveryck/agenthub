---
title: "How to Create an A2A Task Publisher"
weight: 20
description: "Learn how to create an agent that publishes Agent2Agent (A2A) protocol-compliant tasks to other agents through the AgentHub EDA broker."
---

# How to Create an A2A Task Publisher

This guide shows you how to create an agent that publishes Agent2Agent (A2A) protocol-compliant tasks to other agents through the AgentHub Event-Driven Architecture (EDA) broker.

## Basic Setup

Using AgentHub's unified abstractions, creating a publisher is straightforward:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/owulveryck/agenthub/internal/agenthub"
    pb "github.com/owulveryck/agenthub/events/a2a"
)

const (
    myAgentID = "my_publisher_agent"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Create configuration with automatic observability
    config := agenthub.NewGRPCConfig("publisher")
    config.HealthPort = "8081" // Unique port for this publisher

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

    // Create A2A task publisher with automatic tracing and metrics
    taskPublisher := &agenthub.A2ATaskPublisher{
        Client:         client.Client,
        TraceManager:   client.TraceManager,
        MetricsManager: client.MetricsManager,
        Logger:         client.Logger,
        ComponentName:  "publisher",
        AgentID:        myAgentID,
    }

    // Your A2A task publishing code goes here
}
```

## Publishing a Simple A2A Task

Here's how to publish a basic A2A task using the A2ATaskPublisher abstraction:

```go
func publishSimpleTask(ctx context.Context, taskPublisher *agenthub.A2ATaskPublisher) error {
    // Create A2A-compliant content parts
    content := []*pb.Part{
        {
            Part: &pb.Part_Text{
                Text: "Hello! Please provide a greeting for Claude.",
            },
        },
    }

    // Publish A2A task using the unified abstraction
    task, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
        TaskType:         "greeting",
        Content:          content,
        RequesterAgentID: myAgentID,
        ResponderAgentID: "agent_demo_subscriber", // Target agent
        Priority:         pb.Priority_PRIORITY_HIGH,
        ContextID:        "ctx_greeting_demo", // Optional: conversation context
    })
    if err != nil {
        return fmt.Errorf("failed to publish greeting task: %w", err)
    }

    taskPublisher.Logger.InfoContext(ctx, "Published A2A greeting task",
        "task_id", task.GetId(),
        "context_id", task.GetContextId())
    return nil
}
```

## Publishing Different Task Types

### Math Calculation Task with A2A Data Parts

```go
func publishMathTask(ctx context.Context, taskPublisher *agenthub.A2ATaskPublisher) error {
    // Create A2A-compliant content with structured data
    content := []*pb.Part{
        {
            Part: &pb.Part_Text{
                Text: "Please perform the following mathematical calculation:",
            },
        },
        {
            Part: &pb.Part_Data{
                Data: &pb.DataPart{
                    Data: &structpb.Struct{
                        Fields: map[string]*structpb.Value{
                            "operation": structpb.NewStringValue("multiply"),
                            "a":         structpb.NewNumberValue(15.0),
                            "b":         structpb.NewNumberValue(7.0),
                        },
                    },
                },
            },
        },
    }

    // Publish A2A math task
    task, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
        TaskType:         "math_calculation",
        Content:          content,
        RequesterAgentID: myAgentID,
        ResponderAgentID: "agent_demo_subscriber",
        Priority:         pb.Priority_PRIORITY_MEDIUM,
        ContextID:        "ctx_math_demo",
    })
    if err != nil {
        return fmt.Errorf("failed to publish math task: %w", err)
    }

    taskPublisher.Logger.InfoContext(ctx, "Published A2A math task",
        "task_id", task.GetId(),
        "operation", "multiply")
    return nil
}
```

### Data Processing Task

```go
func publishDataProcessingTask(ctx context.Context, taskPublisher *agenthub.TaskPublisher) {
    err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
        TaskType: "data_processing",
        Parameters: map[string]interface{}{
            "dataset_path":   "/data/customer_data.csv",
            "analysis_type":  "summary_statistics",
            "output_format":  "json",
            "filters": map[string]interface{}{
                "date_range": "last_30_days",
                "status":     "active",
            },
            // Metadata is handled automatically by TaskPublisher
            "workflow_id": "workflow_123",
            "user_id":     "user_456",
        },
        RequesterAgentID: myAgentID,
        ResponderAgentID: "data_agent",
        Priority:         pb.Priority_PRIORITY_HIGH,
    })
    if err != nil {
        panic(fmt.Sprintf("Failed to publish data processing task: %v", err))
    }
}
```

## Broadcasting Tasks (No Specific Responder)

To broadcast a task to all available agents, omit the `ResponderAgentID`:

```go
func broadcastTask(ctx context.Context, taskPublisher *agenthub.TaskPublisher) {
    err := taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
        TaskType: "announcement",
        Parameters: map[string]interface{}{
            "announcement":    "Server maintenance in 30 minutes",
            "action_required": false,
        },
        RequesterAgentID: myAgentID,
        // ResponderAgentID omitted - will broadcast to all agents
        ResponderAgentID: "",
        Priority:         pb.Priority_PRIORITY_LOW,
    })
    if err != nil {
        panic(fmt.Sprintf("Failed to publish announcement: %v", err))
    }
}
```

## Subscribing to Task Results

As a publisher, you'll want to receive results from tasks you've requested. You can use the AgentHub client directly:

```go
func subscribeToResults(ctx context.Context, client *agenthub.AgentHubClient) {
    req := &pb.SubscribeToTaskResultsRequest{
        RequesterAgentId: myAgentID,
        // TaskIds: []string{"specific_task_id"}, // Optional: filter specific tasks
    }

    stream, err := client.Client.SubscribeToTaskResults(ctx, req)
    if err != nil {
        client.Logger.ErrorContext(ctx, "Error subscribing to results", "error", err)
        return
    }

    client.Logger.InfoContext(ctx, "Subscribed to task results", "agent_id", myAgentID)

    for {
        result, err := stream.Recv()
        if err != nil {
            client.Logger.ErrorContext(ctx, "Error receiving result", "error", err)
            return
        }

        handleTaskResult(ctx, client, result)
    }
}

func handleTaskResult(ctx context.Context, client *agenthub.AgentHubClient, result *pb.TaskResult) {
    client.Logger.InfoContext(ctx, "Received task result",
        "task_id", result.GetTaskId(),
        "status", result.GetStatus().String())

    switch result.GetStatus() {
    case pb.TaskStatus_TASK_STATUS_COMPLETED:
        client.Logger.InfoContext(ctx, "Task completed successfully",
            "task_id", result.GetTaskId(),
            "result", result.GetResult().AsMap())
    case pb.TaskStatus_TASK_STATUS_FAILED:
        client.Logger.ErrorContext(ctx, "Task failed",
            "task_id", result.GetTaskId(),
            "error", result.GetErrorMessage())
    case pb.TaskStatus_TASK_STATUS_CANCELLED:
        client.Logger.InfoContext(ctx, "Task was cancelled",
            "task_id", result.GetTaskId())
    }
}
```

## Monitoring Task Progress

Subscribe to progress updates to track long-running tasks:

```go
func subscribeToProgress(ctx context.Context, client *agenthub.AgentHubClient) {
    req := &pb.SubscribeToTaskResultsRequest{
        RequesterAgentId: myAgentID,
    }

    stream, err := client.Client.SubscribeToTaskProgress(ctx, req)
    if err != nil {
        client.Logger.ErrorContext(ctx, "Error subscribing to progress", "error", err)
        return
    }

    client.Logger.InfoContext(ctx, "Subscribed to task progress", "agent_id", myAgentID)

    for {
        progress, err := stream.Recv()
        if err != nil {
            client.Logger.ErrorContext(ctx, "Error receiving progress", "error", err)
            return
        }

        client.Logger.InfoContext(ctx, "Task progress update",
            "task_id", progress.GetTaskId(),
            "progress_percentage", progress.GetProgressPercentage(),
            "progress_message", progress.GetProgressMessage())
    }
}
```

## Complete Publisher Example

```go
func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Create configuration with automatic observability
    config := agenthub.NewGRPCConfig("publisher")
    config.HealthPort = "8081"

    // Create AgentHub client with built-in observability
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

    // Start the client (enables observability)
    if err := client.Start(ctx); err != nil {
        client.Logger.ErrorContext(ctx, "Failed to start client", "error", err)
        panic(err)
    }

    // Create task publisher with automatic tracing and metrics
    taskPublisher := &agenthub.TaskPublisher{
        Client:         client.Client,
        TraceManager:   client.TraceManager,
        MetricsManager: client.MetricsManager,
        Logger:         client.Logger,
        ComponentName:  "publisher",
    }

    client.Logger.InfoContext(ctx, "Starting publisher demo")

    // Publish various tasks with automatic observability
    publishMathTask(ctx, taskPublisher)
    time.Sleep(2 * time.Second)

    publishDataProcessingTask(ctx, taskPublisher)
    time.Sleep(2 * time.Second)

    broadcastTask(ctx, taskPublisher)

    client.Logger.InfoContext(ctx, "All tasks published! Check subscriber logs for results")
}
```

## Best Practices

1. **Always set a unique task ID**: Use timestamps, UUIDs, or sequential IDs to ensure uniqueness.

2. **Use appropriate priorities**: Reserve `PRIORITY_CRITICAL` for urgent tasks that must be processed immediately.

3. **Set realistic deadlines**: Include deadlines for time-sensitive tasks to help agents prioritize.

4. **Handle results gracefully**: Always subscribe to task results and handle failures appropriately.

5. **Include helpful metadata**: Add context information that might be useful for debugging or auditing.

6. **Validate parameters**: Ensure task parameters are properly structured before publishing.

7. **Use specific responder IDs when possible**: This ensures tasks go to the most appropriate agent.

Your publisher is now ready to send tasks to agents and receive results!