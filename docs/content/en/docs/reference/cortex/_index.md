---
title: "Cortex Reference"
linkTitle: "Cortex"
weight: 50
description: >
  Technical reference for the Cortex orchestration engine
---

# Cortex Reference Documentation

Complete API and configuration reference for Cortex.

## Overview

Cortex is an asynchronous AI orchestration engine that coordinates multi-agent workflows through event-driven architecture.

**Version**: 0.1.0 (POC)
**Status**: Production-Ready Architecture (Mock LLM)

## Core Interfaces

### StateManager Interface

Manages conversation persistence.

```go
package state

type StateManager interface {
    // Get retrieves conversation state for a session
    Get(sessionID string) (*ConversationState, error)

    // Set persists conversation state
    Set(sessionID string, state *ConversationState) error

    // Delete removes conversation state
    Delete(sessionID string) error

    // WithLock executes a function with exclusive session access
    WithLock(sessionID string, fn func(*ConversationState) error) error
}
```

**Implementations**:
- `InMemoryStateManager` - POC implementation (in-memory)
- Future: `RedisStateManager`, `PostgresStateManager`

**Usage**:
```go
sm := state.NewInMemoryStateManager()

// Get state
state, err := sm.Get("session-123")

// Update with lock
err = sm.WithLock("session-123", func(state *ConversationState) error {
    state.Messages = append(state.Messages, newMessage)
    return nil
})
```

### LLM Client Interface

Abstraction for AI decision-making.

```go
package llm

type Client interface {
    // Decide analyzes state and returns actions to take
    Decide(
        ctx context.Context,
        conversationHistory []*pb.Message,
        availableAgents []*pb.AgentCard,
        newEvent *pb.Message,
    ) (*Decision, error)
}
```

**Decision Structure**:
```go
type Decision struct {
    Reasoning string   // Why these actions were chosen
    Actions   []Action // Actions to execute
}

type Action struct {
    Type string // "chat.response" or "task.request"

    // For chat.response
    ResponseText string

    // For task.request
    TaskType    string
    TaskPayload map[string]interface{}
    TargetAgent string

    CorrelationID string
}
```

**Implementations**:
- `MockClient` - Testing implementation
- Future: `VertexAIClient`, `OpenAIClient`

**Usage**:
```go
llmClient := llm.NewMockClient()

decision, err := llmClient.Decide(
    ctx,
    conversationHistory,
    availableAgents,
    newMessage,
)

for _, action := range decision.Actions {
    // Execute action
}
```

### MessagePublisher Interface

Publishes messages to Event Bus.

```go
package cortex

type MessagePublisher interface {
    PublishMessage(
        ctx context.Context,
        msg *pb.Message,
        routing *pb.AgentEventMetadata,
    ) error
}
```

**Implementation**:
```go
type AgentHubMessagePublisher struct {
    client *agenthub.AgentHubClient
}

func (a *AgentHubMessagePublisher) PublishMessage(
    ctx context.Context,
    msg *pb.Message,
    routing *pb.AgentEventMetadata,
) error {
    _, err := a.client.Client.PublishMessage(ctx, &pb.PublishMessageRequest{
        Message: msg,
        Routing: routing,
    })
    return err
}
```

## Data Structures

### ConversationState

```go
type ConversationState struct {
    SessionID        string                      // Unique session identifier
    Messages         []*pb.Message               // Full conversation history
    PendingTasks     map[string]*TaskContext     // In-flight tasks
    RegisteredAgents map[string]*pb.AgentCard    // Available agents
}
```

**Fields**:
- `SessionID` - Unique identifier for the conversation (e.g., "session-123")
- `Messages` - Complete message history (USER and AGENT messages)
- `PendingTasks` - Tasks awaiting completion (keyed by task_id)
- `RegisteredAgents` - Agents available for this session (keyed by agent_id)

### TaskContext

```go
type TaskContext struct {
    TaskID        string       // Unique task identifier
    TaskType      string       // Type of task (e.g., "transcription")
    RequestedAt   int64        // Unix timestamp when task was created
    OriginalInput *pb.Message  // Original user message that triggered task
    UserNotified  bool         // Whether user received acknowledgment
}
```

## Cortex Core API

### Constructor

```go
func NewCortex(
    stateManager StateManager,
    llmClient llm.Client,
    messagePublisher MessagePublisher,
) *Cortex
```

**Parameters**:
- `stateManager` - State persistence implementation
- `llmClient` - LLM decision engine
- `messagePublisher` - Message publishing adapter

**Returns**: Configured Cortex instance

**Example**:
```go
cortex := cortex.NewCortex(
    state.NewInMemoryStateManager(),
    llm.NewMockClient(),
    &AgentHubMessagePublisher{client: agentHubClient},
)
```

### RegisterAgent

```go
func (c *Cortex) RegisterAgent(agentID string, card *pb.AgentCard)
```

Registers an agent's capabilities with Cortex.

**Parameters**:
- `agentID` - Unique agent identifier
- `card` - Agent capability card

**Example**:
```go
cortex.RegisterAgent("transcriber-1", &pb.AgentCard{
    Name: "Audio Transcriber",
    Description: "Transcribes audio files to text",
    Skills: []*pb.AgentSkill{
        {
            Id: "transcribe",
            Name: "Transcription",
            Description: "Converts speech to text",
        },
    },
})
```

### GetAvailableAgents

```go
func (c *Cortex) GetAvailableAgents() []*pb.AgentCard
```

Returns all registered agents.

**Returns**: Slice of agent cards

**Example**:
```go
agents := cortex.GetAvailableAgents()
for _, agent := range agents {
    fmt.Printf("Agent: %s - %s\n", agent.Name, agent.Description)
}
```

### HandleMessage

```go
func (c *Cortex) HandleMessage(ctx context.Context, msg *pb.Message) error
```

Main entry point for processing messages.

**Parameters**:
- `ctx` - Context for cancellation and tracing
- `msg` - A2A protocol message

**Returns**: Error if processing failed

**Message Types Handled**:
1. Chat requests (role=USER) → Cortex decides response
2. Task results (role=AGENT, task_id set) → Cortex synthesizes result

**Example**:
```go
message := &pb.Message{
    MessageId: "msg-123",
    ContextId: "session-456",
    Role: pb.Role_ROLE_USER,
    Content: []*pb.Part{
        {Part: &pb.Part_Text{Text: "Hello"}},
    },
}

err := cortex.HandleMessage(ctx, message)
```

## Configuration

### Environment Variables

```bash
# LLM Configuration
CORTEX_LLM_MODEL=vertex-ai://gemini-2.0-flash  # LLM model to use

# AgentHub Connection
AGENTHUB_GRPC_PORT=127.0.0.1:50051            # Broker gRPC address
AGENTHUB_BROKER_ADDR=127.0.0.1                # Broker host

# Health Check
CORTEX_HEALTH_PORT=8086                       # Health check HTTP port

# Logging
LOG_LEVEL=info                                # Log level (debug, info, warn, error)
```

### Programmatic Configuration

```go
// Create state manager
stateManager := state.NewInMemoryStateManager()
// Or for production:
// stateManager := redis.NewRedisStateManager(redisClient)

// Create LLM client
llmClient := llm.NewMockClient()
// Or for production:
// llmClient := vertexai.NewClient(os.Getenv("CORTEX_LLM_MODEL"))

// Create message publisher
agentHubClient, _ := agenthub.NewAgentHubClient(config)
messagePublisher := &AgentHubMessagePublisher{client: agentHubClient}

// Create Cortex
cortex := cortex.NewCortex(stateManager, llmClient, messagePublisher)
```

## Message Correlation

### Session Management

Each conversation has a unique `context_id` (session ID).

**Session Lifecycle**:
1. CLI creates session: `cli_session_<timestamp>`
2. All messages in conversation share this `context_id`
3. Cortex maintains state per `context_id`
4. State persists across restarts (if using persistent storage)

### Task Correlation

Tasks are correlated via `task_id`.

**Task Lifecycle**:
1. User message triggers task
2. Cortex creates `task_id`: `task_<timestamp>`
3. Cortex adds to `PendingTasks` map
4. Agent receives task (with `task_id`)
5. Agent publishes result (same `task_id`)
6. Cortex matches result to pending task
7. Cortex removes from `PendingTasks`

## Error Handling

### State Errors

```go
type StateError struct {
    Op  string // Operation that failed
    Err string // Error message
}
```

**Common Errors**:
- Empty session ID
- Nil state
- Lock timeout

**Handling**:
```go
state, err := sm.Get(sessionID)
if err != nil {
    if stateErr, ok := err.(*state.StateError); ok {
        log.Printf("State operation %s failed: %s", stateErr.Op, stateErr.Err)
    }
}
```

### LLM Errors

**Handling**:
```go
decision, err := llmClient.Decide(ctx, history, agents, event)
if err != nil {
    // Fallback: send generic error response to user
    cortex.publishErrorResponse(ctx, sessionID, err)
    return err
}
```

### Message Processing Errors

Errors during `HandleMessage` are logged but don't crash Cortex:

```go
err := cortex.HandleMessage(ctx, msg)
if err != nil {
    logger.ErrorContext(ctx, "Failed to handle message",
        "error", err,
        "message_id", msg.GetMessageId(),
    )
    // Cortex continues processing other messages
}
```

## Performance

### Concurrency

- **Thread-Safe**: All operations use proper locking
- **Per-Session Locks**: No global bottleneck
- **Tested**: 100 concurrent goroutines, zero lost updates

### Complexity

| Operation | Time Complexity | Notes |
|-----------|----------------|-------|
| Get State | O(1) | Map lookup |
| Set State | O(1) | Map insert |
| WithLock | O(1) + fn | Per-session lock |
| HandleMessage | O(n) | n = message history for LLM |

### Scalability

**Vertical Scaling**:
- In-memory state limited by RAM
- Recommendation: ~10,000 active sessions per instance

**Horizontal Scaling** (Future):
- Partition sessions by `context_id` hash
- Multiple Cortex instances
- Shared persistent state (Redis Cluster)

## Testing

### Unit Tests

Run all tests:
```bash
go test -v ./agents/cortex/...
```

**Coverage**:
- State manager: 5 tests
- LLM client: 4 tests
- Cortex core: 4 tests

### Integration Testing

Use demo script:
```bash
./demo_cortex.sh
```

### Mock LLM

For testing custom decision logic:

```go
mockLLM := llm.NewMockClientWithFunc(
    func(ctx context.Context, history []*pb.Message, agents []*pb.AgentCard, event *pb.Message) (*llm.Decision, error) {
        // Custom logic
        return &llm.Decision{
            Actions: []llm.Action{
                {Type: "chat.response", ResponseText: "Test response"},
            },
        }, nil
    },
)

cortex := cortex.NewCortex(stateManager, mockLLM, publisher)
```

## Migration Guide

### From Mock LLM to Real LLM

1. Implement `llm.Client` interface:

```go
type VertexAIClient struct {
    client *genai.Client
    model  string
}

func (v *VertexAIClient) Decide(...) (*Decision, error) {
    // Build prompt
    prompt := buildPrompt(conversationHistory, availableAgents, newEvent)

    // Call Vertex AI
    response, err := v.client.Generate(ctx, prompt)
    if err != nil {
        return nil, err
    }

    // Parse response into Decision
    return parseDecision(response)
}
```

2. Update configuration:

```go
llmClient := vertexai.NewClient(os.Getenv("CORTEX_LLM_MODEL"))
cortex := cortex.NewCortex(stateManager, llmClient, publisher)
```

### From In-Memory to Persistent State

1. Implement `state.StateManager` interface:

```go
type RedisStateManager struct {
    client *redis.Client
}

func (r *RedisStateManager) Get(sessionID string) (*ConversationState, error) {
    data, err := r.client.Get(ctx, sessionID).Bytes()
    // Deserialize and return
}

func (r *RedisStateManager) Set(sessionID string, state *ConversationState) error {
    data, _ := json.Marshal(state)
    return r.client.Set(ctx, sessionID, data, 24*time.Hour).Err()
}
```

2. Update configuration:

```go
redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
stateManager := redis.NewRedisStateManager(redisClient)
cortex := cortex.NewCortex(stateManager, llmClient, publisher)
```

## Resources

- [Cortex Architecture](../../explanation/architecture/cortex_architecture/) - Design explanation
- [Getting Started Tutorial](../../tutorials/cortex/getting-started/) - Hands-on guide
- [Source Code](../../../../agents/cortex/) - Implementation
- [SPEC.md](../../../../agents/cortex/SPEC.md) - Original specification
