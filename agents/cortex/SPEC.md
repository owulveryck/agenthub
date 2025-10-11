# **SPEC.md: Asynchronous AI Orchestration Engine**

**Version:** 0.1.0
**Status:** Proposed

## 1. Introduction

This document outlines the technical specification for a new asynchronous, event-driven AI orchestration engine.

### 1.1. The Problem (The "Why")

The current AI assistant implementation is based on a synchronous, blocking request-response paradigm. When a user request requires a long-running tool (e.g., audio transcription, complex data analysis), the entire conversation is blocked until the tool completes its execution. This leads to a poor user experience, as the user cannot interact with the assistant while background tasks are running. The model is rigid and does not allow for parallel processing of thoughts or proactive notifications, capabilities that are natural for human interaction.

### 1.2. The Proposed Solution

We will re-architect the system around an asynchronous, event-driven paradigm using a central message bus. This new architecture decouples user interaction from task execution.

A central orchestrator, **The Cortex**, will analyze user requests and decompose them into tasks. These tasks will be dispatched as messages to a swarm of independent, specialized **Agents**. Agents process their tasks in the background and report results via messages. The Cortex then synthesizes these results and communicates back to the user when appropriate.

This design enables non-blocking conversations, allows the system to scale with new capabilities by simply adding new agents, and provides a foundation for more complex, proactive AI behaviors.

### 1.3. Scope of this POC

This initial specification is for a Proof of Concept (POC) to validate the core architecture. The scope is strictly limited to the following:

*   **Interaction Model:** User interaction will be handled via a command-line interface (CLI) agent, not a web-based UI.
*   **Core Components:** The focus is on the `Cortex`, the `Event Bus`, and the `Agent` interaction pattern.
*   **State Management:** Conversation state will be managed in-memory.
*   **Agent Liveness:** The system will assume agents are always available after they register. Agent health checks or heartbeats are out of scope.

## 2. Core Concepts & Glossary

*   **Event Bus:** The central message broker for the system. It is a pre-existing, custom component that exposes `publish(message)` and `subscribe(message_type)` functions. All communication between components happens via the Event Bus.
*   **Message:** A structured, stateless packet of data. The exact schema for messages is defined in an external document. All messages must contain sufficient metadata (e.g., `correlation_id`, `session_id`) to be processed statelessly.
*   **The Cortex:** The "brain" of the system. A long-running service that orchestrates tasks. It is the only component that holds conversational state (in-memory for this POC).
*   **Agent:** An independent, specialized microservice or function. It performs a specific task (e.g., transcription). Agents are stateless and communicate exclusively via the Event Bus.
*   **Agent Card:** A registration message (`agent.card`) published by an Agent on startup. It details the agent's capabilities, the message types it subscribes to, and the results it can produce.
*   **Requester CLI Agent:** A command-line tool that acts as the user interface for this POC. It allows a user to send chat requests and displays responses received from the Cortex.

## 3. System Architecture

### 3.1. High-Level Diagram

The flow of information follows this general pattern:

```
+---------------------+      +----------------+      +----------------+
| Requester CLI Agent |----->|   Event Bus    |<-----|     Cortex     |
+---------------------+      +----------------+      +----------------+
      ^      | chat.request         ^                        | task.request
      |      |                        |                        |
      |      +------------------------+------------------------+
      |                               |
      | chat.response                 | task.result
      |                               |
      |      +----------------+       |      +----------------+
      +------|   Event Bus    |<------+------| Generic Agent  |
             +----------------+              +----------------+
```

### 3.2. Component Responsibilities

#### 3.2.1. Requester CLI Agent
*   **Action:** On startup, takes user input from the command line.
*   **Publish:** Publishes user input as a `chat.request` message to the Event Bus.
*   **Subscribe:** Subscribes to `chat.response` messages that match its session.
*   **Action:** Prints the payload of received `chat.response` messages to the standard output.

#### 3.2.2. The Cortex
*   **Subscribe:** Subscribes to `agent.card`, `chat.request`, and all `*.result` message types.
*   **Core Loop:**
    1.  On receipt of a message (e.g., `chat.request` from the user or `transcription.result` from an agent), retrieve the relevant conversation history from the state manager.
    2.  Construct a prompt for an LLM. This prompt will include the conversation history, the newly received message, and a dynamically generated list of available tools/agents (from the `Agent Cards`).
    3.  Execute a call to the configured LLM to decide the next action(s).
    4.  Parse the LLM's response.
    5.  Dispatch zero or more new messages to the Event Bus (e.g., a `chat.response` to acknowledge the user, a `transcription.request` for an agent).
    6.  Update the conversation history in the state manager.
*   **Agent Management:** Maintains an in-memory list of available agents based on the `agent.card` messages it has received. This list is used to build the system prompt for the LLM.

#### 3.2.3. Generic Agent (e.g., `TranscriptionAgent`)
*   **Action:** On startup, publishes an `agent.card` message detailing its function (e.g., "transcribes audio to text"), the message type it listens to (`transcription.request`), and the message type it produces (`transcription.result`).
*   **Subscribe:** Subscribes to the request message type specified in its card.
*   **Action:** Upon receiving a request message, it executes its core logic (e.g., calls a transcription service).
*   **Publish:** Once processing is complete (or has failed), it publishes a result message (e.g., `transcription.result`) containing the outcome.

## 4. Key Workflows

### 4.1. System Startup & Agent Registration
1.  **Cortex** starts and subscribes to `agent.card` messages.
2.  **Agent A** starts.
3.  **Agent A** publishes an `agent.card` message to the Event Bus.
4.  **Cortex** receives the `agent.card` and updates its internal list of available tools, modifying the system prompt it will use for future LLM calls.

### 4.2. Asynchronous Task Execution
1.  **User** types "please transcribe this recording" into the `Requester CLI Agent` and provides an audio file path.
2.  **Requester CLI Agent** publishes a `chat.request` message with the user's text and audio data.
3.  **Cortex** receives the `chat.request`. It makes an LLM call to analyze the request.
4.  The LLM determines two actions are needed: acknowledge the user and start transcription.
5.  **Cortex** publishes two messages:
    *   A `chat.response` message with a payload like "Certainly, I'll start the transcription. I'll let you know when it's done."
    *   A `transcription.request` message with the audio data.
6.  **Requester CLI Agent** receives the `chat.response` and prints it to the console. The user can now type another message and continue the conversation.
7.  **TranscriptionAgent** receives the `transcription.request` and begins processing. This may take a long time.
8.  Once complete, **TranscriptionAgent** publishes a `transcription.result` message containing the transcribed text.
9.  **Cortex** receives the `transcription.result`. It makes another LLM call with the context: "The transcription you requested is now complete. The text is: '...'. What should you do?"
10. The LLM decides to inform the user.
11. **Cortex** publishes a `chat.response` message with the payload: "The transcription is complete. Here is the text: '...'".
12. **Requester CLI Agent** receives this final response and prints it to the console.

## 5. Technical Specifications

### 5.1. Cortex State Management
The Cortex must manage conversational state. This will be abstracted via a Go `interface`.

```go
// state.go
package cortex

// ConversationState represents the history of a single conversation.
type ConversationState struct {
    // Includes messages, context, etc.
}

// StateManager defines the interface for persisting conversation state.
type StateManager interface {
    Get(sessionID string) (*ConversationState, error)
    Set(sessionID string, state *ConversationState) error
}
```
For this POC, a default in-memory implementation (`NewInMemoryStateManager()`) will be provided.

### 5.2. Cortex Configuration
The Cortex must be configurable via environment variables or a configuration file. At a minimum, the following parameter must be exposed:
*   `CORTEX_LLM_MODEL`: The identifier for the LLM model to be used for orchestration and decision-making.

### 5.3. Message Structure
The message structure is considered an external dependency. All components must adhere to the schema defined in the **[Link to Message Schema Documentation]**.

### 5.4. Error Handling Protocol
Agents are responsible for their own internal error handling.
*   If an Agent fails to process a task, it MUST NOT crash.
*   Instead, it MUST publish a standard result message (e.g., `transcription.result`) with a `status` field set to `"failed"` and an `error` field containing a descriptive error message.
*   The **Cortex** is responsible for subscribing to these results and deciding how to handle the failure (e.g., informing the user, attempting a retry, or trying an alternative agent).

## 6. Out of Scope for POC / Future Work
*   **Persistent State Management:** Implementing `StateManager` interfaces for Redis, PostgreSQL, etc.
*   **Web UI & API Gateway:** A full-featured user interface with real-time updates via WebSockets or SSE.
*   **Agent Liveness:** A heartbeat or TTL mechanism to detect and de-register offline agents.
*   **Advanced Error Recovery:** Implementing retry logic or fallback mechanisms within the Cortex.
*   **Message Guarantees:** Investigation of "at-least-once" or "exactly-once" delivery if the custom event bus does not provide it.
*   **Security:** Authentication and authorization for agents and users.
