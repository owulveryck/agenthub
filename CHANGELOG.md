# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.2] - 2025-09-29

### 🚀 Major Features

- **Agent2Agent (A2A) Protocol Compliance**: Complete implementation of A2A protocol v0.2.9 for standardized agent communication
  - Full A2A message structures (`Message`, `Part`, `Task`, `Artifact`)
  - A2A task lifecycle management with proper states (SUBMITTED, WORKING, COMPLETED, FAILED, CANCELLED)
  - A2A conversation context and message threading support
  - A2A agent discovery with `AgentCard` implementation

### 🏗️ Architecture Changes

- **Hybrid EDA+A2A Architecture**: Combines Agent2Agent protocol compliance with Event-Driven Architecture scalability
  - AgentHub service replaces legacy EventBus for A2A compliance
  - EDA event wrapping maintains scalability and resilience benefits
  - Backward compatibility through automatic conversion between legacy and A2A formats

### 📚 Documentation

- **Complete Documentation Overhaul**: All documentation updated for A2A compliance
  - API reference completely rewritten for AgentHub service
  - Tutorial documentation updated with A2A examples and build processes
  - Reference documentation updated for A2A metrics, configuration, and task definitions
  - Explanation documentation updated for A2A concepts and migration guide
  - Documentation structure reorganized with improved organization and subchapters

### 🔧 Developer Experience

- **A2A Client Abstractions**: High-level abstractions for easier A2A integration
  - `A2ATaskPublisher` for simplified task publishing
  - `A2ATaskSubscriber` for streamlined task processing
  - Built-in observability with OpenTelemetry integration

### 📋 Protocol Buffers

- **Updated Protobuf Definitions**:
  - `proto/a2a_core.proto` for core A2A types
  - `proto/eventbus.proto` for AgentHub service definition
  - Automatic protobuf generation through Makefile

### ⚡ Performance & Observability

- **Enhanced Observability**: Full OpenTelemetry integration with A2A task tracing
- **Event-Driven Scalability**: Maintains EDA patterns for high-throughput scenarios
- **Structured Logging**: Comprehensive logging with A2A context information

### 🛠️ Breaking Changes

- **API Migration**: Legacy `EventBus` service replaced with `AgentHub` service
- **Message Format**: Custom message types replaced with A2A-compliant structures
- **Task Handling**: Legacy task handlers need migration to A2A format

### 📦 Build & Deployment

- **Improved Build Process**: Enhanced Makefile with A2A protobuf generation
- **GitHub Actions**: Updated CI/CD pipeline for A2A compliance testing

### 📄 Legal

- **License Added**: MIT License added to the project

---

## [Unreleased]

### Changed
- Initial release preparation

---

**Full Changelog**: https://github.com/owulveryck/agenthub/commits/v0.0.2