package main

import (
	"regexp"
	"strings"
)

// slogFieldRe matches key=value pairs in slog text output.
// Handles: key="quoted value", key=[array], key=simple
var slogFieldRe = regexp.MustCompile(`(\w+)=("(?:[^"\\]|\\.)*"|\[(?:[^\]]*)\]|\S+)`)

// parseSlogFields extracts key-value pairs from a slog text-formatted line.
func parseSlogFields(line string) map[string]string {
	matches := slogFieldRe.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}
	fields := make(map[string]string, len(matches))
	for _, m := range matches {
		key := m[1]
		val := m[2]
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		fields[key] = val
	}
	return fields
}

// extractTime pulls HH:MM:SS from a slog time field.
func extractTime(fields map[string]string) string {
	t, ok := fields["time"]
	if !ok {
		return ""
	}
	if idx := strings.IndexByte(t, 'T'); idx >= 0 {
		rest := t[idx+1:]
		if len(rest) >= 8 {
			return rest[:8]
		}
	}
	return ""
}

// parseLogLine converts a raw log line from a child process into a visual
// WSMessage event. Returns nil for lines that should be suppressed.
func parseLogLine(source, line string) *WSMessage {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	fields := parseSlogFields(line)
	if fields != nil && fields["msg"] != "" {
		return matchSlogEvent(source, fields)
	}

	return matchPlainText(source, line)
}

// suppress lists messages that should never be shown.
var suppress = map[string]bool{
	"No subscribers for event":                                       true,
	"Event delivered to subscriber":                                  true,
	"Timeout sending event to subscriber":                            true,
	"Starting health server":                                         true,
	"Health server failed":                                           true,
	"AgentHub client started with observability":                     true,
	"AgentHub gRPC server with observability listening":              true,
	"Registered task handler":                                        true,
	"Message stream ended":                                           true,
	"Agent event stream ended":                                       true,
	"Task event stream ended":                                        true,
	"Shutting down AgentHub client":                                  true,
	"Shutting down AgentHub server":                                  true,
	"Error shutting down health server":                              true,
	"Observability shutdown failed - likely OTLP trace export issue": true,
	"Error closing gRPC connection":                                  true,
}

func matchSlogEvent(source string, f map[string]string) *WSMessage {
	msg := f["msg"]
	ts := extractTime(f)
	level := f["level"]

	if suppress[msg] {
		return nil
	}

	switch msg {

	// ── Broker ──

	case "Broker received message":
		return &WSMessage{
			Type: "event", Source: source, Icon: "in",
			Content: "Message received",
			Detail:  join(f["from_agent"], " → ", f["to_agent"]),
			Time:    ts,
		}
	case "Message routed successfully":
		return &WSMessage{
			Type: "event", Source: source, Icon: "route",
			Content: "Message routed",
			Detail:  f["subscriber_count"] + " subscribers",
			Time:    ts,
		}
	case "Agent subscribed to messages":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Agent connected",
			Detail:  f["agent_id"],
			Time:    ts,
		}
	case "Agent registered":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Agent registered",
			Detail:  f["agent_name"],
			Time:    ts,
		}
	case "Routing event to subscribers":
		return &WSMessage{
			Type: "event", Source: source, Icon: "route",
			Content: "Event routed",
			Detail:  pair(f["event_type"], f["subscriber_count"]+" subscribers"),
			Time:    ts,
		}
	case "Received shutdown signal":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Shutting down",
			Time:    ts,
		}

	// ── Cortex ──

	case "Cortex received message":
		return &WSMessage{
			Type: "event", Source: source, Icon: "in",
			Content: "Message received",
			Detail:  shortID(f["message_id"]),
			Time:    ts,
		}
	case "Cortex successfully processed message":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Message processed",
			Detail:  shortID(f["message_id"]),
			Time:    ts,
		}
	case "Cortex failed to handle message":
		return &WSMessage{
			Type: "event", Source: source, Icon: "err",
			Content: "Processing failed",
			Detail:  f["error"],
			Status:  "error",
			Time:    ts,
		}
	case "Received agent card event":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Agent discovered",
			Detail:  pair(f["agent_name"], f["skills_count"]+" skills"),
			Time:    ts,
		}
	case "Agent skills registered":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Skills registered",
			Detail:  pair(f["agent_id"], f["skills"]),
			Time:    ts,
		}
	case "Agent registered with Cortex orchestrator":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Agent registered",
			Detail:  pair(f["agent_id"], f["total_agents"]+" total"),
			Time:    ts,
		}
	case "Received task status update":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Task update",
			Detail:  pair(shortID(f["task_id"]), f["state"]),
			Time:    ts,
		}
	case "Received task artifact":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Artifact received",
			Detail:  f["artifact_name"],
			Time:    ts,
		}
	case "Cortex is ready to orchestrate conversations and tasks":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Ready",
			Time:    ts,
		}
	case "Starting Cortex Orchestrator":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Starting...",
			Time:    ts,
		}
	case "Cortex initialized":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Initialized",
			Detail:  "LLM: " + f["llm_client"],
			Time:    ts,
		}
	case "Subscribed to agent registration events":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Watching for agents",
			Time:    ts,
		}
	case "Subscribed to task updates":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Watching for tasks",
			Time:    ts,
		}

	// ── SubAgent / Echo Agent ──

	case "Agent started successfully":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Agent ready",
			Detail:  f["name"],
			Time:    ts,
		}
	case "Agent card registered":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Registered",
			Detail:  pair(f["name"], f["skills"]+" skills"),
			Time:    ts,
		}
	case "Processing task":
		return &WSMessage{
			Type: "event", Source: source, Icon: "in",
			Content: "Processing task",
			Detail:  pair(f["skill"], shortID(f["task_id"])),
			Time:    ts,
		}
	case "Task completed successfully":
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "Task completed",
			Detail:  f["skill"],
			Time:    ts,
		}
	case "Task failed":
		return &WSMessage{
			Type: "event", Source: source, Icon: "err",
			Content: "Task failed",
			Detail:  pair(f["skill"], f["error"]),
			Status:  "error",
			Time:    ts,
		}
	case "Starting task subscription":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Listening for tasks",
			Time:    ts,
		}
	case "Agent shutting down gracefully":
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Shutting down",
			Time:    ts,
		}
	}

	// ── Generic error/warning — always show ──
	if level == "ERROR" || level == "WARN" {
		return &WSMessage{
			Type: "event", Source: source, Icon: "err",
			Content: msg,
			Detail:  f["error"],
			Status:  "error",
			Time:    ts,
		}
	}

	// Suppress unrecognized DEBUG and most INFO noise
	if level == "DEBUG" {
		return nil
	}

	// Unrecognized INFO — show compactly
	return &WSMessage{
		Type: "event", Source: source, Icon: "info",
		Content: msg,
		Time:    ts,
	}
}

func matchPlainText(source, line string) *WSMessage {
	switch {
	case strings.Contains(line, "Initializing VertexAI"):
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Initializing VertexAI",
		}
	case strings.Contains(line, "VertexAI client initialized"):
		return &WSMessage{
			Type: "event", Source: source, Icon: "ok",
			Content: "VertexAI ready",
		}
	case strings.Contains(line, "mock LLM client"):
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Using mock LLM",
		}
	case strings.Contains(line, "Shutting down"):
		return &WSMessage{
			Type: "event", Source: source, Icon: "info",
			Content: "Shutting down",
		}
	}

	// Suppress long or noisy lines
	if len(line) > 200 {
		return nil
	}

	return nil
}

// join concatenates a, sep, b — returning "" if both are empty.
func join(a, sep, b string) string {
	if a == "" && b == "" {
		return ""
	}
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + sep + b
}

// pair formats "a · b", handling empty values.
func pair(a, b string) string {
	return join(a, " · ", b)
}

// shortID truncates long IDs for display.
func shortID(id string) string {
	if len(id) > 16 {
		return id[:16] + "..."
	}
	return id
}
