---
title: "How to Work with A2A Task Lifecycle"
weight: 40
description: "Learn how to manage Agent2Agent protocol task states, handle lifecycle transitions, and coordinate complex task workflows."
---

# How to Work with A2A Task Lifecycle

This guide shows you how to manage the complete lifecycle of Agent2Agent (A2A) protocol tasks, from creation through completion. Understanding task states and transitions is essential for building reliable agent workflows.

## Understanding A2A Task States

A2A tasks progress through the following states:

- **`TASK_STATE_SUBMITTED`**: Task created and submitted for processing
- **`TASK_STATE_WORKING`**: Task accepted and currently being processed
- **`TASK_STATE_COMPLETED`**: Task finished successfully with results
- **`TASK_STATE_FAILED`**: Task failed with error information
- **`TASK_STATE_CANCELLED`**: Task cancelled before completion

Each state transition is recorded with a timestamp and status message.

## Creating A2A Tasks

### Basic Task Creation

Create a new task with initial state:

```go
package main

import (
    "fmt"
    "github.com/google/uuid"
    pb "github.com/owulveryck/agenthub/events/a2a"
    "google.golang.org/protobuf/types/known/timestamppb"
    "google.golang.org/protobuf/types/known/structpb"
)

func createA2ATask(contextID, taskType string, content []*pb.Part) *pb.Task {
    taskID := fmt.Sprintf("task_%s_%s", taskType, uuid.New().String())
    messageID := fmt.Sprintf("msg_%s", uuid.New().String())

    return &pb.Task{
        Id:        taskID,
        ContextId: contextID,
        Status: &pb.TaskStatus{
            State: pb.TaskState_TASK_STATE_SUBMITTED,
            Update: &pb.Message{
                MessageId: messageID,
                ContextId: contextID,
                TaskId:    taskID,
                Role:      pb.Role_USER,
                Content:   content,
                Metadata: &structpb.Struct{
                    Fields: map[string]*structpb.Value{
                        "task_type":      structpb.NewStringValue(taskType),
                        "submitted_by":   structpb.NewStringValue("user_agent"),
                        "priority":       structpb.NewStringValue("medium"),
                    },
                },
            },
            Timestamp: timestamppb.Now(),
        },
        History:   []*pb.Message{},
        Artifacts: []*pb.Artifact{},
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "task_type":    structpb.NewStringValue(taskType),
                "created_at":   structpb.NewStringValue(time.Now().Format(time.RFC3339)),
                "expected_duration": structpb.NewStringValue("5m"),
            },
        },
    }
}
```

### Task with Complex Requirements

Create tasks with detailed specifications:

```go
func createComplexAnalysisTask(contextID string) *pb.Task {
    // Task configuration
    taskConfig, _ := structpb.NewStruct(map[string]interface{}{
        "algorithm":         "advanced_ml_analysis",
        "confidence_level":  0.95,
        "max_processing_time": "30m",
        "output_formats":    []string{"json", "csv", "visualization"},
        "quality_threshold": 0.9,
    })

    // Input data specification
    inputSpec, _ := structpb.NewStruct(map[string]interface{}{
        "dataset_id":       "customer_data_2024",
        "required_fields":  []string{"customer_id", "transaction_amount", "timestamp"},
        "date_range":       map[string]string{"start": "2024-01-01", "end": "2024-12-31"},
        "preprocessing":    true,
    })

    content := []*pb.Part{
        {
            Part: &pb.Part_Text{
                Text: "Perform comprehensive customer behavior analysis on the specified dataset with advanced ML algorithms.",
            },
        },
        {
            Part: &pb.Part_Data{
                Data: &pb.DataPart{
                    Data:        taskConfig,
                    Description: "Analysis configuration parameters",
                },
            },
        },
        {
            Part: &pb.Part_Data{
                Data: &pb.DataPart{
                    Data:        inputSpec,
                    Description: "Input dataset specification",
                },
            },
        },
    }

    task := createA2ATask(contextID, "customer_analysis", content)

    // Add complex task metadata
    task.Metadata = &structpb.Struct{
        Fields: map[string]*structpb.Value{
            "task_type":           structpb.NewStringValue("customer_analysis"),
            "complexity":          structpb.NewStringValue("high"),
            "estimated_duration":  structpb.NewStringValue("30m"),
            "required_resources":  structpb.NewListValue(&structpb.ListValue{
                Values: []*structpb.Value{
                    structpb.NewStringValue("gpu_compute"),
                    structpb.NewStringValue("large_memory"),
                },
            }),
            "deliverables":        structpb.NewListValue(&structpb.ListValue{
                Values: []*structpb.Value{
                    structpb.NewStringValue("analysis_report"),
                    structpb.NewStringValue("customer_segments"),
                    structpb.NewStringValue("predictions"),
                },
            }),
        },
    }

    return task
}
```

## Task State Transitions

### Accepting a Task (SUBMITTED → WORKING)

When an agent accepts a task:

```go
func acceptTask(task *pb.Task, agentID string) *pb.Task {
    // Create acceptance message
    acceptanceMessage := &pb.Message{
        MessageId: fmt.Sprintf("msg_accept_%s", uuid.New().String()),
        ContextId: task.GetContextId(),
        TaskId:    task.GetId(),
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: fmt.Sprintf("Task accepted by agent %s. Beginning processing.", agentID),
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "accepting_agent": structpb.NewStringValue(agentID),
                "estimated_completion": structpb.NewStringValue(
                    time.Now().Add(15*time.Minute).Format(time.RFC3339),
                ),
            },
        },
    }

    // Update task status
    task.Status = &pb.TaskStatus{
        State:     pb.TaskState_TASK_STATE_WORKING,
        Update:    acceptanceMessage,
        Timestamp: timestamppb.Now(),
    }

    // Add to history
    task.History = append(task.History, acceptanceMessage)

    return task
}
```

### Progress Updates (WORKING → WORKING)

Send progress updates during processing:

```go
func sendProgressUpdate(task *pb.Task, progressPercentage int, currentPhase, details string) *pb.Task {
    // Create progress data
    progressData, _ := structpb.NewStruct(map[string]interface{}{
        "progress_percentage": progressPercentage,
        "current_phase":       currentPhase,
        "details":            details,
        "estimated_remaining": calculateRemainingTime(progressPercentage),
        "memory_usage_mb":     getCurrentMemoryUsage(),
        "cpu_usage_percent":   getCurrentCPUUsage(),
    })

    progressMessage := &pb.Message{
        MessageId: fmt.Sprintf("msg_progress_%s_%d", uuid.New().String(), progressPercentage),
        ContextId: task.GetContextId(),
        TaskId:    task.GetId(),
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: fmt.Sprintf("Progress update: %d%% complete. Current phase: %s",
                        progressPercentage, currentPhase),
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        progressData,
                        Description: "Detailed progress information",
                    },
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "update_type":         structpb.NewStringValue("progress"),
                "progress_percentage": structpb.NewNumberValue(float64(progressPercentage)),
                "phase":              structpb.NewStringValue(currentPhase),
            },
        },
    }

    // Update task status (still WORKING, but with new message)
    task.Status = &pb.TaskStatus{
        State:     pb.TaskState_TASK_STATE_WORKING,
        Update:    progressMessage,
        Timestamp: timestamppb.Now(),
    }

    // Add to history
    task.History = append(task.History, progressMessage)

    return task
}

func calculateRemainingTime(progressPercentage int) string {
    if progressPercentage <= 0 {
        return "unknown"
    }
    // Simplified estimation logic
    remainingMinutes := (100 - progressPercentage) * 15 / 100
    return fmt.Sprintf("%dm", remainingMinutes)
}
```

### Completing a Task (WORKING → COMPLETED)

Complete a task with results:

```go
func completeTask(task *pb.Task, results string, artifacts []*pb.Artifact) *pb.Task {
    // Create completion message
    completionMessage := &pb.Message{
        MessageId: fmt.Sprintf("msg_complete_%s", uuid.New().String()),
        ContextId: task.GetContextId(),
        TaskId:    task.GetId(),
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: fmt.Sprintf("Task completed successfully. %s", results),
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "completion_status": structpb.NewStringValue("success"),
                "processing_time":   structpb.NewStringValue(
                    time.Since(getTaskStartTime(task)).String(),
                ),
                "artifact_count":    structpb.NewNumberValue(float64(len(artifacts))),
            },
        },
    }

    // Update task status
    task.Status = &pb.TaskStatus{
        State:     pb.TaskState_TASK_STATE_COMPLETED,
        Update:    completionMessage,
        Timestamp: timestamppb.Now(),
    }

    // Add completion message to history
    task.History = append(task.History, completionMessage)

    // Add artifacts
    task.Artifacts = append(task.Artifacts, artifacts...)

    return task
}
```

### Handling Task Failures (WORKING → FAILED)

Handle task failures with detailed error information:

```go
func failTask(task *pb.Task, errorMessage, errorCode string, errorDetails map[string]interface{}) *pb.Task {
    // Create error data
    errorData, _ := structpb.NewStruct(map[string]interface{}{
        "error_code":    errorCode,
        "error_message": errorMessage,
        "error_details": errorDetails,
        "failure_phase": getCurrentProcessingPhase(task),
        "retry_possible": determineRetryPossibility(errorCode),
        "diagnostic_info": map[string]interface{}{
            "memory_at_failure": getCurrentMemoryUsage(),
            "cpu_at_failure":   getCurrentCPUUsage(),
            "logs_reference":   getLogReference(),
        },
    })

    failureMessage := &pb.Message{
        MessageId: fmt.Sprintf("msg_failure_%s", uuid.New().String()),
        ContextId: task.GetContextId(),
        TaskId:    task.GetId(),
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: fmt.Sprintf("Task failed: %s (Code: %s)", errorMessage, errorCode),
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        errorData,
                        Description: "Detailed error information and diagnostics",
                    },
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "failure_type":  structpb.NewStringValue("processing_error"),
                "error_code":    structpb.NewStringValue(errorCode),
                "retry_possible": structpb.NewBoolValue(determineRetryPossibility(errorCode)),
            },
        },
    }

    // Update task status
    task.Status = &pb.TaskStatus{
        State:     pb.TaskState_TASK_STATE_FAILED,
        Update:    failureMessage,
        Timestamp: timestamppb.Now(),
    }

    // Add failure message to history
    task.History = append(task.History, failureMessage)

    return task
}

func determineRetryPossibility(errorCode string) bool {
    // Determine if the error is retryable
    retryableErrors := []string{
        "TEMPORARY_RESOURCE_UNAVAILABLE",
        "NETWORK_TIMEOUT",
        "RATE_LIMIT_EXCEEDED",
    }

    for _, retryable := range retryableErrors {
        if errorCode == retryable {
            return true
        }
    }
    return false
}
```

### Cancelling Tasks (ANY → CANCELLED)

Handle task cancellation:

```go
func cancelTask(task *pb.Task, reason, cancelledBy string) *pb.Task {
    cancellationMessage := &pb.Message{
        MessageId: fmt.Sprintf("msg_cancel_%s", uuid.New().String()),
        ContextId: task.GetContextId(),
        TaskId:    task.GetId(),
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: fmt.Sprintf("Task cancelled: %s", reason),
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "cancellation_reason": structpb.NewStringValue(reason),
                "cancelled_by":        structpb.NewStringValue(cancelledBy),
                "previous_state":      structpb.NewStringValue(task.GetStatus().GetState().String()),
            },
        },
    }

    // Update task status
    task.Status = &pb.TaskStatus{
        State:     pb.TaskState_TASK_STATE_CANCELLED,
        Update:    cancellationMessage,
        Timestamp: timestamppb.Now(),
    }

    // Add cancellation message to history
    task.History = append(task.History, cancellationMessage)

    return task
}
```

## Publishing Task Updates

### Using AgentHub Client

Publish task updates through the AgentHub broker:

```go
import (
    "context"
    eventbus "github.com/owulveryck/agenthub/events/eventbus"
)

func publishTaskUpdate(ctx context.Context, client eventbus.AgentHubClient, task *pb.Task, fromAgent, toAgent string) error {
    _, err := client.PublishTaskUpdate(ctx, &eventbus.PublishTaskUpdateRequest{
        Task: task,
        Routing: &eventbus.AgentEventMetadata{
            FromAgentId: fromAgent,
            ToAgentId:   toAgent,
            EventType:   fmt.Sprintf("task.%s", task.GetStatus().GetState().String()),
            Priority:    getPriorityFromTaskState(task.GetStatus().GetState()),
        },
    })

    return err
}

func getPriorityFromTaskState(state pb.TaskState) eventbus.Priority {
    switch state {
    case pb.TaskState_TASK_STATE_FAILED:
        return eventbus.Priority_PRIORITY_HIGH
    case pb.TaskState_TASK_STATE_COMPLETED:
        return eventbus.Priority_PRIORITY_MEDIUM
    case pb.TaskState_TASK_STATE_WORKING:
        return eventbus.Priority_PRIORITY_LOW
    default:
        return eventbus.Priority_PRIORITY_MEDIUM
    }
}
```

### Using A2A Abstractions

Use simplified A2A task management:

```go
import (
    "github.com/owulveryck/agenthub/internal/agenthub"
)

func manageTaskWithA2A(ctx context.Context, subscriber *agenthub.A2ATaskSubscriber, task *pb.Task) error {
    // Process the task
    artifact, status, errorMsg := processTaskContent(ctx, task)

    switch status {
    case pb.TaskState_TASK_STATE_COMPLETED:
        return subscriber.CompleteA2ATaskWithArtifact(ctx, task, artifact)
    case pb.TaskState_TASK_STATE_FAILED:
        return subscriber.FailA2ATask(ctx, task, errorMsg)
    default:
        return subscriber.UpdateA2ATaskProgress(ctx, task, 50, "Processing data", "Halfway complete")
    }
}
```

## Task Monitoring and Querying

### Get Task Status

Query task status and history:

```go
func getTaskStatus(ctx context.Context, client eventbus.AgentHubClient, taskID string) (*pb.Task, error) {
    task, err := client.GetTask(ctx, &eventbus.GetTaskRequest{
        TaskId:        taskID,
        HistoryLength: 10, // Get last 10 messages
    })
    if err != nil {
        return nil, err
    }

    // Log current status
    log.Printf("Task %s status: %s", taskID, task.GetStatus().GetState().String())
    log.Printf("Last update: %s", task.GetStatus().GetUpdate().GetContent()[0].GetText())
    log.Printf("History length: %d messages", len(task.GetHistory()))
    log.Printf("Artifacts: %d", len(task.GetArtifacts()))

    return task, nil
}
```

### List Tasks by Context

Get all tasks for a conversation context:

```go
func getTasksInContext(ctx context.Context, client eventbus.AgentHubClient, contextID string) ([]*pb.Task, error) {
    response, err := client.ListTasks(ctx, &eventbus.ListTasksRequest{
        ContextId: contextID,
        States:    []pb.TaskState{}, // All states
        Limit:     100,
    })
    if err != nil {
        return nil, err
    }

    tasks := response.GetTasks()
    log.Printf("Found %d tasks in context %s", len(tasks), contextID)

    // Analyze task distribution
    stateCount := make(map[pb.TaskState]int)
    for _, task := range tasks {
        stateCount[task.GetStatus().GetState()]++
    }

    for state, count := range stateCount {
        log.Printf("  %s: %d tasks", state.String(), count)
    }

    return tasks, nil
}
```

## Workflow Coordination

### Sequential Task Workflow

Create dependent tasks that execute in sequence:

```go
type TaskWorkflow struct {
    ContextID string
    Tasks     []*pb.Task
    Current   int
}

func (tw *TaskWorkflow) ExecuteNext(ctx context.Context, client eventbus.AgentHubClient) error {
    if tw.Current >= len(tw.Tasks) {
        return fmt.Errorf("workflow completed")
    }

    currentTask := tw.Tasks[tw.Current]

    // Add dependency metadata if not first task
    if tw.Current > 0 {
        previousTask := tw.Tasks[tw.Current-1]
        dependencyMetadata := map[string]interface{}{
            "depends_on":     previousTask.GetId(),
            "workflow_step":  tw.Current + 1,
            "total_steps":    len(tw.Tasks),
        }

        metadata, _ := structpb.NewStruct(dependencyMetadata)
        currentTask.Metadata = metadata
    }

    // Publish the task
    err := publishTaskUpdate(ctx, client, currentTask, "workflow_coordinator", "")
    if err != nil {
        return err
    }

    tw.Current++
    return nil
}
```

### Parallel Task Execution

Execute multiple tasks concurrently:

```go
func executeParallelTasks(ctx context.Context, client eventbus.AgentHubClient, tasks []*pb.Task) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(tasks))

    for _, task := range tasks {
        wg.Add(1)
        go func(t *pb.Task) {
            defer wg.Done()

            // Add parallel execution metadata
            t.Metadata = &structpb.Struct{
                Fields: map[string]*structpb.Value{
                    "execution_mode": structpb.NewStringValue("parallel"),
                    "batch_id":       structpb.NewStringValue(uuid.New().String()),
                    "batch_size":     structpb.NewNumberValue(float64(len(tasks))),
                },
            }

            err := publishTaskUpdate(ctx, client, t, "parallel_coordinator", "")
            if err != nil {
                errors <- err
            }
        }(task)
    }

    wg.Wait()
    close(errors)

    // Check for errors
    for err := range errors {
        if err != nil {
            return err
        }
    }

    return nil
}
```

## Best Practices

### 1. Always Update Task Status
```go
// Update status for every significant state change
task = acceptTask(task, agentID)
publishTaskUpdate(ctx, client, task, agentID, "")
```

### 2. Provide Meaningful Progress Updates
```go
// Send regular progress updates during long-running tasks
for progress := 10; progress <= 90; progress += 10 {
    task = sendProgressUpdate(task, progress, currentPhase, details)
    publishTaskUpdate(ctx, client, task, agentID, "")
    time.Sleep(processingInterval)
}
```

### 3. Include Rich Error Information
```go
errorDetails := map[string]interface{}{
    "input_validation_errors": validationErrors,
    "system_resources":        resourceSnapshot,
    "retry_strategy":         "exponential_backoff",
}
task = failTask(task, "Data validation failed", "INVALID_INPUT", errorDetails)
```

### 4. Maintain Complete Message History
```go
// Always append to history, never replace
task.History = append(task.History, statusMessage)
```

### 5. Use Appropriate Metadata
```go
// Include context for debugging and monitoring
metadata := map[string]interface{}{
    "processing_node":  hostname,
    "resource_usage":   resourceMetrics,
    "performance_metrics": performanceData,
}
```

This guide covered the complete A2A task lifecycle management. You now have the tools to create, manage, and coordinate complex task workflows with proper state management and comprehensive observability.