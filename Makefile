# ==============================================================================
# Configuration
# ==============================================================================

# Define your Go module path (must match the one used in go.mod)
MODULE_PATH := github.com/owulveryck/gomcptest/broker # <<<--- CHANGE THIS

# Protobuf compiler and plugins
PROTOC := protoc
GO_PROTOC_GEN := $(shell go env GOPATH)/bin/protoc-gen-go
GO_GRPC_PROTOC_GEN := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

# Source proto file
PROTO_SRC := proto/eventbus.proto

# Target directory for generated Go code
GO_OUT_DIR := internal/grpc

# Go build output names
SERVER_BINARY := eventbus-server
PUBLISHER_BINARY := publisher
SUBSCRIBER_BINARY := subscriber

# Go compiler flags
GO_BUILD_FLAGS := -ldflags="-s -w" # Strip symbols and debug info for smaller binaries

# ==============================================================================
# Targets
# ==============================================================================

.PHONY: all proto build run-server run-publisher run-subscriber clean help

all: build

# Target to generate protobuf Go code
proto: $(GO_OUT_DIR)/eventbus.pb.go $(GO_OUT_DIR)/eventbus_grpc.pb.go

# Rule to generate .pb.go files
$(GO_OUT_DIR)/%.pb.go: $(PROTO_SRC)
	@echo "Generating protobuf code for $<..."
	@mkdir -p $(GO_OUT_DIR) # Ensure output directory exists
	$(PROTOC) --go_out=. --go-grpc_out=. $(PROTO_SRC)
	@echo "Protobuf code generated successfully."

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
	@echo "Building server binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SERVER_BINARY) cmd/eventbus_server/main.go

	@echo "Building publisher binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(PUBLISHER_BINARY) cmd/publisher/main.go

	@echo "Building subscriber binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SUBSCRIBER_BINARY) cmd/subscriber/main.go

	@echo "Build complete. Binaries are in the 'bin/' directory."

# Target to run the event bus server
run-server:
	@echo "Starting Event Bus gRPC Server..."
	go run cmd/eventbus_server/main.go

# Target to run the publisher client
run-publisher:
	@echo "Starting Publisher Client..."
	go run cmd/publisher/main.go

# Target to run the subscriber client
run-subscriber:
	@echo "Starting Subscriber Client..."
	go run cmd/subscriber/main.go

# Target to clean up generated files and binaries
clean:
	@echo "Cleaning up generated files and binaries..."
	rm -rf $(GO_OUT_DIR)/*.pb.go
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
	@echo "  all              Builds all binaries (default)."
	@echo "  proto            Generates Go code from .proto files."
	@echo "  build            Builds the server, publisher, and subscriber binaries."
	@echo "  run-server       Runs the event bus gRPC server."
	@echo "  run-publisher    Runs the publisher client."
	@echo "  run-subscriber   Runs the subscriber client."
	@echo "  clean            Removes generated Go files and build artifacts."
	@echo "  help             Displays this help message."
	@echo ""
	@echo "Configuration:"
	@echo "  MODULE_PATH      Your Go module path (e.g., github.com/user/repo)."
	@echo "                   Ensure this matches your go.mod file."
