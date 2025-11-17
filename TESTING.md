# Testing Guide: Dynamic Agent Discovery

## Quick Start

### Option 1: Run Automated Tests

```bash
# Test agent discovery only (no LLM required)
./test_agent_discovery.sh

# Test with VertexAI LLM integration
export GCP_PROJECT=your-project-id
export GCP_LOCATION=us-central1
export VERTEX_AI_MODEL=gemini-2.0-flash
./test_e2e_vertexai.sh
```

### Option 2: Interactive Demo

```bash
# Set VertexAI credentials (optional, will use mock if not set)
export GCP_PROJECT=your-project-id
export GCP_LOCATION=us-central1
export VERTEX_AI_MODEL=gemini-2.0-flash

# Run interactive demo with tmux
./demo_cortex.sh
```

## What to Expect

### Agent Discovery Flow

1. **Broker starts** - Event bus ready
2. **Cortex starts** - Subscribes to agent events
3. **Echo agent starts** - Registers with detailed AgentCard
4. **Discovery happens automatically:**
   ```
   time=... level=INFO msg="Agent registered" agent_id=agent_echo
   time=... level=INFO msg="Received agent card event" agent_id=agent_echo
   time=... level=INFO msg="Agent skills registered" skills="[Echo Messages: ...]"
   time=... level=INFO msg="Agent registered with Cortex orchestrator" total_agents=1
   ```

### LLM Integration

The VertexAI LLM will see:
```
Available agents:
- agent_echo: A simple echo agent that repeats back messages for testing purposes
  Skills:
    * Echo Messages: Echoes back any text message with an 'Echo: ' prefix for testing and verification
```

Examples that should work:
- "Can you echo hello world?"
- "Please repeat what I say"
- "Test the echo functionality"
- "Echo this message back to me"

## Test Scenarios

### Scenario 1: Verify Agent Registration

**Test:**
```bash
./test_agent_discovery.sh
```

**Expected Output:**
```
✓ Broker started (PID: ...)
✓ Cortex started (PID: ...)
✓ Echo Agent started (PID: ...)
✓ Cortex received agent registration event
✓ Echo agent discovered by Cortex
✓ Echo agent skills registered
```

**What's Being Tested:**
- Agent sends AgentCard via RegisterAgent RPC
- Broker publishes AgentCardEvent
- Cortex subscribes and receives the event
- Cortex registers the agent internally
- Agent count updates correctly

### Scenario 2: Verify LLM Integration

**Test:**
```bash
export GCP_PROJECT=your-project
./test_e2e_vertexai.sh
```

**Expected Output:**
```
✓ VertexAI configured
✓ Agent discovery complete
✓ Agent skills integrated with Cortex
✓ Agent count updated in Cortex

Agent Discovery:
  • Registered agents: 1
  • Skills discovered: 1
```

**What's Being Tested:**
- VertexAI client initialization
- Agent skills visible in LLM prompt
- Cortex can query available agents
- Agent metadata properly formatted for LLM

### Scenario 3: End-to-End Message Delegation

**Manual Test:**
```bash
# Terminal 1: Run demo
./demo_cortex.sh

# In the chat CLI (top pane), type:
Can you echo hello world?
```

**Expected Flow:**
1. User message sent to Cortex
2. Cortex calls VertexAI with agent context
3. LLM recognizes "echo" matches echo agent skill
4. LLM decides to delegate to agent_echo
5. Cortex publishes task message with task_type="echo"
6. Echo agent receives and processes
7. Echo agent responds with "Echo: hello world"
8. Cortex receives result
9. Cortex calls LLM to synthesize response
10. User sees final response

**Watch Logs:**
- Cortex log: Look for "llm_decide" and "delegate" traces
- Echo log: Look for "Received echo request"
- Broker log: Look for message routing events

## Debugging

### Check Agent Registration

```bash
# In one terminal, start broker and cortex:
./bin/broker &
sleep 3
./bin/cortex &

# In another terminal, start echo agent:
LOG_LEVEL=DEBUG ./bin/echo_agent

# Look for:
# "Agent registered" - from echo agent
# "Agent card event" - from Cortex
# "skills registered" - from Cortex
```

### Check LLM Sees Agent

```bash
# Start services
./bin/broker &
sleep 3
LOG_LEVEL=DEBUG ./bin/cortex &
sleep 3
./bin/echo_agent &

# Send a test message and watch Cortex DEBUG logs
# You should see the LLM prompt includes:
# "Available agents:"
# "- agent_echo: A simple echo agent..."
```

### Verify Event Routing

```bash
# Watch broker logs for agent_registered events:
./bin/broker 2>&1 | grep "agent_registered"

# Expected:
# level=DEBUG msg="Routing event to subscribers" event_id=agent_registered_agent_echo_...
# level=DEBUG msg="Event delivered to subscriber" event_id=agent_registered_...
```

## Common Issues

### Issue: "No subscribers for event"

**Cause:** Cortex not subscribed before echo agent registers

**Fix:** Ensure Cortex starts before echo agent:
```bash
./bin/cortex &
sleep 3  # Give Cortex time to subscribe
./bin/echo_agent &
```

### Issue: "Agent not registered with Cortex"

**Cause:** Event routing failed

**Check:**
1. Broker logs for event routing errors
2. Cortex logs for subscription status
3. Network connectivity (gRPC port 50051)

### Issue: LLM doesn't delegate

**Cause:** Agent skills not clear or LLM doesn't recognize match

**Fix:**
1. Check agent card has clear skill descriptions
2. Add more examples to skill.Examples
3. Verify VertexAI credentials are correct
4. Try more explicit requests: "Use the echo agent to repeat this"

## Logs to Monitor

### Broker (`bin/broker`)
```
level=INFO msg="Agent registered" agent_id=agent_echo
level=DEBUG msg="Routing event to subscribers" event_type=agent.registered
level=DEBUG msg="Event delivered to subscriber"
```

### Cortex (`bin/cortex`)
```
level=INFO msg="Subscribed to agent registration events"
level=INFO msg="Received agent card event" agent_id=agent_echo
level=INFO msg="Agent skills registered" skills="[Echo Messages: ...]"
level=INFO msg="Agent registered with Cortex orchestrator" total_agents=1
```

### Echo Agent (`bin/echo_agent`)
```
level=INFO msg="Echo agent registered successfully" agent_id=agent_echo
level=INFO msg="Received echo request" message_id=... context_id=...
level=INFO msg="Published echo response" echo_text="Echo: ..."
```

## Performance Metrics

Expected timings:
- Agent registration: < 100ms
- Event delivery: < 50ms
- Cortex registration processing: < 10ms
- Total discovery time: < 200ms

Check logs for timing:
```bash
grep "Agent registered" logs/*.log | grep -o "time=[^ ]*"
```

## Success Criteria

✅ All automated tests pass
✅ Agent count in Cortex matches running agents
✅ Skills visible in Cortex logs
✅ LLM prompt includes agent details
✅ Event routing delivers to all subscribers
✅ No errors in any service logs
✅ Total discovery time < 1 second

## Next Steps

After successful testing:
1. Review IMPLEMENTATION_SUMMARY.md for architecture details
2. Review AGENT_DECIDE.md for the complete specification
3. Try creating your own agent with custom skills
4. Explore multi-agent scenarios with multiple agents
