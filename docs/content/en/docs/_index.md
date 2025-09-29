---
title: "Documentation"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

# AgentHub Documentation

Welcome to the AgentHub documentation! This comprehensive guide will help you understand, install, and use the Agent2Agent (A2A) protocol-compliant communication broker to build sophisticated multi-agent systems with Event-Driven Architecture scalability.

## 🚀 Quick Start

### New to AgentHub?
Start with our [Installation and Setup](tutorials/installation_and_setup/) tutorial, then follow the [Running the Demo](tutorials/run_demo/) guide to see AgentHub in action.

### Want to understand the concepts?
Read about [The Agent2Agent Principle](explanation/the_agent_to_agent_principle/) to understand the philosophy behind AgentHub.

### Ready to build agents?
Use our guides to [Create a Publisher](howto/create_publisher/) or [Create a Subscriber](howto/create_subscriber/).

### Need technical details?
Check the [API Reference](reference/api_reference/) and [Configuration Reference](reference/configuration_reference/).

## 📚 Documentation Types

Our documentation follows the [Diátaxis framework](https://diataxis.fr/) with four distinct types:

### [Tutorials](tutorials/) - Learning by doing
Step-by-step guides that teach you how to use AgentHub through practical exercises:
- [Installation and Setup](tutorials/installation_and_setup/)
- [Running the Demo](tutorials/run_demo/)
- [Building Multi-Agent Workflows](tutorials/building_multi_agent_workflows/)
- [Dashboard Tour](tutorials/dashboard_tour/)
- [Observability Demo](tutorials/observability_demo/)

### [How-to Guides](howto/) - Solving specific problems
Goal-oriented guides for accomplishing specific tasks:
- [Create a Publisher](howto/create_publisher/)
- [Create a Subscriber](howto/create_subscriber/)
- [Debugging Agent Issues](howto/debugging_agent_issues/)
- [Add Observability](howto/add_observability/)
- [Use Dashboards](howto/use_dashboards/)

### [Reference](reference/) - Technical specifications
Comprehensive technical documentation and API details:
- [API Reference](reference/api_reference/)
- [Configuration Reference](reference/configuration_reference/)
- [Task Reference](reference/the_tasks/)
- [Unified Abstraction API](reference/unified_abstraction_api/)
- [Observability Metrics](reference/observability_metrics/)
- [Health Endpoints](reference/health_endpoints/)
- [Tracing API](reference/tracing_api/)

### [Explanation](explanation/) - Understanding concepts
In-depth discussions that provide context and background:
- [The Agent2Agent Principle](explanation/the_agent_to_agent_principle/)
- [A2A Protocol Migration](explanation/a2a_migration/) - **New: Understanding the A2A compliance migration**
- [Understanding Tasks](explanation/the_tasks/)
- [Broker Architecture](explanation/broker_architecture/)
- [Performance and Scaling](explanation/performance_and_scaling/)
- [Unified Abstraction Library](explanation/unified_abstraction_library/)
- [Distributed Tracing](explanation/distributed_tracing/)
- [Go Build Tags](explanation/go_build_tags/)

## 🎯 Learning Paths

### For Beginners
1. [Installation and Setup](tutorials/installation_and_setup/)
2. [The Agent2Agent Principle](explanation/the_agent_to_agent_principle/)
3. [Running the Demo](tutorials/run_demo/)
4. [Understanding Tasks](explanation/the_tasks/)
5. [Create a Subscriber](howto/create_subscriber/)

### For Experienced Developers
1. [Running the Demo](tutorials/run_demo/)
2. [API Reference](reference/api_reference/)
3. [Create a Publisher](howto/create_publisher/)
4. [Performance and Scaling](explanation/performance_and_scaling/)

### For System Architects
1. [The Agent2Agent Principle](explanation/the_agent_to_agent_principle/)
2. [Broker Architecture](explanation/broker_architecture/)
3. [Performance and Scaling](explanation/performance_and_scaling/)
4. [Configuration Reference](reference/configuration_reference/)

## 🔧 System Overview

AgentHub enables autonomous agents to collaborate through A2A protocol-compliant task delegation with EDA scalability:

- **A2A Protocol Compliance** with standardized Message, Task, and Artifact formats
- **Event-Driven Architecture** for scalable asynchronous communication
- **Flexible agent addressing** (direct, broadcast, topic-based routing)
- **Rich task semantics** with A2A lifecycle states and priorities
- **Built-in resilience** with EDA patterns and graceful failure handling
- **Comprehensive observability** with distributed tracing and metrics

## 🛠️ Key Technologies

- **Protocol**: gRPC with Protocol Buffers
- **Language**: Go 1.24+
- **Architecture**: Event-driven with pub/sub patterns
- **Observability**: OpenTelemetry integration
- **Message Format**: Structured protobuf with flexible JSON parameters