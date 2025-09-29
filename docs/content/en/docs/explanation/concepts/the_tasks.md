---
title: "Understanding Tasks in Agent2Agent Communication"
weight: 60
description: >
  Tasks are the fundamental unit of work exchange in the Agent2Agent protocol. Deep dive into task semantics, lifecycle, and design patterns.
---

# Understanding Tasks in Agent2Agent Communication

Tasks are the fundamental unit of work exchange in the Agent2Agent protocol. This document provides a deep dive into task semantics, lifecycle, and design patterns.

## Task Anatomy

### Core Components

Every task in the Agent2Agent system consists of several key components that define its identity, purpose, and execution context:

#### A2A Task Identity
```protobuf
string id = 1;                         // Unique task identifier
string context_id = 2;                 // Optional conversation context
```

The **id** serves as a unique identifier that allows all participants to track the task throughout its lifecycle. It should be globally unique and meaningful for debugging purposes.

The **context_id** groups related tasks in a conversation or workflow context, enabling sophisticated multi-task coordination patterns.

Task classification in A2A is handled through the initial **Message** content rather than a separate task_type field, providing more flexibility for complex task descriptions.

#### A2A Task Status and History
```protobuf
TaskStatus status = 3;                 // Current task status
repeated Message history = 4;          // Message history for this task
repeated Artifact artifacts = 5;       // Task output artifacts
google.protobuf.Struct metadata = 6;   // Task metadata
```

In A2A, task data is contained within **Message** content using the structured **Part** format:

```protobuf
// A2A task request message
message Message {
  string message_id = 1;
  string context_id = 2;
  string task_id = 3;
  Role role = 4;                    // USER (requester) or AGENT (responder)
  repeated Part content = 5;        // Structured task content
}

message Part {
  oneof part {
    string text = 1;               // Text description
    DataPart data = 2;             // Structured data
    FilePart file = 3;             // File references
  }
}
```

```go
// Example: A2A data analysis task
taskMessage := &a2a.Message{
    MessageId: "msg_" + uuid.New().String(),
    ContextId: "analysis_workflow_123",
    TaskId:    "task_analysis_456",
    Role:      a2a.Role_USER,
    Content: []*a2a.Part{
        {
            Part: &a2a.Part_Text{
                Text: "Please perform trend analysis on Q4 sales data",
            },
        },
        {
            Part: &a2a.Part_Data{
                Data: &a2a.DataPart{
                    Data: analysisParams, // Structured parameters
                    Description: "Analysis configuration",
                },
            },
        },
    },
}
```

**Metadata** in A2A tasks provides additional context for execution, auditing, or debugging:

```go
// A2A task metadata
taskMetadata, _ := structpb.NewStruct(map[string]interface{}{
    "workflow_id":     "workflow_abc123",
    "user_id":         "user_456",
    "request_source":  "web_ui",
    "correlation_id":  "trace_789",
    "priority":        "high",
    "expected_duration": "5m",
})

task := &a2a.Task{
    Id:        "task_analysis_456",
    ContextId: "analysis_workflow_123",
    Metadata:  taskMetadata,
}
```

#### A2A Agent Coordination

In A2A, agent coordination is handled through the EDA routing metadata:

```protobuf
message AgentEventMetadata {
  string from_agent_id = 1;           // Source agent identifier
  string to_agent_id = 2;             // Target agent ID (empty = broadcast)
  string event_type = 3;              // Event classification
  repeated string subscriptions = 4;   // Topic-based routing tags
  Priority priority = 5;              // Delivery priority
}
```

This enables flexible routing patterns:
- **from_agent_id** identifies the requesting agent
- **to_agent_id** can specify a target agent or be empty for broadcast
- **subscriptions** enable topic-based routing for specialized agents
- **priority** ensures urgent tasks get precedence

#### A2A Execution Context

A2A handles execution context through the TaskStatus structure:

```protobuf
message TaskStatus {
  TaskState state = 1;                   // SUBMITTED, WORKING, COMPLETED, FAILED, CANCELLED
  Message update = 2;                    // Latest status message
  google.protobuf.Timestamp timestamp = 3; // Status timestamp
}

enum TaskState {
  TASK_STATE_SUBMITTED = 0;
  TASK_STATE_WORKING = 1;
  TASK_STATE_COMPLETED = 2;
  TASK_STATE_FAILED = 3;
  TASK_STATE_CANCELLED = 4;
}
```

This context helps agents make intelligent scheduling decisions:
- **deadline** enables time-sensitive prioritization
- **priority** provides explicit urgency ranking
- **created_at** enables age-based scheduling policies

## Task Lifecycle

### 1. A2A Task Creation and Publishing

A2A tasks begin their lifecycle when a requesting agent creates a task with an initial message:

```go
// Create A2A task with initial request message
task := &a2a.Task{
    Id:        "task_analysis_" + uuid.New().String(),
    ContextId: "workflow_orchestration_123",
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            MessageId: "msg_" + uuid.New().String(),
            TaskId:    "task_analysis_" + uuid.New().String(),
            Role:      a2a.Role_USER,
            Content: []*a2a.Part{
                {
                    Part: &a2a.Part_Text{
                        Text: "Please analyze the quarterly sales data for trends",
                    },
                },
                {
                    Part: &a2a.Part_Data{
                        Data: &a2a.DataPart{
                            Data: analysisParams,
                            Description: "Analysis configuration",
                        },
                    },
                },
            },
        },
        Timestamp: timestamppb.Now(),
    },
}

// Publish to AgentHub broker
client.PublishTaskUpdate(ctx, &pb.PublishTaskUpdateRequest{
    Task: task,
    Routing: &pb.AgentEventMetadata{
        FromAgentId: "data_orchestrator",
        ToAgentId:   "data_processor_01", // Optional: specific agent
        EventType:   "task.submitted",
        Priority:    pb.Priority_PRIORITY_HIGH,
    },
})
```

### 2. A2A Task Discovery and Acceptance

Agents subscribe to A2A task events and evaluate whether to accept them:

```go
// Agent receives A2A task event
func (a *Agent) evaluateA2ATask(event *pb.AgentEvent) bool {
    task := event.GetTask()
    if task == nil || task.Status.State != a2a.TaskState_TASK_STATE_SUBMITTED {
        return false
    }

    // Analyze task content to understand requirements
    requestMessage := task.Status.Update
    taskDescription := a.extractTaskDescription(requestMessage)

    // Check if agent can handle this task type
    if !a.canHandleTaskType(taskDescription) {
        return false
    }

    // Check capacity constraints
    if a.getCurrentLoad() > a.maxCapacity {
        return false
    }

    // Estimate duration from task content and metadata
    estimatedDuration := a.estimateA2ATaskDuration(task)
    if estimatedDuration > a.maxTaskDuration {
        return false
    }

    return true
}

func (a *Agent) extractTaskDescription(msg *a2a.Message) string {
    for _, part := range msg.Content {
        if textPart := part.GetText(); textPart != "" {
            return textPart
        }
    }
    return ""
}
```

### 3. A2A Task Execution with Progress Reporting

Accepted A2A tasks enter the execution phase with regular status updates:

```go
func (a *Agent) executeA2ATask(task *a2a.Task) {
    // Update task to WORKING state
    a.updateTaskStatus(task, a2a.TaskState_TASK_STATE_WORKING, "Task started")

    // Phase 1: Preparation
    a.updateTaskStatus(task, a2a.TaskState_TASK_STATE_WORKING, "Preparing data analysis")
    prepareResult := a.prepareA2AExecution(task)

    // Phase 2: Main processing
    a.updateTaskStatus(task, a2a.TaskState_TASK_STATE_WORKING, "Processing data - 50% complete")
    processResult := a.processA2AData(prepareResult)

    // Phase 3: Finalization
    a.updateTaskStatus(task, a2a.TaskState_TASK_STATE_WORKING, "Finalizing results - 75% complete")
    finalResult := a.finalizeA2AResults(processResult)

    // Completion with artifacts
    a.completeTaskWithArtifacts(task, finalResult)
}

func (a *Agent) updateTaskStatus(task *a2a.Task, state a2a.TaskState, message string) {
    statusUpdate := &a2a.Message{
        MessageId: "msg_" + uuid.New().String(),
        TaskId:    task.Id,
        Role:      a2a.Role_AGENT,
        Content: []*a2a.Part{
            {
                Part: &a2a.Part_Text{
                    Text: message,
                },
            },
        },
    }

    task.Status = &a2a.TaskStatus{
        State:     state,
        Update:    statusUpdate,
        Timestamp: timestamppb.Now(),
    }

    // Publish task update
    a.client.PublishTaskUpdate(context.Background(), &pb.PublishTaskUpdateRequest{
        Task: task,
        Routing: &pb.AgentEventMetadata{
            FromAgentId: a.agentId,
            EventType:   "task.status_update",
        },
    })
}
```

### 4. A2A Result Delivery

A2A task completion delivers results through structured artifacts:

```go
func (a *Agent) completeTaskWithArtifacts(task *a2a.Task, resultData interface{}) {
    // Create completion message
    completionMessage := &a2a.Message{
        MessageId: "msg_" + uuid.New().String(),
        TaskId:    task.Id,
        Role:      a2a.Role_AGENT,
        Content: []*a2a.Part{
            {
                Part: &a2a.Part_Text{
                    Text: "Analysis completed successfully",
                },
            },
        },
    }

    // Create result artifact
    resultArtifact := &a2a.Artifact{
        ArtifactId:  "artifact_" + uuid.New().String(),
        Name:        "Analysis Results",
        Description: "Quarterly sales trend analysis",
        Parts: []*a2a.Part{
            {
                Part: &a2a.Part_Data{
                    Data: &a2a.DataPart{
                        Data:        resultData.(structpb.Struct),
                        Description: "Analysis results and metrics",
                    },
                },
            },
        },
    }

    // Update task to completed
    task.Status = &a2a.TaskStatus{
        State:     a2a.TaskState_TASK_STATE_COMPLETED,
        Update:    completionMessage,
        Timestamp: timestamppb.Now(),
    }
    task.Artifacts = append(task.Artifacts, resultArtifact)

    // Publish final task update
    a.client.PublishTaskUpdate(context.Background(), &pb.PublishTaskUpdateRequest{
        Task: task,
        Routing: &pb.AgentEventMetadata{
            FromAgentId: a.agentId,
            EventType:   "task.completed",
        },
    })

    // Publish artifact separately
    a.client.PublishTaskArtifact(context.Background(), &pb.PublishTaskArtifactRequest{
        TaskId:   task.Id,
        Artifact: resultArtifact,
        Routing: &pb.AgentEventMetadata{
            FromAgentId: a.agentId,
            EventType:   "task.artifact",
        },
    })
}
```

## A2A Task Design Patterns

### 1. Simple A2A Request-Response

The most basic pattern where one agent requests work from another using A2A messages:

```
Agent A ──[A2A Task]──> AgentHub ──[TaskEvent]──> Agent B
Agent A <─[Artifact]─── AgentHub <─[TaskUpdate]── Agent B
```

**A2A Implementation:**
```go
// Agent A creates task
task := &a2a.Task{
    Id: "simple_task_123",
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            Role: a2a.Role_USER,
            Content: []*a2a.Part{{Part: &a2a.Part_Text{Text: "Convert CSV to JSON"}}},
        },
    },
}

// Agent B responds with artifact
artifact := &a2a.Artifact{
    Name: "Converted Data",
    Parts: []*a2a.Part{{Part: &a2a.Part_File{File: &a2a.FilePart{FileId: "converted.json"}}}},
}
```

**Use cases:**
- File format conversion
- Simple calculations
- Data validation
- Content generation

### 2. A2A Broadcast Processing

One agent broadcasts a task to multiple potential processors using A2A context-aware routing:

```
Agent A ──[A2A Task]──> AgentHub ──[TaskEvent]──> Agent B₁
                                ├─[TaskEvent]──> Agent B₂
                                └─[TaskEvent]──> Agent B₃
```

**A2A Implementation:**
```go
// Broadcast task with shared context
task := &a2a.Task{
    Id:        "broadcast_task_456",
    ContextId: "parallel_processing_context",
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            Role: a2a.Role_USER,
            Content: []*a2a.Part{
                {Part: &a2a.Part_Text{Text: "Process data chunk"}},
                {Part: &a2a.Part_Data{Data: &a2a.DataPart{Data: chunkData}}},
            },
        },
    },
}

// Publish without specific target (broadcast)
client.PublishTaskUpdate(ctx, &pb.PublishTaskUpdateRequest{
    Task: task,
    Routing: &pb.AgentEventMetadata{
        FromAgentId: "orchestrator",
        // No ToAgentId = broadcast
        EventType: "task.broadcast",
    },
})
```

**Use cases:**
- Distributed computation
- Load testing
- Content distribution
- Parallel processing

### 3. A2A Pipeline Processing

Tasks flow through a series of specialized agents using shared A2A context:

```
Agent A ──[A2A Task₁]──> Agent B ──[A2A Task₂]──> Agent C ──[A2A Task₃]──> Agent D
       <──[Final Artifact]───────────────────────────────────────────────────┘
```

**A2A Implementation:**
```go
// Shared context for pipeline
pipelineContext := "data_pipeline_" + uuid.New().String()

// Stage 1: Data extraction
task1 := &a2a.Task{
    Id:        "extract_" + uuid.New().String(),
    ContextId: pipelineContext,
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            Role: a2a.Role_USER,
            Content: []*a2a.Part{{Part: &a2a.Part_Text{Text: "Extract data from source"}}},
        },
    },
}

// Stage 2: Data transformation (triggered by Stage 1 completion)
task2 := &a2a.Task{
    Id:        "transform_" + uuid.New().String(),
    ContextId: pipelineContext, // Same context
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            Role: a2a.Role_USER,
            Content: []*a2a.Part{{Part: &a2a.Part_Text{Text: "Transform extracted data"}}},
        },
    },
}

// Context linking enables pipeline coordination
```

**Use cases:**
- Data processing pipelines
- Image processing workflows
- Document processing chains
- ETL operations

### 4. A2A Hierarchical Decomposition

Complex tasks are broken down into subtasks using A2A context hierarchy:

```
Agent A ──[A2A ComplexTask]──> Coordinator
                                  ├──[A2A SubTask₁]──> Specialist₁
                                  ├──[A2A SubTask₂]──> Specialist₂
                                  └──[A2A SubTask₃]──> Specialist₃
```

**A2A Implementation:**
```go
// Parent task
parentTask := &a2a.Task{
    Id:        "complex_analysis_789",
    ContextId: "business_workflow_123",
    Status: &a2a.TaskStatus{
        State: a2a.TaskState_TASK_STATE_SUBMITTED,
        Update: &a2a.Message{
            Role: a2a.Role_USER,
            Content: []*a2a.Part{{Part: &a2a.Part_Text{Text: "Perform comprehensive business analysis"}}},
        },
    },
}

// Coordinator creates subtasks with hierarchical context
subtask1 := &a2a.Task{
    Id:        "financial_analysis_790",
    ContextId: "business_workflow_123", // Same parent context
    Metadata: map[string]interface{}{
        "parent_task_id": "complex_analysis_789",
        "subtask_type":   "financial",
    },
}

subtask2 := &a2a.Task{
    Id:        "market_analysis_791",
    ContextId: "business_workflow_123", // Same parent context
    Metadata: map[string]interface{}{
        "parent_task_id": "complex_analysis_789",
        "subtask_type":   "market",
    },
}

// Context enables coordination and result aggregation
```

**Use cases:**
- Complex business workflows
- Multi-step analysis
- Orchestrated services
- Batch job coordination

### 5. Competitive Processing

Multiple agents compete to handle the same task (first-come-first-served):

```
Agent A ──[Task]──> Broker ──[Task]──> Agent B₁ (accepts)
                           ├─[Task]──> Agent B₂ (rejects)
                           └─[Task]──> Agent B₃ (rejects)
```

**Use cases:**
- Resource-constrained environments
- Load balancing
- Fault tolerance
- Performance optimization

## A2A Task Content and Semantics

### A2A Message-Based Classification

In A2A, task classification is handled through message content rather than rigid type fields, providing more flexibility:

#### Content-Based Classification
```go
// Data processing task
message := &a2a.Message{
    Content: []*a2a.Part{
        {Part: &a2a.Part_Text{Text: "Analyze quarterly sales data for trends"}},
        {Part: &a2a.Part_Data{Data: &a2a.DataPart{Description: "Analysis parameters"}}},
    },
}

// Image processing task
message := &a2a.Message{
    Content: []*a2a.Part{
        {Part: &a2a.Part_Text{Text: "Generate product image with specifications"}},
        {Part: &a2a.Part_Data{Data: &a2a.DataPart{Description: "Image requirements"}}},
    },
}

// Notification task
message := &a2a.Message{
    Content: []*a2a.Part{
        {Part: &a2a.Part_Text{Text: "Send completion notification to user"}},
        {Part: &a2a.Part_Data{Data: &a2a.DataPart{Description: "Notification details"}}},
    },
}
```

#### Operation-Based Classification
```
create.*        - Creation operations
update.*        - Modification operations
delete.*        - Removal operations
analyze.*       - Analysis operations
transform.*     - Transformation operations
```

#### Complexity-Based Classification
```
simple.*        - Quick, low-resource tasks
standard.*      - Normal processing tasks
complex.*       - Resource-intensive tasks
background.*    - Long-running batch tasks
```

### A2A Content Design Guidelines

**Be Explicit**: Include all information needed for execution in structured Parts
```go
// Good: Explicit A2A content
content := []*a2a.Part{
    {
        Part: &a2a.Part_Text{
            Text: "Convert CSV file to JSON format with specific options",
        },
    },
    {
        Part: &a2a.Part_Data{
            Data: &a2a.DataPart{
                Data: structpb.NewStruct(map[string]interface{}{
                    "source_format":   "csv",
                    "target_format":   "json",
                    "include_headers": true,
                    "delimiter":       ",",
                    "encoding":        "utf-8",
                }),
                Description: "Conversion parameters",
            },
        },
    },
    {
        Part: &a2a.Part_File{
            File: &a2a.FilePart{
                FileId:   "source_data.csv",
                Filename: "data.csv",
                MimeType: "text/csv",
            },
        },
    },
}

// Poor: Ambiguous A2A content
content := []*a2a.Part{
    {
        Part: &a2a.Part_Text{
            Text: "Convert file", // Too vague
        },
    },
}
```

**Use Standard Data Types**: Leverage common formats for interoperability
```json
// Good: Standard formats
{
  "timestamp": "2024-01-15T10:30:00Z",      // ISO 8601
  "amount": "123.45",                        // String for precision
  "coordinates": {"lat": 40.7128, "lng": -74.0060}
}
```

**Include Validation Information**: Help agents validate inputs
```json
{
  "email": "user@example.com",
  "email_format": "rfc5322",
  "max_length": 254,
  "required": true
}
```

## A2A Error Handling and Edge Cases

### A2A Task Rejection

Agents should provide meaningful rejection reasons using A2A message format:

```go
func (a *Agent) rejectA2ATask(task *a2a.Task, reason string) {
    // Create rejection message
    rejectionMessage := &a2a.Message{
        MessageId: "msg_" + uuid.New().String(),
        TaskId:    task.Id,
        Role:      a2a.Role_AGENT,
        Content: []*a2a.Part{
            {
                Part: &a2a.Part_Text{
                    Text: "Task rejected: " + reason,
                },
            },
            {
                Part: &a2a.Part_Data{
                    Data: &a2a.DataPart{
                        Data: structpb.NewStruct(map[string]interface{}{
                            "rejection_reason": reason,
                            "agent_id":         a.agentId,
                            "timestamp":        time.Now().Unix(),
                        }),
                        Description: "Rejection details",
                    },
                },
            },
        },
    }

    // Update task status to failed
    task.Status = &a2a.TaskStatus{
        State:     a2a.TaskState_TASK_STATE_FAILED,
        Update:    rejectionMessage,
        Timestamp: timestamppb.Now(),
    }

    a.publishTaskUpdate(task)
}
```

Common rejection reasons:
- `UNSUPPORTED_TASK_TYPE`: Agent doesn't handle this task type
- `CAPACITY_EXCEEDED`: Agent is at maximum capacity
- `DEADLINE_IMPOSSIBLE`: Cannot complete within deadline
- `INVALID_PARAMETERS`: Task parameters are malformed
- `RESOURCE_UNAVAILABLE`: Required external resources unavailable

### Timeout Handling

Both requesters and processors should handle timeouts gracefully:

```go
// Requester timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

select {
case result := <-resultChannel:
    // Process result
case <-ctx.Done():
    // Handle timeout - possibly retry or fail
}

// Processor timeout
func (a *Agent) executeWithTimeout(task *pb.TaskMessage) {
    deadline := task.GetDeadline().AsTime()
    ctx, cancel := context.WithDeadline(context.Background(), deadline)
    defer cancel()

    select {
    case result := <-a.processTask(ctx, task):
        a.publishResult(task, result, pb.TaskStatus_TASK_STATUS_COMPLETED)
    case <-ctx.Done():
        a.publishResult(task, nil, pb.TaskStatus_TASK_STATUS_FAILED, "Deadline exceeded")
    }
}
```

### Partial Results

For long-running tasks, consider supporting partial results:

```go
type PartialResult struct {
    TaskId          string
    CompletedPortion float64    // 0.0 to 1.0
    IntermediateData interface{}
    CanResume       bool
    ResumeToken     string
}
```

## Best Practices

### Task Design
1. **Make task types granular** but not too fine-grained
2. **Design for idempotency** when possible
3. **Include retry information** in metadata
4. **Use consistent parameter naming** across similar task types
5. **Version your task schemas** to enable evolution

### Performance Considerations
1. **Batch related tasks** when appropriate
2. **Use appropriate priority levels** to avoid starvation
3. **Set realistic deadlines** based on historical performance
4. **Include resource hints** to help with scheduling
5. **Monitor task completion rates** to identify bottlenecks

### Security Considerations
1. **Validate all task parameters** before processing
2. **Sanitize user-provided data** in task parameters
3. **Include authorization context** in metadata
4. **Log task execution** for audit trails
5. **Encrypt sensitive parameters** when necessary

A2A tasks form the foundation of Agent2Agent communication, enabling sophisticated distributed processing patterns through structured messages, artifacts, and context-aware coordination. The A2A protocol's flexible message format and EDA integration provide robust, scalable agent networks with clear semantics and strong observability. Proper A2A task design leverages the protocol's strengths for building maintainable, interoperable agent systems.