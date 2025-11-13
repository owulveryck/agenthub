package agenthub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

// AgentHubService implements the gRPC AgentHub service with A2A compliance and observability
type AgentHubService struct {
	pb.UnimplementedAgentHubServer

	// A2A event streams
	messageSubscribers map[string][]chan *pb.AgentEvent
	taskSubscribers    map[string][]chan *pb.AgentEvent
	eventSubscribers   map[string][]chan *pb.AgentEvent
	agentMu            sync.RWMutex

	// Task storage for A2A compliance
	tasks   map[string]*pb.Task
	tasksMu sync.RWMutex

	// Agent registry
	registeredAgents map[string]*pb.AgentCard
	agentsMu         sync.RWMutex

	// Context and message storage
	contexts   map[string][]*pb.Message
	contextsMu sync.RWMutex

	// AgentHub components
	Server *AgentHubServer
}

// NewAgentHubService creates a new A2A-compliant AgentHub service
func NewAgentHubService(server *AgentHubServer) *AgentHubService {
	return &AgentHubService{
		Server:             server,
		messageSubscribers: make(map[string][]chan *pb.AgentEvent),
		taskSubscribers:    make(map[string][]chan *pb.AgentEvent),
		eventSubscribers:   make(map[string][]chan *pb.AgentEvent),
		tasks:              make(map[string]*pb.Task),
		registeredAgents:   make(map[string]*pb.AgentCard),
		contexts:           make(map[string][]*pb.Message),
	}
}

// ===== A2A Message Publishing (EDA style) =====

// PublishMessage publishes A2A messages through the broker
func (s *AgentHubService) PublishMessage(ctx context.Context, req *pb.PublishMessageRequest) (*pb.PublishResponse, error) {
	ctx, span := s.Server.TraceManager.StartPublishSpan(ctx, "broker", "a2a_message", req.GetMessage().GetRole().String())
	defer span.End()

	timer := s.Server.MetricsManager.StartTimer()
	defer timer(ctx, "a2a_message", "broker")

	// Validate request
	if req.GetMessage() == nil {
		err := status.Error(codes.InvalidArgument, "message cannot be nil")
		s.Server.TraceManager.RecordError(span, err)
		return nil, err
	}

	message := req.GetMessage()
	if message.GetMessageId() == "" {
		err := status.Error(codes.InvalidArgument, "message_id cannot be empty")
		s.Server.TraceManager.RecordError(span, err)
		return nil, err
	}

	// Add comprehensive A2A message attributes to span
	taskType := ""
	if message.Metadata != nil && message.Metadata.Fields != nil {
		if taskTypeValue, exists := message.Metadata.Fields["task_type"]; exists {
			taskType = taskTypeValue.GetStringValue()
		}
	}
	s.Server.TraceManager.AddA2AMessageAttributes(
		span,
		message.GetMessageId(),
		message.GetContextId(),
		message.GetRole().String(),
		taskType,
		len(message.GetContent()),
		message.GetMetadata() != nil,
	)

	// Generate event ID
	eventID := fmt.Sprintf("evt_%s_%d", message.GetMessageId(), time.Now().Unix())

	// Store message in context if context_id is provided
	if message.GetContextId() != "" {
		s.contextsMu.Lock()
		s.contexts[message.GetContextId()] = append(s.contexts[message.GetContextId()], message)
		s.contextsMu.Unlock()
	}

	// Handle task creation/update if this message has a task_id
	var task *pb.Task
	if message.GetTaskId() != "" {
		s.tasksMu.Lock()
		if existingTask, exists := s.tasks[message.GetTaskId()]; exists {
			// Update existing task with new message
			existingTask.History = append(existingTask.History, message)
			existingTask.Status.Update = message
			existingTask.Status.Timestamp = timestamppb.Now()
			task = existingTask
		} else {
			// Create new task for this message
			task = &pb.Task{
				Id:        message.GetTaskId(),
				ContextId: message.GetContextId(),
				Status: &pb.TaskStatus{
					State:     pb.TaskState_TASK_STATE_SUBMITTED,
					Timestamp: timestamppb.Now(),
					Update:    message,
				},
				History:   []*pb.Message{message},
				Artifacts: []*pb.Artifact{},
				Metadata:  message.GetMetadata(),
			}
		}
		s.tasks[message.GetTaskId()] = task
		s.tasksMu.Unlock()
	}

	// Create message event
	messageEvent := &pb.AgentEvent{
		EventId:   eventID,
		Timestamp: timestamppb.Now(),
		Payload:   &pb.AgentEvent_Message{Message: message},
		Routing:   req.GetRouting(),
		TraceId:   span.SpanContext().TraceID().String(),
		SpanId:    span.SpanContext().SpanID().String(),
	}

	// Route message event to subscribers with enhanced tracing
	routeCtx, routeSpan := s.Server.TraceManager.StartA2AEventRouteSpan(
		ctx,
		"broker",
		eventID,
		"message",
		s.getSubscriberCount("message", messageEvent.GetRouting()),
	)
	defer routeSpan.End()

	// Add routing metadata to route span
	if routing := messageEvent.GetRouting(); routing != nil {
		s.Server.TraceManager.AddA2AEventAttributes(
			routeSpan,
			eventID,
			routing.GetEventType(),
			routing.GetFromAgentId(),
			routing.GetToAgentId(),
			s.getSubscriberCount("message", routing),
		)
	}

	err := s.routeEvent(routeCtx, messageEvent)
	if err != nil {
		s.Server.TraceManager.RecordError(span, err)
		s.Server.TraceManager.RecordError(routeSpan, err)
		s.Server.MetricsManager.IncrementEventErrors(ctx, "a2a_message", "broker", "routing_error")
		return &pb.PublishResponse{Success: false, Error: err.Error()}, nil
	}
	s.Server.TraceManager.SetSpanSuccess(routeSpan)

	// If this was a task message, also publish a task event
	if task != nil {
		taskEventID := fmt.Sprintf("task_%s_%d", task.GetId(), time.Now().Unix())
		taskEvent := &pb.AgentEvent{
			EventId:   taskEventID,
			Timestamp: timestamppb.Now(),
			Payload:   &pb.AgentEvent_Task{Task: task},
			Routing:   req.GetRouting(),
			TraceId:   span.SpanContext().TraceID().String(),
			SpanId:    span.SpanContext().SpanID().String(),
		}

		// Route task event to task subscribers
		err = s.routeEvent(ctx, taskEvent)
		if err != nil {
			s.Server.TraceManager.RecordError(span, err)
			s.Server.MetricsManager.IncrementEventErrors(ctx, "a2a_task", "broker", "routing_error")
			return &pb.PublishResponse{Success: false, Error: err.Error()}, nil
		}
	}

	s.Server.MetricsManager.IncrementEventsProcessed(ctx, "a2a_message", "broker", true)
	s.Server.TraceManager.SetSpanSuccess(span)

	return &pb.PublishResponse{
		Success: true,
		EventId: eventID,
	}, nil
}

// PublishTaskUpdate publishes task status updates
func (s *AgentHubService) PublishTaskUpdate(ctx context.Context, req *pb.PublishTaskUpdateRequest) (*pb.PublishResponse, error) {
	ctx, span := s.Server.TraceManager.StartPublishSpan(ctx, "broker", "task_status_update", req.GetUpdate().GetTaskId())
	defer span.End()

	update := req.GetUpdate()
	if update == nil {
		err := status.Error(codes.InvalidArgument, "update cannot be nil")
		s.Server.TraceManager.RecordError(span, err)
		return nil, err
	}

	// Update task in storage
	s.tasksMu.Lock()
	if task, exists := s.tasks[update.GetTaskId()]; exists {
		task.Status = update.GetStatus()
		s.tasks[update.GetTaskId()] = task
	}
	s.tasksMu.Unlock()

	// Generate event
	eventID := fmt.Sprintf("status_%s_%d", update.GetTaskId(), time.Now().Unix())
	agentEvent := &pb.AgentEvent{
		EventId:   eventID,
		Timestamp: timestamppb.Now(),
		Payload:   &pb.AgentEvent_StatusUpdate{StatusUpdate: update},
		Routing:   req.GetRouting(),
		TraceId:   span.SpanContext().TraceID().String(),
		SpanId:    span.SpanContext().SpanID().String(),
	}

	err := s.routeEvent(ctx, agentEvent)
	if err != nil {
		return &pb.PublishResponse{Success: false, Error: err.Error()}, nil
	}

	return &pb.PublishResponse{Success: true, EventId: eventID}, nil
}

// PublishTaskArtifact publishes task artifacts
func (s *AgentHubService) PublishTaskArtifact(ctx context.Context, req *pb.PublishTaskArtifactRequest) (*pb.PublishResponse, error) {
	ctx, span := s.Server.TraceManager.StartPublishSpan(ctx, "broker", "task_artifact", req.GetArtifact().GetTaskId())
	defer span.End()

	artifact := req.GetArtifact()
	if artifact == nil {
		err := status.Error(codes.InvalidArgument, "artifact cannot be nil")
		s.Server.TraceManager.RecordError(span, err)
		return nil, err
	}

	// Update task with artifact
	s.tasksMu.Lock()
	if task, exists := s.tasks[artifact.GetTaskId()]; exists {
		// Add or update artifact
		found := false
		for i, existing := range task.Artifacts {
			if existing.ArtifactId == artifact.GetArtifact().GetArtifactId() {
				if artifact.GetAppend() {
					// Append parts to existing artifact
					existing.Parts = append(existing.Parts, artifact.GetArtifact().GetParts()...)
				} else {
					// Replace artifact
					task.Artifacts[i] = artifact.GetArtifact()
				}
				found = true
				break
			}
		}
		if !found {
			task.Artifacts = append(task.Artifacts, artifact.GetArtifact())
		}
		s.tasks[artifact.GetTaskId()] = task
	}
	s.tasksMu.Unlock()

	// Generate event
	eventID := fmt.Sprintf("artifact_%s_%d", artifact.GetTaskId(), time.Now().Unix())
	agentEvent := &pb.AgentEvent{
		EventId:   eventID,
		Timestamp: timestamppb.Now(),
		Payload:   &pb.AgentEvent_ArtifactUpdate{ArtifactUpdate: artifact},
		Routing:   req.GetRouting(),
		TraceId:   span.SpanContext().TraceID().String(),
		SpanId:    span.SpanContext().SpanID().String(),
	}

	err := s.routeEvent(ctx, agentEvent)
	if err != nil {
		return &pb.PublishResponse{Success: false, Error: err.Error()}, nil
	}

	return &pb.PublishResponse{Success: true, EventId: eventID}, nil
}

// ===== A2A Event Subscriptions (EDA style) =====

// SubscribeToMessages subscribes to A2A messages for a specific agent
func (s *AgentHubService) SubscribeToMessages(req *pb.SubscribeToMessagesRequest, stream pb.AgentHub_SubscribeToMessagesServer) error {
	ctx := stream.Context()
	agentID := req.GetAgentId()

	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id cannot be empty")
	}

	subChan := make(chan *pb.AgentEvent, 10)

	s.agentMu.Lock()
	s.messageSubscribers[agentID] = append(s.messageSubscribers[agentID], subChan)
	s.agentMu.Unlock()

	defer func() {
		s.agentMu.Lock()
		if subs, ok := s.messageSubscribers[agentID]; ok {
			newSubs := []chan *pb.AgentEvent{}
			for _, ch := range subs {
				if ch != subChan {
					newSubs = append(newSubs, ch)
				}
			}
			s.messageSubscribers[agentID] = newSubs
			if len(s.messageSubscribers[agentID]) == 0 {
				delete(s.messageSubscribers, agentID)
			}
		}
		close(subChan)
		s.agentMu.Unlock()
	}()

	for {
		select {
		case event, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// SubscribeToTasks subscribes to A2A task events
func (s *AgentHubService) SubscribeToTasks(req *pb.SubscribeToTasksRequest, stream pb.AgentHub_SubscribeToTasksServer) error {
	ctx := stream.Context()
	agentID := req.GetAgentId()

	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id cannot be empty")
	}

	subChan := make(chan *pb.AgentEvent, 10)

	s.agentMu.Lock()
	s.taskSubscribers[agentID] = append(s.taskSubscribers[agentID], subChan)
	s.agentMu.Unlock()

	defer func() {
		s.agentMu.Lock()
		if subs, ok := s.taskSubscribers[agentID]; ok {
			newSubs := []chan *pb.AgentEvent{}
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
		s.agentMu.Unlock()
	}()

	for {
		select {
		case event, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// SubscribeToAgentEvents subscribes to all events for an agent
func (s *AgentHubService) SubscribeToAgentEvents(req *pb.SubscribeToAgentEventsRequest, stream pb.AgentHub_SubscribeToAgentEventsServer) error {
	ctx := stream.Context()
	agentID := req.GetAgentId()

	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id cannot be empty")
	}

	subChan := make(chan *pb.AgentEvent, 10)

	s.agentMu.Lock()
	s.eventSubscribers[agentID] = append(s.eventSubscribers[agentID], subChan)
	s.agentMu.Unlock()

	defer func() {
		s.agentMu.Lock()
		if subs, ok := s.eventSubscribers[agentID]; ok {
			newSubs := []chan *pb.AgentEvent{}
			for _, ch := range subs {
				if ch != subChan {
					newSubs = append(newSubs, ch)
				}
			}
			s.eventSubscribers[agentID] = newSubs
			if len(s.eventSubscribers[agentID]) == 0 {
				delete(s.eventSubscribers, agentID)
			}
		}
		close(subChan)
		s.agentMu.Unlock()
	}()

	for {
		select {
		case event, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ===== A2A Task Management =====

// GetTask retrieves a task by ID
func (s *AgentHubService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	s.tasksMu.RLock()
	task, exists := s.tasks[req.GetTaskId()]
	s.tasksMu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	// Apply history length limit if specified
	if req.GetHistoryLength() > 0 && len(task.History) > int(req.GetHistoryLength()) {
		// Create a copy with limited history
		limitedTask := *task
		start := len(task.History) - int(req.GetHistoryLength())
		limitedTask.History = task.History[start:]
		return &limitedTask, nil
	}

	return task, nil
}

// CancelTask cancels a task
func (s *AgentHubService) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	s.tasksMu.Lock()
	defer s.tasksMu.Unlock()

	task, exists := s.tasks[req.GetTaskId()]
	if !exists {
		return nil, status.Error(codes.NotFound, "task not found")
	}

	// Check if task can be cancelled
	switch task.Status.State {
	case pb.TaskState_TASK_STATE_COMPLETED, pb.TaskState_TASK_STATE_FAILED, pb.TaskState_TASK_STATE_CANCELLED:
		return nil, status.Error(codes.FailedPrecondition, "task cannot be cancelled in current state")
	}

	// Update task status
	task.Status = &pb.TaskStatus{
		State:     pb.TaskState_TASK_STATE_CANCELLED,
		Timestamp: timestamppb.Now(),
		Update: &pb.Message{
			MessageId: fmt.Sprintf("cancel_%s_%d", req.GetTaskId(), time.Now().Unix()),
			Role:      pb.Role_ROLE_AGENT,
			Content: []*pb.Part{
				{
					Part: &pb.Part_Text{
						Text: fmt.Sprintf("Task cancelled: %s", req.GetReason()),
					},
				},
			},
		},
	}

	s.tasks[req.GetTaskId()] = task

	// Publish cancellation event
	go func() {
		s.PublishTaskUpdate(context.Background(), &pb.PublishTaskUpdateRequest{
			Update: &pb.TaskStatusUpdateEvent{
				TaskId:    req.GetTaskId(),
				ContextId: task.ContextId,
				Status:    task.Status,
				Final:     true,
			},
			Routing: &pb.AgentEventMetadata{
				EventType: "task_cancelled",
				Priority:  pb.Priority_PRIORITY_HIGH,
			},
		})
	}()

	return task, nil
}

// ListTasks lists tasks for an agent
func (s *AgentHubService) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	var tasks []*pb.Task
	for _, task := range s.tasks {
		// Apply filters
		if req.GetAgentId() != "" {
			// Check if agent is involved (in history or as executor)
			involved := false
			for _, msg := range task.History {
				if msg.Role == pb.Role_ROLE_AGENT {
					involved = true
					break
				}
			}
			if !involved {
				continue
			}
		}

		if req.GetContextId() != "" && task.ContextId != req.GetContextId() {
			continue
		}

		if len(req.GetStates()) > 0 {
			stateMatch := false
			for _, state := range req.GetStates() {
				if task.Status.State == state {
					stateMatch = true
					break
				}
			}
			if !stateMatch {
				continue
			}
		}

		tasks = append(tasks, task)
	}

	return &pb.ListTasksResponse{
		Tasks: tasks,
	}, nil
}

// ===== Agent Discovery =====

// GetAgentCard returns the broker's agent card
func (s *AgentHubService) GetAgentCard(ctx context.Context, req *emptypb.Empty) (*pb.AgentCard, error) {
	// Return a default AgentHub broker card
	return &pb.AgentCard{
		ProtocolVersion:    "0.2.9",
		Name:               "AgentHub EDA Broker",
		Description:        "Event-driven architecture broker for Agent2Agent protocol with A2A message compliance",
		Url:                s.Server.Config.ServerAddr,
		PreferredTransport: "GRPC",
		Provider: &pb.AgentProvider{
			Organization: "AgentHub",
			Url:          "https://github.com/owulveryck/agenthub",
		},
		Version: "1.0.0",
		Capabilities: &pb.AgentCapabilities{
			Streaming:         true,
			PushNotifications: false, // Not implemented yet
		},
		Skills: []*pb.AgentSkill{
			{
				Id:          "message-routing",
				Name:        "Message Routing",
				Description: "Routes A2A messages between agents in an event-driven architecture",
				Tags:        []string{"routing", "eda", "broker"},
			},
			{
				Id:          "task-coordination",
				Name:        "Task Coordination",
				Description: "Coordinates A2A tasks between multiple agents",
				Tags:        []string{"coordination", "tasks", "collaboration"},
			},
		},
	}, nil
}

// RegisterAgent registers an agent with the broker
func (s *AgentHubService) RegisterAgent(ctx context.Context, req *pb.RegisterAgentRequest) (*pb.RegisterAgentResponse, error) {
	if req.GetAgentCard() == nil {
		return &pb.RegisterAgentResponse{
			Success: false,
			Error:   "agent_card is required",
		}, nil
	}

	agentID := req.GetAgentCard().GetName()
	if agentID == "" {
		return &pb.RegisterAgentResponse{
			Success: false,
			Error:   "agent name is required",
		}, nil
	}

	s.agentsMu.Lock()
	s.registeredAgents[agentID] = req.GetAgentCard()
	s.agentsMu.Unlock()

	s.Server.Logger.InfoContext(ctx, "Agent registered",
		"agent_id", agentID,
		"agent_name", req.GetAgentCard().GetName(),
		"subscriptions", req.GetSubscriptions(),
	)

	return &pb.RegisterAgentResponse{
		Success: true,
		AgentId: agentID,
	}, nil
}

// ===== Helper Methods =====

// routeEvent routes an agent event to appropriate subscribers
func (s *AgentHubService) routeEvent(ctx context.Context, event *pb.AgentEvent) error {
	routing := event.GetRouting()
	if routing == nil {
		return fmt.Errorf("routing metadata is required")
	}

	s.agentMu.RLock()
	defer s.agentMu.RUnlock()

	var targetChannels []chan *pb.AgentEvent

	// Route based on target agent
	targetAgent := routing.GetToAgentId()
	if targetAgent != "" {
		// Route to specific agent
		switch event.GetPayload().(type) {
		case *pb.AgentEvent_Message:
			if subs, ok := s.messageSubscribers[targetAgent]; ok {
				targetChannels = append(targetChannels, subs...)
			}
		case *pb.AgentEvent_Task, *pb.AgentEvent_StatusUpdate, *pb.AgentEvent_ArtifactUpdate:
			if subs, ok := s.taskSubscribers[targetAgent]; ok {
				targetChannels = append(targetChannels, subs...)
			}
		}
		if subs, ok := s.eventSubscribers[targetAgent]; ok {
			targetChannels = append(targetChannels, subs...)
		}
	} else {
		// Broadcast to all relevant subscribers
		switch event.GetPayload().(type) {
		case *pb.AgentEvent_Message:
			for _, subs := range s.messageSubscribers {
				targetChannels = append(targetChannels, subs...)
			}
		case *pb.AgentEvent_Task, *pb.AgentEvent_StatusUpdate, *pb.AgentEvent_ArtifactUpdate:
			for _, subs := range s.taskSubscribers {
				targetChannels = append(targetChannels, subs...)
			}
		}
		for _, subs := range s.eventSubscribers {
			targetChannels = append(targetChannels, subs...)
		}
	}

	if len(targetChannels) == 0 {
		s.Server.Logger.InfoContext(ctx, "No subscribers for event",
			"event_id", event.GetEventId(),
			"event_type", routing.GetEventType(),
			"target_agent", targetAgent,
		)
		return nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		go func(ch chan *pb.AgentEvent, evt *pb.AgentEvent) {
			defer func() {
				if r := recover(); r != nil {
					s.Server.Logger.ErrorContext(ctx, "Recovered from panic while sending event",
						"event_id", evt.GetEventId(),
						"panic", r,
					)
				}
			}()

			select {
			case ch <- evt:
				// Event sent successfully
			case <-ctx.Done():
				s.Server.Logger.InfoContext(ctx, "Context cancelled while sending event",
					"event_id", evt.GetEventId(),
				)
			case <-time.After(5 * time.Second):
				s.Server.Logger.InfoContext(ctx, "Timeout sending event",
					"event_id", evt.GetEventId(),
				)
			}
		}(subChan, event)
	}

	return nil
}

// getSubscriberCount returns the number of subscribers for a given event type and routing
func (s *AgentHubService) getSubscriberCount(eventType string, routing *pb.AgentEventMetadata) int {
	s.agentMu.RLock()
	defer s.agentMu.RUnlock()

	count := 0
	targetAgent := ""
	if routing != nil {
		targetAgent = routing.GetToAgentId()
	}

	if targetAgent != "" {
		// Count subscribers for specific agent
		switch eventType {
		case "message":
			if subs, ok := s.messageSubscribers[targetAgent]; ok {
				count += len(subs)
			}
		case "task":
			if subs, ok := s.taskSubscribers[targetAgent]; ok {
				count += len(subs)
			}
		}
		if subs, ok := s.eventSubscribers[targetAgent]; ok {
			count += len(subs)
		}
	} else {
		// Count all subscribers for broadcast
		switch eventType {
		case "message":
			for _, subs := range s.messageSubscribers {
				count += len(subs)
			}
		case "task":
			for _, subs := range s.taskSubscribers {
				count += len(subs)
			}
		}
		for _, subs := range s.eventSubscribers {
			count += len(subs)
		}
	}

	return count
}

// StartBroker creates and starts a broker with A2A compliance
func StartBroker(ctx context.Context) error {
	// Create gRPC configuration for broker
	config := NewGRPCConfig("broker")

	// Create AgentHub server
	server, err := NewAgentHubServer(config)
	if err != nil {
		return fmt.Errorf("failed to create AgentHub server: %w", err)
	}

	// Create AgentHub service
	agentHubService := NewAgentHubService(server)

	// Register the AgentHub service
	pb.RegisterAgentHubServer(server.Server, agentHubService)

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
