---
title: "Running the A2A-Compliant AgentHub Demo"
weight: 50
description: "Walk through setting up and running the complete A2A-compliant AgentHub EDA broker system. Learn how agents communicate using Agent2Agent protocol messages through the Event-Driven Architecture broker."
---

# Running the A2A-Compliant AgentHub Demo

This tutorial will walk you through setting up and running the complete Agent2Agent (A2A) protocol-compliant AgentHub Event-Driven Architecture (EDA) broker system. By the end of this tutorial, you'll have agents communicating using standardized A2A messages through the scalable EDA broker.

## Prerequisites

- Go 1.24 or later installed
- Protocol Buffers compiler (protoc) installed
- Basic understanding of gRPC and message brokers

## Step 1: Build the A2A-Compliant Components

First, let's build all the A2A-compliant components using the Makefile:

```bash
# Build all A2A-compliant binaries (generates protobuf files first)
make build
```

This will:
1. Generate A2A protocol files from `proto/a2a_core.proto` and `proto/eventbus.proto`
2. Build the A2A-compliant broker, publisher, and subscriber binaries
3. Place all binaries in the `bin/` directory

You should see output like:
```
Building A2A-compliant server binary...
Building A2A-compliant publisher binary...
Building A2A-compliant subscriber binary...
Build complete. A2A-compliant binaries are in the 'bin/' directory.
```

## Step 2: Start the AgentHub Broker Server

Open a terminal and start the AgentHub broker server:

```bash
./bin/broker
```

You should see output like:
```
time=2025-09-29T11:51:26.612+02:00 level=INFO msg="Starting health server" port=8080
time=2025-09-29T11:51:26.611+02:00 level=INFO msg="AgentHub gRPC server with observability listening" address=[::]:50051 health_endpoint=http://localhost:8080/health metrics_endpoint=http://localhost:8080/metrics component=broker
```

Keep this terminal open - the AgentHub broker needs to run continuously.

## Step 3: Start an Agent (Subscriber)

Open a second terminal and start an agent that can receive and process tasks:

```bash
./bin/subscriber
```

You should see output indicating the agent has started:
```
time=2025-09-29T11:52:04.727+02:00 level=INFO msg="AgentHub client started with observability" broker_addr=localhost:50051 component=subscriber
time=2025-09-29T11:52:04.727+02:00 level=INFO msg="Starting health server" port=8082
time=2025-09-29T11:52:04.728+02:00 level=INFO msg="Agent started with observability. Listening for events and tasks."
time=2025-09-29T11:52:04.728+02:00 level=INFO msg="Subscribing to task results" agent_id=agent_demo_subscriber
time=2025-09-29T11:52:04.728+02:00 level=INFO msg="Subscribing to tasks" agent_id=agent_demo_subscriber
```

This agent can process several types of tasks:
- `greeting`: Simple greeting messages
- `math_calculation`: Basic arithmetic operations
- `random_number`: Random number generation
- Any unknown task type will be rejected

## Step 4: Send A2A-Compliant Tasks

Open a third terminal and run the publisher to send A2A protocol-compliant task messages:

```bash
./bin/publisher
```

You'll see the publisher send various A2A-compliant task messages through the AgentHub EDA broker:

```
time=2025-09-29T14:41:11.237+02:00 level=INFO msg="Starting publisher demo"
time=2025-09-29T14:41:11.237+02:00 level=INFO msg="Testing Agent2Agent Task Publishing via AgentHub with observability"
time=2025-09-29T14:41:11.237+02:00 level=INFO msg="Publishing A2A task" task_id=task_greeting_1759149671 task_type=greeting responder_agent_id=agent_demo_subscriber context_id=ctx_greeting_1759149671
time=2025-09-29T14:41:11.242+02:00 level=INFO msg="A2A task published successfully" task_id=task_greeting_1759149671 task_type=greeting event_id=evt_msg_greeting_1759149671_1759149671
time=2025-09-29T14:41:11.242+02:00 level=INFO msg="Published greeting task" task_id=task_greeting_1759149671
time=2025-09-29T14:41:14.243+02:00 level=INFO msg="Publishing A2A task" task_id=task_math_calculation_1759149674 task_type=math_calculation responder_agent_id=agent_demo_subscriber context_id=ctx_math_calculation_1759149674
time=2025-09-29T14:41:14.247+02:00 level=INFO msg="A2A task published successfully" task_id=task_math_calculation_1759149674 task_type=math_calculation event_id=evt_msg_math_calculation_1759149674_1759149674
time=2025-09-29T14:41:16.248+02:00 level=INFO msg="Publishing A2A task" task_id=task_random_number_1759149676 task_type=random_number responder_agent_id=agent_demo_subscriber context_id=ctx_random_number_1759149676
time=2025-09-29T14:41:16.249+02:00 level=INFO msg="Published random number task" task_id=task_random_number_1759149676
```

Notice how the A2A implementation includes:
- **Context IDs**: Each task is grouped in a conversation context (`ctx_greeting_...`)
- **Event IDs**: EDA wrapper events have unique identifiers for tracing
- **A2A Task Structure**: Tasks use A2A-compliant Message and Part formats

## Step 5: Observe A2A Task Processing

Switch back to the subscriber terminal to see the agent processing A2A tasks in real-time:

```
time=2025-09-29T14:41:11.243+02:00 level=INFO msg="Task processing completed" task_id=task_greeting_1759149671 status=TASK_STATE_COMPLETED has_artifact=true
time=2025-09-29T14:41:14.253+02:00 level=INFO msg="Task processing completed" task_id=task_math_calculation_1759149674 status=TASK_STATE_COMPLETED has_artifact=true
time=2025-09-29T14:41:16.249+02:00 level=INFO msg="Task processing completed" task_id=task_random_number_1759149676 status=TASK_STATE_COMPLETED has_artifact=true
```

Notice the A2A-compliant processing:
- **Task States**: Using A2A standard states (`TASK_STATE_COMPLETED`)
- **Artifacts**: Each completed task generates A2A artifacts (`has_artifact=true`)
- **Structured Processing**: Tasks are processed using A2A Message and Part handlers

## Step 6: Check the Broker Logs

In the first terminal (broker server), you'll see logs showing message routing:

```
2025/09/27 16:34:33 Received task request: task_greeting_1758983673 (type: greeting) from agent: agent_demo_publisher
2025/09/27 16:34:35 Received task result for task: task_greeting_1758983673 from agent: agent_demo_subscriber
2025/09/27 16:34:35 Received task progress for task: task_greeting_1758983673 (100%) from agent: agent_demo_subscriber
```

## Understanding What Happened

1. **A2A Message Creation**: The publisher created A2A-compliant messages with:
   - **Message Structure**: Using A2A Message format with Part content
   - **Context Grouping**: Each task belongs to a conversation context
   - **Task Association**: Messages are linked to specific A2A tasks
   - **Role Definition**: Messages specify USER (requester) or AGENT (responder) roles

2. **EDA Event Routing**: The AgentHub EDA broker:
   - **Wrapped A2A Messages**: A2A messages wrapped in AgentEvent for EDA transport
   - **Event-Driven Routing**: Used EDA patterns for scalable message delivery
   - **Task Storage**: Stored A2A tasks with full message history and artifacts
   - **Status Tracking**: Managed A2A task lifecycle (SUBMITTED → WORKING → COMPLETED)

3. **A2A Task Processing**: The subscriber agent:
   - **A2A Task Reception**: Received A2A tasks via EDA event streams
   - **Message Processing**: Processed A2A Message content using Part handlers
   - **Artifact Generation**: Generated structured A2A artifacts as task output
   - **Status Updates**: Published A2A-compliant status updates through EDA events

4. **Hybrid Architecture Benefits**:
   - **A2A Compliance**: Full interoperability with other A2A-compliant systems
   - **EDA Scalability**: Event-driven patterns for high-throughput scenarios
   - **Standards-Based**: Using industry-standard Agent2Agent protocol
   - **Observable**: Built-in tracing and metrics for production deployment

## Next Steps

Now that you have the basic system working, you can:

1. **Create Multiple Agents**: Run multiple subscriber instances with different agent IDs to see task distribution
2. **Add Custom Task Types**: Modify the subscriber to handle new types of tasks
3. **Build a Request-Response Flow**: Create an agent that both requests and processes tasks
4. **Monitor Task Progress**: Build a dashboard that subscribes to task progress updates

## Troubleshooting

**Port Already in Use**: If you see "bind: address already in use", kill any existing processes:
```bash
lsof -ti:50051 | xargs kill -9
```

**Agent Not Receiving Tasks**: Ensure the agent ID in the publisher matches the subscriber's agent ID (`agent_demo_subscriber`).

**Build Errors**: Regenerate A2A protocol buffer files and ensure all imports are correct:
```bash
# Clean old protobuf files
make clean

# Regenerate A2A protobuf files
make proto

# Rebuild everything
make build
```

**A2A Compliance Issues**: Verify A2A protocol structures are correctly generated:
```bash
# Check A2A core types
ls events/a2a/

# Should show: a2a_core.pb.go eventbus.pb.go eventbus_grpc.pb.go
```

You now have a working A2A-compliant AgentHub EDA broker system! The agents can exchange standardized A2A messages, maintain conversation contexts, generate structured artifacts, and track task lifecycles - all through your scalable Event-Driven Architecture broker with full Agent2Agent protocol compliance.