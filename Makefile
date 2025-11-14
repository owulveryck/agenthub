# ==============================================================================
# Configuration
# ==============================================================================

# Define your Go module path (must match the one used in go.mod)
MODULE_PATH := github.com/owulveryck/agenthub

# Protobuf compiler and plugins
PROTOC := protoc
GO_PROTOC_GEN := $(shell go env GOPATH)/bin/protoc-gen-go
GO_GRPC_PROTOC_GEN := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

# Source proto files
PROTO_SRC := proto/eventbus.proto proto/events.proto proto/a2a_core.proto

# Target directories for generated Go code
A2A_OUT_DIR := events/a2a
OBSERVABILITY_OUT_DIR := internal/events/observability

# Go build output names
SERVER_BINARY := broker
PUBLISHER_BINARY := publisher
SUBSCRIBER_BINARY := subscriber
CHAT_RESPONDER_BINARY := chat_responder
CHAT_REPL_BINARY := chat_repl
CHAT_CLI_BINARY := chat_cli
ECHO_AGENT_BINARY := echo_agent
CORTEX_BINARY := cortex

# Go compiler flags
GO_BUILD_FLAGS := -ldflags="-s -w" # Strip symbols and debug info for smaller binaries

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all proto build build-broker build-agents run-server run-publisher run-subscriber run-chat-responder run-chat-repl run-chat-cli run-echo-agent run-cortex clean help

all: build

# Target to generate protobuf Go code
proto: $(A2A_OUT_DIR)/eventbus.pb.go $(A2A_OUT_DIR)/eventbus_grpc.pb.go $(A2A_OUT_DIR)/a2a_core.pb.go $(OBSERVABILITY_OUT_DIR)/events.pb.go

# Rule to generate A2A protocol files
$(A2A_OUT_DIR)/eventbus.pb.go $(A2A_OUT_DIR)/eventbus_grpc.pb.go $(A2A_OUT_DIR)/a2a_core.pb.go: proto/eventbus.proto proto/a2a_core.proto
	@echo "Generating a2a protobuf code..."
	@mkdir -p $(A2A_OUT_DIR) # Ensure output directory exists
	$(PROTOC) --go_out=. --go-grpc_out=. proto/eventbus.proto proto/a2a_core.proto
	@echo "A2A protobuf code generated successfully."

$(OBSERVABILITY_OUT_DIR)/events.pb.go: proto/events.proto
	@echo "Generating observability protobuf code..."
	@mkdir -p $(OBSERVABILITY_OUT_DIR) # Ensure output directory exists
	$(PROTOC) --go_out=. proto/events.proto
	@echo "Observability protobuf code generated successfully."

# Rule to generate .pb.gw.go files (if you add gRPC-Gateway later)
# $(GO_OUT_DIR)/%.pb.gw.go: $(PROTO_SRC)
# 	@echo "Generating gRPC-Gateway code for $<..."
# 	@mkdir -p $(dir $@)
# 	$(PROTOC) --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
# 	          -I $(dir $(PROTO_SRC)) \
# 	          $<
# 	@echo "gRPC-Gateway code generated successfully."


# Target to build all binaries
build: build-broker build-agents
	@echo "Build complete. All binaries are in the 'bin/' directory."

# Target to build broker
build-broker: proto
	@echo "Building broker binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SERVER_BINARY) broker/main.go
	@echo "✓ Broker built: bin/$(SERVER_BINARY)"

# Target to build all agents
build-agents: proto
	@echo "Building all agent binaries..."

	@echo "  Building publisher..."
	go build $(GO_BUILD_FLAGS) -o bin/$(PUBLISHER_BINARY) agents/publisher/main.go
	@echo "  ✓ Publisher built: bin/$(PUBLISHER_BINARY)"

	@echo "  Building subscriber..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SUBSCRIBER_BINARY) agents/subscriber/main.go
	@echo "  ✓ Subscriber built: bin/$(SUBSCRIBER_BINARY)"

	@echo "  Building chat_responder..."
	go build $(GO_BUILD_FLAGS) -o bin/$(CHAT_RESPONDER_BINARY) agents/chat_responder/main.go
	@echo "  ✓ Chat responder built: bin/$(CHAT_RESPONDER_BINARY)"

	@echo "  Building chat_repl..."
	go build $(GO_BUILD_FLAGS) -o bin/$(CHAT_REPL_BINARY) agents/chat_repl/main.go
	@echo "  ✓ Chat REPL built: bin/$(CHAT_REPL_BINARY)"

	@echo "  Building chat_cli..."
	go build $(GO_BUILD_FLAGS) -o bin/$(CHAT_CLI_BINARY) agents/chat_cli/main.go
	@echo "  ✓ Chat CLI built: bin/$(CHAT_CLI_BINARY)"

	@echo "  Building echo_agent..."
	go build $(GO_BUILD_FLAGS) -o bin/$(ECHO_AGENT_BINARY) agents/echo_agent/main.go
	@echo "  ✓ Echo agent built: bin/$(ECHO_AGENT_BINARY)"

	@echo "  Building cortex..."
	go build $(GO_BUILD_FLAGS) -o bin/$(CORTEX_BINARY) agents/cortex/cmd/main.go
	@echo "  ✓ Cortex built: bin/$(CORTEX_BINARY)"

	@echo "All agents built successfully."

# Target to run the event bus server
run-server:
	@echo "Starting Event Bus gRPC Server..."
	go run broker/main.go

# Target to run the publisher client
run-publisher:
	@echo "Starting Publisher Client..."
	go run agents/publisher/main.go

# Target to run the subscriber client
run-subscriber:
	@echo "Starting Subscriber Client..."
	go run agents/subscriber/main.go

# Target to run the chat responder agent
run-chat-responder:
	@echo "Starting Chat Responder Agent..."
	go run agents/chat_responder/main.go

# Target to run the chat REPL agent
run-chat-repl:
	@echo "Starting Chat REPL Agent..."
	go run agents/chat_repl/main.go

# Target to run the chat CLI agent
run-chat-cli:
	@echo "Starting Chat CLI Agent..."
	go run agents/chat_cli/main.go

# Target to run the echo agent
run-echo-agent:
	@echo "Starting Echo Agent..."
	go run agents/echo_agent/main.go

# Target to run the cortex orchestrator
run-cortex:
	@echo "Starting Cortex Orchestrator..."
	go run agents/cortex/cmd/main.go

# Target to clean up generated files and binaries
clean:
	@echo "Cleaning up generated files and binaries..."
	rm -rf $(A2A_OUT_DIR)/*.pb.go
	rm -rf $(OBSERVABILITY_OUT_DIR)/*.pb.go
	rm -rf internal/grpc/*.pb.go
	rm -rf bin/
	@echo "Clean complete."

# Target to display help
help:
	@echo "Makefile for AgentHub - A2A Agent Orchestration Platform"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Build Targets:"
	@echo "  all                  Builds all binaries (default)."
	@echo "  proto                Generates Go code from .proto files."
	@echo "  build                Builds all binaries (broker + all agents)."
	@echo "  build-broker         Builds only the broker binary."
	@echo "  build-agents         Builds all agent binaries."
	@echo ""
	@echo "Run Targets:"
	@echo "  run-server           Runs the event bus broker."
	@echo "  run-publisher        Runs the publisher agent."
	@echo "  run-subscriber       Runs the subscriber agent."
	@echo "  run-chat-responder   Runs the chat responder agent (requires Vertex AI)."
	@echo "  run-chat-repl        Runs the chat REPL agent."
	@echo "  run-chat-cli         Runs the chat CLI agent."
	@echo "  run-echo-agent       Runs the echo agent."
	@echo "  run-cortex           Runs the Cortex orchestrator (uses VertexAI if configured)."
	@echo ""
	@echo "Utility Targets:"
	@echo "  clean                Removes generated Go files and build artifacts."
	@echo "  help                 Displays this help message."
	@echo ""
	@echo "Configuration:"
	@echo "  MODULE_PATH      Your Go module path (matches go.mod)."
	@echo ""
	@echo "Environment Variables for AI-powered Agents:"
	@echo "  GCP_PROJECT      Your Google Cloud Project ID"
	@echo "  GCP_LOCATION     GCP region (default: us-central1)"
	@echo "  VERTEX_AI_MODEL  Model name (default: gemini-2.0-flash)"
	@echo ""
	@echo "  Note: chat_responder and cortex use VertexAI when GCP_PROJECT is set,"
	@echo "        otherwise they fall back to mock/echo behavior."
