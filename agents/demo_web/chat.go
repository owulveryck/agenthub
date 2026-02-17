package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	chatAgentID = "agent_demo_web"
)

// ChatClient wraps an AgentHubClient to send and receive chat messages.
type ChatClient struct {
	client    *agenthub.AgentHubClient
	hub       *Hub
	logger    *slog.Logger
	sessionID string
}

// NewChatClient creates a ChatClient. Call Start() to connect and begin receiving.
func NewChatClient(hub *Hub, logger *slog.Logger) (*ChatClient, error) {
	config := agenthub.NewGRPCConfig("demo_web")
	config.HealthPort = "8088"

	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AgentHub client: %w", err)
	}

	return &ChatClient{
		client:    client,
		hub:       hub,
		logger:    logger,
		sessionID: fmt.Sprintf("demo_web_%d", time.Now().Unix()),
	}, nil
}

// Start connects to the broker and begins listening for responses.
func (cc *ChatClient) Start(ctx context.Context) error {
	if err := cc.client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start AgentHub client: %w", err)
	}

	// Subscribe to messages destined for this agent
	go cc.receiveMessages(ctx)

	cc.logger.Info("chat client started", "agent_id", chatAgentID, "session", cc.sessionID)
	return nil
}

// Shutdown gracefully shuts down the chat client.
func (cc *ChatClient) Shutdown(ctx context.Context) error {
	return cc.client.Shutdown(ctx)
}

// SendMessage publishes a chat message to the broker (routed to cortex).
func (cc *ChatClient) SendMessage(ctx context.Context, text string) {
	message := &pb.Message{
		MessageId: fmt.Sprintf("demo_msg_%d", time.Now().UnixNano()),
		ContextId: cc.sessionID,
		Role:      pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{Part: &pb.Part_Text{Text: text}},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":  structpb.NewStringValue("chat_request"),
				"from_agent": structpb.NewStringValue(chatAgentID),
				"created_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	pubCtx, pubSpan := cc.client.TraceManager.StartA2AMessageSpan(
		ctx,
		"demo_web.publish_message",
		message.MessageId,
		message.Role.String(),
	)
	defer pubSpan.End()

	cc.client.TraceManager.AddA2AMessageAttributes(
		pubSpan,
		message.MessageId,
		message.ContextId,
		message.Role.String(),
		"chat_request",
		len(message.Content),
		message.Metadata != nil,
	)
	cc.client.TraceManager.AddComponentAttribute(pubSpan, "demo_web")

	_, err := cc.client.Client.PublishMessage(pubCtx, &pb.PublishMessageRequest{
		Message: message,
		Routing: &pb.AgentEventMetadata{
			FromAgentId: chatAgentID,
			ToAgentId:   "cortex",
			EventType:   "a2a.message.chat_request",
			Priority:    pb.Priority_PRIORITY_HIGH,
		},
	})

	if err != nil {
		cc.client.TraceManager.RecordError(pubSpan, err)
		cc.logger.Error("failed to send chat message", "error", err)
		cc.hub.Broadcast(WSMessage{
			Type:    "chat_response",
			Content: fmt.Sprintf("[Error sending message: %v]", err),
		})
	} else {
		cc.client.TraceManager.SetSpanSuccess(pubSpan)
	}
}

func (cc *ChatClient) receiveMessages(ctx context.Context) {
	stream, err := cc.client.Client.SubscribeToMessages(ctx, &pb.SubscribeToMessagesRequest{
		AgentId: chatAgentID,
	})
	if err != nil {
		cc.logger.Error("failed to subscribe to messages", "error", err)
		return
	}

	for {
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return
			}
			cc.logger.Error("error receiving message", "error", err)
			return
		}

		msg := event.GetMessage()
		if msg == nil {
			continue
		}

		// Filter: only our session (or task_result) and only AGENT role
		isTaskResult := false
		if msg.Metadata != nil && msg.Metadata.Fields != nil {
			if taskType, exists := msg.Metadata.Fields["task_type"]; exists {
				if taskType.GetStringValue() == "task_result" {
					isTaskResult = true
				}
			}
		}

		if (msg.ContextId == cc.sessionID || isTaskResult) && msg.Role == pb.Role_ROLE_AGENT {
			if msg.Metadata != nil && msg.Metadata.Fields != nil {
				if taskType, exists := msg.Metadata.Fields["task_type"]; exists {
					tv := taskType.GetStringValue()
					if tv == "chat_response" || tv == "task_result" {
						if len(msg.Content) > 0 {
							cc.hub.Broadcast(WSMessage{
								Type:    "chat_response",
								Content: msg.Content[0].GetText(),
							})
						}
					}
				}
			}
		}
	}
}
