---
title: "Hexagonal Architecture & A2A Protocol Implementation"
linkTitle: "Hexagonal A2A Architecture"
description: "Understanding AgentHub's hexagonal architecture with A2A protocol, gRPC communication, and event-driven design"
weight: 50
---

# Hexagonal Architecture & A2A Protocol Implementation

This document explains how AgentHub implements hexagonal architecture principles with the Agent2Agent (A2A) protocol, gRPC communication, and event-driven design patterns.

## Overview

AgentHub follows hexagonal architecture (also known as Ports and Adapters) to achieve:
- **Domain isolation**: Core A2A protocol logic separated from infrastructure
- **Testability**: Clean interfaces enable comprehensive testing
- **Flexibility**: Multiple adapters for different communication protocols
- **Maintainability**: Clear separation of concerns and dependencies

## System Architecture

```mermaid
graph TB
    subgraph "AgentHub Ecosystem"
        subgraph "External Agents"
            A["Agent A<br/>(Chat REPL)"]
            B["Agent B<br/>(Chat Responder)"]
            C["Agent C<br/>(Custom Agent)"]
        end

        subgraph "AgentHub Broker"
            subgraph "Adapters (Infrastructure)"
                GRPC["gRPC Server<br/>Adapter"]
                HEALTH["Health Check<br/>Adapter"]
                METRICS["Metrics<br/>Adapter"]
                TRACING["Tracing Adapter<br/>(OTLP/Jaeger)"]
            end

            subgraph "Ports (Interfaces)"
                SP["AgentHub<br/>Service Port"]
                PP["Message<br/>Publisher Port"]
                EP["Event<br/>Subscriber Port"]
                OP["Observability<br/>Port"]
            end

            subgraph "Domain (Core Logic)"
                A2A["A2A Protocol<br/>Engine"]
                ROUTER["Event Router<br/>& Broker"]
                VALIDATOR["Message<br/>Validator"]
                CONTEXT["Context<br/>Manager"]
                TASK["Task<br/>Lifecycle"]
            end
        end

        subgraph "External Systems"
            OTLP["OTLP Collector<br/>& Jaeger"]
            STORE["Event Store<br/>(Memory)"]
        end
    end

    %% External agent connections
    A -->|"gRPC calls<br/>(PublishMessage,<br/>SubscribeToMessages)"| GRPC
    B -->|"gRPC calls"| GRPC
    C -->|"gRPC calls"| GRPC

    %% Adapter to Port connections
    GRPC -->|"implements"| SP
    HEALTH -->|"implements"| OP
    METRICS -->|"implements"| OP
    TRACING -->|"implements"| OP

    %% Port to Domain connections
    SP -->|"delegates to"| A2A
    PP -->|"delegates to"| ROUTER
    EP -->|"delegates to"| ROUTER
    OP -->|"observes"| A2A

    %% Domain internal connections
    A2A -->|"uses"| VALIDATOR
    A2A -->|"uses"| CONTEXT
    A2A -->|"uses"| TASK
    ROUTER -->|"persists events"| STORE
    TRACING -->|"exports traces"| OTLP

    %% Styling
    classDef agents fill:#add8e6
    classDef adapters fill:#ffa500
    classDef ports fill:#e0ffff
    classDef domain fill:#ffb6c1
    classDef external fill:#dda0dd

    class A,B,C agents
    class GRPC,HEALTH,METRICS,TRACING adapters
    class SP,PP,EP,OP ports
    class A2A,ROUTER,VALIDATOR,CONTEXT,TASK domain
    class OTLP,STORE external
```

**Architecture Notes:**
- **Domain Core**: Pure A2A protocol logic with message validation, event routing, context correlation, and task state management
- **Ports**: Clean, technology-agnostic interfaces providing testable contracts and dependency inversion
- **Adapters**: Infrastructure concerns including gRPC communication, observability exports, and protocol adaptations

## A2A Message Flow

```mermaid
sequenceDiagram
    participant REPL as Chat REPL<br/>Agent
    participant gRPC as gRPC<br/>Adapter
    participant A2A as A2A Protocol<br/>Engine
    participant Router as Event<br/>Router
    participant Responder as Chat Responder<br/>Agent

    rect rgb(240, 248, 255)
        Note over REPL, Router: A2A Message Publishing
        REPL->>+gRPC: PublishMessage(A2AMessage)
        gRPC->>+A2A: validateA2AMessage()
        A2A->>A2A: check MessageId, Role, Content
        A2A-->>-gRPC: validation result
        gRPC->>+Router: routeA2AEvent(messageEvent)
        Router->>Router: identify subscribers<br/>by agent_id/broadcast
        Router->>Router: create tracing span<br/>with A2A attributes
        Router-->>Responder: deliver message event
        Router-->>-gRPC: routing success
        gRPC-->>-REPL: PublishResponse(event_id)
    end

    rect rgb(255, 248, 240)
        Note over Responder, Router: A2A Message Processing
        Responder->>+gRPC: SubscribeToMessages(agent_id)
        gRPC->>Router: register subscriber
        Router-->>gRPC: subscription stream
        gRPC-->>-Responder: message stream
        Note over Responder: Process A2A message<br/>with tracing spans
        Responder->>+gRPC: PublishMessage(A2AResponse)
        gRPC->>A2A: validateA2AMessage()
        A2A->>A2A: check AGENT role,<br/>ContextId correlation
        gRPC->>Router: routeA2AEvent(responseEvent)
        Router-->>REPL: deliver response event
        gRPC-->>-Responder: PublishResponse
    end

    Note over REPL, Responder: A2A Protocol ensures:<br/>• Message structure compliance<br/>• Role semantics (USER/AGENT)<br/>• Context correlation<br/>• Event-driven routing
```

## Core Components

### 1. A2A Protocol Engine (Domain Core)

The heart of the system implementing A2A protocol specifications:

```go
// Core domain logic - technology agnostic
type A2AProtocolEngine struct {
    messageValidator MessageValidator
    contextManager   ContextManager
    taskLifecycle    TaskLifecycle
}

// A2A message validation
func (e *A2AProtocolEngine) ValidateMessage(msg *Message) error {
    // A2A compliance checks
    if msg.MessageId == "" { return ErrMissingMessageId }
    if msg.Role == ROLE_UNSPECIFIED { return ErrInvalidRole }
    if len(msg.Content) == 0 { return ErrEmptyContent }
    return nil
}
```

### 2. Event Router (Domain Core)

Manages event-driven communication between agents:

```go
type EventRouter struct {
    messageSubscribers map[string][]chan *AgentEvent
    taskSubscribers    map[string][]chan *AgentEvent
    eventSubscribers   map[string][]chan *AgentEvent
}

func (r *EventRouter) RouteEvent(event *AgentEvent) error {
    // Route based on A2A metadata
    routing := event.GetRouting()
    subscribers := r.getSubscribers(routing.ToAgentId, event.PayloadType)

    // Deliver with tracing
    for _, sub := range subscribers {
        go r.deliverWithTracing(sub, event)
    }
}
```

### 3. gRPC Adapter (Infrastructure)

Translates between gRPC and domain logic:

```go
type GrpcAdapter struct {
    a2aEngine    A2AProtocolEngine
    eventRouter  EventRouter
    tracer       TracingAdapter
}

func (a *GrpcAdapter) PublishMessage(ctx context.Context, req *PublishMessageRequest) (*PublishResponse, error) {
    // Start tracing span
    ctx, span := a.tracer.StartA2AMessageSpan(ctx, "publish_message", req.Message.MessageId, req.Message.Role)
    defer span.End()

    // Validate using domain logic
    if err := a.a2aEngine.ValidateMessage(req.Message); err != nil {
        a.tracer.RecordError(span, err)
        return nil, err
    }

    // Route using domain logic
    event := a.createA2AEvent(req)
    if err := a.eventRouter.RouteEvent(event); err != nil {
        return nil, err
    }

    return &PublishResponse{Success: true, EventId: event.EventId}, nil
}
```

## Hexagonal Architecture Benefits

### 1. Domain Isolation
- **A2A protocol logic** is pure, testable business logic
- **No infrastructure dependencies** in the core domain
- **Technology-agnostic** implementation

### 2. Adapter Pattern
- **gRPC Adapter**: Handles Protocol Buffer serialization/deserialization
- **Tracing Adapter**: OTLP/Jaeger integration without domain coupling
- **Health Adapter**: Service health monitoring
- **Metrics Adapter**: Prometheus metrics collection

### 3. Port Interfaces
```go
// Clean, testable interfaces
type MessagePublisher interface {
    PublishMessage(ctx context.Context, msg *Message) (*PublishResponse, error)
}

type EventSubscriber interface {
    SubscribeToMessages(ctx context.Context, agentId string) (MessageStream, error)
}

type ObservabilityPort interface {
    StartSpan(ctx context.Context, operation string) (context.Context, Span)
    RecordMetric(name string, value float64, labels map[string]string)
}
```

### 4. Dependency Inversion
- **Domain depends on abstractions** (ports), not concrete implementations
- **Adapters depend on domain** through well-defined interfaces
- **Easy testing** with mock implementations

## A2A Protocol Integration

### Message Structure Compliance

```mermaid
classDiagram
    class A2AMessage {
        +string MessageId
        +string ContextId
        +Role Role
        +Part Content
        +Metadata Metadata
        +string TaskId
    }

    class Part {
        +string Text
        +bytes Data
        +FileData File
    }

    class EventMetadata {
        +string FromAgentId
        +string ToAgentId
        +string EventType
        +Priority Priority
    }

    class Role {
        <<enumeration>>
        USER
        AGENT
    }

    class Metadata {
        +Fields map
    }

    A2AMessage "1" --> "0..*" Part : contains
    A2AMessage "1" --> "1" EventMetadata : routed_with
    A2AMessage "1" --> "1" Role : has
    A2AMessage "1" --> "0..1" Metadata : includes
```

### Event-Driven Architecture
The system implements pure event-driven architecture:

1. **Publishers** emit A2A-compliant events
2. **Broker** routes events based on metadata
3. **Subscribers** receive relevant events
4. **Correlation** through ContextId maintains conversation flow

## Observability Integration

### Distributed Tracing

```mermaid
sequenceDiagram
    participant A as Agent A
    participant B as Broker
    participant AB as Agent B
    participant OTLP as OTLP Collector
    participant J as Jaeger

    A->>+B: PublishMessage<br/>[trace_id: 123]
    B->>B: Create A2A spans<br/>with structured attributes
    B->>+AB: RouteEvent<br/>[trace_id: 123]
    AB->>AB: Process with<br/>child spans
    AB->>-B: PublishResponse<br/>[trace_id: 123]
    B->>-A: Success<br/>[trace_id: 123]

    par Observability Export
        B->>OTLP: Export spans<br/>with A2A attributes
        OTLP->>J: Store traces
        J->>J: Build trace timeline<br/>with correlation
    end

    Note over A, J: End-to-end tracing<br/>with A2A protocol visibility
```

### Structured Attributes
Each span includes A2A-specific attributes:
- `a2a.message.id`
- `a2a.message.role`
- `a2a.context.id`
- `a2a.event.type`
- `a2a.routing.from_agent`
- `a2a.routing.to_agent`

## Testing Strategy

### Unit Testing (Domain Core)
```go
func TestA2AEngine_ValidateMessage(t *testing.T) {
    engine := NewA2AProtocolEngine()

    // Test A2A compliance
    msg := &Message{
        MessageId: "test_msg_123",
        Role: ROLE_USER,
        Content: []*Part{{Text: "hello"}},
    }

    err := engine.ValidateMessage(msg)
    assert.NoError(t, err)
}
```

### Integration Testing (Adapters)
```go
func TestGrpcAdapter_PublishMessage(t *testing.T) {
    // Mock domain dependencies
    mockEngine := &MockA2AEngine{}
    mockRouter := &MockEventRouter{}

    adapter := NewGrpcAdapter(mockEngine, mockRouter)

    // Test adapter behavior
    resp, err := adapter.PublishMessage(ctx, validRequest)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
}
```

## Conclusion

AgentHub's hexagonal architecture with A2A protocol provides:

1. **Clean Architecture**: Separation of concerns with domain-driven design
2. **A2A Compliance**: Full protocol implementation with validation
3. **Event-Driven Design**: Scalable, loosely-coupled communication
4. **Rich Observability**: Comprehensive tracing and metrics
5. **Testability**: Clean interfaces enable thorough testing
6. **Flexibility**: Easy to extend with new adapters and protocols

This architecture ensures maintainable, scalable, and observable agent communication while maintaining strict A2A protocol compliance.