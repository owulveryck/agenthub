# Agent2Agent Broker Documentation

This documentation provides comprehensive guidance for understanding and using the Agent2Agent communication broker. The documentation is organized according to the [Diátaxis framework](https://diataxis.fr/) to serve different user needs effectively.

## 📚 Documentation Structure

### [Tutorials](tutorials/)
*Learning-oriented guides that take you through practical exercises*

#### Getting Started
- **[Installation and Setup](tutorials/installation_and_setup.md)** - Step-by-step guide to install AgentHub and set up your development environment
- **[Running the Demo](tutorials/run_demo.md)** - Complete walkthrough of setting up and running the Agent2Agent broker system with example agents exchanging tasks

#### Advanced Features
- **[Dashboard Tour](tutorials/dashboard_tour.md)** - Guided tour of the AgentHub monitoring and observability dashboards
- **[Observability Demo](tutorials/observability_demo.md)** - Hands-on tutorial for setting up and using comprehensive observability features
- **[Building Multi-Agent Workflows](tutorials/building_multi_agent_workflows.md)** - Advanced tutorial for creating complex multi-agent systems with workflow orchestration

### [How-to Guides](howto/)
*Goal-oriented guides that solve specific problems*

#### Agent Development
- **[Create a Publisher](howto/create_publisher.md)** - Step-by-step guide to building agents that publish tasks to other agents
- **[Create a Subscriber](howto/create_subscriber.md)** - Complete guide to building agents that receive and process tasks from other agents

#### Operations & Monitoring
- **[Add Observability](howto/add_observability.md)** - How to integrate observability features into your agent systems
- **[Use Dashboards](howto/use_dashboards.md)** - Practical guide to using monitoring dashboards for system insights
- **[Debugging Agent Issues](howto/debugging_agent_issues.md)** - Practical troubleshooting guide for common agent development and deployment issues

### [Explanation](explanation/)
*Understanding-oriented discussions that provide context and background*

#### Core Concepts
- **[The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)** - Deep dive into the philosophy and design principles behind Agent2Agent communication
- **[Understanding Tasks](explanation/the_tasks.md)** - Comprehensive explanation of task semantics, lifecycle, and design patterns
- **[Unified Abstraction Library](explanation/unified_abstraction_library.md)** - Understanding the common abstractions for agent coordination

#### Architecture & Implementation
- **[Broker Architecture](explanation/broker_architecture.md)** - Detailed explanation of AgentHub's internal architecture and design decisions
- **[Performance and Scaling](explanation/performance_and_scaling.md)** - Understanding performance characteristics and scaling strategies
- **[Go Build Tags](explanation/go_build_tags.md)** - Explanation of build tag usage for optional features

#### Observability & Monitoring
- **[Distributed Tracing](explanation/distributed_tracing.md)** - Understanding distributed tracing concepts and implementation in AgentHub

### [Reference](reference/)
*Information-oriented materials that describe the technical details*

#### Core APIs
- **[API Reference](reference/api_reference.md)** - Complete gRPC API documentation with examples and error handling
- **[Task Reference](reference/the_tasks.md)** - Detailed reference for all task-related messages and operations
- **[Unified Abstraction API](reference/unified_abstraction_api.md)** - Technical reference for the unified abstraction interfaces
- **[Configuration Reference](reference/configuration_reference.md)** - Comprehensive guide to configuring brokers and agents

#### Observability & Health
- **[Observability Metrics](reference/observability_metrics.md)** - Complete catalog of all metrics exposed by AgentHub's observability system
- **[Health Endpoints](reference/health_endpoints.md)** - Documentation for health monitoring APIs and endpoint specifications
- **[Tracing API](reference/tracing_api.md)** - Technical reference for distributed tracing APIs and integration

## 🎯 Quick Start

### I'm new to AgentHub and want to get started
→ Begin with **[Installation and Setup](tutorials/installation_and_setup.md)** then **[Running the Demo](tutorials/run_demo.md)**

### I want to understand what this is about
→ Start with **[The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)**

### I want to see it working
→ Follow the **[Running the Demo](tutorials/run_demo.md)** tutorial

### I want to build an agent
→ Use the **[Create a Publisher](howto/create_publisher.md)** or **[Create a Subscriber](howto/create_subscriber.md)** guides

### I want to monitor my system
→ Follow **[Dashboard Tour](tutorials/dashboard_tour.md)** and **[Add Observability](howto/add_observability.md)**

### I need technical details
→ Check the **[API Reference](reference/api_reference.md)** and **[Configuration Reference](reference/configuration_reference.md)**

### I'm having issues
→ Consult the **[Debugging Agent Issues](howto/debugging_agent_issues.md)** guide

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

1. **Install and setup**: Follow [Installation and Setup](tutorials/installation_and_setup.md)
2. **Understand the concepts**: Read [The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)
3. **See it in action**: Follow [Running the Demo](tutorials/run_demo.md)
4. **Build your first agent**: Use the [Create a Subscriber](howto/create_subscriber.md) guide
5. **Create task workflows**: Follow [Create a Publisher](howto/create_publisher.md)
6. **Build complex systems**: Try [Building Multi-Agent Workflows](tutorials/building_multi_agent_workflows.md)

## 📖 Learning Paths

### For Developers New to Agent Systems
1. **[Installation and Setup](tutorials/installation_and_setup.md)** - Get your environment ready
2. **[The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)** - Understand the why
3. **[Running the Demo](tutorials/run_demo.md)** - See it working
4. **[Understanding Tasks](explanation/the_tasks.md)** - Learn the core concepts
5. **[Create a Subscriber](howto/create_subscriber.md)** - Build your first agent
6. **[Create a Publisher](howto/create_publisher.md)** - Build task orchestrators
7. **[Dashboard Tour](tutorials/dashboard_tour.md)** - Learn monitoring capabilities

### For Experienced Distributed Systems Developers
1. **[Installation and Setup](tutorials/installation_and_setup.md)** - Quick setup
2. **[Running the Demo](tutorials/run_demo.md)** - Hands-on experience
3. **[API Reference](reference/api_reference.md)** - Technical specifications
4. **[Unified Abstraction API](reference/unified_abstraction_api.md)** - Advanced coordination patterns
5. **[Create a Publisher](howto/create_publisher.md)** - Build task orchestrators
6. **[Performance and Scaling](explanation/performance_and_scaling.md)** - Optimization strategies
7. **[Observability Demo](tutorials/observability_demo.md)** - Production monitoring

### For System Architects & Operations
1. **[The Agent2Agent Principle](explanation/the_agent_to_agent_principle.md)** - Design philosophy
2. **[Broker Architecture](explanation/broker_architecture.md)** - Internal design and trade-offs
3. **[Performance and Scaling](explanation/performance_and_scaling.md)** - Scaling patterns
4. **[Configuration Reference](reference/configuration_reference.md)** - Production deployment
5. **[Building Multi-Agent Workflows](tutorials/building_multi_agent_workflows.md)** - Complex system patterns
6. **[Add Observability](howto/add_observability.md)** - Production monitoring setup
7. **[Health Endpoints](reference/health_endpoints.md)** - Operational monitoring

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