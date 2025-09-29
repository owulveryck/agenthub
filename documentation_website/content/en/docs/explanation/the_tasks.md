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

#### Task Identity
```protobuf
string task_id = 1;                    // Unique identifier
string task_type = 2;                  // Semantic classification
```

The **task_id** serves as a unique identifier that allows all participants to track the task throughout its lifecycle. It should be globally unique and meaningful for debugging purposes.

The **task_type** provides semantic classification that allows agents to understand what kind of work is being requested. Examples include:
- `data_analysis` - Request for data processing and analysis
- `image_generation` - Request for creating or manipulating images
- `file_conversion` - Request for converting files between formats
- `notification_delivery` - Request for sending notifications to users

#### Task Payload
```protobuf
google.protobuf.Struct parameters = 3; // Task-specific data
google.protobuf.Struct metadata = 8;   // Additional context
```

**Parameters** contain the specific data required to execute the task. The structure is flexible and task-type dependent:

```json
// Example: data_analysis task parameters
{
  "dataset_path": "/data/sales_2024.csv",
  "analysis_type": "trend_analysis",
  "time_period": "quarterly",
  "output_format": "json"
}

// Example: image_generation task parameters
{
  "prompt": "A futuristic cityscape at sunset",
  "style": "photorealistic",
  "resolution": "1920x1080",
  "quality": "high"
}
```

**Metadata** provides additional context that may be useful for execution, auditing, or debugging but isn't directly required for task completion:

```json
{
  "workflow_id": "workflow_abc123",
  "user_id": "user_456",
  "request_source": "web_ui",
  "correlation_id": "trace_789"
}
```

#### Agent Coordination
```protobuf
string requester_agent_id = 4;        // Who initiated the task
string responder_agent_id = 5;        // Who should execute (optional)
```

These fields establish the communication relationship:
- **requester_agent_id** identifies which agent requested the task, enabling result delivery
- **responder_agent_id** can specify a particular agent for execution, or be omitted for broadcast/routing

#### Execution Context
```protobuf
google.protobuf.Timestamp deadline = 6;   // When task must complete
Priority priority = 7;                    // Urgency level
google.protobuf.Timestamp created_at = 9; // Creation timestamp
```

This context helps agents make intelligent scheduling decisions:
- **deadline** enables time-sensitive prioritization
- **priority** provides explicit urgency ranking
- **created_at** enables age-based scheduling policies

## Task Lifecycle

### 1. Task Creation and Publishing

Tasks begin their lifecycle when a requesting agent identifies work that needs to be done:

```go
// Agent identifies need for data analysis
task := &pb.TaskMessage{
    TaskId:           generateUniqueID(),
    TaskType:         "data_analysis",
    Parameters:       analysisParams,
    RequesterAgentId: "data_orchestrator",
    ResponderAgentId: "data_processor_01", // Optional: specific agent
    Priority:         pb.Priority_PRIORITY_HIGH,
    Deadline:         timestamppb.New(time.Now().Add(30 * time.Minute)),
    CreatedAt:        timestamppb.Now(),
}

// Publish to the broker
client.PublishTask(ctx, &pb.PublishTaskRequest{Task: task})
```

### 2. Task Discovery and Acceptance

Agents subscribe to tasks and evaluate whether to accept them:

```go
// Agent receives task notification
func (a *Agent) evaluateTask(task *pb.TaskMessage) bool {
    // Check if agent can handle this task type
    if !a.canHandle(task.GetTaskType()) {
        return false
    }

    // Check capacity constraints
    if a.getCurrentLoad() > a.maxCapacity {
        return false
    }

    // Check deadline feasibility
    estimatedDuration := a.estimateTaskDuration(task)
    if time.Now().Add(estimatedDuration).After(task.GetDeadline().AsTime()) {
        return false
    }

    return true
}
```

### 3. Task Execution with Progress Reporting

Accepted tasks enter the execution phase with regular progress updates:

```go
func (a *Agent) executeTask(task *pb.TaskMessage) {
    // Initial progress
    a.reportProgress(task, 0, "Task started")

    // Phase 1: Preparation
    a.reportProgress(task, 25, "Preparing data")
    prepareResult := a.prepareExecution(task)

    // Phase 2: Main processing
    a.reportProgress(task, 50, "Processing data")
    processResult := a.processData(prepareResult)

    // Phase 3: Finalization
    a.reportProgress(task, 75, "Finalizing results")
    finalResult := a.finalizeResults(processResult)

    // Completion
    a.reportProgress(task, 100, "Task completed")
    a.publishResult(task, finalResult, pb.TaskStatus_TASK_STATUS_COMPLETED)
}
```

### 4. Result Delivery

Task completion triggers result publication back to the requesting agent:

```go
result := &pb.TaskResult{
    TaskId:            task.GetTaskId(),
    Status:            pb.TaskStatus_TASK_STATUS_COMPLETED,
    Result:            resultData,
    ExecutorAgentId:   a.agentId,
    CompletedAt:       timestamppb.Now(),
    ExecutionMetadata: executionContext,
}

client.PublishTaskResult(ctx, &pb.PublishTaskResultRequest{Result: result})
```

## Task Design Patterns

### 1. Simple Request-Response

The most basic pattern where one agent requests work from another:

```
Agent A ──[Task]──> Broker ──[Task]──> Agent B
Agent A <─[Result]─ Broker <─[Result]─ Agent B
```

**Use cases:**
- File format conversion
- Simple calculations
- Data validation
- Content generation

### 2. Broadcast Processing

One agent broadcasts a task to multiple potential processors:

```
Agent A ──[Task]──> Broker ──[Task]──> Agent B₁
                           ├─[Task]──> Agent B₂
                           └─[Task]──> Agent B₃
```

**Use cases:**
- Distributed computation
- Load testing
- Content distribution
- Parallel processing

### 3. Pipeline Processing

Tasks flow through a series of specialized agents:

```
Agent A ──[Task₁]──> Agent B ──[Task₂]──> Agent C ──[Task₃]──> Agent D
       <──[Result]─────────────────────────────────────────────┘
```

**Use cases:**
- Data processing pipelines
- Image processing workflows
- Document processing chains
- ETL operations

### 4. Hierarchical Decomposition

Complex tasks are broken down into subtasks by coordinator agents:

```
Agent A ──[ComplexTask]──> Coordinator
                              ├──[SubTask₁]──> Specialist₁
                              ├──[SubTask₂]──> Specialist₂
                              └──[SubTask₃]──> Specialist₃
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

## Task Types and Semantics

### Classification Strategies

Task types should follow consistent naming conventions that enable agents to understand capabilities:

#### Domain-Based Classification
```
data.*          - Data processing tasks
  data.analysis
  data.transformation
  data.validation

image.*         - Image processing tasks
  image.generation
  image.enhancement
  image.conversion

notification.*  - Communication tasks
  notification.email
  notification.sms
  notification.push
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

### Parameter Design Guidelines

**Be Explicit**: Include all information needed for execution
```json
// Good: Explicit parameters
{
  "source_format": "csv",
  "target_format": "json",
  "include_headers": true,
  "delimiter": ",",
  "encoding": "utf-8"
}

// Poor: Ambiguous parameters
{
  "file": "data.csv",
  "convert": "json"
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

## Error Handling and Edge Cases

### Task Rejection

Agents should provide meaningful rejection reasons:

```go
func (a *Agent) rejectTask(task *pb.TaskMessage, reason string) {
    result := &pb.TaskResult{
        TaskId:       task.GetTaskId(),
        Status:       pb.TaskStatus_TASK_STATUS_FAILED,
        ErrorMessage: reason,
        ExecutorAgentId: a.agentId,
        CompletedAt:  timestamppb.Now(),
    }

    a.publishResult(result)
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

Tasks form the foundation of Agent2Agent communication, enabling sophisticated distributed processing patterns while maintaining clear semantics and strong observability. Proper task design is crucial for building robust, scalable agent networks.