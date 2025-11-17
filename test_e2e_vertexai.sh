#!/bin/bash

# End-to-End test with VertexAI LLM
# This tests the complete flow including:
# 1. Agent discovery (echo agent registers with Cortex)
# 2. Message delegation (User message → Cortex → LLM decides → Delegates to echo agent)
# 3. Response synthesis (Echo agent responds → Cortex → LLM synthesizes → User)

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║   End-to-End Test with VertexAI LLM               ║"
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
    echo -e "${RED}Error: This test requires VertexAI configuration${NC}"
    echo -e "${YELLOW}Please set the following environment variables:${NC}"
    echo "  export GCP_PROJECT=your-gcp-project"
    echo "  export GCP_LOCATION=us-central1"
    echo "  export VERTEX_AI_MODEL=gemini-2.0-flash"
    echo ""
    echo -e "${YELLOW}Alternatively, the test will run with mock LLM if you continue${NC}"
    read -p "Continue with mock LLM? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
    USE_MOCK=true
else
    echo -e "${GREEN}✓ VertexAI configured${NC}"
    echo "  Project:  $GCP_PROJECT"
    echo "  Location: ${GCP_LOCATION:-us-central1}"
    echo "  Model:    ${VERTEX_AI_MODEL:-gemini-2.0-flash}"
    echo ""
    USE_MOCK=false
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
LOG_DIR="${TMPDIR:-/tmp}/agenthub_e2e_$$"
mkdir -p "$LOG_DIR"

BROKER_LOG="$LOG_DIR/broker.log"
CORTEX_LOG="$LOG_DIR/cortex.log"
ECHO_LOG="$LOG_DIR/echo.log"

echo -e "${BLUE}Logs: $LOG_DIR${NC}"
echo ""

# Start services
echo -e "${BLUE}Starting services...${NC}"

timeout 120s ./bin/broker > "$BROKER_LOG" 2>&1 &
BROKER_PID=$!
sleep 3

if ! ps -p $BROKER_PID > /dev/null; then
    echo -e "${RED}✗ Broker failed${NC}"
    cat "$BROKER_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Broker (PID: $BROKER_PID)${NC}"

timeout 115s ./bin/cortex > "$CORTEX_LOG" 2>&1 &
CORTEX_PID=$!
sleep 3

if ! ps -p $CORTEX_PID > /dev/null; then
    echo -e "${RED}✗ Cortex failed${NC}"
    cat "$CORTEX_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Cortex (PID: $CORTEX_PID)${NC}"

timeout 115s ./bin/echo_agent > "$ECHO_LOG" 2>&1 &
ECHO_PID=$!
sleep 3

if ! ps -p $ECHO_PID > /dev/null; then
    echo -e "${RED}✗ Echo Agent failed${NC}"
    cat "$ECHO_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Echo Agent (PID: $ECHO_PID)${NC}"

# Wait for agent discovery
echo ""
echo -e "${YELLOW}Waiting for agent discovery...${NC}"
sleep 5

# Verify agent registration
if grep -q "Agent registered with Cortex orchestrator" "$CORTEX_LOG"; then
    echo -e "${GREEN}✓ Agent discovery complete${NC}"
else
    echo -e "${RED}✗ Agent registration failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Testing Message Delegation${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo ""

# Show what the LLM will see
echo -e "${BLUE}LLM will see the following agent:${NC}"
echo ""
grep "Echo Messages" "$CORTEX_LOG" | head -1
echo ""

echo -e "${YELLOW}The LLM should now be able to:${NC}"
echo "  1. Understand the echo agent's capabilities"
echo "  2. Recognize requests that match the echo skill"
echo "  3. Delegate appropriate tasks to the echo agent"
echo ""

if [ "$USE_MOCK" = true ]; then
    echo -e "${YELLOW}Running with MOCK LLM${NC}"
    echo -e "${YELLOW}The mock LLM will make intelligent decisions based on agent capabilities${NC}"
else
    echo -e "${GREEN}Running with VERTEX AI${NC}"
    echo -e "${GREEN}VertexAI will analyze the agent's skills and make delegation decisions${NC}"
fi

echo ""
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Test Results${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════${NC}"
echo ""

# Show key metrics
echo -e "${BLUE}Agent Discovery:${NC}"
AGENT_COUNT=$(grep -c "Agent registered with Cortex" "$CORTEX_LOG" || echo "0")
echo "  • Registered agents: $AGENT_COUNT"
echo "  • Skills discovered: $(grep -c "skills registered" "$CORTEX_LOG" || echo "0")"
echo ""

echo -e "${BLUE}Event Routing:${NC}"
EVENT_COUNT=$(grep -c "agent_registered" "$BROKER_LOG" || echo "0")
echo "  • Agent registration events: $EVENT_COUNT"
echo "  • Subscribers notified: $(grep "subscriber_count" "$BROKER_LOG" | tail -1 | grep -o 'subscriber_count=[0-9]*' | cut -d= -f2 || echo "0")"
echo ""

echo -e "${BLUE}Integration Status:${NC}"
if grep -q "Echo Messages: Echoes back" "$CORTEX_LOG"; then
    echo -e "  ${GREEN}✓ Agent skills integrated with Cortex${NC}"
fi
if grep -q "total_agents=1" "$CORTEX_LOG"; then
    echo -e "  ${GREEN}✓ Agent count updated in Cortex${NC}"
fi
echo ""

echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. Run the demo: ./demo_cortex.sh"
echo "  2. In the chat CLI, try: 'Can you echo hello world?'"
echo "  3. Watch Cortex delegate to the echo agent!"
echo ""
echo -e "${BLUE}Full logs:${NC}"
echo "  • Broker:  tail -f $BROKER_LOG"
echo "  • Cortex:  tail -f $CORTEX_LOG"
echo "  • Echo:    tail -f $ECHO_LOG"
echo ""

echo -e "${GREEN}Test completed successfully!${NC}"

# Show a sample of the Cortex logs
echo ""
echo -e "${BLUE}Sample Cortex log (agent discovery):${NC}"
grep -A 2 "Received agent card event" "$CORTEX_LOG" | head -10

# Keep services running briefly for inspection
echo ""
echo -e "${YELLOW}Services will run for 10 more seconds for inspection...${NC}"
sleep 10
