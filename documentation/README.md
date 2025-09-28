# Agent2Agent Broker Documentation

This documentation provides comprehensive guidance for understanding and using the Agent2Agent communication broker. The documentation is organized according to the [Diátaxis framework](https://diataxis.fr/) to serve different user needs effectively.

## 📚 Documentation Structure

### [Tutorials](tutorials/)
*Learning-oriented guides that take you through practical exercises*

- **[Running the Demo](tutorials/run_demo.md)** - Complete walkthrough of setting up and running the Agent2Agent broker system with example agents exchanging tasks

### [How-to Guides](howto/)
*Goal-oriented guides that solve specific problems*

- **[Create a Publisher](howto/create_publisher.md)** - Step-by-step guide to building agents that publish tasks to other agents
- **[Create a Subscriber](howto/create_subscriber.md)** - Complete guide to building agents that receive and process tasks from other agents

### [Explanation](explanation/)
*Understanding-oriented discussions that provide context and background*

- **[The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)** - Deep dive into the philosophy and design principles behind Agent2Agent communication
- **[Understanding Tasks](explanation/the_tasks.md)** - Comprehensive explanation of task semantics, lifecycle, and design patterns

### [Reference](reference/)
*Information-oriented materials that describe the technical details*

- **[Task Reference](reference/the_tasks.md)** - Detailed reference for all task-related messages and operations

## 🎯 Quick Start

### I want to understand what this is about
→ Start with **[The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)**

### I want to see it working
→ Follow the **[Running the Demo](tutorials/run_demo.md)** tutorial

### I want to build an agent
→ Use the **[Create a Publisher](howto/create_publisher.md)** or **[Create a Subscriber](howto/create_subscriber.md)** guides

### I need technical details
→ Check the **[Task Reference](reference/the_tasks.md)**

## 🏗️ System Overview

The Agent2Agent broker enables autonomous agents to collaborate by exchanging structured tasks. Key features include:

- **Asynchronous task delegation** with progress tracking
- **Flexible agent addressing** (direct, broadcast, capability-based)
- **Rich task semantics** with priorities and deadlines
- **Built-in resilience** with timeout and retry capabilities

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Publisher     │    │     Broker      │    │   Subscriber    │
│    Agent        │    │     Server      │    │     Agent       │
│                 │    │                 │    │                 │
│ • Creates tasks │    │ • Routes tasks  │    │ • Processes     │
│ • Receives      │◄──►│ • Manages       │◄──►│   tasks         │
│   results       │    │   subscribers   │    │ • Reports       │
│ • Monitors      │    │ • Handles       │    │   progress      │
│   progress      │    │   failures      │    │ • Returns       │
│                 │    │                 │    │   results       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Message Flow

1. **Task Request**: Publisher creates and sends task to broker
2. **Task Routing**: Broker routes task to appropriate subscriber(s)
3. **Task Processing**: Subscriber processes task with progress updates
4. **Result Delivery**: Subscriber sends result back through broker to publisher

## 🛠️ Technologies Used

- **Protocol**: gRPC with Protocol Buffers
- **Language**: Go 1.24+
- **Message Format**: Structured protobuf messages with flexible JSON-like parameters
- **Architecture**: Event-driven with pub/sub patterns

## 📋 Prerequisites

- Go 1.24 or later
- Protocol Buffers compiler (`protoc`)
- Basic understanding of gRPC and distributed systems

## 🚀 Getting Started

1. **Understand the concepts**: Read [The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)
2. **See it in action**: Follow [Running the Demo](tutorials/run_demo.md)
3. **Build your first agent**: Use the [Create a Subscriber](howto/create_subscriber.md) guide
4. **Create task workflows**: Follow [Create a Publisher](howto/create_publisher.md)

## 📖 Learning Path

### For Developers New to Agent Systems
1. [The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md) - Understand the why
2. [Running the Demo](tutorials/run_demo.md) - See it working
3. [Understanding Tasks](explanation/the_tasks.md) - Learn the core concepts
4. [Create a Subscriber](howto/create_subscriber.md) - Build your first agent

### For Experienced Distributed Systems Developers
1. [Running the Demo](tutorials/run_demo.md) - Quick hands-on experience
2. [Task Reference](reference/the_tasks.md) - Technical specifications
3. [Create a Publisher](howto/create_publisher.md) - Build task orchestrators

### For System Architects
1. [The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md) - Design philosophy
2. [Understanding Tasks](explanation/the_tasks.md) - Task patterns and semantics
3. [Task Reference](reference/the_tasks.md) - Integration patterns
4. [Create a Publisher](howto/create_publisher.md) - Orchestration patterns

## 🤝 Contributing

This documentation follows the Diátaxis framework:

- **Tutorials** should be hands-on learning experiences
- **How-to guides** should solve specific problems
- **Explanations** should provide understanding and context
- **References** should be comprehensive and accurate

When contributing:
1. Choose the right documentation type for your content
2. Follow the established patterns and structure
3. Include practical examples where appropriate
4. Test all code examples to ensure they work

## 📄 License

This project is part of the agenthub repository. See the main repository for license information.

---

*This documentation is designed to grow with the project. If you find gaps or have suggestions for improvement, please contribute!*