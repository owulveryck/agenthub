---
title: "How to Create a Task Publisher"
weight: 20
description: "Learn how to create an agent that publishes Agent2Agent protocol tasks to other agents through the AgentHub broker."
---

# How to Create a Task Publisher

This guide shows you how to create an agent that publishes Agent2Agent protocol tasks to other agents through the AgentHub broker.

## Basic Setup

First, establish a connection to the AgentHub broker and create a client:

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/types/known/structpb"
    "google.golang.org/protobuf/types/known/timestamppb"

    pb "github.com/owulveryck/agenthub/internal/grpc"
)

const (
    agentHubAddr = "localhost:50051"
    myAgentID    = "my_publisher_agent"
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

    // Your task publishing code goes here
}
```

## Publishing a Simple Task

Here's how to publish a basic task:

```go
func publishSimpleTask(ctx context.Context, client pb.EventBusClient) {
    // Create task parameters
    params, err := structpb.NewStruct(map[string]interface{}{
        "message": "Hello from publisher!",
        "priority": "high",
    })
    if err != nil {
        log.Printf("Error creating parameters: %v", err)
        return
    }

    // Create the task message
    task := &pb.TaskMessage{
        TaskId:           generateTaskID("greeting"),
        TaskType:         "greeting",
        Parameters:       params,
        RequesterAgentId: myAgentID,
        ResponderAgentId: "target_agent_id", // Optional: specify target agent
        Priority:         pb.Priority_PRIORITY_HIGH,
        CreatedAt:        timestamppb.Now(),
    }

    // Publish the task
    req := &pb.PublishTaskRequest{Task: task}

    res, err := client.PublishTask(ctx, req)
    if err != nil {
        log.Printf("Error publishing task: %v", err)
        return
    }

    if !res.GetSuccess() {
        log.Printf("Failed to publish task: %s", res.GetError())
        return
    }

    log.Printf("Task %s published successfully", task.GetTaskId())
}

func generateTaskID(taskType string) string {
    return fmt.Sprintf("task_%s_%d", taskType, time.Now().Unix())
}
```

## Publishing Different Task Types

### Math Calculation Task

```go
func publishMathTask(ctx context.Context, client pb.EventBusClient) {
    params, _ := structpb.NewStruct(map[string]interface{}{
        "operation": "multiply",
        "a":         15.0,
        "b":         7.0,
    })

    task := &pb.TaskMessage{
        TaskId:           generateTaskID("math_calculation"),
        TaskType:         "math_calculation",
        Parameters:       params,
        RequesterAgentId: myAgentID,
        ResponderAgentId: "math_agent",
        Priority:         pb.Priority_PRIORITY_MEDIUM,
        CreatedAt:        timestamppb.Now(),
    }

    publishTask(ctx, client, task)
}
```

### Data Processing Task

```go
func publishDataProcessingTask(ctx context.Context, client pb.EventBusClient) {
    params, _ := structpb.NewStruct(map[string]interface{}{
        "dataset_path": "/data/customer_data.csv",
        "analysis_type": "summary_statistics",
        "output_format": "json",
        "filters": map[string]interface{}{
            "date_range": "last_30_days",
            "status": "active",
        },
    })

    task := &pb.TaskMessage{
        TaskId:           generateTaskID("data_processing"),
        TaskType:         "data_processing",
        Parameters:       params,
        RequesterAgentId: myAgentID,
        ResponderAgentId: "data_agent",
        Priority:         pb.Priority_PRIORITY_HIGH,
        Deadline:         timestamppb.New(time.Now().Add(30 * time.Minute)),
        CreatedAt:        timestamppb.Now(),
        Metadata: createMetadata(map[string]interface{}{
            "workflow_id": "workflow_123",
            "user_id": "user_456",
        }),
    }

    publishTask(ctx, client, task)
}

func createMetadata(data map[string]interface{}) *structpb.Struct {
    metadata, _ := structpb.NewStruct(data)
    return metadata
}
```

## Broadcasting Tasks (No Specific Responder)

To broadcast a task to all available agents, omit the `ResponderAgentId`:

```go
func broadcastTask(ctx context.Context, client pb.EventBusClient) {
    params, _ := structpb.NewStruct(map[string]interface{}{
        "announcement": "Server maintenance in 30 minutes",
        "action_required": false,
    })

    task := &pb.TaskMessage{
        TaskId:           generateTaskID("announcement"),
        TaskType:         "announcement",
        Parameters:       params,
        RequesterAgentId: myAgentID,
        // ResponderAgentId omitted - will broadcast to all agents
        Priority:         pb.Priority_PRIORITY_LOW,
        CreatedAt:        timestamppb.Now(),
    }

    publishTask(ctx, client, task)
}
```

## Subscribing to Task Results

As a publisher, you'll want to receive results from tasks you've requested:

```go
func subscribeToResults(ctx context.Context, client pb.EventBusClient) {
    req := &pb.SubscribeToTaskResultsRequest{
        RequesterAgentId: myAgentID,
        // TaskIds: []string{"specific_task_id"}, // Optional: filter specific tasks
    }

    stream, err := client.SubscribeToTaskResults(ctx, req)
    if err != nil {
        log.Printf("Error subscribing to results: %v", err)
        return
    }

    log.Printf("Subscribed to task results for agent %s", myAgentID)

    for {
        result, err := stream.Recv()
        if err != nil {
            log.Printf("Error receiving result: %v", err)
            return
        }

        handleTaskResult(result)
    }
}

func handleTaskResult(result *pb.TaskResult) {
    log.Printf("Received result for task %s: status=%s",
        result.GetTaskId(), result.GetStatus().String())

    switch result.GetStatus() {
    case pb.TaskStatus_TASK_STATUS_COMPLETED:
        log.Printf("Task completed successfully: %+v", result.GetResult().AsMap())
    case pb.TaskStatus_TASK_STATUS_FAILED:
        log.Printf("Task failed: %s", result.GetErrorMessage())
    case pb.TaskStatus_TASK_STATUS_CANCELLED:
        log.Printf("Task was cancelled")
    }
}
```

## Monitoring Task Progress

Subscribe to progress updates to track long-running tasks:

```go
func subscribeToProgress(ctx context.Context, client pb.EventBusClient) {
    req := &pb.SubscribeToTaskResultsRequest{
        RequesterAgentId: myAgentID,
    }

    stream, err := client.SubscribeToTaskProgress(ctx, req)
    if err != nil {
        log.Printf("Error subscribing to progress: %v", err)
        return
    }

    for {
        progress, err := stream.Recv()
        if err != nil {
            log.Printf("Error receiving progress: %v", err)
            return
        }

        log.Printf("Task %s progress: %d%% - %s",
            progress.GetTaskId(),
            progress.GetProgressPercentage(),
            progress.GetProgressMessage())
    }
}
```

## Complete Publisher Example

```go
func main() {
    conn, err := grpc.Dial(eventBusAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewEventBusClient(conn)
    ctx := context.Background()

    // Start result and progress subscribers
    go subscribeToResults(ctx, client)
    go subscribeToProgress(ctx, client)

    // Publish various tasks
    publishMathTask(ctx, client)
    time.Sleep(2 * time.Second)

    publishDataProcessingTask(ctx, client)
    time.Sleep(2 * time.Second)

    broadcastTask(ctx, client)

    // Keep running to receive results
    log.Println("Publisher running. Press Ctrl+C to stop.")
    select {} // Block forever
}

// Helper function to publish any task
func publishTask(ctx context.Context, client pb.EventBusClient, task *pb.TaskMessage) {
    req := &pb.PublishTaskRequest{Task: task}

    res, err := client.PublishTask(ctx, req)
    if err != nil {
        log.Printf("Error publishing task %s: %v", task.GetTaskId(), err)
        return
    }

    if !res.GetSuccess() {
        log.Printf("Failed to publish task %s: %s", task.GetTaskId(), res.GetError())
        return
    }

    log.Printf("Task %s published successfully", task.GetTaskId())
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