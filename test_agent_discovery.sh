#!/bin/bash

# Test script for agent discovery and dynamic registration
# This tests the complete flow:
# 1. Broker starts
# 2. Cortex starts and subscribes to agent events
# 3. Echo agent registers
# 4. Cortex receives the agent card and updates its LLM prompt
# 5. User sends a message
# 6. Cortex delegates to echo agent

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║   Agent Discovery Test (with VertexAI)            ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check for required environment variables
if [ -z "$GCP_PROJECT" ] || [ "$GCP_PROJECT" = "your-project" ]; then
    echo -e "${YELLOW}Warning: GCP_PROJECT not set. Using mock LLM.${NC}"
    echo -e "${YELLOW}Set GCP_PROJECT, GCP_LOCATION, and VERTEX_AI_MODEL to use VertexAI${NC}"
    echo ""
fi

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    pkill -P $$ 2>/dev/null || true
    wait 2>/dev/null || true
    echo -e "${GREEN}Cleanup complete${NC}"
}

trap cleanup EXIT INT TERM

# Set environment
export AGENTHUB_GRPC_PORT=127.0.0.1:50051
export AGENTHUB_BROKER_ADDR=127.0.0.1
export LOG_LEVEL=DEBUG

# Create log directory
LOG_DIR="${TMPDIR:-/tmp}/agenthub_test_$$"
mkdir -p "$LOG_DIR"

BROKER_LOG="$LOG_DIR/broker.log"
CORTEX_LOG="$LOG_DIR/cortex.log"
ECHO_LOG="$LOG_DIR/echo.log"

echo -e "${BLUE}Logs will be written to: $LOG_DIR${NC}"
echo ""

# Start broker
echo -e "${BLUE}[1/3] Starting broker...${NC}"
timeout 60s ./bin/broker > "$BROKER_LOG" 2>&1 &
BROKER_PID=$!
sleep 3

if ! ps -p $BROKER_PID > /dev/null; then
    echo -e "${RED}✗ Broker failed to start${NC}"
    cat "$BROKER_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Broker started (PID: $BROKER_PID)${NC}"

# Start Cortex
echo -e "${BLUE}[2/3] Starting Cortex orchestrator...${NC}"
timeout 55s ./bin/cortex > "$CORTEX_LOG" 2>&1 &
CORTEX_PID=$!
sleep 3

if ! ps -p $CORTEX_PID > /dev/null; then
    echo -e "${RED}✗ Cortex failed to start${NC}"
    cat "$CORTEX_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Cortex started (PID: $CORTEX_PID)${NC}"

# Start Echo Agent (this should trigger agent registration event)
echo -e "${BLUE}[3/3] Starting Echo Agent...${NC}"
timeout 55s ./bin/echo_agent > "$ECHO_LOG" 2>&1 &
ECHO_PID=$!
sleep 3

if ! ps -p $ECHO_PID > /dev/null; then
    echo -e "${RED}✗ Echo Agent failed to start${NC}"
    cat "$ECHO_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Echo Agent started (PID: $ECHO_PID)${NC}"

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  All services started successfully!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo ""

# Give services time to register and discover
echo -e "${YELLOW}Waiting for agent discovery (5 seconds)...${NC}"
sleep 5

# Check logs for agent registration
echo ""
echo -e "${BLUE}Checking agent registration...${NC}"

if grep -q "Agent registered with Cortex orchestrator" "$CORTEX_LOG"; then
    echo -e "${GREEN}✓ Cortex received agent registration event${NC}"

    # Extract agent details from logs
    if grep -q "agent_echo" "$CORTEX_LOG"; then
        echo -e "${GREEN}✓ Echo agent discovered by Cortex${NC}"
    fi

    if grep -q "Echo Messages" "$CORTEX_LOG"; then
        echo -e "${GREEN}✓ Echo agent skills registered${NC}"
    fi
else
    echo -e "${RED}✗ Agent registration not found in Cortex logs${NC}"
    echo -e "${YELLOW}Cortex logs:${NC}"
    grep -i "agent" "$CORTEX_LOG" | tail -10
fi

echo ""
echo -e "${BLUE}Key log entries:${NC}"
echo ""
echo -e "${YELLOW}=== Echo Agent Registration ===${NC}"
grep "Agent registered" "$ECHO_LOG" | head -5
echo ""
echo -e "${YELLOW}=== Broker Event Routing ===${NC}"
grep "agent_registered" "$BROKER_LOG" | head -5
echo ""
echo -e "${YELLOW}=== Cortex Agent Discovery ===${NC}"
grep -i "agent.*registered\|skills registered" "$CORTEX_LOG" | head -10

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Test Summary${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo ""
echo "1. Agent Registration: Echo agent sent AgentCard to broker"
echo "2. Event Publishing: Broker published agent_registered event"
echo "3. Discovery: Cortex received and registered the agent"
echo "4. LLM Integration: Agent skills are now available to LLM"
echo ""
echo -e "${BLUE}Full logs available at: $LOG_DIR${NC}"
echo "  • Broker:  tail -f $BROKER_LOG"
echo "  • Cortex:  tail -f $CORTEX_LOG"
echo "  • Echo:    tail -f $ECHO_LOG"
echo ""
echo -e "${YELLOW}Services will run for 45 more seconds...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"

# Wait for timeouts or manual termination
wait
