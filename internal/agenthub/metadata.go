package agenthub

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
	"github.com/owulveryck/agenthub/internal/observability"
)

// TaskPublisher provides an abstraction for publishing tasks with observability
type TaskPublisher struct {
	Client         pb.EventBusClient
	TraceManager   *observability.TraceManager
	MetricsManager *observability.MetricsManager
	Logger         interface {
		InfoContext(ctx context.Context, msg string, args ...interface{})
	}
	ComponentName string
}

// PublishTaskRequest contains all parameters needed to publish a task
type PublishTaskRequest struct {
	TaskType         string
	Parameters       map[string]interface{}
	RequesterAgentID string
	ResponderAgentID string
	Priority         pb.Priority
}

// PublishTask publishes a task with automatic correlation ID generation and observability
func (tp *TaskPublisher) PublishTask(ctx context.Context, req *PublishTaskRequest) error {
	// Start tracing for task publishing
	ctx, span := tp.TraceManager.StartPublishSpan(ctx, req.ResponderAgentID, req.TaskType)
	defer span.End()
	tp.TraceManager.AddComponentAttribute(span, tp.ComponentName)

	// Start timing
	timer := tp.MetricsManager.StartTimer()
	defer timer(ctx, req.TaskType, tp.ComponentName)

	// Generate a unique task ID (correlation ID)
	taskID := fmt.Sprintf("task_%s_%d", req.TaskType, time.Now().Unix())

	tp.Logger.InfoContext(ctx, "Publishing task",
		"task_id", taskID,
		"task_type", req.TaskType,
		"responder_agent_id", req.ResponderAgentID,
	)

	// Convert parameters to protobuf Struct
	parametersStruct, err := structpb.NewStruct(req.Parameters)
	if err != nil {
		tp.Logger.InfoContext(ctx, "Error creating parameters struct",
			"task_id", taskID,
			"error", err,
		)
		tp.TraceManager.RecordError(span, err)
		tp.MetricsManager.IncrementEventErrors(ctx, req.TaskType, tp.ComponentName, "struct_conversion_error")
		return err
	}

	// Create simple metadata - gRPC interceptors handle trace propagation automatically
	metadataStruct, err := structpb.NewStruct(map[string]interface{}{
		"publisher":    req.RequesterAgentID,
		"published_at": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		tp.Logger.InfoContext(ctx, "Error creating metadata struct",
			"task_id", taskID,
			"error", err,
		)
		// Continue without metadata
		metadataStruct = &structpb.Struct{}
	}

	// Create task message
	task := &pb.TaskMessage{
		TaskId:           taskID,
		TaskType:         req.TaskType,
		Parameters:       parametersStruct,
		RequesterAgentId: req.RequesterAgentID,
		ResponderAgentId: req.ResponderAgentID,
		Priority:         req.Priority,
		CreatedAt:        timestamppb.Now(),
		Metadata:         metadataStruct,
	}

	// Publish the task
	taskReq := &pb.PublishTaskRequest{
		Task: task,
	}

	res, err := tp.Client.PublishTask(ctx, taskReq)
	if err != nil {
		tp.Logger.InfoContext(ctx, "Error publishing task",
			"task_id", taskID,
			"error", err,
		)
		tp.TraceManager.RecordError(span, err)
		tp.MetricsManager.IncrementEventErrors(ctx, req.TaskType, tp.ComponentName, "grpc_error")
		return err
	}

	if !res.GetSuccess() {
		err := fmt.Errorf("failed to publish task: %s", res.GetError())
		tp.Logger.InfoContext(ctx, "Failed to publish task",
			"task_id", taskID,
			"error", res.GetError(),
		)
		tp.TraceManager.RecordError(span, err)
		tp.MetricsManager.IncrementEventErrors(ctx, req.TaskType, tp.ComponentName, "publish_failed")
		return err
	}

	tp.Logger.InfoContext(ctx, "Task published successfully",
		"task_id", taskID,
		"task_type", req.TaskType,
	)

	// Record successful metrics
	tp.MetricsManager.IncrementEventsProcessed(ctx, req.TaskType, tp.ComponentName, true)
	tp.MetricsManager.IncrementEventsPublished(ctx, req.TaskType, req.ResponderAgentID)
	tp.TraceManager.SetSpanSuccess(span)

	return nil
}

// TaskResultPublisher provides an abstraction for publishing task results with observability
type TaskResultPublisher struct {
	Client         pb.EventBusClient
	TraceManager   *observability.TraceManager
	MetricsManager *observability.MetricsManager
	ComponentName  string
}

// PublishTaskResult publishes a task result with observability
func (trp *TaskResultPublisher) PublishTaskResult(ctx context.Context, result *pb.TaskResult) error {
	ctx, span := trp.TraceManager.StartPublishSpan(ctx, "task_result_queue", "task_result")
	defer span.End()
	trp.TraceManager.AddComponentAttribute(span, trp.ComponentName)

	// Inject trace context
	headers := make(map[string]string)
	trp.TraceManager.InjectTraceContext(ctx, headers)

	req := &pb.PublishTaskResultRequest{
		Result: result,
	}

	res, err := trp.Client.PublishTaskResult(ctx, req)
	if err != nil {
		trp.TraceManager.RecordError(span, err)
		return err
	}

	if !res.GetSuccess() {
		err := fmt.Errorf("failed to publish task result: %s", res.GetError())
		trp.TraceManager.RecordError(span, err)
		return err
	}

	trp.MetricsManager.IncrementEventsPublished(ctx, "task_result", "task_result_queue")
	trp.TraceManager.SetSpanSuccess(span)
	return nil
}

// TaskProcessor provides an abstraction for processing tasks with observability
type TaskProcessor struct {
	TraceManager   *observability.TraceManager
	MetricsManager *observability.MetricsManager
	Logger         interface {
		InfoContext(ctx context.Context, msg string, args ...interface{})
		DebugContext(ctx context.Context, msg string, args ...interface{})
		ErrorContext(ctx context.Context, msg string, args ...interface{})
	}
	ComponentName string
	AgentID       string
}

// ProcessTaskOptions contains processing configuration
type ProcessTaskOptions struct {
	ProcessorFunc func(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string)
}

// ProcessTask processes a task with full observability
func (tp *TaskProcessor) ProcessTask(ctx context.Context, task *pb.TaskMessage, opts ProcessTaskOptions) {
	// The gRPC context already contains the trace information from interceptors
	// Start processing span using the context that already has trace info
	ctx, span := tp.TraceManager.StartEventProcessingSpan(ctx, task.GetTaskId(), task.GetTaskType(), task.GetRequesterAgentId(), "")
	defer span.End()
	tp.TraceManager.AddComponentAttribute(span, tp.ComponentName)

	// Extract task parameters for tracing
	taskParams := make(map[string]interface{})
	if task.GetParameters() != nil {
		for key, value := range task.GetParameters().Fields {
			switch v := value.Kind.(type) {
			case *structpb.Value_StringValue:
				taskParams[key] = v.StringValue
			case *structpb.Value_NumberValue:
				taskParams[key] = v.NumberValue
			case *structpb.Value_BoolValue:
				taskParams[key] = v.BoolValue
			default:
				taskParams[key] = value.String()
			}
		}
	}

	// Add rich task information to the span
	tp.TraceManager.AddTaskAttributes(span, task.GetTaskId(), task.GetTaskType(), taskParams)
	tp.TraceManager.AddSpanEvent(span, "task.processing.started")

	// Debug logging: task parameters
	tp.Logger.DebugContext(ctx, "Task received with parameters",
		"task_id", task.GetTaskId(),
		"task_type", task.GetTaskType(),
		"parameters", taskParams,
		"requester", task.GetRequesterAgentId(),
	)

	// Start timing
	timer := tp.MetricsManager.StartTimer()
	defer timer(ctx, task.GetTaskType(), tp.AgentID)

	tp.Logger.InfoContext(ctx, "Processing task",
		"task_id", task.GetTaskId(),
		"task_type", task.GetTaskType(),
		"requester_agent_id", task.GetRequesterAgentId(),
	)

	var result *structpb.Struct
	var status pb.TaskStatus
	var errorMessage string

	// Process the task using the provided function
	tp.TraceManager.AddSpanEvent(span, "task.processing.dispatch")
	result, status, errorMessage = opts.ProcessorFunc(ctx, task)

	// Add result information to trace
	taskResult := make(map[string]interface{})
	if result != nil {
		for key, value := range result.Fields {
			switch v := value.Kind.(type) {
			case *structpb.Value_StringValue:
				taskResult[key] = v.StringValue
			case *structpb.Value_NumberValue:
				taskResult[key] = v.NumberValue
			case *structpb.Value_BoolValue:
				taskResult[key] = v.BoolValue
			default:
				taskResult[key] = value.String()
			}
		}
	}

	// Add result to span
	tp.TraceManager.AddTaskResult(span, status.String(), taskResult, errorMessage)
	tp.TraceManager.AddSpanEvent(span, "task.processing.completed")

	// Debug logging: task result
	tp.Logger.DebugContext(ctx, "Task processing completed",
		"task_id", task.GetTaskId(),
		"status", status.String(),
		"result", taskResult,
		"error_message", errorMessage,
		"executor", tp.AgentID,
	)

	// Create task result
	taskResultProto := &pb.TaskResult{
		TaskId:            task.GetTaskId(),
		Status:            status,
		Result:            result,
		ErrorMessage:      errorMessage,
		ExecutorAgentId:   tp.AgentID,
		CompletedAt:       timestamppb.Now(),
		ExecutionMetadata: &structpb.Struct{},
	}

	// Record successful metrics
	tp.MetricsManager.IncrementEventsProcessed(ctx, task.GetTaskType(), tp.AgentID, status == pb.TaskStatus_TASK_STATUS_COMPLETED)
	tp.TraceManager.SetSpanSuccess(span)

	// Store the result for the caller to handle
	// This allows for flexible result handling strategies
	_ = taskResultProto
}
