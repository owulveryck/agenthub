---
title: "AgentHub API Reference"
weight: 10
description: "Complete technical reference for the AgentHub API, including all gRPC services, message types, and operational details."
---

# AgentHub API Reference

This document provides complete technical reference for the AgentHub API, including all gRPC services, message types, and operational details.

## gRPC Service Definition

The AgentHub broker implements the `EventBus` service as defined in [proto/eventbus.proto:100](proto/eventbus.proto:100):

```protobuf
service EventBus {
  // Publisher methods
  rpc PublishTask (PublishTaskRequest) returns (PublishResponse);
  rpc PublishTaskResult (PublishTaskResultRequest) returns (PublishResponse);
  rpc PublishTaskProgress (PublishTaskProgressRequest) returns (PublishResponse);

  // Subscriber methods
  rpc SubscribeToTasks (SubscribeToTasksRequest) returns (stream TaskMessage);
  rpc SubscribeToTaskResults (SubscribeToTaskResultsRequest) returns (stream TaskResult);
  rpc SubscribeToTaskProgress (SubscribeToTaskResultsRequest) returns (stream TaskProgress);
}
```

## Message Types

### Core Task Messages

#### TaskMessage

Represents a task to be executed by an agent.

```protobuf
message TaskMessage {
  string task_id = 1;                                    // Required: Unique identifier
  string task_type = 2;                                  // Required: Type of task
  google.protobuf.Struct parameters = 3;                // Optional: Task parameters
  string requester_agent_id = 4;                        // Required: Requesting agent ID
  string responder_agent_id = 5;                        // Optional: Target agent ID
  google.protobuf.Timestamp deadline = 6;               // Optional: Task deadline
  Priority priority = 7;                                // Optional: Task priority (default: UNSPECIFIED)
  google.protobuf.Struct metadata = 8;                  // Optional: Additional metadata
  google.protobuf.Timestamp created_at = 9;             // Required: Creation timestamp
}
```

**Field Details:**
- `task_id`: Must be unique across all tasks. Recommended format: `{task_type}_{timestamp}_{uuid}`
- `task_type`: Semantic identifier for task category (e.g., "data_analysis", "image_processing")
- `parameters`: Flexible JSON-like structure containing task-specific parameters
- `requester_agent_id`: ID of the agent requesting the task
- `responder_agent_id`: If specified, task is routed only to this agent; otherwise broadcast
- `deadline`: RFC3339 timestamp indicating when task must complete
- `priority`: Task priority level (see Priority enum)
- `metadata`: Additional context information for debugging, routing, or processing
- `created_at`: Timestamp when task was created

#### TaskResult

Represents the completion result of a task.

```protobuf
message TaskResult {
  string task_id = 1;                                    // Required: Reference to original task
  TaskStatus status = 2;                                 // Required: Completion status
  google.protobuf.Struct result = 3;                    // Optional: Task results
  string error_message = 4;                             // Optional: Error details if failed
  string executor_agent_id = 5;                         // Required: Executing agent ID
  google.protobuf.Timestamp completed_at = 6;           // Required: Completion timestamp
  google.protobuf.Struct execution_metadata = 7;        // Optional: Execution details
}
```

**Field Details:**
- `task_id`: Must match the original task's `task_id`
- `status`: Final status of task execution (see TaskStatus enum)
- `result`: Structured result data if task completed successfully
- `error_message`: Human-readable error description if status is FAILED
- `executor_agent_id`: ID of the agent that executed the task
- `completed_at`: Timestamp when task execution finished
- `execution_metadata`: Additional execution details (timing, resource usage, etc.)

#### TaskProgress

Represents progress updates during task execution.

```protobuf
message TaskProgress {
  string task_id = 1;                                    // Required: Reference to original task
  TaskStatus status = 2;                                 // Required: Current status
  string progress_message = 3;                           // Optional: Human-readable progress
  int32 progress_percentage = 4;                         // Optional: Progress as percentage (0-100)
  google.protobuf.Struct progress_data = 5;             // Optional: Structured progress data
  string executor_agent_id = 6;                         // Required: Executing agent ID
  google.protobuf.Timestamp updated_at = 7;             // Required: Progress update timestamp
}
```

**Field Details:**
- `task_id`: Must match the original task's `task_id`
- `status`: Current execution status (typically IN_PROGRESS)
- `progress_message`: Human-readable description of current activity
- `progress_percentage`: Numeric progress indicator (0-100)
- `progress_data`: Structured data about progress (e.g., records processed, files completed)
- `executor_agent_id`: ID of the agent executing the task
- `updated_at`: Timestamp of this progress update

### Enums

#### Priority

Defines task priority levels:

```protobuf
enum Priority {
  PRIORITY_UNSPECIFIED = 0;  // Default priority
  PRIORITY_LOW = 1;          // Background tasks
  PRIORITY_MEDIUM = 2;       // Normal tasks
  PRIORITY_HIGH = 3;         // Important tasks
  PRIORITY_CRITICAL = 4;     // Urgent tasks requiring immediate attention
}
```

**Usage Guidelines:**
- `PRIORITY_LOW`: Batch jobs, maintenance tasks, background processing
- `PRIORITY_MEDIUM`: Standard user requests, regular business operations
- `PRIORITY_HIGH`: User-facing operations, time-sensitive tasks
- `PRIORITY_CRITICAL`: Emergency operations, system alerts, health checks

#### TaskStatus

Defines task execution states:

```protobuf
enum TaskStatus {
  TASK_STATUS_UNSPECIFIED = 0;  // Default/unknown status
  TASK_STATUS_PENDING = 1;      // Task queued, waiting for execution
  TASK_STATUS_IN_PROGRESS = 2;  // Task currently being processed
  TASK_STATUS_COMPLETED = 3;    // Task finished successfully
  TASK_STATUS_FAILED = 4;       // Task failed with error
  TASK_STATUS_CANCELLED = 5;    // Task was cancelled
}
```

**State Transitions:**
```
PENDING → IN_PROGRESS → COMPLETED
         ↓
         FAILED
         ↓
         CANCELLED (from any state)
```

### Request/Response Messages

#### PublishTaskRequest

```protobuf
message PublishTaskRequest {
  TaskMessage task = 1;  // Required: Task to publish
}
```

#### PublishTaskResultRequest

```protobuf
message PublishTaskResultRequest {
  TaskResult result = 1;  // Required: Task result to publish
}
```

#### PublishTaskProgressRequest

```protobuf
message PublishTaskProgressRequest {
  TaskProgress progress = 1;  // Required: Progress update to publish
}
```

#### PublishResponse

```protobuf
message PublishResponse {
  bool success = 1;     // True if message was accepted
  string error = 2;     // Error message if success is false
}
```

#### SubscribeToTasksRequest

```protobuf
message SubscribeToTasksRequest {
  string agent_id = 1;                    // Required: Agent ID to receive tasks for
  repeated string task_types = 2;         // Optional: Filter by task types
}
```

**Filtering Behavior:**
- If `task_types` is empty: Agent receives all tasks addressed to them
- If `task_types` is specified: Agent only receives tasks with matching `task_type`

#### SubscribeToTaskResultsRequest

```protobuf
message SubscribeToTaskResultsRequest {
  string requester_agent_id = 1;          // Required: Agent ID that requested tasks
  repeated string task_ids = 2;           // Optional: Filter by specific task IDs
}
```

**Filtering Behavior:**
- If `task_ids` is empty: Agent receives results for all tasks they requested
- If `task_ids` is specified: Agent only receives results for those specific tasks

## API Operations

### Publishing Operations

#### PublishTask

Publishes a new task for execution.

**Go Example:**
```go
task := &pb.TaskMessage{
    TaskId:           "task_data_analysis_123",
    TaskType:         "data_analysis",
    Parameters:       parametersStruct,
    RequesterAgentId: "analytics_coordinator",
    ResponderAgentId: "data_processor_agent", // Optional
    Priority:         pb.Priority_PRIORITY_HIGH,
    CreatedAt:        timestamppb.Now(),
}

response, err := client.PublishTask(ctx, &pb.PublishTaskRequest{
    Task: task,
})
```

**Validation Rules:**
- `task_id` must be non-empty and unique
- `task_type` must be non-empty
- `requester_agent_id` must be non-empty
- `created_at` must be set

**Error Conditions:**
- `InvalidArgument`: Required fields missing or invalid
- `Internal`: Server error during processing

#### PublishTaskResult

Publishes the result of task execution.

**Go Example:**
```go
result := &pb.TaskResult{
    TaskId:          originalTask.GetTaskId(),
    Status:          pb.TaskStatus_TASK_STATUS_COMPLETED,
    Result:          resultStruct,
    ExecutorAgentId: "data_processor_agent",
    CompletedAt:     timestamppb.Now(),
}

response, err := client.PublishTaskResult(ctx, &pb.PublishTaskResultRequest{
    Result: result,
})
```

#### PublishTaskProgress

Publishes progress updates during task execution.

**Go Example:**
```go
progress := &pb.TaskProgress{
    TaskId:             originalTask.GetTaskId(),
    Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
    ProgressMessage:    "Processing batch 3 of 10",
    ProgressPercentage: 30,
    ExecutorAgentId:    "data_processor_agent",
    UpdatedAt:          timestamppb.Now(),
}

response, err := client.PublishTaskProgress(ctx, &pb.PublishTaskProgressRequest{
    Progress: progress,
})
```

### Subscription Operations

#### SubscribeToTasks

Subscribes to receive tasks assigned to an agent.

**Go Example:**
```go
req := &pb.SubscribeToTasksRequest{
    AgentId:   "data_processor_agent",
    TaskTypes: []string{"data_analysis", "data_transformation"}, // Optional
}

stream, err := client.SubscribeToTasks(ctx, req)
if err != nil {
    return err
}

for {
    task, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    // Process task
    go processTask(ctx, task, client)
}
```

**Stream Behavior:**
- Long-lived bidirectional stream
- Messages are pushed immediately when available
- Stream closes when client disconnects or context is cancelled
- Automatic cleanup removes subscription when stream closes

#### SubscribeToTaskResults

Subscribes to receive results of tasks requested by an agent.

**Go Example:**
```go
req := &pb.SubscribeToTaskResultsRequest{
    RequesterAgentId: "analytics_coordinator",
    TaskIds:          []string{"task_123", "task_456"}, // Optional
}

stream, err := client.SubscribeToTaskResults(ctx, req)
if err != nil {
    return err
}

for {
    result, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    // Handle result
    handleTaskResult(result)
}
```

#### SubscribeToTaskProgress

Subscribes to receive progress updates for tasks requested by an agent.

**Go Example:**
```go
req := &pb.SubscribeToTaskResultsRequest{
    RequesterAgentId: "analytics_coordinator",
}

stream, err := client.SubscribeToTaskProgress(ctx, req)
if err != nil {
    return err
}

for {
    progress, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    // Handle progress update
    updateProgressUI(progress)
}
```

## Error Handling

### gRPC Status Codes

AgentHub uses standard gRPC status codes:

#### InvalidArgument (Code: 3)
**Causes:**
- Missing required fields in request
- Invalid field values (empty task_id, etc.)
- Malformed timestamps or enums

**Example Response:**
```
rpc error: code = InvalidArgument desc = task_id cannot be empty
```

#### Internal (Code: 13)
**Causes:**
- Server-side processing errors
- Message serialization failures
- Resource allocation failures

**Example Response:**
```
rpc error: code = Internal desc = failed to route task
```

#### Unavailable (Code: 14)
**Causes:**
- Broker server not running
- Network connectivity issues
- Server overload

**Example Response:**
```
rpc error: code = Unavailable desc = connection refused
```

### Error Recovery Patterns

#### Retry Logic
```go
func publishTaskWithRetry(ctx context.Context, client pb.EventBusClient, task *pb.TaskMessage) error {
    var lastErr error

    for attempt := 0; attempt < 3; attempt++ {
        _, err := client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
        if err == nil {
            return nil
        }

        lastErr = err

        // Check if error is retryable
        if grpc.Code(err) == codes.InvalidArgument {
            return err // Don't retry validation errors
        }

        // Exponential backoff
        time.Sleep(time.Duration(1<<attempt) * time.Second)
    }

    return lastErr
}
```

#### Stream Reconnection
```go
func subscribeWithReconnect(ctx context.Context, client pb.EventBusClient, agentID string) {
    for {
        err := subscribeToTasks(ctx, client, agentID)
        if ctx.Err() != nil {
            return // Context cancelled
        }

        log.Printf("Subscription failed: %v, reconnecting in 5s...", err)
        time.Sleep(5 * time.Second)
    }
}
```

## Performance Considerations

### Message Size Limits

- **Maximum message size**: 4MB (gRPC default)
- **Recommended message size**: <100KB for optimal performance
- **Large payloads**: Consider using external storage with references

### Throughput Optimization

#### Batch Operations
For high-throughput scenarios, consider batching multiple operations:

```go
// Instead of individual calls
for _, task := range tasks {
    client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
}

// Consider client-side batching
taskBatch := make([]*pb.TaskMessage, 0, 100)
for _, task := range tasks {
    taskBatch = append(taskBatch, task)
    if len(taskBatch) >= 100 {
        publishBatch(ctx, client, taskBatch)
        taskBatch = taskBatch[:0]
    }
}
```

#### Connection Reuse
Reuse gRPC connections for better performance:

```go
// Single connection for multiple operations
conn, err := grpc.Dial(address, opts...)
if err != nil {
    return err
}
defer conn.Close()

client := pb.NewEventBusClient(conn)

// Use client for multiple operations
```

### Memory Management

#### Struct Reuse
Reuse message structs to reduce garbage collection:

```go
var taskPool = sync.Pool{
    New: func() interface{} {
        return &pb.TaskMessage{}
    },
}

func publishTask(params TaskParams) {
    task := taskPool.Get().(*pb.TaskMessage)
    defer taskPool.Put(task)

    // Reset and populate task
    *task = pb.TaskMessage{
        TaskId:   params.ID,
        TaskType: params.Type,
        // ... other fields
    }

    // Publish task
}
```

This completes the comprehensive API reference for AgentHub. All message types, operations, and integration patterns are documented with practical examples and error handling guidance.