---
title: "A2A Task Reference"
weight: 50
description: "Comprehensive reference for all task-related message types and operations in the Agent2Agent protocol implementation."
---

# A2A Task Reference

This document provides a comprehensive reference for all task-related message types and operations in the Agent2Agent (A2A) protocol implementation within AgentHub's hybrid Event-Driven Architecture.

## Core A2A Task Types

### A2A Task

The primary message type for managing work requests between agents in the Agent2Agent protocol.

```protobuf
message Task {
  string id = 1;                    // Required: Task identifier
  string context_id = 2;            // Optional: Conversation context
  TaskStatus status = 3;            // Required: Current task status
  repeated Message history = 4;     // Message history for this task
  repeated Artifact artifacts = 5;  // Task output artifacts
  google.protobuf.Struct metadata = 6; // Task metadata
}
```

#### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Globally unique identifier for the task |
| `context_id` | string | No | Groups related tasks in a workflow or conversation |
| `status` | TaskStatus | Yes | Current execution state and last update |
| `history` | Message[] | No | Complete message history for this task |
| `artifacts` | Artifact[] | No | Output artifacts produced by the task |
| `metadata` | Struct | No | Additional context information |

#### Task ID Format

Task IDs should be globally unique and meaningful for debugging:

```go
// Recommended formats:
taskID := fmt.Sprintf("task_%s_%d", taskType, time.Now().Unix())
taskID := fmt.Sprintf("task_%s_%s", taskType, uuid.New().String())
taskID := fmt.Sprintf("%s_%s_%d", requesterID, taskType, sequence)
```

### A2A TaskStatus

Represents the current state and latest update for a task.

```protobuf
message TaskStatus {
  TaskState state = 1;              // Current task state
  Message update = 2;               // Status update message
  google.protobuf.Timestamp timestamp = 3; // Status timestamp
}
```

#### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `state` | TaskState | Yes | Current execution state |
| `update` | Message | No | Latest status message from the executing agent |
| `timestamp` | Timestamp | Yes | When this status was last updated |

### A2A Message

Agent-to-agent communication within task context.

```protobuf
message Message {
  string message_id = 1;       // Required: Unique message identifier
  string context_id = 2;       // Optional: Conversation context
  string task_id = 3;          // Optional: Associated task
  Role role = 4;               // Required: USER or AGENT
  repeated Part content = 5;   // Required: Message content parts
  google.protobuf.Struct metadata = 6; // Optional: Additional metadata
  repeated string extensions = 7;       // Optional: Protocol extensions
}
```

#### Message Content Parts

Messages contain structured content using A2A Part definitions:

```protobuf
message Part {
  oneof part {
    string text = 1;           // Text content
    DataPart data = 2;         // Structured data
    FilePart file = 3;         // File reference
  }
}

message DataPart {
  google.protobuf.Struct data = 1;    // Structured data content
  string description = 2;             // Optional data description
}

message FilePart {
  string file_id = 1;                 // File identifier or URI
  string filename = 2;                // Original filename
  string mime_type = 3;               // MIME type
  int64 size_bytes = 4;               // File size in bytes
  google.protobuf.Struct metadata = 5; // Additional file metadata
}
```

### A2A Artifact

Structured output produced by completed tasks.

```protobuf
message Artifact {
  string artifact_id = 1;           // Required: Artifact identifier
  string name = 2;                  // Human-readable name
  string description = 3;           // Artifact description
  repeated Part parts = 4;          // Artifact content parts
  google.protobuf.Struct metadata = 5; // Artifact metadata
}
```

#### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `artifact_id` | string | Yes | Unique identifier for this artifact |
| `name` | string | No | Human-readable artifact name |
| `description` | string | No | Description of the artifact contents |
| `parts` | Part[] | Yes | Structured content using A2A Part format |
| `metadata` | Struct | No | Additional artifact information |

## Enumerations

### TaskState

Current state of A2A task execution.

```protobuf
enum TaskState {
  TASK_STATE_SUBMITTED = 0;    // Task created and submitted
  TASK_STATE_WORKING = 1;      // Task in progress
  TASK_STATE_COMPLETED = 2;    // Task completed successfully
  TASK_STATE_FAILED = 3;       // Task failed with error
  TASK_STATE_CANCELLED = 4;    // Task cancelled
}
```

#### State Transition Rules

Valid state transitions:

```
TASK_STATE_SUBMITTED → TASK_STATE_WORKING → TASK_STATE_COMPLETED
TASK_STATE_SUBMITTED → TASK_STATE_WORKING → TASK_STATE_FAILED
TASK_STATE_SUBMITTED → TASK_STATE_WORKING → TASK_STATE_CANCELLED
TASK_STATE_SUBMITTED → TASK_STATE_CANCELLED (before execution starts)
```

Invalid transitions:
- Any state → TASK_STATE_SUBMITTED
- TASK_STATE_COMPLETED → any other state
- TASK_STATE_FAILED → any other state (except for retry scenarios)

### Role

Identifies the role of the message sender in A2A communication.

```protobuf
enum Role {
  USER = 0;    // Message from requesting agent
  AGENT = 1;   // Message from responding agent
}
```

### Priority

Task priority levels for scheduling and resource allocation.

```protobuf
enum Priority {
  PRIORITY_UNSPECIFIED = 0;  // Default value, treated as MEDIUM
  PRIORITY_LOW = 1;          // Low priority, can be delayed
  PRIORITY_MEDIUM = 2;       // Normal priority
  PRIORITY_HIGH = 3;         // High priority, expedited processing
  PRIORITY_CRITICAL = 4;     // Critical priority, immediate processing
}
```

#### Priority Usage Guidelines

| Priority | Use Cases | SLA Expectations |
|----------|-----------|------------------|
| `LOW` | Background jobs, cleanup tasks, analytics | Hours to days |
| `MEDIUM` | Standard user requests, routine processing | Minutes to hours |
| `HIGH` | User-visible operations, time-sensitive tasks | Seconds to minutes |
| `CRITICAL` | Emergency operations, system health tasks | Immediate |

## AgentHub EDA Request/Response Messages

### Task Publishing

#### PublishTaskUpdateRequest

Request to publish a task status update through the EDA broker.

```protobuf
message PublishTaskUpdateRequest {
  a2a.Task task = 1;                      // Updated A2A task
  AgentEventMetadata routing = 2;         // EDA routing metadata
}
```

#### PublishTaskArtifactRequest

Request to publish a task artifact through the EDA broker.

```protobuf
message PublishTaskArtifactRequest {
  string task_id = 1;                     // Associated task ID
  a2a.Artifact artifact = 2;              // A2A artifact
  AgentEventMetadata routing = 3;         // EDA routing metadata
}
```

### Task Subscription

#### SubscribeToTasksRequest

Request to subscribe to A2A task events through the EDA broker.

```protobuf
message SubscribeToTasksRequest {
  string agent_id = 1;                    // Agent ID for subscription
  repeated string task_types = 2;         // Optional task type filter
  repeated a2a.TaskState states = 3;      // Optional state filter
}
```

#### Usage Examples

```go
// Subscribe to all tasks for this agent
req := &pb.SubscribeToTasksRequest{
    AgentId: "data_processor_01",
}

// Subscribe only to working and completed tasks
req := &pb.SubscribeToTasksRequest{
    AgentId: "workflow_orchestrator",
    States: []a2a.TaskState{
        a2a.TaskState_TASK_STATE_WORKING,
        a2a.TaskState_TASK_STATE_COMPLETED,
    },
}
```

### Task Management

#### GetTaskRequest

Request to retrieve the current state of an A2A task.

```protobuf
message GetTaskRequest {
  string task_id = 1;                     // Task identifier
  int32 history_length = 2;               // History limit (optional)
}
```

#### CancelTaskRequest

Request to cancel an active A2A task.

```protobuf
message CancelTaskRequest {
  string task_id = 1;                     // Task to cancel
  string reason = 2;                      // Cancellation reason
}
```

#### ListTasksRequest

Request to list A2A tasks matching criteria.

```protobuf
message ListTasksRequest {
  string agent_id = 1;                    // Filter by agent
  repeated a2a.TaskState states = 2;      // Filter by states
  google.protobuf.Timestamp since = 3;    // Filter by timestamp
  int32 limit = 4;                        // Results limit
}
```

## gRPC Service Methods

### Task Publishing Methods

#### PublishTaskUpdate

Publishes a task status update to the EDA broker.

```go
rpc PublishTaskUpdate (PublishTaskUpdateRequest) returns (PublishResponse);
```

**Example:**
```go
// Create updated task status
status := &a2a.TaskStatus{
    State: a2a.TaskState_TASK_STATE_WORKING,
    Update: &a2a.Message{
        MessageId: "msg_" + uuid.New().String(),
        TaskId:    taskID,
        Role:      a2a.Role_AGENT,
        Content: []*a2a.Part{
            {
                Part: &a2a.Part_Text{
                    Text: "Processing data analysis...",
                },
            },
        },
    },
    Timestamp: timestamppb.Now(),
}

task := &a2a.Task{
    Id:     taskID,
    Status: status,
}

req := &pb.PublishTaskUpdateRequest{
    Task: task,
    Routing: &pb.AgentEventMetadata{
        FromAgentId: "processor_01",
        EventType:   "task.status_update",
    },
}
res, err := client.PublishTaskUpdate(ctx, req)
```

#### PublishTaskArtifact

Publishes a task artifact to the EDA broker.

```go
rpc PublishTaskArtifact (PublishTaskArtifactRequest) returns (PublishResponse);
```

**Example:**
```go
// Create artifact with results
artifact := &a2a.Artifact{
    ArtifactId:  "artifact_" + uuid.New().String(),
    Name:        "Analysis Results",
    Description: "Statistical analysis of sales data",
    Parts: []*a2a.Part{
        {
            Part: &a2a.Part_Data{
                Data: &a2a.DataPart{
                    Data: structData, // Contains analysis results
                    Description: "Sales analysis summary statistics",
                },
            },
        },
        {
            Part: &a2a.Part_File{
                File: &a2a.FilePart{
                    FileId:   "file_123",
                    Filename: "analysis_report.pdf",
                    MimeType: "application/pdf",
                    SizeBytes: 1024576,
                },
            },
        },
    },
}

req := &pb.PublishTaskArtifactRequest{
    TaskId:   taskID,
    Artifact: artifact,
    Routing: &pb.AgentEventMetadata{
        FromAgentId: "processor_01",
        EventType:   "task.artifact",
    },
}
res, err := client.PublishTaskArtifact(ctx, req)
```

### Task Subscription Methods

#### SubscribeToTasks

Subscribes to receive A2A task events through the EDA broker.

```go
rpc SubscribeToTasks (SubscribeToTasksRequest) returns (stream AgentEvent);
```

**Returns:** Stream of `AgentEvent` objects containing A2A task updates

**Example:**
```go
req := &pb.SubscribeToTasksRequest{
    AgentId: "processor_01",
    States: []a2a.TaskState{a2a.TaskState_TASK_STATE_SUBMITTED},
}
stream, err := client.SubscribeToTasks(ctx, req)

for {
    event, err := stream.Recv()
    if err != nil {
        break
    }

    // Extract A2A task from event
    if task := event.GetTask(); task != nil {
        go processA2ATask(task)
    }
}
```

### Task Management Methods

#### GetTask

Retrieves the current state of an A2A task by ID.

```go
rpc GetTask (GetTaskRequest) returns (a2a.Task);
```

#### CancelTask

Cancels an active A2A task and notifies subscribers.

```go
rpc CancelTask (CancelTaskRequest) returns (a2a.Task);
```

#### ListTasks

Returns A2A tasks matching the specified criteria.

```go
rpc ListTasks (ListTasksRequest) returns (ListTasksResponse);
```

## A2A Task Workflow Patterns

### Simple Request-Response

```go
// 1. Agent A creates and publishes task request
task := &a2a.Task{
    Id:        "task_analysis_123",
    ContextId: "workflow_456",
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            MessageId: "msg_" + uuid.New().String(),
            TaskId:    "task_analysis_123",
            Role:      a2a.Role_USER,
            Content: []*a2a.Part{
                {
                    Part: &a2a.Part_Text{
                        Text: "Please analyze the Q4 sales data",
                    },
                },
                {
                    Part: &a2a.Part_Data{
                        Data: &a2a.DataPart{
                            Data: dataStruct, // Contains parameters
                        },
                    },
                },
            },
        },
        Timestamp: timestamppb.Now(),
    },
}

// 2. Agent B receives task and updates status to WORKING
// 3. Agent B publishes progress updates during execution
// 4. Agent B publishes final artifacts and COMPLETED status
```

### Multi-Step Workflow

```go
// 1. Orchestrator creates main task
mainTask := &a2a.Task{
    Id:        "workflow_main_789",
    ContextId: "workflow_context_789",
    // ... initial message
}

// 2. Create subtasks with same context_id
subtask1 := &a2a.Task{
    Id:        "subtask_data_prep_790",
    ContextId: "workflow_context_789", // Same context
    // ... data preparation request
}

subtask2 := &a2a.Task{
    Id:        "subtask_analysis_791",
    ContextId: "workflow_context_789", // Same context
    // ... analysis request (depends on subtask1)
}

// 3. Tasks linked by context_id for workflow tracking
```

## Error Handling Reference

### A2A Task Error Patterns

#### Parameter Validation Errors
```go
// Task fails with validation error
failedTask := &a2a.Task{
    Id: taskID,
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_FAILED,
        Update: &a2a.Message{
            Role: a2a.Role_AGENT,
            Content: []*a2a.Part{
                {
                    Part: &a2a.Part_Text{
                        Text: "Task failed: Required parameter 'dataset_path' is missing",
                    },
                },
                {
                    Part: &a2a.Part_Data{
                        Data: &a2a.DataPart{
                            Data: errorDetails, // Structured error info
                            Description: "Validation error details",
                        },
                    },
                },
            },
        },
        Timestamp: timestamppb.Now(),
    },
}
```

#### Resource Errors
```go
// Task fails due to resource unavailability
failedTask := &a2a.Task{
    Id: taskID,
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_FAILED,
        Update: &a2a.Message{
            Role: a2a.Role_AGENT,
            Content: []*a2a.Part{
                {
                    Part: &a2a.Part_Text{
                        Text: "Cannot access dataset file: /data/sales_2024.csv",
                    },
                },
            },
        },
        Timestamp: timestamppb.Now(),
    },
}
```

### Error Handling Best Practices

1. **Use structured error messages** in A2A format for programmatic handling
2. **Include actionable error descriptions** in text parts for human operators
3. **Add detailed error data** in data parts for debugging and retry logic
4. **Maintain task history** to preserve error context
5. **Consider partial results** using artifacts for partially successful operations

## Migration from Legacy EventBus

### Message Type Mappings

| Legacy EventBus | A2A Equivalent | Notes |
|-----------------|----------------|-------|
| `TaskMessage` | `a2a.Task` with initial `Message` | Task creation with request message |
| `TaskResult` | `a2a.Task` with final `Artifact` | Task completion with result artifacts |
| `TaskProgress` | `a2a.Task` with status `Message` | Progress updates via status messages |
| `TaskStatus` enum | `a2a.TaskState` enum | State names updated (e.g., `IN_PROGRESS` → `TASK_STATE_WORKING`) |

### API Method Mappings

| Legacy EventBus | A2A Equivalent | Notes |
|-----------------|----------------|-------|
| `PublishTask` | `PublishTaskUpdate` | Now publishes A2A task objects |
| `PublishTaskResult` | `PublishTaskArtifact` | Results published as artifacts |
| `PublishTaskProgress` | `PublishTaskUpdate` | Progress via task status updates |
| `SubscribeToTasks` | `SubscribeToTasks` | Now returns A2A task events |
| `SubscribeToTaskResults` | `SubscribeToTasks` (filtered) | Filter by COMPLETED state |

This reference provides the complete specification for A2A task-related messages and operations in the AgentHub Event-Driven Architecture, enabling robust distributed task coordination with full Agent2Agent protocol compliance.