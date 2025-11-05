# Cortex - Asynchronous AI Orchestration Engine

**Version:** 0.1.0 (POC)
**Status:** Implemented

## Overview

Cortex is an asynchronous, event-driven AI orchestration engine that manages conversations and coordinates tasks across a swarm of specialized agents. It implements the architecture specified in [SPEC.md](SPEC.md).

## Architecture

```
┌─────────────────┐      ┌────────────────┐      ┌─────────────┐
│   Chat CLI      │─────>│   Event Bus    │<─────│   Cortex    │
└─────────────────┘      └────────────────┘      └─────────────┘
        ▲                        ▲                      │
        │                        │                      │
        │ chat.response          │ task.result          │ task.request
        │                        │                      │
        │                  ┌─────────────┐              │
        └──────────────────│  Echo Agent │◀─────────────┘
                           └─────────────┘
```

### Components Implemented

1. **Cortex Core** (`cortex.go`)
   - Orchestrates conversations and tasks
   - Maintains session state
   - Decides actions using LLM
   - Routes messages to agents

2. **State Manager** (`state/`)
   - Interface for conversation persistence
   - In-memory implementation (POC)
   - Thread-safe session locking
   - Tracks pending tasks and agent cards

3. **LLM Interface** (`llm/`)
   - Abstraction for AI decision-making
   - Mock implementation for testing
   - Extensible for real LLM clients (Vertex AI, OpenAI, etc.)

4. **Agents**
   - **Echo Agent**: Simple agent that echoes messages back
   - **Chat CLI**: Command-line interface for user interaction

## Key Features

### ✅ Implemented (POC)

- [x] Event-driven architecture with pub/sub
- [x] Asynchronous task execution
- [x] Stateful conversation management
- [x] Dynamic agent registration
- [x] LLM-based decision making (mock)
- [x] Thread-safe state operations
- [x] Message correlation with session/context IDs
- [x] CLI for user interaction

### 🚧 Future Work (Out of Scope for POC)

- [ ] Persistent state (Redis, PostgreSQL)
- [ ] Real LLM integration (Vertex AI, OpenAI)
- [ ] Agent health monitoring
- [ ] Web UI with real-time updates
- [ ] Advanced error recovery & retries
- [ ] Authentication & authorization

## Project Structure

```
agents/cortex/
├── README.md              # This file
├── SPEC.md                # Original specification
├── cortex.go              # Core orchestrator
├── cortex_test.go         # Core tests
├── state/
│   ├── interface.go       # State manager interface
│   ├── memory.go          # In-memory implementation
│   └── memory_test.go     # State manager tests
├── llm/
│   ├── interface.go       # LLM client interface
│   ├── mock.go            # Mock LLM for testing
│   └── mock_test.go       # LLM tests
└── cmd/
    └── main.go            # Service entry point
```

## Running the POC

### Prerequisites

- Go 1.21+
- Event Bus (broker) running

### Quick Start

Use the demo script to start all services:

```bash
./demo_cortex.sh
```

This will:
1. Start the Event Bus (broker)
2. Start Cortex orchestrator
3. Start Echo Agent
4. Launch the Chat CLI

### Manual Start

```bash
# Terminal 1: Start broker
make run-broker

# Terminal 2: Start Cortex
go run ./agents/cortex/cmd

# Terminal 3: Start Echo Agent
go run ./agents/echo_agent

# Terminal 4: Start CLI
go run ./agents/chat_cli
```

## Usage Example

```
> Hello Cortex
🤖 Cortex: Echo: Hello Cortex

> How are you?
🤖 Cortex: Echo: How are you?
```

## Testing

### Run All Tests

```bash
# Test state manager
go test -v ./agents/cortex/state

# Test LLM interface
go test -v ./agents/cortex/llm

# Test Cortex core
go test -v ./agents/cortex

# Run all tests
go test -v ./agents/cortex/...
```

### Test Coverage

```bash
go test -cover ./agents/cortex/...
```

## Design Decisions

### 1. Interface-Based Architecture

All major components (StateManager, LLM Client, MessagePublisher) are interfaces. This enables:
- Easy testing with mocks
- Swappable implementations (in-memory → Redis)
- Clear contracts between components

### 2. Test-Driven Development

All core functionality was developed test-first:
- StateManager: 5 tests covering CRUD, concurrency, locking
- LLM: 4 tests covering mocks, decision functions
- Cortex: 4 tests covering registration, chat, tasks

### 3. Message Self-Containment

Every message includes:
- `message_id`: Unique identifier
- `context_id`: Session/conversation ID
- `task_id`: For task correlation
- `role`: USER or AGENT
- `metadata`: Additional context

This ensures stateless agents and proper correlation.

### 4. Session-Level Locking

The StateManager uses per-session locks (not global locking):
- Allows concurrent updates to different sessions
- Prevents race conditions within a session
- Scales better than global locks

### 5. LLM as Control Plane

Cortex uses an LLM to decide "what to do next" rather than hard-coded rules:
- More flexible and adaptable
- Easy to add new capabilities (just update prompt)
- Mimics human decision-making

## Message Flow Example

### Simple Chat Request

```
1. User types "Hello" in CLI
2. CLI publishes Message (role=USER, context_id=session-1)
3. Cortex receives message
4. Cortex calls LLM: Decide(history, agents, newMsg)
5. LLM returns Decision: [chat.response: "Echo: Hello"]
6. Cortex publishes Message (role=AGENT, response text)
7. CLI receives and displays response
8. State updated with both messages
```

### Async Task Execution (Future)

```
1. User: "Transcribe this audio"
2. Cortex: Decide → [chat.response + task.request]
3. Cortex publishes: "I'll start transcription" (to user)
4. Cortex publishes: task.request (to transcription agent)
5. User sees acknowledgment immediately
6. Transcription agent processes (minutes)
7. Agent publishes: task.result
8. Cortex receives result
9. Cortex: Decide → [chat.response: "Transcription complete: ..."]
10. User sees final result
```

## Extending Cortex

### Adding a New Agent

1. Create agent that:
   - Publishes AgentCard on startup
   - Subscribes to task requests
   - Publishes task results

2. Agent will be automatically discovered by Cortex
3. LLM will include agent in decision-making

Example: See `agents/echo_agent/main.go`

### Adding Real LLM

Replace mock in `cmd/main.go`:

```go
func createLLMClient() llm.Client {
    if model := os.Getenv("CORTEX_LLM_MODEL"); model != "" {
        return vertexai.NewClient(model) // Implement this
    }
    return llm.NewMockClient()
}
```

### Adding Persistent State

Implement `state.StateManager` interface:

```go
type PostgresStateManager struct {
    db *sql.DB
}

func (p *PostgresStateManager) Get(sessionID string) (*ConversationState, error) {
    // Load from Postgres
}

func (p *PostgresStateManager) Set(sessionID string, state *ConversationState) error {
    // Save to Postgres
}
```

## Performance Characteristics

### State Manager (In-Memory)

- Get: O(1) with RLock
- Set: O(1) with Lock
- WithLock: O(1) per-session lock (no contention across sessions)

### Concurrency

- Tested with 100 concurrent updates to same session
- No lost updates (atomic operations)
- Scales horizontally by session partitioning

## Known Limitations (POC)

1. **State Loss**: In-memory state lost on restart
2. **No Agent Discovery**: Agents must be started manually
3. **Simple LLM**: Mock LLM uses basic echo logic
4. **No Retries**: Failed tasks are not retried
5. **No Timeouts**: No timeout mechanism for tasks
6. **No Persistence**: Messages not persisted to disk

## Contributing

This is a POC implementation. For production use:

1. Implement persistent StateManager (Redis/Postgres)
2. Integrate real LLM (Vertex AI, OpenAI)
3. Add agent health monitoring
4. Implement retry logic
5. Add comprehensive logging/tracing
6. Build Web UI with WebSockets

## References

- [SPEC.md](SPEC.md) - Original specification
- [A2A Protocol](../../proto/a2a_core.proto) - Agent-to-Agent message protocol
- [Event Bus](../../proto/eventbus.proto) - Event-driven architecture
