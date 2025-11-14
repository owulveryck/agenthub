#!/bin/bash

# Cortex POC Demo Script with tmux split-screen view
# This script starts the broker, Cortex, and agents for testing
# Displays logs in real-time using tmux panes

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║      Cortex POC Demo Launcher (tmux mode)         ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
    echo "Error: tmux is not installed"
    echo "Please install tmux:"
    echo "  macOS:         brew install tmux"
    echo "  Ubuntu/Debian: sudo apt-get install tmux"
    echo "  RHEL/CentOS:   sudo yum install tmux"
    exit 1
fi

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# tmux session name
TMUX_SESSION="agenthub_demo"

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

# Function to cleanup background processes and tmux
cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"

    # Kill tmux session if it exists
    tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true

    # Kill all child processes
    pkill -P $$ 2>/dev/null || true

    # Wait for processes to finish
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

# Enable DEBUG logging to capture logs to stdout
export LOG_LEVEL=DEBUG

echo -e "${BLUE}Starting Event Bus (Broker)...${NC}"
timeout 120s ./bin/broker > "$BROKER_LOG" 2>&1 &
BROKER_PID=$!
sleep 2

# Check if broker is running
if ! ps -p $BROKER_PID > /dev/null; then
    echo -e "${YELLOW}Error: Broker failed to start${NC}"
    echo "Check logs: tail -f $BROKER_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Broker started (PID: $BROKER_PID)${NC}"

echo -e "${BLUE}Starting Cortex Orchestrator...${NC}"
timeout 110s ./bin/cortex > "$CORTEX_LOG" 2>&1 &
CORTEX_PID=$!
sleep 2

if ! ps -p $CORTEX_PID > /dev/null; then
    echo -e "${YELLOW}Error: Cortex failed to start${NC}"
    echo "Check logs: tail -f $CORTEX_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Cortex started (PID: $CORTEX_PID)${NC}"

echo -e "${BLUE}Starting Echo Agent...${NC}"
timeout 110s ./bin/echo_agent > "$ECHO_LOG" 2>&1 &
ECHO_PID=$!
sleep 2

if ! ps -p $ECHO_PID > /dev/null; then
    echo -e "${YELLOW}Error: Echo Agent failed to start${NC}"
    echo "Check logs: tail -f $ECHO_LOG"
    exit 1
fi
echo -e "${GREEN}✓ Echo Agent started (PID: $ECHO_PID)${NC}"

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
echo -e "${BLUE}Launching tmux with split-screen view...${NC}"
echo ""
echo "Layout:"
echo "  ┌─────────────────────────────────┐"
echo "  │      Chat CLI (focused)         │ ← Your interactive session"
echo "  ├──────────┬──────────┬───────────┤"
echo "  │  Broker  │  Cortex  │   Echo    │ ← Live logs"
echo "  │   Logs   │   Logs   │   Logs    │"
echo "  └──────────┴──────────┴───────────┘"
echo ""
echo -e "${YELLOW}Tip: Ctrl-B then arrow keys to navigate panes${NC}"
echo -e "${YELLOW}     Ctrl-D or 'quit' to exit${NC}"
echo ""
sleep 2

# Kill any existing session with same name
tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true

# Create new tmux session (detached)
tmux new-session -d -s "$TMUX_SESSION"

# Split into top (CLI - 70%) and bottom (logs - 30%)
tmux split-window -v -p 30 -t "$TMUX_SESSION"

# Select bottom pane and split into 3 equal parts for logs
tmux select-pane -t "$TMUX_SESSION:0.1"
tmux split-window -h -p 66 -t "$TMUX_SESSION"
tmux select-pane -t "$TMUX_SESSION:0.2"
tmux split-window -h -p 50 -t "$TMUX_SESSION"

# Final layout:
# Pane 0: Top (CLI) - 70% height, full width
# Pane 1: Bottom-left (Broker log) - 30% height, 33% width
# Pane 2: Bottom-center (Cortex log) - 30% height, 33% width
# Pane 3: Bottom-right (Echo log) - 30% height, 33% width

# Set pane titles (works with tmux 3.0+, silently fails on older versions)
tmux select-pane -t "$TMUX_SESSION:0.0" -T "Chat CLI" 2>/dev/null || true
tmux select-pane -t "$TMUX_SESSION:0.1" -T "Broker Logs" 2>/dev/null || true
tmux select-pane -t "$TMUX_SESSION:0.2" -T "Cortex Logs" 2>/dev/null || true
tmux select-pane -t "$TMUX_SESSION:0.3" -T "Echo Agent Logs" 2>/dev/null || true

# Enable pane borders with titles (tmux 3.0+)
tmux set-option -t "$TMUX_SESSION" pane-border-status top 2>/dev/null || true
tmux set-option -t "$TMUX_SESSION" pane-border-format " #{pane_title} " 2>/dev/null || true

# Send tail commands to log panes
tmux send-keys -t "$TMUX_SESSION:0.1" "tail -f '$BROKER_LOG'" C-m
tmux send-keys -t "$TMUX_SESSION:0.2" "tail -f '$CORTEX_LOG'" C-m
tmux send-keys -t "$TMUX_SESSION:0.3" "tail -f '$ECHO_LOG'" C-m

# Send CLI command to main pane
tmux send-keys -t "$TMUX_SESSION:0.0" "./bin/chat_cli" C-m

# Select CLI pane (focus)
tmux select-pane -t "$TMUX_SESSION:0.0"

# Attach to the session (this blocks until session is killed/detached)
tmux attach-session -t "$TMUX_SESSION"

# When we reach here, tmux has exited
# Cleanup will be called automatically via trap
