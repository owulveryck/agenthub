#!/bin/bash

# Cortex POC Demo Script
# This script starts the broker, Cortex, and agents for testing

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║            Cortex POC Demo Launcher                ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if binaries exist
if [ ! -f "bin/broker" ]; then
    echo -e "${YELLOW}Building broker...${NC}"
    make build-broker || go build -o bin/broker ./broker
fi

if [ ! -f "bin/cortex" ]; then
    echo -e "${YELLOW}Building Cortex...${NC}"
    go build -o bin/cortex ./agents/cortex/cmd
fi

if [ ! -f "bin/echo_agent" ]; then
    echo -e "${YELLOW}Building Echo Agent...${NC}"
    go build -o bin/echo_agent ./agents/echo_agent
fi

if [ ! -f "bin/chat_cli" ]; then
    echo -e "${YELLOW}Building Chat CLI...${NC}"
    go build -o bin/chat_cli ./agents/chat_cli
fi

# Function to cleanup background processes
cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"
    jobs -p | xargs -r kill 2>/dev/null || true
    wait 2>/dev/null || true
    echo -e "${GREEN}Cleanup complete${NC}"
}

trap cleanup EXIT INT TERM

# Create temporary log files
LOG_DIR="${TMPDIR:-/tmp}"
BROKER_LOG="${LOG_DIR}/agenthub_broker_$$.log"
CORTEX_LOG="${LOG_DIR}/agenthub_cortex_$$.log"
ECHO_LOG="${LOG_DIR}/agenthub_echo_$$.log"

echo -e "${BLUE}Log files:${NC}"
echo "  • Broker:      $BROKER_LOG"
echo "  • Cortex:      $CORTEX_LOG"
echo "  • Echo Agent:  $ECHO_LOG"
echo ""
echo -e "${YELLOW}Tip: tail -f $BROKER_LOG $CORTEX_LOG $ECHO_LOG${NC}"
echo ""

# Set environment variables
export AGENTHUB_GRPC_PORT=127.0.0.1:50051
export AGENTHUB_BROKER_ADDR=127.0.0.1

echo -e "${BLUE}Starting Event Bus (Broker)...${NC}"
timeout 120s ./bin/broker > "$BROKER_LOG" 2>&1 &
BROKER_PID=$!
sleep 2

# Check if broker is running
if ! ps -p $BROKER_PID > /dev/null; then
    echo -e "${YELLOW}Error: Broker failed to start${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Broker started (PID: $BROKER_PID, log: $BROKER_LOG)${NC}"

echo -e "${BLUE}Starting Cortex Orchestrator...${NC}"
timeout 110s ./bin/cortex > "$CORTEX_LOG" 2>&1 &
CORTEX_PID=$!
sleep 2

if ! ps -p $CORTEX_PID > /dev/null; then
    echo -e "${YELLOW}Error: Cortex failed to start${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Cortex started (PID: $CORTEX_PID, log: $CORTEX_LOG)${NC}"

echo -e "${BLUE}Starting Echo Agent...${NC}"
timeout 110s ./bin/echo_agent > "$ECHO_LOG" 2>&1 &
ECHO_PID=$!
sleep 2

if ! ps -p $ECHO_PID > /dev/null; then
    echo -e "${YELLOW}Error: Echo Agent failed to start${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Echo Agent started (PID: $ECHO_PID, log: $ECHO_LOG)${NC}"

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  All services started successfully!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo ""
echo "Services running:"
echo "  • Broker:      PID $BROKER_PID (port 50051)"
echo "  • Cortex:      PID $CORTEX_PID"
echo "  • Echo Agent:  PID $ECHO_PID"
echo ""
echo "To monitor logs in another terminal:"
echo -e "${YELLOW}  tail -f $BROKER_LOG $CORTEX_LOG $ECHO_LOG${NC}"
echo ""
echo -e "${BLUE}Starting Chat CLI...${NC}"
echo ""

# Run the CLI in foreground
./bin/chat_cli

# Cleanup will be called automatically via trap
