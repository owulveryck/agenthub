---
title: "How to Work with A2A Conversation Context"
weight: 20
description: "Learn how to manage conversation contexts in Agent2Agent protocol for multi-turn interactions and workflow coordination."
---

# How to Work with A2A Conversation Context

This guide shows you how to use A2A conversation contexts to group related messages, maintain state across interactions, and coordinate multi-agent workflows.

## Understanding A2A Conversation Context

A2A conversation context is identified by a `context_id` that groups related messages and tasks. This enables:

- **Multi-turn conversations** between agents
- **Workflow coordination** across multiple tasks
- **State preservation** throughout long-running processes
- **Message threading** for audit trails
- **Context-aware routing** based on conversation history

## Creating Conversation Contexts

### Simple Conversation Context

Start a basic conversation context:

```go
package main

import (
    "fmt"
    "github.com/google/uuid"
    pb "github.com/owulveryck/agenthub/events/a2a"
)

func createConversationContext(workflowType string) string {
    return fmt.Sprintf("ctx_%s_%s", workflowType, uuid.New().String())
}

func startConversation() *pb.Message {
    contextID := createConversationContext("user_onboarding")

    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: contextID,
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Please start the user onboarding process for new user.",
                },
            },
        },
    }
}
```

### Workflow-Specific Contexts

Create contexts for different workflow types:

```go
func createWorkflowContexts() map[string]string {
    return map[string]string{
        "data_analysis":    createConversationContext("data_analysis"),
        "image_processing": createConversationContext("image_processing"),
        "user_support":     createConversationContext("user_support"),
        "integration_test": createConversationContext("integration_test"),
    }
}
```

## Multi-Turn Conversations

### Conversation Initiation

Start a conversation with initial context:

```go
import (
    "google.golang.org/protobuf/types/known/structpb"
)

func initiateDataAnalysisConversation() *pb.Message {
    contextID := createConversationContext("data_analysis")

    // Initial conversation metadata
    contextMetadata, _ := structpb.NewStruct(map[string]interface{}{
        "workflow_type":    "data_analysis",
        "initiated_by":     "user_12345",
        "priority":         "high",
        "expected_steps":   []string{"validation", "processing", "analysis", "report"},
        "timeout_minutes":  30,
    })

    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: contextID,
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Please analyze the uploaded dataset and provide insights.",
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        contextMetadata,
                        Description: "Conversation context and workflow parameters",
                    },
                },
            },
        },
        Metadata: contextMetadata,
    }
}
```

### Continuing the Conversation

Add follow-up messages to the same context:

```go
func continueConversation(contextID, previousMessageID string) *pb.Message {
    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: contextID, // Same context as initial message
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Can you also include trend analysis in the report?",
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "follows_message": structpb.NewStringValue(previousMessageID),
                "conversation_turn": structpb.NewNumberValue(2),
            },
        },
    }
}
```

### Agent Responses in Context

Agents respond within the same conversation context:

```go
func createAgentResponse(contextID, requestMessageID, response string) *pb.Message {
    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: contextID, // Same context as request
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: response,
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "responding_to": structpb.NewStringValue(requestMessageID),
                "agent_id":      structpb.NewStringValue("data_analysis_agent"),
            },
        },
    }
}
```

## Context-Aware Task Management

### Creating Tasks with Context

Link tasks to conversation contexts:

```go
import (
    "google.golang.org/protobuf/types/known/timestamppb"
)

func createContextAwareTask(contextID string) *pb.Task {
    taskID := fmt.Sprintf("task_%s_%s", "analysis", uuid.New().String())

    return &pb.Task{
        Id:        taskID,
        ContextId: contextID, // Link to conversation
        Status: &pb.TaskStatus{
            State: pb.TaskState_TASK_STATE_SUBMITTED,
            Update: &pb.Message{
                MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
                ContextId: contextID,
                TaskId:    taskID,
                Role:      pb.Role_USER,
                Content: []*pb.Part{
                    {
                        Part: &pb.Part_Text{
                            Text: "Task submitted for data analysis workflow",
                        },
                    },
                },
            },
            Timestamp: timestamppb.Now(),
        },
        History: []*pb.Message{}, // Will be populated during processing
        Artifacts: []*pb.Artifact{}, // Will be populated on completion
    }
}
```

### Context-Based Task Querying

Retrieve all tasks for a conversation context:

```go
func getTasksForContext(ctx context.Context, client pb.AgentHubClient, contextID string) ([]*pb.Task, error) {
    response, err := client.ListTasks(ctx, &pb.ListTasksRequest{
        ContextId: contextID,
        Limit:     100,
    })
    if err != nil {
        return nil, err
    }

    return response.GetTasks(), nil
}
```

## Workflow Coordination

### Multi-Agent Workflow with Shared Context

Coordinate multiple agents within a single conversation:

```go
type WorkflowCoordinator struct {
    client    pb.AgentHubClient
    contextID string
    logger    *log.Logger
}

func (wc *WorkflowCoordinator) ExecuteDataPipeline(ctx context.Context) error {
    // Step 1: Data Validation
    validationTask := &pb.Task{
        Id:        fmt.Sprintf("task_validation_%s", uuid.New().String()),
        ContextId: wc.contextID,
        Status: &pb.TaskStatus{
            State: pb.TaskState_TASK_STATE_SUBMITTED,
            Update: &pb.Message{
                MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
                ContextId: wc.contextID,
                Role:      pb.Role_USER,
                Content: []*pb.Part{
                    {
                        Part: &pb.Part_Text{
                            Text: "Validate uploaded dataset for quality and completeness",
                        },
                    },
                },
            },
            Timestamp: timestamppb.Now(),
        },
    }

    // Publish validation task
    _, err := wc.client.PublishTaskUpdate(ctx, &pb.PublishTaskUpdateRequest{
        Task: validationTask,
        Routing: &pb.AgentEventMetadata{
            FromAgentId: "workflow_coordinator",
            ToAgentId:   "data_validator",
            EventType:   "task.validation",
            Priority:    pb.Priority_PRIORITY_HIGH,
        },
    })
    if err != nil {
        return err
    }

    // Step 2: Data Processing (after validation)
    processingTask := &pb.Task{
        Id:        fmt.Sprintf("task_processing_%s", uuid.New().String()),
        ContextId: wc.contextID, // Same context
        Status: &pb.TaskStatus{
            State: pb.TaskState_TASK_STATE_SUBMITTED,
            Update: &pb.Message{
                MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
                ContextId: wc.contextID,
                Role:      pb.Role_USER,
                Content: []*pb.Part{
                    {
                        Part: &pb.Part_Text{
                            Text: "Process validated dataset and extract features",
                        },
                    },
                },
                Metadata: &structpb.Struct{
                    Fields: map[string]*structpb.Value{
                        "depends_on": structpb.NewStringValue(validationTask.GetId()),
                        "workflow_step": structpb.NewNumberValue(2),
                    },
                },
            },
            Timestamp: timestamppb.Now(),
        },
    }

    // Publish processing task
    _, err = wc.client.PublishTaskUpdate(ctx, &pb.PublishTaskUpdateRequest{
        Task: processingTask,
        Routing: &pb.AgentEventMetadata{
            FromAgentId: "workflow_coordinator",
            ToAgentId:   "data_processor",
            EventType:   "task.processing",
            Priority:    pb.Priority_PRIORITY_MEDIUM,
        },
    })

    return err
}
```

## Context State Management

### Tracking Conversation State

Maintain state throughout the conversation:

```go
type ConversationState struct {
    ContextID     string                 `json:"context_id"`
    WorkflowType  string                 `json:"workflow_type"`
    CurrentStep   int                    `json:"current_step"`
    TotalSteps    int                    `json:"total_steps"`
    CompletedTasks []string              `json:"completed_tasks"`
    PendingTasks   []string              `json:"pending_tasks"`
    Variables      map[string]interface{} `json:"variables"`
    CreatedAt      time.Time             `json:"created_at"`
    UpdatedAt      time.Time             `json:"updated_at"`
}

func (cs *ConversationState) ToMetadata() (*structpb.Struct, error) {
    data := map[string]interface{}{
        "context_id":      cs.ContextID,
        "workflow_type":   cs.WorkflowType,
        "current_step":    cs.CurrentStep,
        "total_steps":     cs.TotalSteps,
        "completed_tasks": cs.CompletedTasks,
        "pending_tasks":   cs.PendingTasks,
        "variables":       cs.Variables,
        "updated_at":      cs.UpdatedAt.Format(time.RFC3339),
    }

    return structpb.NewStruct(data)
}

func (cs *ConversationState) UpdateFromMessage(message *pb.Message) {
    cs.UpdatedAt = time.Now()

    // Extract state updates from message metadata
    if metadata := message.GetMetadata(); metadata != nil {
        if step, ok := metadata.GetFields()["current_step"]; ok {
            cs.CurrentStep = int(step.GetNumberValue())
        }

        if vars, ok := metadata.GetFields()["variables"]; ok {
            if varsStruct := vars.GetStructValue(); varsStruct != nil {
                for key, value := range varsStruct.GetFields() {
                    cs.Variables[key] = value
                }
            }
        }
    }
}
```

### State-Aware Message Creation

Include conversation state in messages:

```go
func createStateAwareMessage(contextID string, state *ConversationState, content string) *pb.Message {
    stateMetadata, _ := state.ToMetadata()

    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: contextID,
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: content,
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        stateMetadata,
                        Description: "Current conversation state",
                    },
                },
            },
        },
        Metadata: stateMetadata,
    }
}
```

## Context-Based Routing

### Route Messages Based on Context

Use conversation context for intelligent routing:

```go
func routeByContext(contextID string) *pb.AgentEventMetadata {
    // Determine routing based on context type
    var targetAgent string
    var eventType string

    if strings.Contains(contextID, "data_analysis") {
        targetAgent = "data_analysis_agent"
        eventType = "data.analysis"
    } else if strings.Contains(contextID, "image_processing") {
        targetAgent = "image_processor"
        eventType = "image.processing"
    } else if strings.Contains(contextID, "user_support") {
        targetAgent = "support_agent"
        eventType = "support.request"
    } else {
        targetAgent = "" // Broadcast to all agents
        eventType = "general.message"
    }

    return &pb.AgentEventMetadata{
        FromAgentId:   "context_router",
        ToAgentId:     targetAgent,
        EventType:     eventType,
        Subscriptions: []string{eventType},
        Priority:      pb.Priority_PRIORITY_MEDIUM,
    }
}
```

### Subscribe to Context-Specific Events

Agents can subscribe to specific conversation contexts:

```go
func subscribeToContextEvents(ctx context.Context, client pb.AgentHubClient, agentID, contextPattern string) error {
    stream, err := client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
        AgentId: agentID,
        ContextPattern: contextPattern, // e.g., "ctx_data_analysis_*"
    })
    if err != nil {
        return err
    }

    for {
        event, err := stream.Recv()
        if err != nil {
            return err
        }

        if message := event.GetMessage(); message != nil {
            log.Printf("Received context message: %s in context: %s",
                message.GetMessageId(), message.GetContextId())

            // Process message within context
            processContextMessage(ctx, message)
        }
    }
}
```

## Best Practices

### 1. Use Descriptive Context IDs
```go
contextID := fmt.Sprintf("ctx_%s_%s_%s", workflowType, userID, uuid.New().String())
```

### 2. Preserve Context Across All Related Messages
```go
// All messages in the same workflow should use the same context_id
message.ContextId = existingContextID
```

### 3. Include Context Metadata for State Tracking
```go
contextMetadata := map[string]interface{}{
    "workflow_type":   "data_pipeline",
    "initiated_by":    userID,
    "current_step":    stepNumber,
    "total_steps":     totalSteps,
}
```

### 4. Use Context for Task Dependencies
```go
taskMetadata := map[string]interface{}{
    "context_id":     contextID,
    "depends_on":     previousTaskID,
    "workflow_step":  stepNumber,
}
```

### 5. Handle Context Cleanup
```go
// Set context expiration for long-running workflows
contextMetadata["expires_at"] = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
```

This guide covered conversation context management in A2A protocol. Next, learn about [Working with A2A Artifacts](../work_with_a2a_artifacts/) to understand how to create and manage structured outputs from completed tasks.