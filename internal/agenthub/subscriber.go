package agenthub

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// TaskHandler defines the interface for handling different task types
type TaskHandler func(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string)

// TaskSubscriber provides abstraction for subscribing to and processing tasks
type TaskSubscriber struct {
	Client        *AgentHubClient
	TaskResultPub *TaskResultPublisher
	AgentID       string
	TaskHandlers  map[string]TaskHandler
	TaskProcessor *TaskProcessor
}

// NewTaskSubscriber creates a new task subscriber
func NewTaskSubscriber(client *AgentHubClient, agentID string) *TaskSubscriber {
	taskResultPub := &TaskResultPublisher{
		Client:         client.Client,
		TraceManager:   client.TraceManager,
		MetricsManager: client.MetricsManager,
		ComponentName:  "subscriber",
	}

	taskProcessor := &TaskProcessor{
		TraceManager:   client.TraceManager,
		MetricsManager: client.MetricsManager,
		Logger:         client.Logger,
		ComponentName:  "subscriber",
		AgentID:        agentID,
	}

	return &TaskSubscriber{
		Client:        client,
		TaskResultPub: taskResultPub,
		AgentID:       agentID,
		TaskHandlers:  make(map[string]TaskHandler),
		TaskProcessor: taskProcessor,
	}
}

// RegisterTaskHandler registers a handler for a specific task type
func (ts *TaskSubscriber) RegisterTaskHandler(taskType string, handler TaskHandler) {
	ts.TaskHandlers[taskType] = handler
}

// RegisterDefaultHandlers registers default handlers for common task types
func (ts *TaskSubscriber) RegisterDefaultHandlers() {
	ts.RegisterTaskHandler("greeting", ts.handleGreetingTask)
	ts.RegisterTaskHandler("math_calculation", ts.handleMathTask)
	ts.RegisterTaskHandler("random_number", ts.handleRandomNumberTask)
}

// SubscribeToTasks subscribes to tasks and processes them using registered handlers
func (ts *TaskSubscriber) SubscribeToTasks(ctx context.Context) error {
	ts.Client.Logger.InfoContext(ctx, "Subscribing to tasks", "agent_id", ts.AgentID)

	req := &pb.SubscribeToTasksRequest{
		AgentId: ts.AgentID,
	}

	stream, err := ts.Client.Client.SubscribeToTasks(ctx, req)
	if err != nil {
		ts.Client.Logger.ErrorContext(ctx, "Failed to subscribe to tasks", "error", err)
		return err
	}

	for {
		task, err := stream.Recv()
		if err == io.EOF {
			ts.Client.Logger.InfoContext(ctx, "Task stream ended")
			break
		}
		if err != nil {
			ts.Client.Logger.ErrorContext(ctx, "Error receiving task", "error", err)
			ts.Client.MetricsManager.IncrementEventErrors(ctx, "task_subscription", ts.AgentID, "receive_error")
			return err
		}

		// Process task in a separate goroutine
		go ts.processTask(ctx, task)
	}

	return nil
}

// SubscribeToTaskResults subscribes to task results
func (ts *TaskSubscriber) SubscribeToTaskResults(ctx context.Context) error {
	ts.Client.Logger.InfoContext(ctx, "Subscribing to task results", "agent_id", ts.AgentID)

	req := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: ts.AgentID,
	}

	stream, err := ts.Client.Client.SubscribeToTaskResults(ctx, req)
	if err != nil {
		ts.Client.Logger.ErrorContext(ctx, "Failed to subscribe to task results", "error", err)
		return err
	}

	for {
		result, err := stream.Recv()
		if err == io.EOF {
			ts.Client.Logger.InfoContext(ctx, "Task result stream ended")
			break
		}
		if err != nil {
			ts.Client.Logger.ErrorContext(ctx, "Error receiving task result", "error", err)
			ts.Client.MetricsManager.IncrementEventErrors(ctx, "task_result_subscription", ts.AgentID, "receive_error")
			return err
		}

		ts.Client.Logger.InfoContext(ctx, "Received task result",
			"task_id", result.GetTaskId(),
			"status", result.GetStatus().String(),
			"executor_agent_id", result.GetExecutorAgentId(),
		)

		ts.Client.MetricsManager.IncrementEventsProcessed(ctx, "task_result_received", ts.AgentID, true)
	}

	return nil
}

// processTask processes a task using registered handlers
func (ts *TaskSubscriber) processTask(ctx context.Context, task *pb.TaskMessage) {
	var result *structpb.Struct
	var status pb.TaskStatus
	var errorMessage string

	// Look up handler for this task type
	if handler, ok := ts.TaskHandlers[task.GetTaskType()]; ok {
		result, status, errorMessage = handler(ctx, task)
	} else {
		// Unknown task type
		status = pb.TaskStatus_TASK_STATUS_FAILED
		errorMessage = fmt.Sprintf("Unknown task type: %s", task.GetTaskType())
	}

	// Use TaskProcessor for observability (but get actual results from handler)
	ts.TaskProcessor.ProcessTask(ctx, task, ProcessTaskOptions{
		ProcessorFunc: func(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
			return result, status, errorMessage
		},
	})

	// Create and publish task result
	taskResult := &pb.TaskResult{
		TaskId:            task.GetTaskId(),
		Status:            status,
		Result:            result,
		ErrorMessage:      errorMessage,
		ExecutorAgentId:   ts.AgentID,
		CompletedAt:       timestamppb.Now(),
		ExecutionMetadata: &structpb.Struct{},
	}

	// Publish the result
	if err := ts.TaskResultPub.PublishTaskResult(ctx, taskResult); err != nil {
		ts.Client.Logger.ErrorContext(ctx, "Failed to publish task result",
			"task_id", task.GetTaskId(),
			"error", err,
		)
		ts.Client.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), ts.AgentID, "result_publish_error")
	} else {
		ts.Client.Logger.InfoContext(ctx, "Task completed and result published",
			"task_id", task.GetTaskId(),
			"status", status.String(),
		)
		ts.Client.MetricsManager.IncrementEventsProcessed(ctx, task.GetTaskType(), ts.AgentID, status == pb.TaskStatus_TASK_STATUS_COMPLETED)
	}
}

// Default task handlers

func (ts *TaskSubscriber) handleGreetingTask(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
	params := task.GetParameters()
	name := params.Fields["name"].GetStringValue()

	if name == "" {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Name parameter is required"
	}

	greeting := fmt.Sprintf("Hello, %s! Nice to meet you.", name)

	result, err := structpb.NewStruct(map[string]interface{}{
		"greeting":     greeting,
		"processed_by": ts.AgentID,
		"processed_at": time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to create result: %v", err)
	}

	return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}

func (ts *TaskSubscriber) handleMathTask(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
	params := task.GetParameters()
	operation := params.Fields["operation"].GetStringValue()
	a := params.Fields["a"].GetNumberValue()
	b := params.Fields["b"].GetNumberValue()

	var result float64
	var err error

	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Division by zero"
		}
		result = a / b
	default:
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Unknown operation: %s", operation)
	}

	resultStruct, err := structpb.NewStruct(map[string]interface{}{
		"operation":    operation,
		"a":            a,
		"b":            b,
		"result":       result,
		"processed_by": ts.AgentID,
		"processed_at": time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to create result: %v", err)
	}

	return resultStruct, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}

func (ts *TaskSubscriber) handleRandomNumberTask(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
	params := task.GetParameters()
	seed := int64(params.Fields["seed"].GetNumberValue())

	r := rand.New(rand.NewSource(seed))
	randomNumber := r.Intn(1000)

	result, err := structpb.NewStruct(map[string]interface{}{
		"seed":          seed,
		"random_number": randomNumber,
		"processed_by":  ts.AgentID,
		"processed_at":  time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to create result: %v", err)
	}

	return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}
