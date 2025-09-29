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
SERVER_BINARY := eventbus-server
PUBLISHER_BINARY := publisher
SUBSCRIBER_BINARY := subscriber
CHAT_RESPONDER_BINARY := chat_responder
CHAT_REPL_BINARY := chat_repl

# Go compiler flags
GO_BUILD_FLAGS := -ldflags="-s -w" # Strip symbols and debug info for smaller binaries

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all proto build run-server run-publisher run-subscriber run-chat-responder run-chat-repl clean help

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
build: proto
	@echo "Building A2A-compliant server binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SERVER_BINARY) broker/main.go

	@echo "Building A2A-compliant publisher binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(PUBLISHER_BINARY) agents/publisher/main.go

	@echo "Building A2A-compliant subscriber binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SUBSCRIBER_BINARY) agents/subscriber/main.go

	@echo "Building A2A-compliant chat responder binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(CHAT_RESPONDER_BINARY) agents/chat_responder/main.go

	@echo "Building A2A-compliant chat REPL binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(CHAT_REPL_BINARY) agents/chat_repl/main.go

	@echo "Build complete. A2A-compliant binaries are in the 'bin/' directory."

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
	@echo "Makefile for gRPC Event Bus"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all                  Builds all binaries (default)."
	@echo "  proto                Generates Go code from .proto files."
	@echo "  build                Builds all binaries (server, agents)."
	@echo "  run-server           Runs the event bus gRPC server."
	@echo "  run-publisher        Runs the publisher client."
	@echo "  run-subscriber       Runs the subscriber client."
	@echo "  run-chat-responder   Runs the chat responder agent (requires Vertex AI config)."
	@echo "  run-chat-repl        Runs the chat REPL agent."
	@echo "  clean                Removes generated Go files and build artifacts."
	@echo "  help                 Displays this help message."
	@echo ""
	@echo "Configuration:"
	@echo "  MODULE_PATH      Your Go module path (e.g., github.com/user/repo)."
	@echo "                   Ensure this matches your go.mod file."
	@echo ""
	@echo "Environment Variables for Chat Responder:"
	@echo "  GCP_PROJECT      Your Google Cloud Project ID"
	@echo "  GCP_LOCATION     GCP region (default: us-central1)"
	@echo "  VERTEX_AI_MODEL  Model name (default: gemini-2.0-flash)"
