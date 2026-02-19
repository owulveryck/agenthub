package vertexai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"google.golang.org/genai"

	"github.com/owulveryck/agenthub/agents/cortex/llm"
	pb "github.com/owulveryck/agenthub/events/a2a"
)

// Config holds the configuration for the VertexAI client
type Config struct {
	Project  string
	Location string
	Model    string
}

// NewConfigFromEnv creates a VertexAI config from environment variables
// matching the pattern used in agents/chat_responder
func NewConfigFromEnv() *Config {
	return &Config{
		Project:  getEnvOrDefault("GCP_PROJECT", "your-project"),
		Location: getEnvOrDefault("GCP_LOCATION", "us-central1"),
		Model:    getEnvOrDefault("VERTEX_AI_MODEL", "gemini-2.0-flash"),
	}
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Client implements the llm.Client interface using VertexAI
type Client struct {
	config *Config
	client *genai.Client
	logger *slog.Logger
}

// NewClient creates a new VertexAI client for Cortex orchestration
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  config.Project,
		Location: config.Location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	// Create logger for VertexAI client
	// Use DEBUG level by default if LOG_LEVEL=DEBUG, otherwise INFO
	logLevel := slog.LevelInfo
	if strings.ToUpper(os.Getenv("LOG_LEVEL")) == "DEBUG" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	return &Client{
		config: config,
		client: genaiClient,
		logger: logger,
	}, nil
}

// Decide implements the llm.Client interface
// It analyzes the conversation history and available agents to decide what actions to take
func (c *Client) Decide(
	ctx context.Context,
	conversationHistory []*pb.Message,
	availableAgents map[string]*pb.AgentCard,
	newEvent *pb.Message,
) (*llm.Decision, error) {
	if newEvent == nil {
		return &llm.Decision{
			Reasoning: "No new event to process",
			Actions:   []llm.Action{},
		}, nil
	}

	// Build the orchestration prompt
	prompt := c.buildOrchestrationPrompt(conversationHistory, availableAgents, newEvent)

	// Log the prompt being sent to VertexAI
	c.logger.DebugContext(ctx, "Sending prompt to VertexAI",
		"model", c.config.Model,
		"project", c.config.Project,
		"prompt_length", len(prompt),
	)
	c.logger.DebugContext(ctx, "VertexAI prompt content",
		"prompt", prompt,
	)

	// Query VertexAI for orchestration decision
	response, err := c.queryVertexAI(ctx, prompt)
	if err != nil {
		c.logger.ErrorContext(ctx, "VertexAI query failed", "error", err)
		return nil, fmt.Errorf("failed to query VertexAI: %w", err)
	}

	// Log the response from VertexAI
	c.logger.DebugContext(ctx, "Received response from VertexAI",
		"response_length", len(response),
	)
	c.logger.DebugContext(ctx, "VertexAI response content",
		"response", response,
	)

	// Parse the response into a Decision
	decision, err := c.parseDecision(response)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to parse VertexAI response",
			"error", err,
			"response", response,
		)
		// Fallback: return a simple acknowledgment if parsing fails
		return &llm.Decision{
			Reasoning: fmt.Sprintf("Failed to parse LLM response: %v. Providing default response.", err),
			Actions: []llm.Action{
				{
					Type:         "chat.response",
					ResponseText: "I received your message but had trouble processing it. Could you please rephrase?",
				},
			},
		}, nil
	}

	// Log the parsed decision
	c.logger.DebugContext(ctx, "Successfully parsed LLM decision",
		"action_count", len(decision.Actions),
		"reasoning", decision.Reasoning,
	)

	return decision, nil
}

// buildOrchestrationPrompt creates the prompt for the LLM orchestrator
func (c *Client) buildOrchestrationPrompt(
	conversationHistory []*pb.Message,
	availableAgents map[string]*pb.AgentCard,
	newEvent *pb.Message,
) string {
	var prompt strings.Builder

	// System instructions
	prompt.WriteString("You are Cortex, an AI orchestrator. You receive user requests and agent results,\n")
	prompt.WriteString("and you decide the best course of action using the tools available to you.\n\n")
	prompt.WriteString("Your goal is to bring maximum value to the end user by:\n")
	prompt.WriteString("- Analyzing each request and determining which agents (if any) can handle it\n")
	prompt.WriteString("- Building optimal processing pipelines: if one agent's output can be enriched\n")
	prompt.WriteString("  by another agent, chain them together\n")
	prompt.WriteString("- Presenting results directly when no further agent processing adds value\n")
	prompt.WriteString("- Never adding unnecessary wrapper text around self-contained content\n\n")

	// List available agents
	if len(availableAgents) > 0 {
		prompt.WriteString("Available agents:\n")
		for agentID, agent := range availableAgents {
			prompt.WriteString(fmt.Sprintf("- %s (id: %s): %s\n", agent.GetName(), agentID, agent.GetDescription()))
			for _, skill := range agent.GetSkills() {
				prompt.WriteString(fmt.Sprintf("  * Skill: %s — %s\n", skill.GetName(), skill.GetDescription()))
				if len(skill.GetInputModes()) > 0 {
					prompt.WriteString(fmt.Sprintf("    Input: %s\n", strings.Join(skill.GetInputModes(), ", ")))
				}
				if len(skill.GetOutputModes()) > 0 {
					prompt.WriteString(fmt.Sprintf("    Output: %s\n", strings.Join(skill.GetOutputModes(), ", ")))
				}
			}
		}
		prompt.WriteString("\n")
	} else {
		prompt.WriteString("No agents are currently available. You must respond directly to all requests.\n\n")
	}

	// Add conversation history
	if len(conversationHistory) > 1 {
		prompt.WriteString("Conversation history:\n")
		for _, msg := range conversationHistory[:len(conversationHistory)-1] { // Exclude the new event
			role := "User"
			if msg.GetRole() == pb.Role_ROLE_AGENT {
				role = "Agent"
			}
			var content string
			if len(msg.GetContent()) > 0 {
				content = msg.GetContent()[0].GetText()
			}
			prompt.WriteString(fmt.Sprintf("%s: %s\n", role, content))
		}
		prompt.WriteString("\n")
	}

	// Add the new event
	var newEventContent string
	if len(newEvent.GetContent()) > 0 {
		newEventContent = newEvent.GetContent()[0].GetText()
	}

	eventType := "user message"
	if newEvent.GetRole() == pb.Role_ROLE_AGENT && newEvent.GetTaskId() != "" {
		eventType = "task result"
	}

	prompt.WriteString(fmt.Sprintf("New %s: %s\n", eventType, newEventContent))

	// Add metadata context for task results
	if newEvent.GetRole() == pb.Role_ROLE_AGENT && newEvent.GetMetadata() != nil {
		if fields := newEvent.GetMetadata().GetFields(); fields != nil {
			if ct, ok := fields["content_type"]; ok {
				prompt.WriteString(fmt.Sprintf("Content type: %s\n", ct.GetStringValue()))
			}
		}
	}
	prompt.WriteString("\n")

	// Instructions for response format
	prompt.WriteString("Respond with a JSON object containing your decision:\n")
	prompt.WriteString("{\n")
	prompt.WriteString(`  "reasoning": "explain your decision",` + "\n")
	prompt.WriteString(`  "actions": [` + "\n")
	prompt.WriteString("    {\n")
	prompt.WriteString(`      "type": "chat.response",` + "\n")
	prompt.WriteString(`      "responseText": "your response to the user"` + "\n")
	prompt.WriteString("    },\n")
	prompt.WriteString("    {\n")
	prompt.WriteString(`      "type": "task.request",` + "\n")
	prompt.WriteString(`      "taskType": "the type of task",` + "\n")
	prompt.WriteString(`      "targetAgent": "the agent id from the list above"` + "\n")
	prompt.WriteString("    }\n")
	prompt.WriteString("  ]\n")
	prompt.WriteString("}\n\n")
	prompt.WriteString("Action types:\n")
	prompt.WriteString("- chat.response: Send a message to the user (has 'responseText' field)\n")
	prompt.WriteString("- task.request: Delegate a task to an agent (has 'taskType' and 'targetAgent' fields). IMPORTANT: 'targetAgent' must be the agent id value (e.g. 'agent_mp3'), NOT the display name.\n")
	prompt.WriteString("- status.request: Ask an agent for its current status (has 'targetAgent' field)\n\n")
	prompt.WriteString("Decision principles:\n")
	prompt.WriteString("- Examine the available agents' skills and descriptions above. Only use agents\n")
	prompt.WriteString("  that are listed — never invent or assume agents that aren't shown.\n")
	prompt.WriteString("- For a user request: if an agent's skills match the request, delegate to it.\n")
	prompt.WriteString("  If no agent matches, respond directly.\n")
	prompt.WriteString("- For a task result: check whether another available agent could further\n")
	prompt.WriteString("  process or enrich this output based on its skill descriptions.\n")
	prompt.WriteString("  If yes, chain the result to that agent with a brief user acknowledgment.\n")
	prompt.WriteString("  If no, present the content directly to the user.\n")
	prompt.WriteString("- When presenting content directly, return it as-is. Self-contained content\n")
	prompt.WriteString("  (transcriptions, summaries, analyses) is already meaningful — do not wrap\n")
	prompt.WriteString("  it in \"Here's the result:\" or similar boilerplate.\n")
	prompt.WriteString("- If the user asks about what an agent is doing or its status, use status.request\n")
	prompt.WriteString("  to query the agent. Pair it with a brief chat.response acknowledgment.\n")
	prompt.WriteString("- You may include multiple actions: e.g., a chat.response acknowledgment\n")
	prompt.WriteString("  AND a task.request delegation in the same decision.\n")
	prompt.WriteString("- Always explain your reasoning: what agents you considered and why you chose\n")
	prompt.WriteString("  to delegate, chain, or respond directly.\n\n")
	prompt.WriteString("Now, decide what actions to take:")

	return prompt.String()
}

// queryVertexAI sends a prompt to VertexAI and returns the response
func (c *Client) queryVertexAI(ctx context.Context, prompt string) (string, error) {
	chat, err := c.client.Chats.Create(ctx, c.config.Model, nil, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create chat: %w", err)
	}

	result, err := chat.SendMessage(ctx, genai.Part{Text: prompt})
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	// Extract the text response from the result
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		part := result.Candidates[0].Content.Parts[0]
		if part.Text != "" {
			return part.Text, nil
		}
	}

	return "", fmt.Errorf("no response from VertexAI")
}

// parseDecision parses the LLM response into a Decision structure
func (c *Client) parseDecision(response string) (*llm.Decision, error) {
	// Try to extract JSON from the response
	// LLMs sometimes wrap JSON in markdown code blocks
	jsonStr := response
	if strings.Contains(response, "```json") {
		// Extract JSON from markdown code block
		start := strings.Index(response, "```json")
		if start != -1 {
			start += len("```json")
			end := strings.Index(response[start:], "```")
			if end != -1 {
				jsonStr = strings.TrimSpace(response[start : start+end])
			}
		}
	} else if strings.Contains(response, "```") {
		// Extract from generic code block
		start := strings.Index(response, "```")
		if start != -1 {
			start += 3
			end := strings.Index(response[start:], "```")
			if end != -1 {
				jsonStr = strings.TrimSpace(response[start : start+end])
			}
		}
	}

	// Try to find JSON object in the response
	if !strings.HasPrefix(strings.TrimSpace(jsonStr), "{") {
		// Search for first { and last }
		start := strings.Index(jsonStr, "{")
		end := strings.LastIndex(jsonStr, "}")
		if start != -1 && end != -1 && end > start {
			jsonStr = jsonStr[start : end+1]
		}
	}

	// Parse the JSON
	var rawDecision struct {
		Reasoning string `json:"reasoning"`
		Actions   []struct {
			Type         string `json:"type"`
			ResponseText string `json:"responseText,omitempty"`
			TaskType     string `json:"taskType,omitempty"`
			TargetAgent  string `json:"targetAgent,omitempty"`
		} `json:"actions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawDecision); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (response: %s)", err, response)
	}

	// Convert to llm.Decision
	decision := &llm.Decision{
		Reasoning: rawDecision.Reasoning,
		Actions:   make([]llm.Action, len(rawDecision.Actions)),
	}

	for i, rawAction := range rawDecision.Actions {
		decision.Actions[i] = llm.Action{
			Type:         rawAction.Type,
			ResponseText: rawAction.ResponseText,
			TaskType:     rawAction.TaskType,
			TargetAgent:  rawAction.TargetAgent,
		}
	}

	// Validate at least one action
	if len(decision.Actions) == 0 {
		return nil, fmt.Errorf("decision must contain at least one action")
	}

	return decision, nil
}
