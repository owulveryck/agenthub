# Running the AgentHub Broker Demo

This tutorial will walk you through setting up and running the complete AgentHub broker system with Agent2Agent protocol task exchange capabilities. By the end of this tutorial, you'll have agents communicating and exchanging Agent2Agent-structured tasks through the AgentHub broker.

## Prerequisites

- Go 1.19 or later installed
- Protocol Buffers compiler (protoc) installed
- Basic understanding of gRPC and message brokers

## Step 1: Build the Components

First, let's build all the necessary components:

```bash
# Build the event bus server
go build -o bin/eventbus-server ./cmd/eventbus_server

# Build the subscriber (agent)
go build -o bin/subscriber ./cmd/subscriber

# Build the publisher
go build -o bin/publisher ./cmd/publisher
```

If you encounter any build errors, ensure the protocol buffer files are generated:

```bash
protoc --go_out=. --go-grpc_out=. proto/eventbus.proto
```

## Step 2: Start the AgentHub Broker Server

Open a terminal and start the AgentHub broker server:

```bash
./bin/eventbus-server
```

You should see output like:
```
2025/09/27 16:34:00 AgentHub gRPC server listening on [::]:50051
```

Keep this terminal open - the AgentHub broker needs to run continuously.

## Step 3: Start an Agent (Subscriber)

Open a second terminal and start an agent that can receive and process tasks:

```bash
./bin/subscriber
```

You should see output indicating the agent has started:
```
Agent started. Listening for events and tasks. Press Enter to stop.
2025/09/27 16:34:18 Agent agent_demo_subscriber subscribing to task results...
2025/09/27 16:34:18 Successfully subscribed to tasks for agent agent_demo_subscriber. Waiting for tasks...
2025/09/27 16:34:18 Successfully subscribed to task results for agent agent_demo_subscriber.
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
=== Testing Agent2Agent Task Publishing via AgentHub ===
Publishing task: task_greeting_1758983673 (type: greeting) to agent: agent_demo_subscriber
Task task_greeting_1758983673 published successfully.
Publishing task: task_math_calculation_1758983676 (type: math_calculation) to agent: agent_demo_subscriber
Task task_math_calculation_1758983676 published successfully.
Publishing task: task_random_number_1758983678 (type: random_number) to agent: agent_demo_subscriber
Task task_random_number_1758983678 published successfully.
Publishing task: task_unknown_task_1758983680 (type: unknown_task) to agent: agent_demo_subscriber
Task task_unknown_task_1758983680 published successfully.
```

## Step 5: Observe Task Processing

Switch back to the subscriber terminal to see the agent processing tasks in real-time:

```
2025/09/27 16:34:33 Received task: task_greeting_1758983673 (type: greeting) from agent: agent_demo_publisher
2025/09/27 16:34:33 Processing task task_greeting_1758983673 of type 'greeting'
2025/09/27 16:34:33 Published progress for task task_greeting_1758983673: 25% - Starting task processing
2025/09/27 16:34:35 Published progress for task task_greeting_1758983673: 75% - Generating greeting
2025/09/27 16:34:35 Published progress for task task_greeting_1758983673: 100% - Task completed
2025/09/27 16:34:35 Published result for task task_greeting_1758983673 with status TASK_STATUS_COMPLETED
```

The agent will also display macOS notifications (if on macOS) for each completed task.

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