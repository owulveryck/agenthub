---
title: "Agent2Agent (A2A) Protocol Migration"
weight: 40
description: "Understanding the migration to Agent2Agent protocol compliance while maintaining Event-Driven Architecture benefits."
---

# Agent2Agent (A2A) Protocol Migration

This document explains the migration of AgentHub to full Agent2Agent (A2A) protocol compliance while maintaining the essential Event-Driven Architecture (EDA) patterns that make the system scalable and resilient.

## What is the Agent2Agent Protocol?

The Agent2Agent (A2A) protocol is a standardized specification for communication between AI agents. It defines:

- **Standardized Message Formats**: Using `Message`, `Part`, `Task`, and `Artifact` structures
- **Task Lifecycle Management**: Clear states (SUBMITTED, WORKING, COMPLETED, FAILED, CANCELLED)
- **Agent Discovery**: Using `AgentCard` for capability advertisement
- **Interoperability**: Ensuring agents can communicate across different platforms

## Why Migrate to A2A?

### Benefits of A2A Compliance

1. **Interoperability**: AgentHub can now communicate with any A2A-compliant agent or system
2. **Standardization**: Clear, well-defined message formats reduce integration complexity
3. **Ecosystem Compatibility**: Join the growing ecosystem of A2A-compatible tools
4. **Future-Proofing**: Built on industry standards rather than custom protocols

### Maintained EDA Benefits

- **Scalability**: Event-driven routing scales to thousands of agents
- **Resilience**: Asynchronous communication handles network partitions gracefully
- **Flexibility**: Topic-based routing and priority queues enable sophisticated workflows
- **Observability**: Built-in tracing and metrics for production deployments

## Hybrid Architecture

AgentHub implements a **hybrid approach** that combines the best of both worlds:

```
┌─────────────────────────────────────────────────────────────────┐
│                   A2A Protocol Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐│
│  │ A2A Message │  │  A2A Task   │  │ A2A Artifact│  │A2A Agent││
│  │  (standard) │  │ (standard)  │  │ (standard)  │  │  Card   ││
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘│
├─────────────────────────────────────────────────────────────────┤
│                    EDA Transport Layer                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐│
│  │ AgentEvent  │  │Event Router │  │ Subscribers │  │Priority ││
│  │  Wrapper    │  │             │  │  Manager    │  │ Queues  ││
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘│
├─────────────────────────────────────────────────────────────────┤
│                      gRPC Infrastructure                       │
└─────────────────────────────────────────────────────────────────┘
```

### How It Works

1. **A2A Messages** are created using standard A2A structures (`Message`, `Task`, etc.)
2. **EDA Wrapper** wraps A2A messages in `AgentEvent` for transport
3. **Event Routing** uses EDA patterns (pub/sub, priority, topics) for delivery
4. **A2A Compliance** ensures messages follow A2A protocol semantics

## API Changes

### Before (Legacy API)

```go
// Legacy TaskMessage (deprecated)
taskPublisher.PublishTask(ctx, &agenthub.PublishTaskRequest{
    TaskType: "greeting",
    Parameters: map[string]interface{}{
        "name": "Claude",
    },
    RequesterAgentID: "my_agent",
    ResponderAgentID: "target_agent",
})
```

### After (A2A-Compliant API)

```go
// A2A-compliant task publishing
content := []*pb.Part{
    {
        Part: &pb.Part_Text{
            Text: "Hello! Please provide a greeting for Claude.",
        },
    },
}

task, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
    TaskType:         "greeting",
    Content:          content,
    RequesterAgentID: "my_agent",
    ResponderAgentID: "target_agent",
    Priority:         pb.Priority_PRIORITY_MEDIUM,
    ContextID:        "conversation_123",
})
```

## Message Structure Changes

### A2A Message Format

```protobuf
message Message {
  string message_id = 1;       // Unique message identifier
  string context_id = 2;       // Conversation context
  string task_id = 3;          // Associated task (optional)
  Role role = 4;               // USER or AGENT
  repeated Part content = 5;   // Message content parts
  google.protobuf.Struct metadata = 6; // Additional metadata
}

message Part {
  oneof part {
    string text = 1;           // Text content
    DataPart data = 2;         // Structured data
    FilePart file = 3;         // File reference
  }
}
```

### A2A Task Format

```protobuf
message Task {
  string id = 1;                    // Task identifier
  string context_id = 2;            // Conversation context
  TaskStatus status = 3;            // Current status
  repeated Message history = 4;     // Message history
  repeated Artifact artifacts = 5;  // Task outputs
  google.protobuf.Struct metadata = 6; // Task metadata
}

enum TaskState {
  TASK_STATE_SUBMITTED = 0;    // Task created
  TASK_STATE_WORKING = 1;      // Task in progress
  TASK_STATE_COMPLETED = 2;    // Task completed successfully
  TASK_STATE_FAILED = 3;       // Task failed
  TASK_STATE_CANCELLED = 4;    // Task cancelled
}
```

## Migration Guide

### For Publishers

1. Replace `TaskPublisher` with `A2ATaskPublisher`
2. Use `A2APublishTaskRequest` with A2A `Part` structures
3. Handle returned A2A `Task` objects

### For Subscribers

1. Replace `TaskSubscriber` with `A2ATaskSubscriber`
2. Update handlers to process A2A `Task` and `Message` objects
3. Return A2A `Artifact` objects instead of custom results

### For Custom Integrations

1. Update protobuf imports to use `events/a2a` package
2. Replace custom message structures with A2A equivalents
3. Use `AgentHub` service instead of `EventBus`

## Backward Compatibility

The migration maintains **wire-level compatibility** through:

- **Deprecated Types**: Legacy message types marked as deprecated but still supported
- **Automatic Conversion**: EDA broker converts between legacy and A2A formats when needed
- **Graceful Migration**: Existing agents can migrate incrementally

## Testing A2A Compliance

Run the demo to verify A2A compliance:

```bash
# Terminal 1: Start A2A broker
make run-server

# Terminal 2: Start A2A subscriber
make run-subscriber

# Terminal 3: Start A2A publisher
make run-publisher
```

Expected output shows successful A2A task processing:
- Publisher: "Published A2A task"
- Subscriber: "Task processing completed"
- Artifacts generated in A2A format

## Best Practices

1. **Use A2A Types**: Always use A2A message structures for new code
2. **Context Management**: Use `context_id` to group related messages
3. **Proper Parts**: Structure content using appropriate `Part` types
4. **Artifact Returns**: Return structured `Artifact` objects from tasks
5. **Status Updates**: Properly manage task lifecycle states

The A2A migration ensures AgentHub remains both standards-compliant and highly scalable through its hybrid EDA+A2A architecture.