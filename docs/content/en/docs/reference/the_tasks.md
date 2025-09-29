---
title: "Task Reference"
weight: 50
description: "Comprehensive reference for all task-related message types and operations in the Agent2Agent protocol implementation."
---

# Task Reference

This document provides a comprehensive reference for all task-related message types and operations in the Agent2Agent protocol implementation.

## Core Task Messages

### TaskMessage

The primary message type for requesting work from other agents.

```protobuf
message TaskMessage {
  string task_id = 1;                                    // Unique identifier
  string task_type = 2;                                  // Type of task
  google.protobuf.Struct parameters = 3;                // Task parameters
  string requester_agent_id = 4;                        // ID of requesting agent
  string responder_agent_id = 5;                        // ID of target agent (optional)
  google.protobuf.Timestamp deadline = 6;               // Optional deadline
  Priority priority = 7;                                // Task priority
  google.protobuf.Struct metadata = 8;                  // Optional metadata
  google.protobuf.Timestamp created_at = 9;             // Creation timestamp
}
```

#### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task_id` | string | Yes | Globally unique identifier for the task |
| `task_type` | string | Yes | Semantic classification of the work to be done |
| `parameters` | Struct | No | Task-specific input parameters |
| `requester_agent_id` | string | Yes | Agent ID that initiated the task |
| `responder_agent_id` | string | No | Specific agent to handle task (blank = broadcast) |
| `deadline` | Timestamp | No | Latest acceptable completion time |
| `priority` | Priority | No | Urgency level (default: PRIORITY_MEDIUM) |
| `metadata` | Struct | No | Additional context information |
| `created_at` | Timestamp | Yes | When the task was created |

#### Task ID Format

Task IDs should be globally unique and meaningful for debugging:

```go
// Recommended formats:
taskID := fmt.Sprintf("task_%s_%d", taskType, time.Now().Unix())
taskID := fmt.Sprintf("task_%s_%s", taskType, uuid.New().String())
taskID := fmt.Sprintf("%s_%s_%d", requesterID, taskType, sequence)
```

#### Task Type Conventions

Use hierarchical naming for task types:

```
domain.operation[.variant]

Examples:
- data.analysis
- data.analysis.trend
- image.generation
- image.generation.portrait
- notification.email
- notification.email.marketing
```

### TaskResult

Response message containing task execution results.

```protobuf
message TaskResult {
  string task_id = 1;                                    // Reference to original task
  TaskStatus status = 2;                                 // Final status
  google.protobuf.Struct result = 3;                    // Task results
  string error_message = 4;                             // Error details if failed
  string executor_agent_id = 5;                         // ID of executing agent
  google.protobuf.Timestamp completed_at = 6;           // Completion timestamp
  google.protobuf.Struct execution_metadata = 7;        // Optional execution details
}
```

#### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task_id` | string | Yes | Must match original TaskMessage.task_id |
| `status` | TaskStatus | Yes | Final execution status |
| `result` | Struct | No | Structured result data (null if failed) |
| `error_message` | string | No | Human-readable error description |
| `executor_agent_id` | string | Yes | Agent that executed the task |
| `completed_at` | Timestamp | Yes | When execution finished |
| `execution_metadata` | Struct | No | Additional execution context |

#### Result Data Patterns

Structure result data for easy consumption:

```go
// Simple scalar result
result, _ := structpb.NewStruct(map[string]interface{}{
    "value": 42.0,
    "unit": "seconds",
})

// Complex structured result
result, _ := structpb.NewStruct(map[string]interface{}{
    "analysis": map[string]interface{}{
        "total_records": 1500,
        "mean_value": 42.7,
        "trends": []string{"increasing", "seasonal"},
    },
    "metadata": map[string]interface{}{
        "processing_time": "2.3s",
        "data_quality": "high",
    },
})

// File-based result
result, _ := structpb.NewStruct(map[string]interface{}{
    "output_file": "/tmp/results/analysis_20240115.json",
    "file_size": 2048576,
    "format": "json",
    "checksum": "sha256:abc123...",
})
```

### TaskProgress

Intermediate progress updates during task execution.

```protobuf
message TaskProgress {
  string task_id = 1;                                    // Reference to original task
  TaskStatus status = 2;                                 // Current status
  string progress_message = 3;                           // Human-readable description
  int32 progress_percentage = 4;                         // Progress as percentage (0-100)
  google.protobuf.Struct progress_data = 5;             // Optional structured progress
  string executor_agent_id = 6;                         // ID of executing agent
  google.protobuf.Timestamp updated_at = 7;             // When this progress was reported
}
```

#### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `task_id` | string | Yes | Must match original TaskMessage.task_id |
| `status` | TaskStatus | Yes | Current execution status (typically IN_PROGRESS) |
| `progress_message` | string | No | Human-readable progress description |
| `progress_percentage` | int32 | No | Completion percentage (0-100) |
| `progress_data` | Struct | No | Structured progress information |
| `executor_agent_id` | string | Yes | Agent reporting the progress |
| `updated_at` | Timestamp | Yes | When this update was generated |

#### Progress Reporting Patterns

```go
// Simple percentage progress
progress := &pb.TaskProgress{
    TaskId:             taskID,
    Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
    ProgressMessage:    "Processing data",
    ProgressPercentage: 45,
    ExecutorAgentId:    agentID,
    UpdatedAt:          timestamppb.Now(),
}

// Detailed progress with structured data
progressData, _ := structpb.NewStruct(map[string]interface{}{
    "phase": "data_analysis",
    "records_processed": 750,
    "total_records": 1500,
    "current_operation": "correlation_analysis",
    "estimated_remaining": "2m30s",
})

progress := &pb.TaskProgress{
    TaskId:           taskID,
    Status:           pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
    ProgressMessage:  "Analyzing correlations",
    ProgressPercentage: 50,
    ProgressData:     progressData,
    ExecutorAgentId:  agentID,
    UpdatedAt:        timestamppb.Now(),
}
```

## Enumerations

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

### TaskStatus

Current state of task execution.

```protobuf
enum TaskStatus {
  TASK_STATUS_UNSPECIFIED = 0;  // Invalid/unknown status
  TASK_STATUS_PENDING = 1;      // Waiting to be processed
  TASK_STATUS_IN_PROGRESS = 2;  // Currently being executed
  TASK_STATUS_COMPLETED = 3;    // Successfully completed
  TASK_STATUS_FAILED = 4;       // Failed during execution
  TASK_STATUS_CANCELLED = 5;    // Cancelled before/during execution
}
```

#### Status Transition Rules

Valid status transitions:

```
PENDING → IN_PROGRESS → COMPLETED
PENDING → IN_PROGRESS → FAILED
PENDING → IN_PROGRESS → CANCELLED
PENDING → CANCELLED (before execution starts)
```

Invalid transitions:
- Any status → PENDING
- COMPLETED → any other status
- FAILED → any other status (except for retry scenarios)

## Request/Response Messages

### PublishTaskRequest

Request to publish a task to the broker.

```protobuf
message PublishTaskRequest {
  TaskMessage task = 1;
}
```

### PublishTaskResultRequest

Request to publish a task result.

```protobuf
message PublishTaskResultRequest {
  TaskResult result = 1;
}
```

### PublishTaskProgressRequest

Request to publish task progress.

```protobuf
message PublishTaskProgressRequest {
  TaskProgress progress = 1;
}
```

### SubscribeToTasksRequest

Request to subscribe to tasks for a specific agent.

```protobuf
message SubscribeToTasksRequest {
  string agent_id = 1;                   // Agent ID to receive tasks for
  repeated string task_types = 2;        // Optional: filter by task types
}
```

#### Usage Examples

```go
// Subscribe to all tasks for this agent
req := &pb.SubscribeToTasksRequest{
    AgentId: "data_processor_01",
}

// Subscribe only to specific task types
req := &pb.SubscribeToTasksRequest{
    AgentId:   "image_processor",
    TaskTypes: []string{"image.generation", "image.enhancement"},
}
```

### SubscribeToTaskResultsRequest

Request to subscribe to task results.

```protobuf
message SubscribeToTaskResultsRequest {
  string requester_agent_id = 1;         // Agent ID that requested tasks
  repeated string task_ids = 2;          // Optional: filter by specific task IDs
}
```

#### Usage Examples

```go
// Subscribe to all results for tasks this agent requested
req := &pb.SubscribeToTaskResultsRequest{
    RequesterAgentId: "workflow_orchestrator",
}

// Subscribe only to specific task results
req := &pb.SubscribeToTaskResultsRequest{
    RequesterAgentId: "workflow_orchestrator",
    TaskIds: []string{"task_analysis_123", "task_report_456"},
}
```

## RPC Service Methods

### Task Publishing Methods

#### PublishTask

Publishes a task request to the broker.

```go
rpc PublishTask (PublishTaskRequest) returns (PublishResponse);
```

**Parameters:**
- `PublishTaskRequest` containing the `TaskMessage`

**Returns:**
- `PublishResponse` with success status and optional error message

**Example:**
```go
task := &pb.TaskMessage{
    TaskId:           "task_analysis_123",
    TaskType:         "data.analysis",
    RequesterAgentId: "orchestrator",
    // ... other fields
}

req := &pb.PublishTaskRequest{Task: task}
res, err := client.PublishTask(ctx, req)
```

#### PublishTaskResult

Publishes a task completion result.

```go
rpc PublishTaskResult (PublishTaskResultRequest) returns (PublishResponse);
```

#### PublishTaskProgress

Publishes a task progress update.

```go
rpc PublishTaskProgress (PublishTaskProgressRequest) returns (PublishResponse);
```

### Task Subscription Methods

#### SubscribeToTasks

Subscribes to receive tasks assigned to a specific agent.

```go
rpc SubscribeToTasks (SubscribeToTasksRequest) returns (stream TaskMessage);
```

**Returns:** Stream of `TaskMessage` objects

**Example:**
```go
req := &pb.SubscribeToTasksRequest{AgentId: "processor_01"}
stream, err := client.SubscribeToTasks(ctx, req)

for {
    task, err := stream.Recv()
    if err != nil {
        break
    }
    go processTask(task)
}
```

#### SubscribeToTaskResults

Subscribes to receive results for tasks requested by an agent.

```go
rpc SubscribeToTaskResults (SubscribeToTaskResultsRequest) returns (stream TaskResult);
```

#### SubscribeToTaskProgress

Subscribes to receive progress updates for tasks requested by an agent.

```go
rpc SubscribeToTaskProgress (SubscribeToTaskResultsRequest) returns (stream TaskProgress);
```

## Common Task Types Reference

### Data Processing Tasks

#### data.analysis
Analyzes datasets and produces statistical results.

**Parameters:**
```json
{
  "dataset_path": "/data/sales_2024.csv",
  "analysis_type": "summary_statistics|trend_analysis|correlation",
  "time_period": "daily|weekly|monthly|quarterly",
  "output_format": "json|csv|html"
}
```

**Result:**
```json
{
  "analysis_type": "summary_statistics",
  "total_records": 1500,
  "metrics": {
    "mean": 42.7,
    "median": 41.2,
    "std_dev": 8.3
  },
  "charts": ["/tmp/chart1.png", "/tmp/chart2.png"]
}
```

#### data.transformation
Transforms data between formats or structures.

**Parameters:**
```json
{
  "input_path": "/data/input.csv",
  "output_path": "/data/output.json",
  "transformation_rules": {
    "format": "csv_to_json",
    "schema_mapping": {...}
  }
}
```

### Image Processing Tasks

#### image.generation
Generates images based on text prompts or parameters.

**Parameters:**
```json
{
  "prompt": "A futuristic cityscape at sunset",
  "style": "photorealistic|artistic|cartoon",
  "resolution": "1920x1080",
  "quality": "low|medium|high"
}
```

**Result:**
```json
{
  "image_path": "/tmp/generated_image.png",
  "resolution": "1920x1080",
  "file_size": 2048576,
  "generation_time": "15.2s"
}
```

### Mathematical Tasks

#### math.calculation
Performs mathematical operations.

**Parameters:**
```json
{
  "operation": "add|subtract|multiply|divide|power",
  "operands": [42.0, 58.0],
  "precision": 2
}
```

**Result:**
```json
{
  "operation": "add",
  "operands": [42.0, 58.0],
  "result": 100.0
}
```

### Communication Tasks

#### notification.email
Sends email notifications.

**Parameters:**
```json
{
  "to": ["user@example.com"],
  "subject": "Task Completed",
  "body": "Your analysis task has completed successfully.",
  "template": "task_completion",
  "attachments": ["/tmp/report.pdf"]
}
```

## Error Handling Reference

### Common Error Codes

Task execution may fail with these common error patterns:

#### Parameter Validation Errors
```json
{
  "error_code": "INVALID_PARAMETERS",
  "error_message": "Required parameter 'dataset_path' is missing",
  "details": {
    "missing_parameters": ["dataset_path"],
    "invalid_parameters": {"timeout": "must be positive integer"}
  }
}
```

#### Resource Errors
```json
{
  "error_code": "RESOURCE_UNAVAILABLE",
  "error_message": "Cannot access dataset file",
  "details": {
    "resource_type": "file",
    "resource_path": "/data/sales_2024.csv",
    "error_details": "File not found"
  }
}
```

#### Timeout Errors
```json
{
  "error_code": "DEADLINE_EXCEEDED",
  "error_message": "Task deadline exceeded during processing",
  "details": {
    "deadline": "2024-01-15T11:00:00Z",
    "actual_completion": "2024-01-15T11:05:00Z",
    "phase": "data_analysis"
  }
}
```

### Error Handling Best Practices

1. **Provide specific error codes** for programmatic handling
2. **Include actionable error messages** for human operators
3. **Add structured details** for debugging and retry logic
4. **Log errors appropriately** based on severity
5. **Consider partial results** for partially successful operations

This reference provides the complete specification for task-related messages and operations in the Agent2Agent protocol, enabling robust distributed task coordination and execution.