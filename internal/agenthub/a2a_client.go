package agenthub

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/observability"
)

// A2ATaskPublisher provides an abstraction for publishing A2A tasks with observability
type A2ATaskPublisher struct {
	Client         pb.AgentHubClient
	TraceManager   *observability.TraceManager
	MetricsManager *observability.MetricsManager
	Logger         interface {
		InfoContext(ctx context.Context, msg string, args ...interface{})
	}
	ComponentName string
	AgentID       string
}

// A2APublishTaskRequest contains all parameters needed to publish an A2A task
type A2APublishTaskRequest struct {
	TaskType         string
	Content          []*pb.Part // A2A-compliant content parts
	RequesterAgentID string
	ResponderAgentID string
	Priority         pb.Priority
	ContextID        string // Optional context grouping
}

// PublishTask publishes an A2A task with automatic correlation ID generation and observability
func (tp *A2ATaskPublisher) PublishTask(ctx context.Context, req *A2APublishTaskRequest) (*pb.Task, error) {
	// Start tracing for task publishing
	ctx, span := tp.TraceManager.StartPublishSpan(ctx, tp.ComponentName, req.ResponderAgentID, req.TaskType)
	defer span.End()

	// Start timing
	timer := tp.MetricsManager.StartTimer()
	defer timer(ctx, req.TaskType, tp.ComponentName)

	// Generate unique IDs
	taskID := fmt.Sprintf("task_%s_%d", req.TaskType, time.Now().Unix())
	messageID := fmt.Sprintf("msg_%s_%d", req.TaskType, time.Now().Unix())
	contextID := req.ContextID
	if contextID == "" {
		contextID = fmt.Sprintf("ctx_%s_%d", req.TaskType, time.Now().Unix())
	}

	tp.Logger.InfoContext(ctx, "Publishing A2A task",
		"task_id", taskID,
		"task_type", req.TaskType,
		"responder_agent_id", req.ResponderAgentID,
		"context_id", contextID,
	)

	// Create A2A message for the task
	message := &pb.Message{
		MessageId: messageID,
		ContextId: contextID,
		TaskId:    taskID,
		Role:      pb.Role_ROLE_USER,
		Content:   req.Content,
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":  structpb.NewStringValue(req.TaskType),
				"publisher":  structpb.NewStringValue(req.RequesterAgentID),
				"created_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	// Create task object
	task := &pb.Task{
		Id:        taskID,
		ContextId: contextID,
		Status: &pb.TaskStatus{
			State:     pb.TaskState_TASK_STATE_SUBMITTED,
			Timestamp: timestamppb.Now(),
			Update:    message,
		},
		History: []*pb.Message{message},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"task_type":          structpb.NewStringValue(req.TaskType),
				"requester_agent_id": structpb.NewStringValue(req.RequesterAgentID),
				"responder_agent_id": structpb.NewStringValue(req.ResponderAgentID),
				"priority":           structpb.NewStringValue(req.Priority.String()),
				"created_at":         structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	// Publish the message through the broker
	publishReq := &pb.PublishMessageRequest{
		Message: message,
		Routing: &pb.AgentEventMetadata{
			FromAgentId: req.RequesterAgentID,
			ToAgentId:   req.ResponderAgentID,
			EventType:   "task_message",
			Priority:    req.Priority,
		},
	}

	res, err := tp.Client.PublishMessage(ctx, publishReq)
	if err != nil {
		tp.Logger.InfoContext(ctx, "Error publishing A2A task",
			"task_id", taskID,
			"error", err,
		)
		tp.TraceManager.RecordError(span, err)
		tp.MetricsManager.IncrementEventErrors(ctx, req.TaskType, tp.ComponentName, "grpc_error")
		return nil, err
	}

	if !res.GetSuccess() {
		err := fmt.Errorf("failed to publish A2A task: %s", res.GetError())
		tp.Logger.InfoContext(ctx, "Failed to publish A2A task",
			"task_id", taskID,
			"error", res.GetError(),
		)
		tp.TraceManager.RecordError(span, err)
		tp.MetricsManager.IncrementEventErrors(ctx, req.TaskType, tp.ComponentName, "publish_failed")
		return nil, err
	}

	tp.Logger.InfoContext(ctx, "A2A task published successfully",
		"task_id", taskID,
		"task_type", req.TaskType,
		"event_id", res.GetEventId(),
	)

	// Record successful metrics
	tp.MetricsManager.IncrementEventsProcessed(ctx, req.TaskType, tp.ComponentName, true)
	tp.MetricsManager.IncrementEventsPublished(ctx, req.TaskType, req.ResponderAgentID)
	tp.TraceManager.SetSpanSuccess(span)

	return task, nil
}

// A2ATaskSubscriber provides abstraction for subscribing to and processing A2A tasks
type A2ATaskSubscriber struct {
	Client       *AgentHubClient
	AgentID      string
	TaskHandlers map[string]A2ATaskHandler
}

// A2ATaskHandler defines the interface for handling different A2A task types
type A2ATaskHandler func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string)

// NewA2ATaskSubscriber creates a new A2A task subscriber
func NewA2ATaskSubscriber(client *AgentHubClient, agentID string) *A2ATaskSubscriber {
	return &A2ATaskSubscriber{
		Client:       client,
		AgentID:      agentID,
		TaskHandlers: make(map[string]A2ATaskHandler),
	}
}

// RegisterTaskHandler registers a handler for a specific task type
func (ts *A2ATaskSubscriber) RegisterTaskHandler(taskType string, handler A2ATaskHandler) {
	ts.TaskHandlers[taskType] = handler
}

// RegisterDefaultHandlers registers default handlers for common task types
func (ts *A2ATaskSubscriber) RegisterDefaultHandlers() {
	ts.RegisterTaskHandler("greeting", ts.handleGreetingTask)
	ts.RegisterTaskHandler("math_calculation", ts.handleMathTask)
	ts.RegisterTaskHandler("random_number", ts.handleRandomNumberTask)
}

// SubscribeToTasks subscribes to A2A tasks and processes them using registered handlers
func (ts *A2ATaskSubscriber) SubscribeToTasks(ctx context.Context) error {
	ts.Client.Logger.InfoContext(ctx, "Subscribing to A2A tasks", "agent_id", ts.AgentID)

	req := &pb.SubscribeToTasksRequest{
		AgentId: ts.AgentID,
	}

	stream, err := ts.Client.Client.SubscribeToTasks(ctx, req)
	if err != nil {
		ts.Client.Logger.ErrorContext(ctx, "Failed to subscribe to A2A tasks", "error", err)
		return err
	}

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			ts.Client.Logger.InfoContext(ctx, "A2A task stream ended")
			break
		}
		if err != nil {
			ts.Client.Logger.ErrorContext(ctx, "Error receiving A2A task event", "error", err)
			ts.Client.MetricsManager.IncrementEventErrors(ctx, "a2a_task_subscription", ts.AgentID, "receive_error")
			return err
		}

		// Process event based on type
		switch payload := event.GetPayload().(type) {
		case *pb.AgentEvent_Message:
			if payload.Message.GetTaskId() != "" {
				go ts.processTaskMessage(ctx, payload.Message)
			}
		case *pb.AgentEvent_Task:
			go ts.processTask(ctx, payload.Task)
		}
	}

	return nil
}

// processTaskMessage processes a task message
func (ts *A2ATaskSubscriber) processTaskMessage(ctx context.Context, message *pb.Message) {
	taskID := message.GetTaskId()
	if taskID == "" {
		return
	}

	// Get the full task
	taskReq := &pb.GetTaskRequest{
		TaskId: taskID,
	}

	task, err := ts.Client.Client.GetTask(ctx, taskReq)
	if err != nil {
		ts.Client.Logger.ErrorContext(ctx, "Failed to get task for message",
			"task_id", taskID,
			"message_id", message.GetMessageId(),
			"error", err,
		)
		return
	}

	ts.processTask(ctx, task)
}

// processTask processes a complete A2A task
func (ts *A2ATaskSubscriber) processTask(ctx context.Context, task *pb.Task) {
	// Extract task type from metadata
	taskType := ""
	if task.Metadata != nil && task.Metadata.Fields != nil {
		if taskTypeValue, ok := task.Metadata.Fields["task_type"]; ok {
			taskType = taskTypeValue.GetStringValue()
		}
	}

	if taskType == "" {
		ts.Client.Logger.ErrorContext(ctx, "Task missing task_type in metadata",
			"task_id", task.GetId(),
		)
		return
	}

	// Get the initial message from history
	var initialMessage *pb.Message
	for _, msg := range task.History {
		if msg.Role == pb.Role_ROLE_USER && msg.TaskId == task.Id {
			initialMessage = msg
			break
		}
	}

	if initialMessage == nil {
		ts.Client.Logger.ErrorContext(ctx, "No user message found in task history",
			"task_id", task.GetId(),
		)
		return
	}

	// Look up handler for this task type
	var artifact *pb.Artifact
	var status pb.TaskState
	var errorMessage string

	if handler, ok := ts.TaskHandlers[taskType]; ok {
		artifact, status, errorMessage = handler(ctx, task, initialMessage)
	} else {
		// Unknown task type
		status = pb.TaskState_TASK_STATE_FAILED
		errorMessage = fmt.Sprintf("Unknown task type: %s", taskType)
	}

	// Publish task completion
	ts.publishTaskCompletion(ctx, task, artifact, status, errorMessage)
}

// publishTaskCompletion publishes task completion with artifact
func (ts *A2ATaskSubscriber) publishTaskCompletion(ctx context.Context, task *pb.Task, artifact *pb.Artifact, status pb.TaskState, errorMessage string) {
	// Create completion message
	completionMessage := &pb.Message{
		MessageId: fmt.Sprintf("completion_%s_%d", task.GetId(), time.Now().Unix()),
		ContextId: task.GetContextId(),
		TaskId:    task.GetId(),
		Role:      pb.Role_ROLE_AGENT,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: fmt.Sprintf("Task completed with status: %s", status.String()),
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"executor_agent_id": structpb.NewStringValue(ts.AgentID),
				"completed_at":      structpb.NewStringValue(time.Now().Format(time.RFC3339)),
				"status":            structpb.NewStringValue(status.String()),
			},
		},
	}

	// Publish status update
	statusUpdate := &pb.TaskStatusUpdateEvent{
		TaskId:    task.GetId(),
		ContextId: task.GetContextId(),
		Status: &pb.TaskStatus{
			State:     status,
			Update:    completionMessage,
			Timestamp: timestamppb.Now(),
		},
		Final: true,
	}

	_, err := ts.Client.Client.PublishTaskUpdate(ctx, &pb.PublishTaskUpdateRequest{
		Update: statusUpdate,
		Routing: &pb.AgentEventMetadata{
			FromAgentId: ts.AgentID,
			EventType:   "task_completion",
			Priority:    pb.Priority_PRIORITY_MEDIUM,
		},
	})

	if err != nil {
		ts.Client.Logger.ErrorContext(ctx, "Failed to publish task status update",
			"task_id", task.GetId(),
			"error", err,
		)
	}

	// Publish artifact if available
	if artifact != nil {
		artifactUpdate := &pb.TaskArtifactUpdateEvent{
			TaskId:    task.GetId(),
			ContextId: task.GetContextId(),
			Artifact:  artifact,
			Append:    false,
			LastChunk: true,
		}

		_, err := ts.Client.Client.PublishTaskArtifact(ctx, &pb.PublishTaskArtifactRequest{
			Artifact: artifactUpdate,
			Routing: &pb.AgentEventMetadata{
				FromAgentId: ts.AgentID,
				EventType:   "task_artifact",
				Priority:    pb.Priority_PRIORITY_MEDIUM,
			},
		})

		if err != nil {
			ts.Client.Logger.ErrorContext(ctx, "Failed to publish task artifact",
				"task_id", task.GetId(),
				"artifact_id", artifact.GetArtifactId(),
				"error", err,
			)
		}
	}

	ts.Client.Logger.InfoContext(ctx, "Task processing completed",
		"task_id", task.GetId(),
		"status", status.String(),
		"has_artifact", artifact != nil,
	)
}

// Default task handlers (A2A-compliant versions)

func (ts *A2ATaskSubscriber) handleGreetingTask(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
	// Extract name from message content
	name := ""
	for _, part := range message.Content {
		if textPart := part.GetText(); textPart != "" {
			// Simple extraction - in real implementation, use proper parsing
			name = "Claude" // Default for demo
		}
	}

	if name == "" {
		return nil, pb.TaskState_TASK_STATE_FAILED, "Name parameter is required"
	}

	greeting := fmt.Sprintf("Hello, %s! Nice to meet you.", name)

	artifact := &pb.Artifact{
		ArtifactId:  fmt.Sprintf("greeting_%s_%d", task.GetId(), time.Now().Unix()),
		Name:        "greeting_response",
		Description: "Greeting message response",
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: greeting,
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"processed_by": structpb.NewStringValue(ts.AgentID),
				"processed_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
}

func (ts *A2ATaskSubscriber) handleMathTask(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
	// For demo purposes, perform simple addition
	result := 42.0 + 58.0

	artifact := &pb.Artifact{
		ArtifactId:  fmt.Sprintf("math_%s_%d", task.GetId(), time.Now().Unix()),
		Name:        "math_result",
		Description: "Mathematical calculation result",
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Data{
					Data: &pb.DataPart{
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"operation": structpb.NewStringValue("add"),
								"a":         structpb.NewNumberValue(42.0),
								"b":         structpb.NewNumberValue(58.0),
								"result":    structpb.NewNumberValue(result),
							},
						},
					},
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"processed_by": structpb.NewStringValue(ts.AgentID),
				"processed_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
}

func (ts *A2ATaskSubscriber) handleRandomNumberTask(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
	// Generate a random number (for demo, use fixed value)
	randomNumber := 42

	artifact := &pb.Artifact{
		ArtifactId:  fmt.Sprintf("random_%s_%d", task.GetId(), time.Now().Unix()),
		Name:        "random_number",
		Description: "Generated random number",
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Data{
					Data: &pb.DataPart{
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"random_number": structpb.NewNumberValue(float64(randomNumber)),
								"seed":          structpb.NewNumberValue(12345),
							},
						},
					},
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"processed_by": structpb.NewStringValue(ts.AgentID),
				"processed_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
}
