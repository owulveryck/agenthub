---
title: "A2A-Compliant AgentHub API Reference"
weight: 10
description: "Complete technical reference for the A2A-compliant AgentHub API, including all gRPC services, message types, and operational details."
---

# A2A-Compliant AgentHub API Reference

This document provides complete technical reference for the Agent2Agent (A2A) protocol-compliant AgentHub API, including all gRPC services, message types, and operational details.

## gRPC Service Definition

The AgentHub broker implements the `AgentHub` service as defined in [proto/eventbus.proto](proto/eventbus.proto):

```protobuf
service AgentHub {
  // ===== A2A Message Publishing (EDA style) =====

  // PublishMessage submits an A2A message for delivery through the broker
  rpc PublishMessage(PublishMessageRequest) returns (PublishResponse);

  // PublishTaskUpdate notifies subscribers about A2A task state changes
  rpc PublishTaskUpdate(PublishTaskUpdateRequest) returns (PublishResponse);

  // PublishTaskArtifact delivers A2A task output artifacts to subscribers
  rpc PublishTaskArtifact(PublishTaskArtifactRequest) returns (PublishResponse);

  // ===== A2A Event Subscriptions (EDA style) =====

  // SubscribeToMessages creates a stream of A2A message events for an agent
  rpc SubscribeToMessages(SubscribeToMessagesRequest) returns (stream AgentEvent);

  // SubscribeToTasks creates a stream of A2A task events for an agent
  rpc SubscribeToTasks(SubscribeToTasksRequest) returns (stream AgentEvent);

  // SubscribeToAgentEvents creates a unified stream of all events for an agent
  rpc SubscribeToAgentEvents(SubscribeToAgentEventsRequest) returns (stream AgentEvent);

  // ===== A2A Task Management (compatible with A2A spec) =====

  // GetTask retrieves the current state of an A2A task by ID
  rpc GetTask(GetTaskRequest) returns (a2a.Task);

  // CancelTask cancels an active A2A task and notifies subscribers
  rpc CancelTask(CancelTaskRequest) returns (a2a.Task);

  // ListTasks returns A2A tasks matching the specified criteria
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);

  // ===== Agent Discovery (A2A compatible) =====

  // GetAgentCard returns the broker's A2A agent card for discovery
  rpc GetAgentCard(google.protobuf.Empty) returns (a2a.AgentCard);

  // RegisterAgent registers an agent with the broker for event routing
  rpc RegisterAgent(RegisterAgentRequest) returns (RegisterAgentResponse);
}
```

## A2A Message Types

### Core A2A Types

#### A2A Message

Represents an A2A-compliant message for agent communication.

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

**Field Details:**
- `message_id`: Must be unique across all messages. Generated automatically if not provided
- `context_id`: Groups related messages in a conversation or workflow
- `task_id`: Links message to a specific A2A task
- `role`: Indicates whether message is from USER (requesting agent) or AGENT (responding agent)
- `content`: Array of A2A Part structures containing the actual message content
- `metadata`: Additional context for routing, processing, or debugging
- `extensions`: Protocol extension identifiers for future compatibility

#### A2A Part

Represents content within an A2A message.

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

#### A2A Task

Represents an A2A-compliant task with lifecycle management.

```protobuf
message Task {
  string id = 1;                    // Required: Task identifier
  string context_id = 2;            // Optional: Conversation context
  TaskStatus status = 3;            // Required: Current task status
  repeated Message history = 4;     // Message history for this task
  repeated Artifact artifacts = 5;  // Task output artifacts
  google.protobuf.Struct metadata = 6; // Task metadata
}

message TaskStatus {
  TaskState state = 1;              // Current task state
  Message update = 2;               // Status update message
  google.protobuf.Timestamp timestamp = 3; // Status timestamp
}

enum TaskState {
  TASK_STATE_SUBMITTED = 0;    // Task created and submitted
  TASK_STATE_WORKING = 1;      // Task in progress
  TASK_STATE_COMPLETED = 2;    // Task completed successfully
  TASK_STATE_FAILED = 3;       // Task failed with error
  TASK_STATE_CANCELLED = 4;    // Task cancelled
}
```

#### A2A Artifact

Represents structured output from completed tasks.

```protobuf
message Artifact {
  string artifact_id = 1;           // Required: Artifact identifier
  string name = 2;                  // Human-readable name
  string description = 3;           // Artifact description
  repeated Part parts = 4;          // Artifact content parts
  google.protobuf.Struct metadata = 5; // Artifact metadata
}
```

### EDA Event Wrapper Types

#### AgentEvent

Wraps A2A messages for Event-Driven Architecture transport.

```protobuf
message AgentEvent {
  string event_id = 1;                     // Unique event identifier
  google.protobuf.Timestamp timestamp = 2; // Event timestamp

  // A2A-compliant payload
  oneof payload {
    a2a.Message message = 10;              // A2A Message
    a2a.Task task = 11;                    // A2A Task
    TaskStatusUpdateEvent status_update = 12; // Task status change
    TaskArtifactUpdateEvent artifact_update = 13; // Artifact update
  }

  // EDA routing metadata
  AgentEventMetadata routing = 20;

  // Observability context
  string trace_id = 30;
  string span_id = 31;
}
```

#### AgentEventMetadata

Provides routing and delivery information for events.

```protobuf
message AgentEventMetadata {
  string from_agent_id = 1;               // Source agent identifier
  string to_agent_id = 2;                 // Target agent ID (empty = broadcast)
  string event_type = 3;                  // Event classification
  repeated string subscriptions = 4;      // Topic-based routing tags
  Priority priority = 5;                  // Delivery priority
}
```

### Request/Response Messages

#### PublishMessageRequest

```protobuf
message PublishMessageRequest {
  a2a.Message message = 1;                // A2A message to publish
  AgentEventMetadata routing = 2;         // EDA routing info
}
```

#### SubscribeToTasksRequest

```protobuf
message SubscribeToTasksRequest {
  string agent_id = 1;                    // Agent ID for subscription
  repeated string task_types = 2;         // Optional task type filter
  repeated a2a.TaskState states = 3;      // Optional state filter
}
```

#### GetTaskRequest

```protobuf
message GetTaskRequest {
  string task_id = 1;                     // Task identifier
  int32 history_length = 2;               // History limit (optional)
}
```

## API Operations

### Publishing A2A Messages

#### PublishMessage

Publishes an A2A message for delivery through the EDA broker.

**Go Example:**
```go
// Create A2A message content
content := []*pb.Part{
    {
        Part: &pb.Part_Text{
            Text: "Hello! Please process this request.",
        },
    },
    {
        Part: &pb.Part_Data{
            Data: &pb.DataPart{
                Data: &structpb.Struct{
                    Fields: map[string]*structpb.Value{
                        "operation": structpb.NewStringValue("process_data"),
                        "dataset_id": structpb.NewStringValue("dataset_123"),
                    },
                },
            },
        },
    },
}

// Create A2A message
message := &pb.Message{
    MessageId: "msg_12345",
    ContextId: "conversation_abc",
    TaskId:    "task_67890",
    Role:      pb.Role_ROLE_USER,
    Content:   content,
    Metadata: &structpb.Struct{
        Fields: map[string]*structpb.Value{
            "priority": structpb.NewStringValue("high"),
        },
    },
}

// Publish through AgentHub
response, err := client.PublishMessage(ctx, &pb.PublishMessageRequest{
    Message: message,
    Routing: &pb.AgentEventMetadata{
        FromAgentId: "requester_agent",
        ToAgentId:   "processor_agent",
        EventType:   "task_message",
        Priority:    pb.Priority_PRIORITY_HIGH,
    },
})
```

### Subscribing to A2A Events

#### SubscribeToTasks

Creates a stream of A2A task events for an agent.

**Go Example:**
```go
req := &pb.SubscribeToTasksRequest{
    AgentId: "processor_agent",
    TaskTypes: []string{"data_processing", "image_analysis"}, // Optional filter
}

stream, err := client.SubscribeToTasks(ctx, req)
if err != nil {
    return err
}

for {
    event, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    // Process different event types
    switch payload := event.GetPayload().(type) {
    case *pb.AgentEvent_Task:
        task := payload.Task
        log.Printf("Received A2A task: %s", task.GetId())

        // Process task using A2A handler
        artifact, status, errorMsg := processA2ATask(ctx, task)

        // Publish completion
        publishTaskCompletion(ctx, client, task, artifact, status, errorMsg)

    case *pb.AgentEvent_StatusUpdate:
        update := payload.StatusUpdate
        log.Printf("Task %s status: %s", update.GetTaskId(), update.GetStatus().GetState())

    case *pb.AgentEvent_ArtifactUpdate:
        artifact := payload.ArtifactUpdate
        log.Printf("Received artifact for task %s", artifact.GetTaskId())
    }
}
```

### A2A Task Management

#### GetTask

Retrieves the current state of an A2A task.

**Go Example:**
```go
req := &pb.GetTaskRequest{
    TaskId: "task_67890",
    HistoryLength: 10, // Optional: limit message history
}

task, err := client.GetTask(ctx, req)
if err != nil {
    return err
}

log.Printf("Task %s status: %s", task.GetId(), task.GetStatus().GetState())
log.Printf("Message history: %d messages", len(task.GetHistory()))
log.Printf("Artifacts: %d artifacts", len(task.GetArtifacts()))
```

#### CancelTask

Cancels an active A2A task.

**Go Example:**
```go
req := &pb.CancelTaskRequest{
    TaskId: "task_67890",
    Reason: "User requested cancellation",
}

task, err := client.CancelTask(ctx, req)
if err != nil {
    return err
}

log.Printf("Task %s cancelled", task.GetId())
```

### Agent Discovery

#### GetAgentCard

Returns the broker's A2A agent card for discovery.

**Go Example:**
```go
card, err := client.GetAgentCard(ctx, &emptypb.Empty{})
if err != nil {
    return err
}

log.Printf("AgentHub broker: %s v%s", card.GetName(), card.GetVersion())
log.Printf("Protocol version: %s", card.GetProtocolVersion())
log.Printf("Capabilities: streaming=%v", card.GetCapabilities().GetStreaming())

for _, skill := range card.GetSkills() {
    log.Printf("Skill: %s - %s", skill.GetName(), skill.GetDescription())
}
```

#### RegisterAgent

Registers an agent with the broker.

**Go Example:**
```go
agentCard := &pb.AgentCard{
    ProtocolVersion: "0.2.9",
    Name:           "my-processor-agent",
    Description:    "Data processing agent with A2A compliance",
    Version:        "1.0.0",
    Capabilities: &pb.AgentCapabilities{
        Streaming: true,
    },
    Skills: []*pb.AgentSkill{
        {
            Id:          "data_processing",
            Name:        "Data Processing",
            Description: "Process structured datasets",
            Tags:        []string{"data", "analysis"},
        },
    },
}

response, err := client.RegisterAgent(ctx, &pb.RegisterAgentRequest{
    AgentCard: agentCard,
    Subscriptions: []string{"data_processing", "analytics"},
})

if response.GetSuccess() {
    log.Printf("Agent registered with ID: %s", response.GetAgentId())
} else {
    log.Printf("Registration failed: %s", response.GetError())
}
```

## High-Level A2A Client Abstractions

### A2ATaskPublisher

Simplified interface for publishing A2A tasks.

```go
taskPublisher := &agenthub.A2ATaskPublisher{
    Client:         client,
    TraceManager:   traceManager,
    MetricsManager: metricsManager,
    Logger:         logger,
    ComponentName:  "my-publisher",
    AgentID:        "my-agent-id",
}

task, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
    TaskType:         "data_analysis",
    Content:          contentParts,
    RequesterAgentID: "my-agent-id",
    ResponderAgentID: "data-processor",
    Priority:         pb.Priority_PRIORITY_MEDIUM,
    ContextID:        "analysis-session-123",
})
```

### A2ATaskSubscriber

Simplified interface for processing A2A tasks.

```go
taskSubscriber := agenthub.NewA2ATaskSubscriber(client, "my-agent-id")

// Register task handlers
taskSubscriber.RegisterTaskHandler("data_analysis", func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
    // Process the A2A task
    result := processDataAnalysis(task, message)

    // Return A2A artifact
    artifact := &pb.Artifact{
        ArtifactId:  fmt.Sprintf("result_%s", task.GetId()),
        Name:        "analysis_result",
        Description: "Data analysis results",
        Parts: []*pb.Part{
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data: result,
                    },
                },
            },
        },
    }

    return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
})

// Start processing A2A tasks
err := taskSubscriber.SubscribeToTasks(ctx)
```

## Error Handling

### gRPC Status Codes

AgentHub uses standard gRPC status codes:

#### InvalidArgument (Code: 3)
- Missing required fields (message_id, role, content)
- Invalid A2A message structure
- Malformed Part content

#### NotFound (Code: 5)
- Task ID not found in GetTask/CancelTask
- Agent not registered

#### Internal (Code: 13)
- Server-side processing errors
- Message routing failures
- A2A validation errors

### Retry Patterns

```go
func publishWithRetry(ctx context.Context, client pb.AgentHubClient, req *pb.PublishMessageRequest) error {
    for attempt := 0; attempt < 3; attempt++ {
        _, err := client.PublishMessage(ctx, req)
        if err == nil {
            return nil
        }

        // Check if error is retryable
        if status.Code(err) == codes.InvalidArgument {
            return err // Don't retry validation errors
        }

        // Exponential backoff
        time.Sleep(time.Duration(1<<attempt) * time.Second)
    }
    return fmt.Errorf("max retries exceeded")
}
```

## Performance Considerations

### Message Size Limits
- **Maximum message size**: 4MB (gRPC default)
- **Recommended size**: <100KB for optimal A2A compliance
- **Large content**: Use FilePart references for large data

### A2A Best Practices
1. **Use structured Parts**: Prefer DataPart for structured data over text
2. **Context management**: Group related messages with context_id
3. **Artifact structure**: Return well-formed Artifact objects
4. **Task lifecycle**: Properly manage TaskState transitions
5. **Connection reuse**: Maintain persistent gRPC connections

This completes the comprehensive A2A-compliant API reference for AgentHub, covering all message types, operations, and integration patterns with practical examples.