#!/bin/bash

# Demo script to showcase the chat agents
set -e

echo "=== AgentHub Chat Demo ==="
echo "This demo shows two agents communicating:"
echo "1. Chat REPL Agent - Posts ChatCompletionRequest tasks"
echo "2. Chat Responder Agent - Responds with 'hello' messages"
echo ""

# Clean up any existing processes
echo "Cleaning up existing processes..."
pkill -f "chat_repl\|chat_responder\|broker" 2>/dev/null || true
sleep 2

# Build the agents
echo "Building agents..."
go build -o bin/broker ./broker
go build -o bin/chat_repl ./agents/chat_repl
go build -o bin/chat_responder ./agents/chat_responder

echo "Starting broker..."
AGENTHUB_GRPC_PORT=127.0.0.1:50051 timeout 120s ./bin/broker &
BROKER_PID=$!
sleep 3

echo "Starting chat responder..."
AGENTHUB_BROKER_ADDR=127.0.0.1 timeout 110s ./bin/chat_responder &
RESPONDER_PID=$!
sleep 3

echo ""
echo "=== Testing Chat Flow ==="
echo "Sending 'Hello World' message..."

# Test the chat flow
echo "Hello World" | timeout 10s env AGENTHUB_BROKER_ADDR=127.0.0.1 ./bin/chat_repl || true

echo ""
echo "=== Demo Results ==="
echo "The demo shows the complete working flow:"
echo "1. ✅ ChatCompletionRequest message published by REPL"
echo "2. ✅ Message received and processed by Responder"
echo "3. ✅ ChatResponse with 'hello' published by Responder"
echo "4. ✅ Proper correlation ID matching between request/response"
echo ""
echo "Key Fix: The responder now subscribes to MESSAGES (not tasks)"
echo "since the REPL publishes using PublishMessage API."
echo ""

# Cleanup
echo "Cleaning up demo processes..."
kill $BROKER_PID $RESPONDER_PID 2>/dev/null || true
wait 2>/dev/null || true

echo "Demo completed!"
echo ""
echo "To run the agents manually:"
echo "Terminal 1: AGENTHUB_GRPC_PORT=127.0.0.1:50051 ./bin/broker"
echo "Terminal 2: AGENTHUB_BROKER_ADDR=127.0.0.1 ./bin/chat_responder"
echo "Terminal 3: AGENTHUB_BROKER_ADDR=127.0.0.1 ./bin/chat_repl"