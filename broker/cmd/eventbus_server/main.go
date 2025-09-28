package main

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/owulveryck/agenthub/broker/internal/grpc"
)

const (
	port = ":50051" // Port for the gRPC server
)

// AgentHub broker server - implements in-memory routing for Agent2Agent protocol tasks
type eventBusServer struct {
	pb.UnimplementedEventBusServer // Embed for forward compatibility

	// Task-specific subscribers
	taskSubscribers         map[string][]chan *pb.TaskMessage  // Agent ID -> task channels
	taskResultSubscribers   map[string][]chan *pb.TaskResult   // Requester Agent ID -> result channels
	taskProgressSubscribers map[string][]chan *pb.TaskProgress // Requester Agent ID -> progress channels
	taskMu                  sync.RWMutex                       // Protects task subscribers
}

// NewEventBusServer creates a new instance of the event bus server.
func NewEventBusServer() *eventBusServer {
	return &eventBusServer{
		taskSubscribers:         make(map[string][]chan *pb.TaskMessage),
		taskResultSubscribers:   make(map[string][]chan *pb.TaskResult),
		taskProgressSubscribers: make(map[string][]chan *pb.TaskProgress),
	}
}

// PublishTask handles incoming Agent2Agent protocol tasks via AgentHub broker
func (s *eventBusServer) PublishTask(ctx context.Context, req *pb.PublishTaskRequest) (*pb.PublishResponse, error) {
	if req.GetTask() == nil {
		return nil, status.Error(codes.InvalidArgument, "task cannot be nil")
	}
	if req.GetTask().GetTaskId() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id cannot be empty")
	}
	if req.GetTask().GetTaskType() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_type cannot be empty")
	}
	if req.GetTask().GetRequesterAgentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
	}

	log.Printf("Received task request: %s (type: %s) from agent: %s",
		req.GetTask().GetTaskId(), req.GetTask().GetTaskType(), req.GetTask().GetRequesterAgentId())

	s.taskMu.RLock()
	// Route to specific agent or broadcast to all if no specific responder
	var targetChannels []chan *pb.TaskMessage
	if responderID := req.GetTask().GetResponderAgentId(); responderID != "" {
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
		log.Printf("No subscribers for task from agent '%s'", req.GetTask().GetRequesterAgentId())
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		taskToSend := *req.GetTask()
		go func(ch chan *pb.TaskMessage, task pb.TaskMessage) {
			select {
			case ch <- &task:
				// Message sent successfully
			case <-ctx.Done():
				log.Printf("Context cancelled while sending task %s", task.GetTaskId())
			case <-time.After(5 * time.Second):
				log.Printf("Timeout sending task %s. Dropping message.", task.GetTaskId())
			}
		}(subChan, taskToSend)
	}

	return &pb.PublishResponse{Success: true}, nil
}

// PublishTaskResult handles task completion responses
func (s *eventBusServer) PublishTaskResult(ctx context.Context, req *pb.PublishTaskResultRequest) (*pb.PublishResponse, error) {
	if req.GetResult() == nil {
		return nil, status.Error(codes.InvalidArgument, "result cannot be nil")
	}
	if req.GetResult().GetTaskId() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id cannot be empty")
	}

	log.Printf("Received task result for task: %s from agent: %s",
		req.GetResult().GetTaskId(), req.GetResult().GetExecutorAgentId())

	s.taskMu.RLock()
	// Find subscribers for this task result - we need to determine the original requester
	// For now, broadcast to all result subscribers
	var targetChannels []chan *pb.TaskResult
	for _, subs := range s.taskResultSubscribers {
		targetChannels = append(targetChannels, subs...)
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		log.Printf("No subscribers for task result %s", req.GetResult().GetTaskId())
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		resultToSend := *req.GetResult()
		go func(ch chan *pb.TaskResult, result pb.TaskResult) {
			select {
			case ch <- &result:
				// Message sent successfully
			case <-ctx.Done():
				log.Printf("Context cancelled while sending task result %s", result.GetTaskId())
			case <-time.After(5 * time.Second):
				log.Printf("Timeout sending task result %s. Dropping message.", result.GetTaskId())
			}
		}(subChan, resultToSend)
	}

	return &pb.PublishResponse{Success: true}, nil
}

// PublishTaskProgress handles task progress updates
func (s *eventBusServer) PublishTaskProgress(ctx context.Context, req *pb.PublishTaskProgressRequest) (*pb.PublishResponse, error) {
	if req.GetProgress() == nil {
		return nil, status.Error(codes.InvalidArgument, "progress cannot be nil")
	}
	if req.GetProgress().GetTaskId() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_id cannot be empty")
	}

	log.Printf("Received task progress for task: %s (%d%%) from agent: %s",
		req.GetProgress().GetTaskId(), req.GetProgress().GetProgressPercentage(), req.GetProgress().GetExecutorAgentId())

	s.taskMu.RLock()
	// Broadcast progress to all progress subscribers
	var targetChannels []chan *pb.TaskProgress
	for _, subs := range s.taskProgressSubscribers {
		targetChannels = append(targetChannels, subs...)
	}
	s.taskMu.RUnlock()

	if len(targetChannels) == 0 {
		log.Printf("No subscribers for task progress %s", req.GetProgress().GetTaskId())
		return &pb.PublishResponse{Success: true}, nil
	}

	// Send to each subscriber
	for _, subChan := range targetChannels {
		progressToSend := *req.GetProgress()
		go func(ch chan *pb.TaskProgress, progress pb.TaskProgress) {
			select {
			case ch <- &progress:
				// Message sent successfully
			case <-ctx.Done():
				log.Printf("Context cancelled while sending task progress %s", progress.GetTaskId())
			case <-time.After(5 * time.Second):
				log.Printf("Timeout sending task progress %s. Dropping message.", progress.GetTaskId())
			}
		}(subChan, progressToSend)
	}

	return &pb.PublishResponse{Success: true}, nil
}

// SubscribeToTasks allows agents to subscribe to tasks assigned to them
func (s *eventBusServer) SubscribeToTasks(req *pb.SubscribeToTasksRequest, stream pb.EventBus_SubscribeToTasksServer) error {
	agentID := req.GetAgentId()
	if agentID == "" {
		return status.Error(codes.InvalidArgument, "agent_id cannot be empty")
	}

	log.Printf("Agent %s subscribed to tasks", agentID)

	// Create a channel for this subscriber
	subChan := make(chan *pb.TaskMessage, 10)

	// Add the subscriber channel to our map
	s.taskMu.Lock()
	s.taskSubscribers[agentID] = append(s.taskSubscribers[agentID], subChan)
	s.taskMu.Unlock()

	// Clean up when done
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
		log.Printf("Agent %s unsubscribed from tasks", agentID)
	}()

	// Stream tasks back to the client
	for {
		select {
		case task, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(task); err != nil {
				log.Printf("Error sending task to agent %s: %v", agentID, err)
				return err
			}
		case <-stream.Context().Done():
			log.Printf("Task subscription context done for agent %s", agentID)
			return stream.Context().Err()
		}
	}
}

// SubscribeToTaskResults allows agents to subscribe to results of tasks they requested
func (s *eventBusServer) SubscribeToTaskResults(req *pb.SubscribeToTaskResultsRequest, stream pb.EventBus_SubscribeToTaskResultsServer) error {
	requesterID := req.GetRequesterAgentId()
	if requesterID == "" {
		return status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
	}

	log.Printf("Agent %s subscribed to task results", requesterID)

	// Create a channel for this subscriber
	subChan := make(chan *pb.TaskResult, 10)

	// Add the subscriber channel to our map
	s.taskMu.Lock()
	s.taskResultSubscribers[requesterID] = append(s.taskResultSubscribers[requesterID], subChan)
	s.taskMu.Unlock()

	// Clean up when done
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
		log.Printf("Agent %s unsubscribed from task results", requesterID)
	}()

	// Stream results back to the client
	for {
		select {
		case result, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(result); err != nil {
				log.Printf("Error sending task result to agent %s: %v", requesterID, err)
				return err
			}
		case <-stream.Context().Done():
			log.Printf("Task result subscription context done for agent %s", requesterID)
			return stream.Context().Err()
		}
	}
}

// SubscribeToTaskProgress allows agents to subscribe to progress updates of tasks they requested
func (s *eventBusServer) SubscribeToTaskProgress(req *pb.SubscribeToTaskResultsRequest, stream pb.EventBus_SubscribeToTaskProgressServer) error {
	requesterID := req.GetRequesterAgentId()
	if requesterID == "" {
		return status.Error(codes.InvalidArgument, "requester_agent_id cannot be empty")
	}

	log.Printf("Agent %s subscribed to task progress", requesterID)

	// Create a channel for this subscriber
	subChan := make(chan *pb.TaskProgress, 10)

	// Add the subscriber channel to our map
	s.taskMu.Lock()
	s.taskProgressSubscribers[requesterID] = append(s.taskProgressSubscribers[requesterID], subChan)
	s.taskMu.Unlock()

	// Clean up when done
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
		log.Printf("Agent %s unsubscribed from task progress", requesterID)
	}()

	// Stream progress back to the client
	for {
		select {
		case progress, ok := <-subChan:
			if !ok {
				return nil
			}
			if err := stream.Send(progress); err != nil {
				log.Printf("Error sending task progress to agent %s: %v", requesterID, err)
				return err
			}
		case <-stream.Context().Done():
			log.Printf("Task progress subscription context done for agent %s", requesterID)
			return stream.Context().Err()
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	eventBusService := NewEventBusServer()
	pb.RegisterEventBusServer(grpcServer, eventBusService)

	log.Printf("AgentHub broker gRPC server listening on %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
