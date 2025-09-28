//go:build observability
// +build observability

package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/owulveryck/agenthub/internal/grpc"
	"github.com/owulveryck/agenthub/internal/observability"
)

const (
	observabilityPort = ":8080"  // Port for observability endpoints
	grpcPort          = ":50051" // Port for the gRPC server
)

type observableEventBusServer struct {
	pb.UnimplementedEventBusServer

	// Core functionality
	taskSubscribers         map[string][]chan *pb.TaskMessage
	taskResultSubscribers   map[string][]chan *pb.TaskResult
	taskProgressSubscribers map[string][]chan *pb.TaskProgress
	taskMu                  sync.RWMutex

	// Observability components
	obs            *observability.Observability
	traceManager   *observability.TraceManager
	metricsManager *observability.MetricsManager
	healthServer   *observability.HealthServer
	logger         *slog.Logger
}

func NewObservableEventBusServer() (*observableEventBusServer, error) {
	// Initialize observability
	config := observability.DefaultConfig("agenthub-broker")
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
	healthServer := observability.NewHealthServer("8080", config.ServiceName, config.ServiceVersion)

	// Add health checks
	healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
		return nil // Simple health check
	}))

	return &observableEventBusServer{
		taskSubscribers:         make(map[string][]chan *pb.TaskMessage),
		taskResultSubscribers:   make(map[string][]chan *pb.TaskResult),
		taskProgressSubscribers: make(map[string][]chan *pb.TaskProgress),
		obs:                     obs,
		traceManager:            traceManager,
		metricsManager:          metricsManager,
		healthServer:            healthServer,
		logger:                  obs.Logger,
	}, nil
}

func (s *observableEventBusServer) PublishTask(ctx context.Context, req *pb.PublishTaskRequest) (*pb.PublishResponse, error) {
	// Start tracing
	ctx, span := s.traceManager.StartPublishSpan(ctx, "task_queue", req.GetTask().GetTaskType())
	defer span.End()

	// Start timing
	timer := s.metricsManager.StartTimer()
	defer timer(ctx, req.GetTask().GetTaskType(), "broker")

	// Validate request
	if req.GetTask() == nil {
		s.traceManager.RecordError(span, status.Error(codes.InvalidArgument, "task cannot be nil"))
		s.metricsManager.IncrementEventErrors(ctx, req.GetTask().GetTaskType(), "broker", "validation_error")
		return nil, status.Error(codes.InvalidArgument, "task cannot be nil")
	}

	task := req.GetTask()
	if task.GetTaskId() == "" {
		err := status.Error(codes.InvalidArgument, "task_id cannot be empty")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	if task.GetTaskType() == "" {
		err := status.Error(codes.InvalidArgument, "task_type cannot be empty")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	if task.GetRequesterAgentId() == "" {
		err := status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	// Log the task with structured logging
	s.logger.InfoContext(ctx, "Received task request",
		slog.String("task_id", task.GetTaskId()),
		slog.String("task_type", task.GetTaskType()),
		slog.String("requester_agent_id", task.GetRequesterAgentId()),
		slog.String("responder_agent_id", task.GetResponderAgentId()),
	)

	s.taskMu.RLock()
	// Route to specific agent or broadcast to all if no specific responder
	var targetChannels []chan *pb.TaskMessage
	if responderID := task.GetResponderAgentId(); responderID != "" {
		if subs, ok := s.taskSubscribers[responderID]; ok {
			targetChannels = subs
		}
	} else {
		// Broadcast to all task subscribers
		for _, subs := range s.taskSubscribers {
			targetChannels = append(targetChannels, subs...)
		}
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		s.logger.WarnContext(ctx, "No subscribers for task",
			slog.String("task_id", task.GetTaskId()),
			slog.String("requester_agent_id", task.GetRequesterAgentId()),
		)
		s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "no_subscribers")
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	published := 0
	for _, subChan := range targetChannels {
		taskToSend := *task
		go func(ch chan *pb.TaskMessage, task pb.TaskMessage) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.ErrorContext(ctx, "Recovered from panic while sending task",
						slog.String("task_id", task.GetTaskId()),
						slog.Any("panic", r),
					)
					s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "panic")
				}
			}()

			select {
			case ch <- &task:
				s.metricsManager.IncrementEventsPublished(ctx, task.GetTaskType(), "task_queue")
			case <-ctx.Done():
				s.logger.WarnContext(ctx, "Context cancelled while sending task",
					slog.String("task_id", task.GetTaskId()),
				)
				s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "context_cancelled")
			case <-time.After(5 * time.Second):
				s.logger.WarnContext(ctx, "Timeout sending task",
					slog.String("task_id", task.GetTaskId()),
				)
				s.metricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "timeout")
			}
		}(subChan, taskToSend)
		published++
	}

	// Record metrics
	s.metricsManager.IncrementEventsProcessed(ctx, task.GetTaskType(), "broker", true)
	s.traceManager.SetSpanSuccess(span)

	s.logger.InfoContext(ctx, "Task published successfully",
		slog.String("task_id", task.GetTaskId()),
		slog.Int("published_to", published),
	)

	return &pb.PublishResponse{Success: true}, nil
}

func (s *observableEventBusServer) PublishTaskResult(ctx context.Context, req *pb.PublishTaskResultRequest) (*pb.PublishResponse, error) {
	ctx, span := s.traceManager.StartPublishSpan(ctx, "task_result_queue", "task_result")
	defer span.End()

	timer := s.metricsManager.StartTimer()
	defer timer(ctx, "task_result", "broker")

	if req.GetResult() == nil {
		err := status.Error(codes.InvalidArgument, "result cannot be nil")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, "task_result", "broker", "validation_error")
		return nil, err
	}

	result := req.GetResult()
	if result.GetTaskId() == "" {
		err := status.Error(codes.InvalidArgument, "task_id cannot be empty")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, "task_result", "broker", "validation_error")
		return nil, err
	}

	s.logger.InfoContext(ctx, "Received task result",
		slog.String("task_id", result.GetTaskId()),
		slog.String("executor_agent_id", result.GetExecutorAgentId()),
		slog.String("status", result.GetStatus().String()),
	)

	s.taskMu.RLock()
	var targetChannels []chan *pb.TaskResult
	for _, subs := range s.taskResultSubscribers {
		targetChannels = append(targetChannels, subs...)
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		s.logger.WarnContext(ctx, "No subscribers for task result",
			slog.String("task_id", result.GetTaskId()),
		)
		s.metricsManager.IncrementEventErrors(ctx, "task_result", "broker", "no_subscribers")
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		resultToSend := *result
		go func(ch chan *pb.TaskResult, result pb.TaskResult) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.ErrorContext(ctx, "Recovered from panic while sending task result",
						slog.String("task_id", result.GetTaskId()),
						slog.Any("panic", r),
					)
					s.metricsManager.IncrementEventErrors(ctx, "task_result", "broker", "panic")
				}
			}()

			select {
			case ch <- &result:
				s.metricsManager.IncrementEventsPublished(ctx, "task_result", "task_result_queue")
			case <-ctx.Done():
				s.logger.WarnContext(ctx, "Context cancelled while sending task result",
					slog.String("task_id", result.GetTaskId()),
				)
			case <-time.After(5 * time.Second):
				s.logger.WarnContext(ctx, "Timeout sending task result",
					slog.String("task_id", result.GetTaskId()),
				)
			}
		}(subChan, resultToSend)
	}

	s.metricsManager.IncrementEventsProcessed(ctx, "task_result", "broker", true)
	s.traceManager.SetSpanSuccess(span)

	return &pb.PublishResponse{Success: true}, nil
}

func (s *observableEventBusServer) PublishTaskProgress(ctx context.Context, req *pb.PublishTaskProgressRequest) (*pb.PublishResponse, error) {
	ctx, span := s.traceManager.StartPublishSpan(ctx, "task_progress_queue", "task_progress")
	defer span.End()

	timer := s.metricsManager.StartTimer()
	defer timer(ctx, "task_progress", "broker")

	if req.GetProgress() == nil {
		err := status.Error(codes.InvalidArgument, "progress cannot be nil")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "validation_error")
		return nil, err
	}

	progress := req.GetProgress()
	if progress.GetTaskId() == "" {
		err := status.Error(codes.InvalidArgument, "task_id cannot be empty")
		s.traceManager.RecordError(span, err)
		s.metricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "validation_error")
		return nil, err
	}

	s.logger.InfoContext(ctx, "Received task progress",
		slog.String("task_id", progress.GetTaskId()),
		slog.Int("progress_percentage", int(progress.GetProgressPercentage())),
		slog.String("executor_agent_id", progress.GetExecutorAgentId()),
	)

	s.taskMu.RLock()
	var targetChannels []chan *pb.TaskProgress
	for _, subs := range s.taskProgressSubscribers {
		targetChannels = append(targetChannels, subs...)
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		s.logger.WarnContext(ctx, "No subscribers for task progress",
			slog.String("task_id", progress.GetTaskId()),
		)
		s.metricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "no_subscribers")
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		progressToSend := *progress
		go func(ch chan *pb.TaskProgress, progress pb.TaskProgress) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.ErrorContext(ctx, "Recovered from panic while sending task progress",
						slog.String("task_id", progress.GetTaskId()),
						slog.Any("panic", r),
					)
					s.metricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "panic")
				}
			}()

			select {
			case ch <- &progress:
				s.metricsManager.IncrementEventsPublished(ctx, "task_progress", "task_progress_queue")
			case <-ctx.Done():
				s.logger.WarnContext(ctx, "Context cancelled while sending task progress",
					slog.String("task_id", progress.GetTaskId()),
				)
			case <-time.After(5 * time.Second):
				s.logger.WarnContext(ctx, "Timeout sending task progress",
					slog.String("task_id", progress.GetTaskId()),
				)
			}
		}(subChan, progressToSend)
	}

	s.metricsManager.IncrementEventsProcessed(ctx, "task_progress", "broker", true)
	s.traceManager.SetSpanSuccess(span)

	return &pb.PublishResponse{Success: true}, nil
}

func (s *observableEventBusServer) SubscribeToTasks(req *pb.SubscribeToTasksRequest, stream pb.EventBus_SubscribeToTasksServer) error {
	ctx := stream.Context()
	ctx, span := s.traceManager.StartConsumeSpan(ctx, "task_subscription", "task_subscription")
	defer span.End()

	agentID := req.GetAgentId()
	if agentID == "" {
		err := status.Error(codes.InvalidArgument, "agent_id cannot be empty")
		s.traceManager.RecordError(span, err)
		return err
	}

	s.logger.InfoContext(ctx, "Agent subscribed to tasks",
		slog.String("agent_id", agentID),
	)

	subChan := make(chan *pb.TaskMessage, 10)

	s.taskMu.Lock()
	s.taskSubscribers[agentID] = append(s.taskSubscribers[agentID], subChan)
	s.taskMu.Unlock()

	defer func() {
		s.taskMu.Lock()
		if subs, ok := s.taskSubscribers[agentID]; ok {
			newSubs := [](chan *pb.TaskMessage){}
			for _, ch := range subs {
				if ch != subChan {
					newSubs = append(newSubs, ch)
				}
			}
			s.taskSubscribers[agentID] = newSubs
			if len(s.taskSubscribers[agentID]) == 0 {
				delete(s.taskSubscribers, agentID)
			}
		}
		close(subChan)
		s.taskMu.Unlock()
		s.logger.InfoContext(ctx, "Agent unsubscribed from tasks",
			slog.String("agent_id", agentID),
		)
	}()

	// Stream tasks back to the client
	for {
		select {
		case task, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(task); err != nil {
				s.logger.ErrorContext(ctx, "Error sending task to agent",
					slog.String("agent_id", agentID),
					slog.Any("error", err),
				)
				s.metricsManager.IncrementEventErrors(ctx, "task_delivery", "broker", "send_error")
				return err
			}
			s.metricsManager.IncrementEventsProcessed(ctx, "task_delivery", "broker", true)
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "Task subscription context done",
				slog.String("agent_id", agentID),
			)
			return ctx.Err()
		}
	}
}

func (s *observableEventBusServer) SubscribeToTaskResults(req *pb.SubscribeToTaskResultsRequest, stream pb.EventBus_SubscribeToTaskResultsServer) error {
	ctx := stream.Context()
	ctx, span := s.traceManager.StartConsumeSpan(ctx, "task_result_subscription", "task_result_subscription")
	defer span.End()

	requesterID := req.GetRequesterAgentId()
	if requesterID == "" {
		err := status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
		s.traceManager.RecordError(span, err)
		return err
	}

	s.logger.InfoContext(ctx, "Agent subscribed to task results",
		slog.String("requester_agent_id", requesterID),
	)

	subChan := make(chan *pb.TaskResult, 10)

	s.taskMu.Lock()
	s.taskResultSubscribers[requesterID] = append(s.taskResultSubscribers[requesterID], subChan)
	s.taskMu.Unlock()

	defer func() {
		s.taskMu.Lock()
		if subs, ok := s.taskResultSubscribers[requesterID]; ok {
			newSubs := [](chan *pb.TaskResult){}
			for _, ch := range subs {
				if ch != subChan {
					newSubs = append(newSubs, ch)
				}
			}
			s.taskResultSubscribers[requesterID] = newSubs
			if len(s.taskResultSubscribers[requesterID]) == 0 {
				delete(s.taskResultSubscribers, requesterID)
			}
		}
		close(subChan)
		s.taskMu.Unlock()
		s.logger.InfoContext(ctx, "Agent unsubscribed from task results",
			slog.String("requester_agent_id", requesterID),
		)
	}()

	// Stream results back to the client
	for {
		select {
		case result, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(result); err != nil {
				s.logger.ErrorContext(ctx, "Error sending task result to agent",
					slog.String("requester_agent_id", requesterID),
					slog.Any("error", err),
				)
				s.metricsManager.IncrementEventErrors(ctx, "task_result_delivery", "broker", "send_error")
				return err
			}
			s.metricsManager.IncrementEventsProcessed(ctx, "task_result_delivery", "broker", true)
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "Task result subscription context done",
				slog.String("requester_agent_id", requesterID),
			)
			return ctx.Err()
		}
	}
}

func (s *observableEventBusServer) SubscribeToTaskProgress(req *pb.SubscribeToTaskResultsRequest, stream pb.EventBus_SubscribeToTaskProgressServer) error {
	ctx := stream.Context()
	ctx, span := s.traceManager.StartConsumeSpan(ctx, "task_progress_subscription", "task_progress_subscription")
	defer span.End()

	requesterID := req.GetRequesterAgentId()
	if requesterID == "" {
		err := status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
		s.traceManager.RecordError(span, err)
		return err
	}

	s.logger.InfoContext(ctx, "Agent subscribed to task progress",
		slog.String("requester_agent_id", requesterID),
	)

	subChan := make(chan *pb.TaskProgress, 10)

	s.taskMu.Lock()
	s.taskProgressSubscribers[requesterID] = append(s.taskProgressSubscribers[requesterID], subChan)
	s.taskMu.Unlock()

	defer func() {
		s.taskMu.Lock()
		if subs, ok := s.taskProgressSubscribers[requesterID]; ok {
			newSubs := [](chan *pb.TaskProgress){}
			for _, ch := range subs {
				if ch != subChan {
					newSubs = append(newSubs, ch)
				}
			}
			s.taskProgressSubscribers[requesterID] = newSubs
			if len(s.taskProgressSubscribers[requesterID]) == 0 {
				delete(s.taskProgressSubscribers, requesterID)
			}
		}
		close(subChan)
		s.taskMu.Unlock()
		s.logger.InfoContext(ctx, "Agent unsubscribed from task progress",
			slog.String("requester_agent_id", requesterID),
		)
	}()

	// Stream progress back to the client
	for {
		select {
		case progress, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(progress); err != nil {
				s.logger.ErrorContext(ctx, "Error sending task progress to agent",
					slog.String("requester_agent_id", requesterID),
					slog.Any("error", err),
				)
				s.metricsManager.IncrementEventErrors(ctx, "task_progress_delivery", "broker", "send_error")
				return err
			}
			s.metricsManager.IncrementEventsProcessed(ctx, "task_progress_delivery", "broker", true)
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "Task progress subscription context done",
				slog.String("requester_agent_id", requesterID),
			)
			return ctx.Err()
		}
	}
}

func (s *observableEventBusServer) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Shutting down observable event bus server")

	// Shutdown observability components
	if err := s.healthServer.Shutdown(ctx); err != nil {
		s.logger.ErrorContext(ctx, "Error shutting down health server", slog.Any("error", err))
	}

	if err := s.obs.Shutdown(ctx); err != nil {
		s.logger.ErrorContext(ctx, "Error shutting down observability", slog.Any("error", err))
		return err
	}

	return nil
}

func runWithObservability() error {
	ctx := context.Background()

	// Create observable server
	server, err := NewObservableEventBusServer()
	if err != nil {
		return err
	}

	// Start health server
	go func() {
		server.logger.Info("Starting health server on port 8080")
		if err := server.healthServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			server.logger.Error("Health server failed", slog.Any("error", err))
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		server.logger.Error("Failed to listen", slog.Any("error", err))
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterEventBusServer(grpcServer, server)

	// Start metrics collection
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				server.metricsManager.UpdateSystemMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		server.logger.Info("Received shutdown signal")

		// Graceful shutdown
		grpcServer.GracefulStop()

		// Shutdown observability
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	server.logger.Info("AgentHub broker gRPC server with observability listening",
		slog.String("address", lis.Addr().String()),
		slog.String("health_endpoint", "http://localhost:8080/health"),
		slog.String("metrics_endpoint", "http://localhost:8080/metrics"),
	)

	return grpcServer.Serve(lis)
}

func main() {
	if err := runWithObservability(); err != nil {
		panic(err)
	}
}
