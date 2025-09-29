package agenthub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

// EventBusService implements the gRPC EventBus service with observability
type EventBusService struct {
	pb.UnimplementedEventBusServer

	// Core functionality
	taskSubscribers         map[string][]chan *pb.TaskMessage
	taskResultSubscribers   map[string][]chan *pb.TaskResult
	taskProgressSubscribers map[string][]chan *pb.TaskProgress
	taskMu                  sync.RWMutex

	// AgentHub components
	Server *AgentHubServer
}

// NewEventBusService creates a new EventBus service
func NewEventBusService(server *AgentHubServer) *EventBusService {
	return &EventBusService{
		Server:                  server,
		taskSubscribers:         make(map[string][]chan *pb.TaskMessage),
		taskResultSubscribers:   make(map[string][]chan *pb.TaskResult),
		taskProgressSubscribers: make(map[string][]chan *pb.TaskProgress),
	}
}

// PublishTask implements the gRPC method with observability
func (s *EventBusService) PublishTask(ctx context.Context, req *pb.PublishTaskRequest) (*pb.PublishResponse, error) {
	// Start tracing
	ctx, span := s.Server.TraceManager.StartPublishSpan(ctx, "task_queue", req.GetTask().GetTaskType())
	defer span.End()
	s.Server.TraceManager.AddComponentAttribute(span, "broker")

	// Start timing
	timer := s.Server.MetricsManager.StartTimer()
	defer timer(ctx, req.GetTask().GetTaskType(), "broker")

	// Validate request
	if req.GetTask() == nil {
		err := status.Error(codes.InvalidArgument, "task cannot be nil")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, req.GetTask().GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	task := req.GetTask()
	if task.GetTaskId() == "" {
		err := status.Error(codes.InvalidArgument, "task_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	if task.GetTaskType() == "" {
		err := status.Error(codes.InvalidArgument, "task_type cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	if task.GetRequesterAgentId() == "" {
		err := status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "validation_error")
		return nil, err
	}

	// Log the task with structured logging
	s.Server.Logger.InfoContext(ctx, "Received task request",
		"task_id", task.GetTaskId(),
		"task_type", task.GetTaskType(),
		"requester_agent_id", task.GetRequesterAgentId(),
		"responder_agent_id", task.GetResponderAgentId(),
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
		s.Server.Logger.InfoContext(ctx, "No subscribers for task",
			"task_id", task.GetTaskId(),
			"requester_agent_id", task.GetRequesterAgentId(),
		)
		s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "no_subscribers")
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	published := 0
	for _, subChan := range targetChannels {
		taskToSend := *task
		go func(ch chan *pb.TaskMessage, task pb.TaskMessage) {
			defer func() {
				if r := recover(); r != nil {
					s.Server.Logger.ErrorContext(ctx, "Recovered from panic while sending task",
						"task_id", task.GetTaskId(),
						"panic", r,
					)
					s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "panic")
				}
			}()

			select {
			case ch <- &task:
				s.Server.MetricsManager.IncrementEventsPublished(ctx, task.GetTaskType(), "task_queue")
			case <-ctx.Done():
				s.Server.Logger.InfoContext(ctx, "Context cancelled while sending task",
					"task_id", task.GetTaskId(),
				)
				s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "context_cancelled")
			case <-time.After(5 * time.Second):
				s.Server.Logger.InfoContext(ctx, "Timeout sending task",
					"task_id", task.GetTaskId(),
				)
				s.Server.MetricsManager.IncrementEventErrors(ctx, task.GetTaskType(), "broker", "timeout")
			}
		}(subChan, taskToSend)
		published++
	}

	// Record metrics
	s.Server.MetricsManager.IncrementEventsProcessed(ctx, task.GetTaskType(), "broker", true)
	s.Server.TraceManager.SetSpanSuccess(span)

	s.Server.Logger.InfoContext(ctx, "Task published successfully",
		"task_id", task.GetTaskId(),
		"published_to", published,
	)

	return &pb.PublishResponse{Success: true}, nil
}

// PublishTaskResult implements the gRPC method with observability
func (s *EventBusService) PublishTaskResult(ctx context.Context, req *pb.PublishTaskResultRequest) (*pb.PublishResponse, error) {
	ctx, span := s.Server.TraceManager.StartPublishSpan(ctx, "task_result_queue", "task_result")
	defer span.End()
	s.Server.TraceManager.AddComponentAttribute(span, "broker")

	timer := s.Server.MetricsManager.StartTimer()
	defer timer(ctx, "task_result", "broker")

	if req.GetResult() == nil {
		err := status.Error(codes.InvalidArgument, "result cannot be nil")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "task_result", "broker", "validation_error")
		return nil, err
	}

	result := req.GetResult()
	if result.GetTaskId() == "" {
		err := status.Error(codes.InvalidArgument, "task_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "task_result", "broker", "validation_error")
		return nil, err
	}

	s.Server.Logger.InfoContext(ctx, "Received task result",
		"task_id", result.GetTaskId(),
		"executor_agent_id", result.GetExecutorAgentId(),
		"status", result.GetStatus().String(),
	)

	s.taskMu.RLock()
	var targetChannels []chan *pb.TaskResult
	for _, subs := range s.taskResultSubscribers {
		targetChannels = append(targetChannels, subs...)
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		s.Server.Logger.InfoContext(ctx, "No subscribers for task result",
			"task_id", result.GetTaskId(),
		)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "task_result", "broker", "no_subscribers")
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		resultToSend := *result
		go func(ch chan *pb.TaskResult, result pb.TaskResult) {
			defer func() {
				if r := recover(); r != nil {
					s.Server.Logger.ErrorContext(ctx, "Recovered from panic while sending task result",
						"task_id", result.GetTaskId(),
						"panic", r,
					)
					s.Server.MetricsManager.IncrementEventErrors(ctx, "task_result", "broker", "panic")
				}
			}()

			select {
			case ch <- &result:
				s.Server.MetricsManager.IncrementEventsPublished(ctx, "task_result", "task_result_queue")
			case <-ctx.Done():
				s.Server.Logger.InfoContext(ctx, "Context cancelled while sending task result",
					"task_id", result.GetTaskId(),
				)
			case <-time.After(5 * time.Second):
				s.Server.Logger.InfoContext(ctx, "Timeout sending task result",
					"task_id", result.GetTaskId(),
				)
			}
		}(subChan, resultToSend)
	}

	s.Server.MetricsManager.IncrementEventsProcessed(ctx, "task_result", "broker", true)
	s.Server.TraceManager.SetSpanSuccess(span)

	return &pb.PublishResponse{Success: true}, nil
}

// PublishTaskProgress implements the gRPC method with observability
func (s *EventBusService) PublishTaskProgress(ctx context.Context, req *pb.PublishTaskProgressRequest) (*pb.PublishResponse, error) {
	ctx, span := s.Server.TraceManager.StartPublishSpan(ctx, "task_progress_queue", "task_progress")
	defer span.End()
	s.Server.TraceManager.AddComponentAttribute(span, "broker")

	timer := s.Server.MetricsManager.StartTimer()
	defer timer(ctx, "task_progress", "broker")

	if req.GetProgress() == nil {
		err := status.Error(codes.InvalidArgument, "progress cannot be nil")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "validation_error")
		return nil, err
	}

	progress := req.GetProgress()
	if progress.GetTaskId() == "" {
		err := status.Error(codes.InvalidArgument, "task_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "validation_error")
		return nil, err
	}

	s.Server.Logger.InfoContext(ctx, "Received task progress",
		"task_id", progress.GetTaskId(),
		"progress_percentage", int(progress.GetProgressPercentage()),
		"executor_agent_id", progress.GetExecutorAgentId(),
	)

	s.taskMu.RLock()
	var targetChannels []chan *pb.TaskProgress
	for _, subs := range s.taskProgressSubscribers {
		targetChannels = append(targetChannels, subs...)
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		s.Server.Logger.InfoContext(ctx, "No subscribers for task progress",
			"task_id", progress.GetTaskId(),
		)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "no_subscribers")
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		progressToSend := *progress
		go func(ch chan *pb.TaskProgress, progress pb.TaskProgress) {
			defer func() {
				if r := recover(); r != nil {
					s.Server.Logger.ErrorContext(ctx, "Recovered from panic while sending task progress",
						"task_id", progress.GetTaskId(),
						"panic", r,
					)
					s.Server.MetricsManager.IncrementEventErrors(ctx, "task_progress", "broker", "panic")
				}
			}()

			select {
			case ch <- &progress:
				s.Server.MetricsManager.IncrementEventsPublished(ctx, "task_progress", "task_progress_queue")
			case <-ctx.Done():
				s.Server.Logger.InfoContext(ctx, "Context cancelled while sending task progress",
					"task_id", progress.GetTaskId(),
				)
			case <-time.After(5 * time.Second):
				s.Server.Logger.InfoContext(ctx, "Timeout sending task progress",
					"task_id", progress.GetTaskId(),
				)
			}
		}(subChan, progressToSend)
	}

	s.Server.MetricsManager.IncrementEventsProcessed(ctx, "task_progress", "broker", true)
	s.Server.TraceManager.SetSpanSuccess(span)

	return &pb.PublishResponse{Success: true}, nil
}

// SubscribeToTasks implements the gRPC streaming method
func (s *EventBusService) SubscribeToTasks(req *pb.SubscribeToTasksRequest, stream pb.EventBus_SubscribeToTasksServer) error {
	ctx := stream.Context()
	ctx, span := s.Server.TraceManager.StartConsumeSpan(ctx, "task_subscription", "task_subscription")
	defer span.End()
	s.Server.TraceManager.AddComponentAttribute(span, "broker")

	agentID := req.GetAgentId()
	if agentID == "" {
		err := status.Error(codes.InvalidArgument, "agent_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		return err
	}

	s.Server.Logger.InfoContext(ctx, "Agent subscribed to tasks",
		"agent_id", agentID,
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
		s.Server.Logger.InfoContext(ctx, "Agent unsubscribed from tasks",
			"agent_id", agentID,
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
				s.Server.Logger.ErrorContext(ctx, "Error sending task to agent",
					"agent_id", agentID,
					"error", err,
				)
				s.Server.MetricsManager.IncrementEventErrors(ctx, "task_delivery", "broker", "send_error")
				return err
			}
			s.Server.MetricsManager.IncrementEventsProcessed(ctx, "task_delivery", "broker", true)
		case <-ctx.Done():
			s.Server.Logger.InfoContext(ctx, "Task subscription context done",
				"agent_id", agentID,
			)
			return ctx.Err()
		}
	}
}

// SubscribeToTaskResults implements the gRPC streaming method
func (s *EventBusService) SubscribeToTaskResults(req *pb.SubscribeToTaskResultsRequest, stream pb.EventBus_SubscribeToTaskResultsServer) error {
	ctx := stream.Context()
	ctx, span := s.Server.TraceManager.StartConsumeSpan(ctx, "task_result_subscription", "task_result_subscription")
	defer span.End()
	s.Server.TraceManager.AddComponentAttribute(span, "broker")

	requesterID := req.GetRequesterAgentId()
	if requesterID == "" {
		err := status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		return err
	}

	s.Server.Logger.InfoContext(ctx, "Agent subscribed to task results",
		"requester_agent_id", requesterID,
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
		s.Server.Logger.InfoContext(ctx, "Agent unsubscribed from task results",
			"requester_agent_id", requesterID,
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
				s.Server.Logger.ErrorContext(ctx, "Error sending task result to agent",
					"requester_agent_id", requesterID,
					"error", err,
				)
				s.Server.MetricsManager.IncrementEventErrors(ctx, "task_result_delivery", "broker", "send_error")
				return err
			}
			s.Server.MetricsManager.IncrementEventsProcessed(ctx, "task_result_delivery", "broker", true)
		case <-ctx.Done():
			s.Server.Logger.InfoContext(ctx, "Task result subscription context done",
				"requester_agent_id", requesterID,
			)
			return ctx.Err()
		}
	}
}

// SubscribeToTaskProgress implements the gRPC streaming method
func (s *EventBusService) SubscribeToTaskProgress(req *pb.SubscribeToTaskResultsRequest, stream pb.EventBus_SubscribeToTaskProgressServer) error {
	ctx := stream.Context()
	ctx, span := s.Server.TraceManager.StartConsumeSpan(ctx, "task_progress_subscription", "task_progress_subscription")
	defer span.End()
	s.Server.TraceManager.AddComponentAttribute(span, "broker")

	requesterID := req.GetRequesterAgentId()
	if requesterID == "" {
		err := status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		return err
	}

	s.Server.Logger.InfoContext(ctx, "Agent subscribed to task progress",
		"requester_agent_id", requesterID,
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
		s.Server.Logger.InfoContext(ctx, "Agent unsubscribed from task progress",
			"requester_agent_id", requesterID,
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
				s.Server.Logger.ErrorContext(ctx, "Error sending task progress to agent",
					"requester_agent_id", requesterID,
					"error", err,
				)
				s.Server.MetricsManager.IncrementEventErrors(ctx, "task_progress_delivery", "broker", "send_error")
				return err
			}
			s.Server.MetricsManager.IncrementEventsProcessed(ctx, "task_progress_delivery", "broker", true)
		case <-ctx.Done():
			s.Server.Logger.InfoContext(ctx, "Task progress subscription context done",
				"requester_agent_id", requesterID,
			)
			return ctx.Err()
		}
	}
}

// StartBroker creates and starts a broker with the new abstraction
func StartBroker(ctx context.Context) error {
	// Create gRPC configuration for broker
	config := NewGRPCConfig("broker")

	// Create AgentHub server
	server, err := NewAgentHubServer(config)
	if err != nil {
		return fmt.Errorf("failed to create AgentHub server: %w", err)
	}

	// Create EventBus service
	eventBusService := NewEventBusService(server)

	// Register the EventBus service
	pb.RegisterEventBusServer(server.Server, eventBusService)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		server.Logger.Info("Received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Start the server
	return server.Start(ctx)
}
