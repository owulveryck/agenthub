//go:build observability
// +build observability

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
	"github.com/owulveryck/agenthub/internal/observability"
	"go.opentelemetry.io/otel/attribute"
)

const (
	agentHubAddr = "localhost:50051"       // Address of the AgentHub broker server
	agentID      = "agent_demo_subscriber" // This agent's ID
)

type ObservableSubscriber struct {
	client         pb.EventBusClient
	obs            *observability.Observability
	traceManager   *observability.TraceManager
	metricsManager *observability.MetricsManager
	healthServer   *observability.HealthServer
	logger         *slog.Logger
}

func NewObservableSubscriber() (*ObservableSubscriber, error) {
	// Initialize observability
	config := observability.DefaultConfig("agenthub-subscriber")
	obs, err := observability.NewObservability(config)
	if err != nil {
		return nil, err
	}

	// Initialize metrics manager
	metricsManager, err := observability.NewMetricsManager(obs.Meter)
	if err != nil {
		return nil, err
	}

	// Initialize trace manager
	traceManager := observability.NewTraceManager(config.ServiceName)

	// Initialize health server
	healthServer := observability.NewHealthServer("8082", config.ServiceName, config.ServiceVersion)

	// Add health checks
	healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
		return nil // Simple health check
	}))

	// Set up gRPC connection
	conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent hub: %w", err)
	}

	client := pb.NewEventBusClient(conn)

	// Add gRPC connection health check
	healthServer.AddChecker("agenthub_connection", observability.NewGRPCHealthChecker("agenthub_connection", agentHubAddr))

	return &ObservableSubscriber{
		client:         client,
		obs:            obs,
		traceManager:   traceManager,
		metricsManager: metricsManager,
		healthServer:   healthServer,
		logger:         obs.Logger,
	}, nil
}

func (s *ObservableSubscriber) processTask(ctx context.Context, task *pb.TaskMessage) {
	// Extract trace context from task metadata
	if metadata := task.GetMetadata(); metadata != nil {
		if traceHeaders, ok := metadata.Fields["trace_headers"]; ok {
			if headersStruct := traceHeaders.GetStructValue(); headersStruct != nil {
				headers := make(map[string]string)
				for k, v := range headersStruct.Fields {
					headers[k] = v.GetStringValue()
				}
				ctx = s.traceManager.ExtractTraceContext(ctx, headers)
			}
		}
	}

	// Start processing span
	ctx, span := s.traceManager.StartEventProcessingSpan(ctx, task.GetTaskId(), task.GetTaskType(), task.GetRequesterAgentId(), "")
	defer span.End()

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
	s.traceManager.AddTaskAttributes(span, task.GetTaskId(), task.GetTaskType(), taskParams)
	s.traceManager.AddSpanEvent(span, "task.processing.started",
		attribute.String("requester", task.GetRequesterAgentId()),
		attribute.String("executor", agentID))

	// Debug logging: task parameters
	s.logger.DebugContext(ctx, "Task received with parameters",
		slog.String("task_id", task.GetTaskId()),
		slog.String("task_type", task.GetTaskType()),
		slog.Any("parameters", taskParams),
		slog.String("requester", task.GetRequesterAgentId()))

	// Start timing
	timer := s.metricsManager.StartTimer()
	defer timer(ctx, task.GetTaskType(), agentID)

	s.logger.InfoContext(ctx, "Processing task",
		slog.String("task_id", task.GetTaskId()),
		slog.String("task_type", task.GetTaskType()),
		slog.String("requester_agent_id", task.GetRequesterAgentId()),
	)

	var result *structpb.Struct
	var status pb.TaskStatus
	var errorMessage string

	// Process different task types
	s.traceManager.AddSpanEvent(span, "task.processing.dispatch",
		attribute.String("task_type", task.GetTaskType()))

	switch task.GetTaskType() {
	case "greeting":
		s.traceManager.AddSpanEvent(span, "task.type.greeting.started")
		if name, ok := taskParams["name"]; ok {
			s.logger.DebugContext(ctx, "Processing greeting task", slog.Any("name", name))
		}
		result, status, errorMessage = s.processGreetingTask(ctx, task)
	case "math_calculation":
		s.traceManager.AddSpanEvent(span, "task.type.math.started")
		s.logger.DebugContext(ctx, "Processing math calculation",
			slog.Any("operation", taskParams["operation"]),
			slog.Any("a", taskParams["a"]),
			slog.Any("b", taskParams["b"]))
		result, status, errorMessage = s.processMathTask(ctx, task)
	case "random_number":
		s.traceManager.AddSpanEvent(span, "task.type.random.started")
		s.logger.DebugContext(ctx, "Processing random number task")
		result, status, errorMessage = s.processRandomNumberTask(ctx, task)
	default:
		s.traceManager.AddSpanEvent(span, "task.type.unknown.error",
			attribute.String("unknown_type", task.GetTaskType()))
		errorMessage = fmt.Sprintf("Unknown task type: %s", task.GetTaskType())
		status = pb.TaskStatus_TASK_STATUS_FAILED
		s.logger.ErrorContext(ctx, "Unknown task type",
			slog.String("task_id", task.GetTaskId()),
			slog.String("task_type", task.GetTaskType()),
		)
		s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), agentID, "unknown_task_type")
	}

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
	s.traceManager.AddTaskResult(span, status.String(), taskResult, errorMessage)
	s.traceManager.AddSpanEvent(span, "task.processing.completed",
		attribute.String("status", status.String()),
		attribute.String("executor", agentID))

	// Debug logging: task result
	s.logger.DebugContext(ctx, "Task processing completed",
		slog.String("task_id", task.GetTaskId()),
		slog.String("status", status.String()),
		slog.Any("result", taskResult),
		slog.String("error_message", errorMessage),
		slog.String("executor", agentID))

	// Create task result
	taskResultProto := &pb.TaskResult{
		TaskId:            task.GetTaskId(),
		Status:            status,
		Result:            result,
		ErrorMessage:      errorMessage,
		ExecutorAgentId:   agentID,
		CompletedAt:       timestamppb.Now(),
		ExecutionMetadata: &structpb.Struct{},
	}

	// Publish the result
	if err := s.publishTaskResult(ctx, taskResultProto); err != nil {
		s.logger.ErrorContext(ctx, "Failed to publish task result",
			slog.String("task_id", task.GetTaskId()),
			slog.Any("error", err),
		)
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), agentID, "result_publish_error")
	} else {
		s.logger.InfoContext(ctx, "Task completed and result published",
			slog.String("task_id", task.GetTaskId()),
			slog.String("status", status.String()),
		)
		s.metricsManager.IncrementEventsProcessed(ctx, task.GetTaskType(), agentID, status == pb.TaskStatus_TASK_STATUS_COMPLETED)
		s.traceManager.SetSpanSuccess(span)
	}
}

func (s *ObservableSubscriber) processGreetingTask(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
	params := task.GetParameters()
	name := params.Fields["name"].GetStringValue()

	if name == "" {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, "Name parameter is required"
	}

	greeting := fmt.Sprintf("Hello, %s! Nice to meet you.", name)

	result, err := structpb.NewStruct(map[string]interface{}{
		"greeting":     greeting,
		"processed_by": agentID,
		"processed_at": time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to create result: %v", err)
	}

	return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}

func (s *ObservableSubscriber) processMathTask(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
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
		"processed_by": agentID,
		"processed_at": time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to create result: %v", err)
	}

	return resultStruct, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}

func (s *ObservableSubscriber) processRandomNumberTask(ctx context.Context, task *pb.TaskMessage) (*structpb.Struct, pb.TaskStatus, string) {
	params := task.GetParameters()
	seed := int64(params.Fields["seed"].GetNumberValue())

	r := rand.New(rand.NewSource(seed))
	randomNumber := r.Intn(1000)

	result, err := structpb.NewStruct(map[string]interface{}{
		"seed":          seed,
		"random_number": randomNumber,
		"processed_by":  agentID,
		"processed_at":  time.Now().Format(time.RFC3339),
	})

	if err != nil {
		return nil, pb.TaskStatus_TASK_STATUS_FAILED, fmt.Sprintf("Failed to create result: %v", err)
	}

	return result, pb.TaskStatus_TASK_STATUS_COMPLETED, ""
}

func (s *ObservableSubscriber) publishTaskResult(ctx context.Context, result *pb.TaskResult) error {
	ctx, span := s.traceManager.StartPublishSpan(ctx, "task_result_queue", "task_result")
	defer span.End()

	// Inject trace context
	headers := make(map[string]string)
	s.traceManager.InjectTraceContext(ctx, headers)

	req := &pb.PublishTaskResultRequest{
		Result: result,
	}

	res, err := s.client.PublishTaskResult(ctx, req)
	if err != nil {
		s.traceManager.RecordError(span, err)
		return err
	}

	if !res.GetSuccess() {
		err := fmt.Errorf("failed to publish task result: %s", res.GetError())
		s.traceManager.RecordError(span, err)
		return err
	}

	s.metricsManager.IncrementEventsPublished(ctx, "task_result", "task_result_queue")
	s.traceManager.SetSpanSuccess(span)
	return nil
}

func (s *ObservableSubscriber) subscribeToTasks(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Subscribing to tasks", slog.String("agent_id", agentID))

	req := &pb.SubscribeToTasksRequest{
		AgentId: agentID,
	}

	stream, err := s.client.SubscribeToTasks(ctx, req)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to subscribe to tasks", slog.Any("error", err))
		return err
	}

	for {
		task, err := stream.Recv()
		if err == io.EOF {
			s.logger.InfoContext(ctx, "Task stream ended")
			break
		}
		if err != nil {
			s.logger.ErrorContext(ctx, "Error receiving task", slog.Any("error", err))
			s.metricsManager.IncrementEventErrors(ctx, "task_subscription", agentID, "receive_error")
			return err
		}

		// Process task in a separate goroutine
		go s.processTask(ctx, task)
	}

	return nil
}

func (s *ObservableSubscriber) subscribeToTaskResults(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Subscribing to task results", slog.String("agent_id", agentID))

	req := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: agentID,
	}

	stream, err := s.client.SubscribeToTaskResults(ctx, req)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to subscribe to task results", slog.Any("error", err))
		return err
	}

	for {
		result, err := stream.Recv()
		if err == io.EOF {
			s.logger.InfoContext(ctx, "Task result stream ended")
			break
		}
		if err != nil {
			s.logger.ErrorContext(ctx, "Error receiving task result", slog.Any("error", err))
			s.metricsManager.IncrementEventErrors(ctx, "task_result_subscription", agentID, "receive_error")
			return err
		}

		s.logger.InfoContext(ctx, "Received task result",
			slog.String("task_id", result.GetTaskId()),
			slog.String("status", result.GetStatus().String()),
			slog.String("executor_agent_id", result.GetExecutorAgentId()),
		)

		s.metricsManager.IncrementEventsProcessed(ctx, "task_result_received", agentID, true)
	}

	return nil
}

func (s *ObservableSubscriber) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Starting observable subscriber")

	// Start health server
	go func() {
		s.logger.Info("Starting health server on port 8082")
		if err := s.healthServer.Start(ctx); err != nil {
			s.logger.Error("Health server failed", slog.Any("error", err))
		}
	}()

	// Start metrics collection
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.metricsManager.UpdateSystemMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Task subscriber: Listen for tasks assigned to this agent
	go func() {
		if err := s.subscribeToTasks(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Task subscription failed", slog.Any("error", err))
		}
	}()

	// Task result subscriber: Listen for results of tasks this agent requested
	go func() {
		if err := s.subscribeToTaskResults(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Task result subscription failed", slog.Any("error", err))
		}
	}()

	s.logger.InfoContext(ctx, "Agent started with observability. Listening for events and tasks.")

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

func (s *ObservableSubscriber) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Shutting down observable subscriber")

	// Shutdown observability components
	if err := s.healthServer.Shutdown(ctx); err != nil {
		s.logger.ErrorContext(ctx, "Error shutting down health server", slog.Any("error", err))
	}

	if err := s.obs.Shutdown(ctx); err != nil {
		s.logger.ErrorContext(ctx, "Subscriber observability shutdown failed - likely OTLP trace export issue",
			slog.Any("error", err),
			slog.String("service", "subscriber"),
			slog.String("otlp_endpoint", s.obs.Config.JaegerEndpoint),
		)
		return err
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriber, err := NewObservableSubscriber()
	if err != nil {
		panic(fmt.Sprintf("Failed to create observable subscriber: %v", err))
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := subscriber.Shutdown(shutdownCtx); err != nil {
			subscriber.logger.Error("Error during shutdown", slog.Any("error", err))
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		subscriber.logger.Info("Received shutdown signal")
		cancel()
	}()

	if err := subscriber.Run(ctx); err != nil && err != context.Canceled {
		subscriber.logger.Error("Subscriber run failed", slog.Any("error", err))
		panic(err)
	}

	subscriber.logger.Info("Subscriber shutdown complete")
}
