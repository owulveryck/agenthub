package llm

import (
	"context"
	"fmt"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

// MockClient is a mock LLM client for testing.
// It allows you to define custom decision logic via a DecideFunc.
type MockClient struct {
	// DecideFunc is called when Decide is invoked.
	// If nil, returns a simple echo response.
	DecideFunc func(
		ctx context.Context,
		conversationHistory []*pb.Message,
		availableAgents []*pb.AgentCard,
		newEvent *pb.Message,
	) (*Decision, error)

	// Track calls for testing
	CallCount int
	LastEvent *pb.Message
}

// NewMockClient creates a new mock LLM client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// NewMockClientWithFunc creates a mock client with a custom decide function.
func NewMockClientWithFunc(fn func(
	ctx context.Context,
	conversationHistory []*pb.Message,
	availableAgents []*pb.AgentCard,
	newEvent *pb.Message,
) (*Decision, error)) *MockClient {
	return &MockClient{
		DecideFunc: fn,
	}
}

// Decide implements the Client interface.
func (m *MockClient) Decide(
	ctx context.Context,
	conversationHistory []*pb.Message,
	availableAgents []*pb.AgentCard,
	newEvent *pb.Message,
) (*Decision, error) {
	m.CallCount++
	m.LastEvent = newEvent

	// Use custom function if provided
	if m.DecideFunc != nil {
		return m.DecideFunc(ctx, conversationHistory, availableAgents, newEvent)
	}

	// Default: simple echo behavior
	if newEvent == nil {
		return &Decision{
			Reasoning: "No new event, nothing to do",
			Actions:   []Action{},
		}, nil
	}

	// Extract user message
	var userText string
	if len(newEvent.Content) > 0 {
		userText = newEvent.Content[0].GetText()
	}

	// Default response: acknowledge the user
	return &Decision{
		Reasoning: "User sent a message, acknowledging",
		Actions: []Action{
			{
				Type:         "chat.response",
				ResponseText: fmt.Sprintf("I received your message: %s", userText),
			},
		},
	}, nil
}

// SimpleEchoDecider returns a decision function that echoes user messages.
func SimpleEchoDecider() func(context.Context, []*pb.Message, []*pb.AgentCard, *pb.Message) (*Decision, error) {
	return func(ctx context.Context, history []*pb.Message, agents []*pb.AgentCard, event *pb.Message) (*Decision, error) {
		if event == nil {
			return &Decision{Actions: []Action{}}, nil
		}

		var text string
		if len(event.Content) > 0 {
			text = event.Content[0].GetText()
		}

		return &Decision{
			Reasoning: "Echoing user message",
			Actions: []Action{
				{
					Type:         "chat.response",
					ResponseText: fmt.Sprintf("Echo: %s", text),
				},
			},
		}, nil
	}
}

// TaskDispatcherDecider returns a decision function that dispatches tasks to agents.
// This is useful for testing task routing.
// It intelligently handles both user requests and task results:
// - USER messages: dispatch task to agent
// - AGENT task results: synthesize final response (no new task)
func TaskDispatcherDecider(taskType, targetAgent string) func(context.Context, []*pb.Message, []*pb.AgentCard, *pb.Message) (*Decision, error) {
	return func(ctx context.Context, history []*pb.Message, agents []*pb.AgentCard, event *pb.Message) (*Decision, error) {
		// Check if this is a task result (from an agent)
		if event.GetRole() == pb.Role_ROLE_AGENT && event.GetTaskId() != "" {
			// This is a task result - synthesize final response
			var resultText string
			if len(event.Content) > 0 {
				resultText = event.Content[0].GetText()
			}

			return &Decision{
				Reasoning: fmt.Sprintf("Task result received from %s, sending final response to user", targetAgent),
				Actions: []Action{
					{
						Type:         "chat.response",
						ResponseText: fmt.Sprintf("Task completed! Result: %s", resultText),
					},
				},
			}, nil
		}

		// This is a user message - dispatch task
		return &Decision{
			Reasoning: fmt.Sprintf("User request received, dispatching %s task to %s", taskType, targetAgent),
			Actions: []Action{
				{
					Type:         "chat.response",
					ResponseText: fmt.Sprintf("I'll start the %s task for you.", taskType),
				},
				{
					Type:        "task.request",
					TaskType:    taskType,
					TargetAgent: targetAgent,
					TaskPayload: map[string]interface{}{
						"input": event.Content[0].GetText(),
					},
				},
			},
		}, nil
	}
}
