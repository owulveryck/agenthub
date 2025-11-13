---
title: "Cortex Architecture"
linkTitle: "Cortex Architecture"
weight: 20
description: >
  Understanding the Cortex asynchronous AI orchestration engine
---

# Cortex Architecture

Cortex is an asynchronous, event-driven AI orchestration engine that serves as the "brain" of multi-agent systems. It manages conversations, coordinates tasks across specialized agents, and uses LLM-based decision-making to route work intelligently.

## Overview

Traditional chatbots block on long-running operations. Cortex enables **non-blocking conversations** where users can interact while background tasks execute asynchronously.

### Key Innovation

```
Traditional:  User → Request → [BLOCKED] → Response
Cortex:       User → Request → Immediate Ack → [Async Work] → Final Response
```

Users receive instant acknowledgments and can continue conversing while agents process tasks in the background.

## Architecture Diagram

```
┌─────────────────┐      ┌────────────────┐      ┌─────────────┐
│   Chat CLI      │─────>│   Event Bus    │<─────│   Cortex    │
│  (User I/O)     │      │   (Broker)     │      │ Orchestrator│
└─────────────────┘      └────────────────┘      └─────────────┘
        ▲                        ▲                      │
        │ chat.response          │ task.result          │ task.request
        │                        │                      │
        │                  ┌─────────────┐              │
        └──────────────────│  Agent(s)   │◄─────────────┘
                           │ (Workers)   │
                           └─────────────┘
```

## Core Components

### 1. Cortex Orchestrator

The central decision-making engine that:

- **Maintains conversation state** - Full history per session
- **Registers agents dynamically** - Discovers capabilities via Agent Cards
- **Decides actions via LLM** - Uses AI to route work intelligently
- **Coordinates tasks** - Tracks pending work and correlates results

**File**: `agents/cortex/cortex.go`

### 2. State Manager

Manages conversational state with thread-safe operations:

```go
type ConversationState struct {
    SessionID        string
    Messages         []*pb.Message
    PendingTasks     map[string]*TaskContext
    RegisteredAgents map[string]*pb.AgentCard
}
```

**Key Features**:
- Per-session locking (no global bottleneck)
- Interface-based (swappable implementations)
- Currently in-memory (POC), production uses Redis/PostgreSQL

**Files**: `agents/cortex/state/`

### 3. LLM Client

Abstraction for AI-powered decision-making:

```go
type Client interface {
    Decide(
        ctx context.Context,
        conversationHistory []*pb.Message,
        availableAgents []*pb.AgentCard,
        newEvent *pb.Message,
    ) (*Decision, error)
}
```

The LLM analyzes:
- Conversation history
- Available agent capabilities
- New incoming messages

And returns decisions:
- `chat.response` - Reply to user
- `task.request` - Dispatch work to agent

**Files**: `agents/cortex/llm/`

#### IntelligentDecider: Context-Aware Orchestration

The `IntelligentDecider` is a mock LLM implementation that demonstrates intelligent, intent-based task orchestration. Unlike simple dispatchers that route every message to agents, it analyzes user intent before deciding whether to orchestrate with specialized agents or respond directly.

**Key Characteristics:**

1. **Intent Detection**: Analyzes message content for keywords indicating specific needs
   - Echo requests: "echo", "repeat", "say back"
   - Future: "translate", "summarize", "transcribe", etc.

2. **Conditional Orchestration**: Only dispatches to agents when user explicitly requests functionality
   - User: "echo hello" → Dispatches to echo_agent
   - User: "hello" → Responds directly (no agent needed)

3. **Transparent Reasoning**: Always explains decision-making process
   - All decisions include detailed reasoning visible in observability traces
   - Users understand why Cortex chose specific actions

**Example Flow:**

```go
// User message: "echo hello world"
decision := IntelligentDecider()(ctx, history, agents, userMsg)

// Returns:
Decision{
    Reasoning: "User message 'echo hello world' contains an explicit echo request (detected keywords: echo/repeat/say back). I'm dispatching this to the echo_agent which specializes in repeating messages back.",
    Actions: [
        {
            Type: "chat.response",
            ResponseText: "I detected you want me to echo something. I'm asking the echo agent to handle this for you.",
        },
        {
            Type: "task.request",
            TaskType: "echo",
            TargetAgent: "agent_echo",
            TaskPayload: {"input": "echo hello world"},
        },
    ],
}
```

**Comparison to Simple Dispatchers:**

| Approach | Every Message | Intent Detection | Explains Reasoning | Responds Directly |
|----------|---------------|------------------|-------------------|-------------------|
| TaskDispatcherDecider (deprecated) | Dispatches to agent | No | Minimal | No |
| **IntelligentDecider** | **Analyzes first** | **Yes** | **Detailed** | **Yes** |

**Design Benefits:**

- **Reduced Latency**: Simple queries get immediate responses without agent roundtrip
- **Resource Efficiency**: Agents only invoked when their specialized capabilities are needed
- **Better UX**: Users understand what Cortex is doing and why
- **Debuggability**: Reasoning in traces makes orchestration logic transparent
- **Extensibility**: Easy to add new intent patterns for new agent types

**Future Evolution:**

In production, the IntelligentDecider pattern should be replaced with a real LLM that performs function calling:

```go
// Production LLM receives tools/functions
tools := convertAgentCardsToTools(availableAgents)
decision := realLLM.Decide(history, tools, newMsg)

// LLM naturally decides:
// - "hello" → No function call, direct response
// - "echo hello" → Calls echo_agent function
// - "translate this to French" → Calls translation_agent function
```

The IntelligentDecider serves as a working example of the decision patterns a real LLM would follow.

### 4. Message Publisher

Interface for publishing messages to the Event Bus:

```go
type MessagePublisher interface {
    PublishMessage(
        ctx context.Context,
        msg *pb.Message,
        routing *pb.AgentEventMetadata,
    ) error
}
```

Adapts AgentHub client to Cortex's needs.

## Message Flow

### Simple Chat Request

```
1. User types "Hello" in CLI
   ↓
2. CLI publishes A2A Message (role=USER, context_id=session-1)
   ↓
3. Event Bus routes to Cortex
   ↓
4. Cortex retrieves conversation state
   ↓
5. Cortex calls LLM.Decide(history, agents, newMsg)
   ↓
6. LLM returns Decision: [chat.response: "Hello! How can I help?"]
   ↓
7. Cortex publishes A2A Message (role=AGENT, response text)
   ↓
8. Event Bus routes to CLI
   ↓
9. CLI displays response
   ↓
10. Cortex updates state with both messages
```

### Asynchronous Task Execution

```
1. User: "Transcribe this audio file"
   ↓
2. Cortex LLM decides: [chat.response + task.request]
   ↓
3. Cortex publishes:
   - Message to user: "I'll start transcription, this may take a few minutes"
   - Task request to transcription agent
   ↓
4. User sees immediate acknowledgment ✅
   User can continue chatting!
   ↓
5. Transcription agent processes (background, may take minutes)
   ↓
6. Agent publishes task.result with transcribed text
   ↓
7. Cortex receives result, calls LLM.Decide()
   ↓
8. LLM decides: [chat.response: "Transcription complete: <text>"]
   ↓
9. Cortex publishes final response to user
   ↓
10. User sees final result
```

## Design Patterns

### 1. Interface Segregation

All major components are interfaces:

- **StateManager** - Easy to swap (in-memory → Redis)
- **LLM Client** - Easy to test (mock → real AI)
- **MessagePublisher** - Decoupled from transport

**Benefits**:
- Testability (use mocks)
- Flexibility (swap implementations)
- Clear contracts

### 2. Session-Level Concurrency

Each session has its own lock:

```go
// NOT this (global bottleneck):
globalMutex.Lock()
updateState()
globalMutex.Unlock()

// But this (per-session):
sessionLock := getSessionLock(sessionID)
sessionLock.Lock()
updateState()
sessionLock.Unlock()
```

**Benefits**:
- Multiple sessions can update concurrently
- No contention across sessions
- Scales horizontally

### 3. LLM as Control Plane

Instead of hard-coded if/else routing:

```go
// Old way:
if strings.Contains(input, "transcribe") {
    dispatchToTranscriber()
} else if strings.Contains(input, "translate") {
    dispatchToTranslator()
}

// Cortex way:
decision := llm.Decide(history, agents, input)
executeActions(decision.Actions)
```

**Benefits**:
- Flexible - LLM adapts to context
- Extensible - Add agents, LLM discovers them
- Natural - Mimics human reasoning

**Implementation**: The `IntelligentDecider` (see LLM Client section above) demonstrates this pattern by analyzing user intent and making intelligent routing decisions with transparent reasoning.

### 4. Message Self-Containment

Every message is fully self-describing:

```protobuf
message Message {
    string message_id = 1;   // Unique ID
    string context_id = 2;   // Session/conversation ID
    string task_id = 3;      // Task correlation (if applicable)
    Role role = 4;           // USER or AGENT
    repeated Part content = 5;
    Struct metadata = 6;
}
```

**Benefits**:
- Agents are stateless (all context in message)
- Easy correlation (context_id, task_id)
- Traceable (message_id)

## State Management

### ConversationState Structure

```go
type ConversationState struct {
    SessionID        string
    Messages         []*pb.Message      // Full history
    PendingTasks     map[string]*TaskContext
    RegisteredAgents map[string]*pb.AgentCard
}
```

### TaskContext Tracking

```go
type TaskContext struct {
    TaskID        string
    TaskType      string
    RequestedAt   int64
    OriginalInput *pb.Message
    UserNotified  bool  // Did we acknowledge?
}
```

Cortex tracks:
- Which tasks are in-flight
- What the user originally requested
- Whether we've acknowledged the request

### State Lifecycle

```
1. Get session state (or create new)
2. Lock session for updates
3. Add new message to history
4. Call LLM to decide actions
5. Execute actions (publish messages)
6. Update pending tasks
7. Save state
8. Release lock
```

## Agent Discovery

### Agent Card Registration

Agents publish capabilities on startup:

```go
type AgentCard struct {
    Name        string
    Description string
    Skills      []*AgentSkill
}
```

Cortex maintains a registry:

```go
registeredAgents map[string]*pb.AgentCard
```

### Dynamic Tool List

When making LLM decisions, Cortex provides available agents:

```go
decision := llm.Decide(
    ctx,
    conversationHistory,
    cortex.GetAvailableAgents(),  // ← Dynamic list
    newEvent,
)
```

The LLM sees which tools are available and chooses appropriately.

## Scaling & Performance

### Concurrency Model

- **Lock Granularity**: Per-session (not global)
- **State Access**: O(1) lookups via map
- **Message Processing**: Asynchronous (non-blocking)

### Horizontal Scaling

Future: Partition sessions across multiple Cortex instances:

```
Cortex-1: handles sessions A-M
Cortex-2: handles sessions N-Z
```

Event Bus routes messages to correct instance based on context_id.

### Performance Characteristics

- **State Get**: O(1) with read lock
- **State Set**: O(1) with write lock
- **Concurrent Sessions**: No contention (per-session locks)

**Tested**: 100 goroutines updating same session → zero lost updates ✅

## Error Handling

### Agent Failures

When an agent fails:

1. Agent publishes `task.result` with status="failed"
2. Cortex receives result
3. LLM decides how to handle (inform user, retry, try alternative)
4. Cortex publishes response

### LLM Failures

If LLM client errors:

```go
decision, err := llm.Decide(...)
if err != nil {
    // Fallback: publish generic error response
    publishErrorResponse(ctx, session, err)
    return err
}
```

### State Corruption

Protected by:
- Transaction-like WithLock pattern
- Copy-on-read to prevent external mutations
- Validation on state load/save

## Implementation Status

### ✅ Implemented (POC)

- Core orchestrator logic
- In-memory state management
- Mock LLM client with IntelligentDecider (intent-based routing)
- Agent registration
- Message routing
- Task correlation
- CLI interface
- Echo agent (demo)
- Distributed tracing with OpenTelemetry

### 🚧 Future Work

- Persistent state (Redis, PostgreSQL)
- Real LLM integration (Vertex AI, OpenAI)
- Agent health monitoring
- Web UI with WebSockets
- Retry logic & timeouts
- Advanced error recovery

## Code Organization

```
agents/cortex/
├── cortex.go              # Core orchestrator with full observability
├── cortex_test.go         # Core tests (4 tests)
├── state/
│   ├── interface.go       # StateManager interface
│   ├── memory.go          # In-memory implementation
│   └── memory_test.go     # State tests (5 tests)
├── llm/
│   ├── interface.go       # LLM Client interface
│   ├── mock.go            # Mock implementations
│   │                      # - IntelligentDecider (intent-based)
│   │                      # - TaskDispatcherDecider (deprecated)
│   │                      # - SimpleEchoDecider
│   └── mock_test.go       # LLM tests (4 tests)
└── cmd/
    └── main.go            # Service entry point
```

**Total**: ~1,200 lines of production code + 500 lines of tests

## Testing Strategy

### Unit Tests

- **State Manager**: CRUD, concurrency, locking (5 tests)
- **LLM Client**: Mock behavior, decision functions (4 tests)
- **Cortex Core**: Registration, chat, tasks (4 tests)

All tests use interfaces and mocks (no external dependencies).

### Concurrency Testing

```go
func TestInMemoryStateManager_Concurrency(t *testing.T) {
    // Launch 100 goroutines updating same session
    for i := 0; i < 100; i++ {
        go func() {
            sm.WithLock(sessionID, func(state *ConversationState) error {
                state.Messages = append(state.Messages, msg)
                return nil
            })
        }()
    }

    // Assert: Exactly 100 messages (no lost updates)
}
```

### Integration Testing

Demo script (`demo_cortex.sh`) tests:
- Broker startup
- Cortex initialization
- Agent registration
- End-to-end message flow

## Configuration

### Environment Variables

```bash
# LLM Model (future)
CORTEX_LLM_MODEL=vertex-ai://gemini-2.0-flash

# AgentHub connection
AGENTHUB_GRPC_PORT=127.0.0.1:50051
AGENTHUB_BROKER_ADDR=127.0.0.1

# Health check
CORTEX_HEALTH_PORT=8086
```

### Programmatic Configuration

```go
cortex := cortex.NewCortex(
    state.NewInMemoryStateManager(),  // or Redis/Postgres
    llm.NewVertexAIClient(model),     // or Mock for testing
    messagePublisher,
)
```

## Observability

### Logging

Structured logging with context:

```go
client.Logger.InfoContext(ctx, "Cortex received message",
    "message_id", message.GetMessageId(),
    "context_id", message.GetContextId(),
    "role", message.GetRole().String(),
)
```

### Tracing

OpenTelemetry spans already in AgentHub client:

- Trace ID propagation
- Span relationships (parent → child)
- Error recording

### Metrics (Future)

- Messages processed per session
- LLM decision latency
- Task completion rates
- Error rates by type

## Security Considerations

### Current (POC)

- No authentication (all agents trusted)
- No authorization (all agents can do anything)
- No message validation (trusts well-formed protobufs)

### Future

- Agent authentication via mTLS
- Message signing & verification
- Rate limiting per agent
- Input sanitization for LLM prompts

## Best Practices

### For Cortex Operators

1. **Monitor state size** - Large conversation histories impact memory
2. **Configure LLM timeouts** - Prevent hanging on slow AI responses
3. **Use persistent state** - In-memory is POC only
4. **Enable tracing** - Essential for debugging async flows

### For Agent Developers

1. **Publish clear Agent Cards** - Cortex needs good descriptions
2. **Handle errors gracefully** - Publish failed task results, don't crash
3. **Use correlation IDs** - Essential for Cortex to track work
4. **Be stateless** - All context should be in messages

## Comparison to Alternatives

| Approach | Blocking | State Management | Extensibility |
|----------|----------|------------------|---------------|
| Traditional Chatbot | Yes ✗ | Simple | Hard-coded |
| Function Calling | Yes ✗ | Per-request | Config files |
| **Cortex** | **No ✓** | **Persistent** | **Dynamic** |

Cortex enables truly asynchronous, extensible AI systems.

## Resources

- [SPEC.md](../../../../agents/cortex/SPEC.md) - Original specification
- [Implementation Summary](../../../../agents/cortex/IMPLEMENTATION_SUMMARY.md) - Build details
- [README](../../../../agents/cortex/README.md) - Usage guide
- [Source Code](../../../../agents/cortex/) - Full implementation

## Next Steps

1. Read the [Cortex Tutorial](../../tutorials/cortex/getting-started/) to build your first orchestrator
2. See [How to Create Agents](../../howto/agents/create-cortex-agent/) for agent development
3. Check [Cortex API Reference](../../reference/cortex/api/) for detailed interface documentation
