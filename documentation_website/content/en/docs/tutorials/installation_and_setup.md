---
title: "Installation and Setup Tutorial"
weight: 20
description: "Guide for installing AgentHub and setting up your development environment from scratch. Get a working AgentHub installation ready for building agent systems."
---

# Installation and Setup Tutorial

This tutorial will guide you through installing AgentHub and setting up your development environment from scratch. By the end, you'll have a working AgentHub installation ready for building agent systems.

## Prerequisites Check

Before we begin, let's verify you have the required software installed.

### Step 1: Verify Go Installation

Check if Go 1.24+ is installed:

```bash
go version
```

You should see output like:
```
go version go1.24.0 darwin/amd64
```

If Go is not installed or the version is older than 1.24:

**macOS (using Homebrew):**
```bash
brew install go
```

**Linux (using package manager):**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install golang-go

# CentOS/RHEL
sudo yum install golang

# Arch Linux
sudo pacman -S go
```

**Windows:**
Download from [https://golang.org/dl/](https://golang.org/dl/) and run the installer.

### Step 2: Verify Protocol Buffers Compiler

Check if `protoc` is installed:

```bash
protoc --version
```

You should see output like:
```
libprotoc 3.21.12
```

If `protoc` is not installed:

**macOS (using Homebrew):**
```bash
brew install protobuf
```

**Linux:**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install protobuf-compiler

# CentOS/RHEL
sudo yum install protobuf-compiler

# Arch Linux
sudo pacman -S protobuf
```

**Windows:**
Download from [Protocol Buffers releases](https://github.com/protocolbuffers/protobuf/releases) and add to PATH.

### Step 3: Install Go Protocol Buffer Plugins

Install the required Go plugins for Protocol Buffers:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Verify the plugins are in your PATH:

```bash
which protoc-gen-go
which protoc-gen-go-grpc
```

Both commands should return paths to the installed plugins.

## Installing AgentHub

### Step 4: Clone the Repository

Clone the AgentHub repository:

```bash
git clone https://github.com/owulveryck/agenthub.git
cd agenthub
```

### Step 5: Verify Project Structure

Let's explore what we have:

```bash
ls -la
```

You should see:
```
drwxr-xr-x agents/           # Sample agent implementations
drwxr-xr-x broker/           # AgentHub broker server
drwxr-xr-x documentation/    # Complete documentation
drwxr-xr-x internal/         # Generated code
-rw-r--r-- go.mod            # Go module definition
-rw-r--r-- Makefile         # Build automation
drwxr-xr-x proto/           # Protocol definitions
-rw-r--r-- README.md        # Project overview
```

### Step 6: Initialize Go Module

Ensure Go modules are properly initialized:

```bash
go mod tidy
```

This downloads all required dependencies. You should see output about downloading packages.

### Step 7: Generate Protocol Buffer Code

Generate the Go code from Protocol Buffer definitions:

```bash
make proto
```

You should see:
```
Generating protobuf code for proto/eventbus.proto...
Protobuf code generated successfully.
```

Verify the generated files exist:

```bash
ls internal/grpc/
```

You should see:
```
eventbus.pb.go
eventbus_grpc.pb.go
```

### Step 8: Build All Components

Build the AgentHub components:

```bash
make build
```

You should see:
```
Building server binary...
Building publisher binary...
Building subscriber binary...
Build complete. Binaries are in the 'bin/' directory.
```

Verify the binaries were created:

```bash
ls bin/
```

You should see:
```
eventbus-server
publisher
subscriber
```

## Verification Test

Let's verify everything works by running a quick test.

### Step 9: Test the Installation

Start the broker server in the background:

```bash
./bin/eventbus-server &
```

You should see:
```
2025/09/28 10:00:00 AgentHub broker gRPC server listening on [::]:50051
```

Start a subscriber agent:

```bash
./bin/subscriber &
```

You should see:
```
Agent started. Listening for events and tasks. Press Enter to stop.
2025/09/28 10:00:05 Agent agent_demo_subscriber subscribing to tasks...
2025/09/28 10:00:05 Successfully subscribed to tasks for agent agent_demo_subscriber. Waiting for tasks...
```

Run the publisher to send test tasks:

```bash
./bin/publisher
```

You should see tasks being published and processed.

Clean up the test processes:

```bash
pkill -f eventbus-server
pkill -f subscriber
```

## Development Environment Setup

### Step 10: Configure Your Editor

**For VS Code users:**

Install the Go extension:
1. Open VS Code
2. Go to Extensions (Ctrl+Shift+X)
3. Search for "Go" and install the official Go extension
4. Open the AgentHub project folder

**For other editors:**

Ensure your editor has Go language support and Protocol Buffer syntax highlighting.

### Step 11: Set Up Environment Variables (Optional)

Create a `.env` file for local development:

```bash
cat > .env << EOF
# AgentHub Configuration
AGENTHUB_PORT=50051
AGENTHUB_LOG_LEVEL=info

# Development Settings
GO_ENV=development
EOF
```

### Step 12: Verify Make Targets

Test all available make targets:

```bash
make help
```

You should see all available commands:
```
Makefile for gRPC Event Bus

Usage:
  make <target>

Targets:
  all              Builds all binaries (default).
  proto            Generates Go code from .proto files.
  build            Builds the server, publisher, and subscriber binaries.
  run-server       Runs the event bus gRPC server.
  run-publisher    Runs the publisher client.
  run-subscriber   Runs the subscriber client.
  clean            Removes generated Go files and build artifacts.
  help             Displays this help message.
```

## Common Issues and Solutions

### Issue: "protoc-gen-go: program not found"

**Solution:** Ensure Go bin directory is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

### Issue: "go.mod not found"

**Solution:** Ensure you're in the AgentHub project directory:

```bash
pwd  # Should show .../agenthub
ls go.mod  # Should exist
```

### Issue: Port 50051 already in use

**Solution:** Kill existing processes or change the port:

```bash
lsof -ti:50051 | xargs kill -9
```

### Issue: Permission denied on binaries

**Solution:** Make binaries executable:

```bash
chmod +x bin/*
```

## Next Steps

Now that you have AgentHub installed and verified:

1. **Learn the basics**: Follow the [Running the Demo](run_demo.md) tutorial
2. **Build your first agent**: Try [Create a Subscriber](../howto/create_subscriber.md)
3. **Understand the concepts**: Read [The Agent2Agent Principle](../explanation/the_agent_to_agent_principle.md)

## Getting Help

If you encounter issues:

1. Check the [troubleshooting section](#common-issues-and-solutions) above
2. Review the [complete documentation](../README.md)
3. Open an issue on the [GitHub repository](https://github.com/owulveryck/agenthub/issues)

Congratulations! You now have a fully functional AgentHub development environment ready for building autonomous agent systems.