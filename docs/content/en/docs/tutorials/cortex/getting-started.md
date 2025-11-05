---
title: "Getting Started with Cortex"
linkTitle: "Getting Started"
weight: 10
description: >
  Run your first Cortex orchestration demo and understand how it works
---

# Getting Started with Cortex

This tutorial will guide you through running your first Cortex demo and understanding the asynchronous orchestration pattern.

## What You'll Learn

By the end of this tutorial, you will:
- Understand what Cortex does and why it's useful
- Run the complete Cortex demo system
- Send messages through Cortex and see orchestration in action
- Understand the message flow between components

## Prerequisites

- AgentHub repository cloned locally
- Go 1.21+ installed
- Basic terminal/command-line knowledge

## Step 1: Build the Components

First, let's build all the necessary binaries:

```bash
cd /path/to/agenthub

# Build the broker (Event Bus)
go build -o bin/broker ./broker

# Build Cortex orchestrator
go build -o bin/cortex ./agents/cortex/cmd

# Build Echo agent (example agent)
go build -o bin/echo_agent ./agents/echo_agent

# Build CLI interface
go build -o bin/chat_cli ./agents/chat_cli
```

Verify all binaries were created:

```bash
ls -lh bin/ | grep -E "(broker|cortex|echo|chat_cli)"
```

You should see all four executables listed.

## Step 2: Understanding the Architecture

Before we run the demo, let's understand what each component does:

```
┌─────────────┐      ┌────────────┐      ┌──────────┐
│  Chat CLI   │─────>│ Event Bus  │<─────│ Cortex   │
│ (You type)  │      │  (Broker)  │      │ (Brain)  │
└─────────────┘      └────────────┘      └──────────┘
      ▲                     ▲                   │
      │                     │                   │
      │ Responses           │ Results           │ Tasks
      │                     │                   │
      │               ┌─────────────┐           │
      └───────────────│ Echo Agent  │◄──────────┘
                      │  (Worker)   │
                      └─────────────┘
```

**Components**:
1. **Event Bus (Broker)** - Routes all messages between components
2. **Cortex** - The "brain" that decides what to do with messages
3. **Echo Agent** - A simple worker that echoes messages back
4. **Chat CLI** - Your interface to interact with the system

## Step 3: Run the Demo (Automated)

The easiest way to start everything is using the demo script:

```bash
./demo_cortex.sh
```

This script will:
1. ✅ Start the Event Bus (broker)
2. ✅ Start Cortex orchestrator
3. ✅ Start Echo agent
4. ✅ Launch the interactive CLI

You should see:

```
╔════════════════════════════════════════════════════╗
║            Cortex POC Demo Launcher                ║
╚════════════════════════════════════════════════════╝

Starting Event Bus (Broker)...
✓ Broker started (PID: 12345)

Starting Cortex Orchestrator...
✓ Cortex started (PID: 12346)

Starting Echo Agent...
✓ Echo Agent started (PID: 12347)

════════════════════════════════════════════════════
  All services started successfully!
════════════════════════════════════════════════════

╔════════════════════════════════════════════════════╗
║         Cortex Chat CLI - POC Demo                ║
╚════════════════════════════════════════════════════╝

Session ID: cli_session_1234567890

Type your messages and press Enter.
Type 'exit' or 'quit' to end the session.
Press Ctrl+C to shutdown.

>
```

## Step 4: Interact with Cortex

Now you can type messages! Try these:

```
> Hello Cortex

🤖 Cortex: Echo: Hello Cortex

> How are you today?

🤖 Cortex: Echo: How are you today?

> Testing async orchestration

🤖 Cortex: Echo: Testing async orchestration
```

### What Just Happened?

Let's trace what happens when you type "Hello Cortex":

1. **You type** → CLI creates an A2A Message (role=USER)
2. **CLI publishes** → Event Bus receives message
3. **Cortex receives** → Retrieves conversation state
4. **Cortex decides** → LLM analyzes: "This is a greeting, respond friendly"
5. **Cortex publishes** → Sends response back through Event Bus
6. **CLI receives** → Displays the response to you

All of this happens asynchronously through event-driven architecture!

## Step 5: Understanding Message Flow

Let's look at what's happening under the hood.

### Message Structure

Every message contains:

```json
{
  "message_id": "cli_msg_1234567890",
  "context_id": "cli_session_1234567890",  // Session ID
  "role": "ROLE_USER",  // or ROLE_AGENT
  "content": [
    {
      "text": "Hello Cortex"
    }
  ],
  "metadata": {
    "task_type": "chat_request",
    "from_agent": "agent_chat_cli"
  }
}
```

**Key Fields**:
- `message_id` - Unique identifier for this message
- `context_id` - Groups messages in the same conversation
- `role` - USER (from human) or AGENT (from AI/system)
- `content` - The actual message text
- `metadata` - Additional context

### Conversation State

Cortex maintains state for each session:

```go
ConversationState {
    SessionID: "cli_session_1234567890"
    Messages: [
        {role: USER, text: "Hello Cortex"},
        {role: AGENT, text: "Echo: Hello Cortex"},
        {role: USER, text: "How are you today?"},
        {role: AGENT, text: "Echo: How are you today?"},
        // ... full history
    ]
    PendingTasks: {}
    RegisteredAgents: {"agent_echo": {...}}
}
```

This allows Cortex to:
- Remember conversation history
- Track which tasks are in-flight
- Know which agents are available

## Step 6: Run Manually (Optional)

For learning purposes, you can run each component manually in separate terminals:

**Terminal 1: Event Bus**
```bash
export AGENTHUB_GRPC_PORT=127.0.0.1:50051
./bin/broker
```

**Terminal 2: Cortex**
```bash
export AGENTHUB_BROKER_ADDR=127.0.0.1
./bin/cortex
```

**Terminal 3: Echo Agent**
```bash
export AGENTHUB_BROKER_ADDR=127.0.0.1
./bin/echo_agent
```

**Terminal 4: CLI**
```bash
export AGENTHUB_BROKER_ADDR=127.0.0.1
./bin/chat_cli
```

This gives you visibility into each component's logs.

## Step 7: Observing the Logs

When running manually, you'll see detailed logs from each component.

### Cortex Logs

```
INFO Cortex initialized agent_id=cortex llm_client=mock state_manager=in-memory
INFO Starting Cortex Orchestrator
INFO Cortex received message message_id=cli_msg_... context_id=cli_session_... role=ROLE_USER
INFO Cortex successfully processed message message_id=cli_msg_...
```

### Echo Agent Logs

```
INFO Echo agent registered successfully agent_id=agent_echo
INFO Received echo request message_id=task_request_... context_id=cli_session_...
INFO Published echo response message_id=msg_echo_response_... echo_text="Echo: Hello"
```

### Event Bus Logs

```
INFO Agent registered agent_id=cortex
INFO Agent registered agent_id=agent_echo
INFO Agent registered agent_id=agent_chat_cli
```

## Step 8: Shutting Down

To stop the demo:

1. In the CLI, type `exit` or `quit`
2. Or press `Ctrl+C`

The demo script will automatically clean up all processes.

If running manually, press `Ctrl+C` in each terminal (start with Terminal 4 and work backwards).

## What You've Learned

✅ **Architecture** - You understand the four main components
✅ **Message Flow** - You know how messages route through the system
✅ **Orchestration** - You see how Cortex coordinates agents
✅ **State Management** - You understand conversation state
✅ **Async Pattern** - You grasp the non-blocking nature

## Next Steps

Now that you've run the basic demo:

1. **[Build a Custom Agent](../building-custom-agent/)** - Create your own worker agent
2. **[Understand Cortex Architecture](../../explanation/architecture/cortex_architecture/)** - Deep dive into design
3. **[Async Task Orchestration](../async-orchestration/)** - Handle long-running tasks

## Troubleshooting

### Broker fails to start

**Error**: `failed to listen on port 50051`

**Solution**: Port is already in use. Kill existing process:
```bash
lsof -ti:50051 | xargs kill -9
```

### Cortex can't connect to broker

**Error**: `Failed to create AgentHub client`

**Solution**: Ensure broker is running first and environment variables are set:
```bash
export AGENTHUB_BROKER_ADDR=127.0.0.1
export AGENTHUB_GRPC_PORT=127.0.0.1:50051
```

### No response from Cortex

**Check**:
1. All services running? `ps aux | grep -E "(broker|cortex|echo)"`
2. Check logs for errors
3. Ensure Echo agent started successfully

### Messages not routing

**Debug**:
1. Check broker logs for registration confirmations
2. Verify all agents registered successfully
3. Ensure `context_id` is consistent in your session

## Key Concepts Recap

| Concept | What It Does |
|---------|-------------|
| **Event Bus** | Routes all messages between components |
| **Cortex** | Decides what to do with each message |
| **Agent** | Performs specific tasks (echo, transcribe, etc.) |
| **Session** | Groups related messages (context_id) |
| **State** | Remembers conversation history |
| **Async** | Non-blocking - user can chat while work happens |

## Code to Explore

If you want to dive into the code:

- **Cortex core logic**: `agents/cortex/cortex.go`
- **State management**: `agents/cortex/state/memory.go`
- **LLM interface**: `agents/cortex/llm/interface.go`
- **Echo agent**: `agents/echo_agent/main.go`
- **CLI**: `agents/chat_cli/main.go`

Each file is well-documented with comments explaining the logic.

## Resources

- [Cortex Architecture](../../explanation/architecture/cortex_architecture/) - Design deep-dive
- [SPEC.md](../../../../agents/cortex/SPEC.md) - Original specification
- [Implementation Summary](../../../../agents/cortex/IMPLEMENTATION_SUMMARY.md) - Build notes
- [Source Code](../../../../agents/cortex/) - Full implementation

---

**Congratulations!** You've successfully run your first Cortex orchestration demo. You're now ready to build custom agents and create sophisticated multi-agent systems.
