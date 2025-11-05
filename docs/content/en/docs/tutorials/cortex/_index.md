---
title: "Cortex Tutorials"
linkTitle: "Cortex"
weight: 40
description: >
  Learn how to use Cortex, the asynchronous AI orchestration engine
---

# Cortex Tutorials

These hands-on tutorials will teach you how to use Cortex to build asynchronous, AI-powered multi-agent systems.

## What is Cortex?

Cortex is an asynchronous orchestration engine that:
- Manages conversations across multiple agents
- Uses LLMs to make intelligent routing decisions
- Enables non-blocking task execution
- Maintains conversation state and context

## Prerequisites

- AgentHub installed and configured
- Go 1.21 or later
- Basic understanding of the A2A protocol

## Available Tutorials

1. **[Getting Started with Cortex](getting-started/)** - Run your first Cortex demo
2. **[Building a Custom Agent](building-custom-agent/)** - Create agents that work with Cortex
3. **[Async Task Orchestration](async-orchestration/)** - Handle long-running tasks

## Quick Start

Run the Cortex demo to see it in action:

```bash
cd /path/to/agenthub
./demo_cortex.sh
```

This starts:
- Event Bus (broker)
- Cortex orchestrator
- Echo agent (example)
- Interactive CLI

Type messages and see how Cortex orchestrates responses!

## Learning Path

1. Start with [Getting Started](getting-started/) to understand the basics
2. Read [Cortex Architecture](../../explanation/architecture/cortex_architecture/) for deeper understanding
3. Try [Building a Custom Agent](building-custom-agent/) to extend functionality
4. Explore [Advanced Orchestration](async-orchestration/) for complex workflows
