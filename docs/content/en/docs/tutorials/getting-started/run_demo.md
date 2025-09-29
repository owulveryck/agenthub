---
title: "Running the AgentHub Broker Demo"
weight: 50
description: "Walk through setting up and running the complete AgentHub broker system with Agent2Agent protocol task exchange capabilities. Learn how agents communicate and exchange structured tasks through the AgentHub broker."
---

# Running the AgentHub Broker Demo

This tutorial will walk you through setting up and running the complete AgentHub broker system with Agent2Agent protocol task exchange capabilities. By the end of this tutorial, you'll have agents communicating and exchanging Agent2Agent-structured tasks through the AgentHub broker.

## Prerequisites

- Go 1.24 or later installed
- Protocol Buffers compiler (protoc) installed
- Basic understanding of gRPC and message brokers

## Step 1: Build the Components

First, let's build all the necessary components:

```bash
# Build the broker
go build -o bin/broker ./broker

# Build the subscriber (agent)
go build -o bin/subscriber ./agents/subscriber

# Build the publisher
go build -o bin/publisher ./agents/publisher
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

## Step 4: Send Agent2Agent Tasks

Open a third terminal and run the publisher to send Agent2Agent protocol task messages:

```bash
./bin/publisher
```

You'll see the publisher send various Agent2Agent protocol task messages through the AgentHub broker:

```
time=2025-09-29T11:53:50.903+02:00 level=INFO msg="Starting publisher demo"
time=2025-09-29T11:53:50.905+02:00 level=INFO msg="Testing Agent2Agent Task Publishing via AgentHub with observability"
time=2025-09-29T11:53:50.905+02:00 level=INFO msg="Publishing task" task_id=task_greeting_1759139630 task_type=greeting responder_agent_id=agent_demo_subscriber
time=2025-09-29T11:53:53.907+02:00 level=INFO msg="Task published successfully" task_id=task_greeting_1759139630
time=2025-09-29T11:53:53.908+02:00 level=INFO msg="Publishing task" task_id=task_math_calculation_1759139633 task_type=math_calculation responder_agent_id=agent_demo_subscriber
time=2025-09-29T11:53:56.912+02:00 level=INFO msg="Publishing task" task_id=task_random_number_1759139636 task_type=random_number responder_agent_id=agent_demo_subscriber
time=2025-09-29T11:53:58.915+02:00 level=INFO msg="All tasks published! Check subscriber logs for results"
```

## Step 5: Observe Task Processing

Switch back to the subscriber terminal to see the agent processing tasks in real-time:

```
time=2025-09-29T11:54:15.123+02:00 level=INFO msg="Processing task" task_id=task_greeting_1759139630 task_type=greeting requester_agent_id=agent_demo_publisher
time=2025-09-29T11:54:15.125+02:00 level=INFO msg="Task completed successfully" task_id=task_greeting_1759139630 task_type=greeting status=TASK_STATUS_COMPLETED
time=2025-09-29T11:54:18.135+02:00 level=INFO msg="Processing task" task_id=task_math_calculation_1759139633 task_type=math_calculation requester_agent_id=agent_demo_publisher
time=2025-09-29T11:54:18.137+02:00 level=INFO msg="Task completed successfully" task_id=task_math_calculation_1759139633 task_type=math_calculation status=TASK_STATUS_COMPLETED
time=2025-09-29T11:54:21.142+02:00 level=INFO msg="Processing task" task_id=task_random_number_1759139636 task_type=random_number requester_agent_id=agent_demo_publisher
```

The agent will process each task and log the results with structured logging.

## Step 6: Check the Broker Logs

In the first terminal (broker server), you'll see logs showing message routing:

```
2025/09/27 16:34:33 Received task request: task_greeting_1758983673 (type: greeting) from agent: agent_demo_publisher
2025/09/27 16:34:35 Received task result for task: task_greeting_1758983673 from agent: agent_demo_subscriber
2025/09/27 16:34:35 Received task progress for task: task_greeting_1758983673 (100%) from agent: agent_demo_subscriber
```

## Understanding What Happened

1. **Agent2Agent Task Requests**: The publisher sent structured Agent2Agent protocol task requests with:
   - Unique task IDs
   - Task types and parameters
   - Requester and responder agent IDs
   - Priority levels

2. **AgentHub Routing**: The AgentHub broker:
   - Received Agent2Agent tasks from publishers
   - Routed tasks to appropriate subscriber agents
   - Forwarded task results and progress updates back to requesters

3. **Agent Task Processing**: The subscriber agent:
   - Received Agent2Agent tasks assigned to its agent ID
   - Processed them based on task type
   - Sent progress updates during execution using Agent2Agent progress structures
   - Published final results back through the AgentHub broker

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

**Build Errors**: Regenerate protocol buffer files and ensure all imports are correct:
```bash
find . -name "*.pb.go" -delete
protoc --go_out=. --go-grpc_out=. proto/eventbus.proto
```

You now have a working AgentHub broker system implementing Agent2Agent protocol task exchange! The agents can exchange structured Agent2Agent tasks, track progress, and receive results - all through your AgentHub broker.