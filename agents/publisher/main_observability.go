//go:build observability
// +build observability

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
	"github.com/owulveryck/agenthub/internal/observability"
)

const (
	agentHubAddr     = "localhost:50051"      // Address of the AgentHub broker server
	publisherAgentID = "agent_demo_publisher" // This publisher's agent ID
)

type ObservablePublisher struct {
	client         pb.EventBusClient
	obs            *observability.Observability
	traceManager   *observability.TraceManager
	metricsManager *observability.MetricsManager
	healthServer   *observability.HealthServer
	logger         *slog.Logger
}

func NewObservablePublisher() (*ObservablePublisher, error) {
	// Initialize observability
	config := observability.DefaultConfig("agenthub-publisher")
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
	healthServer := observability.NewHealthServer("8081", config.ServiceName, config.ServiceVersion)

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

	return &ObservablePublisher{
		client:         client,
		obs:            obs,
		traceManager:   traceManager,
		metricsManager: metricsManager,
		healthServer:   healthServer,
		logger:         obs.Logger,
	}, nil
}

func (p *ObservablePublisher) publishTask(ctx context.Context, taskType string, params map[string]interface{}, responderAgentID string) error {
	// Start tracing for task publishing
	ctx, span := p.traceManager.StartPublishSpan(ctx, responderAgentID, taskType)
	defer span.End()

	// Start timing
	timer := p.metricsManager.StartTimer()
	defer timer(ctx, taskType, publisherAgentID)

	// Generate a unique task ID
	taskID := fmt.Sprintf("task_%s_%d", taskType, time.Now().Unix())

	p.logger.InfoContext(ctx, "Publishing task",
		slog.String("task_id", taskID),
		slog.String("task_type", taskType),
		slog.String("responder_agent_id", responderAgentID),
	)

	// Convert parameters to protobuf Struct
	parametersStruct, err := structpb.NewStruct(params)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error creating parameters struct",
			slog.String("task_id", taskID),
			slog.Any("error", err),
		)
		p.traceManager.RecordError(span, err)
		p.metricsManager.IncrementEventErrors(ctx, taskType, publisherAgentID, "struct_conversion_error")
		return err
	}

	// Inject trace context into task metadata
	headers := make(map[string]string)
	p.traceManager.InjectTraceContext(ctx, headers)

	// Convert headers to protobuf Struct
	metadataStruct, err := structpb.NewStruct(map[string]interface{}{
		"trace_headers": headers,
		"publisher":     publisherAgentID,
		"published_at":  time.Now().Format(time.RFC3339),
	})
	if err != nil {
		p.logger.WarnContext(ctx, "Error creating metadata struct",
			slog.String("task_id", taskID),
			slog.Any("error", err),
		)
		// Continue without metadata
		metadataStruct = &structpb.Struct{}
	}

	// Create task message
	task := &pb.TaskMessage{
		TaskId:           taskID,
		TaskType:         taskType,
		Parameters:       parametersStruct,
		RequesterAgentId: publisherAgentID,
		ResponderAgentId: responderAgentID,
		Priority:         pb.Priority_PRIORITY_MEDIUM,
		CreatedAt:        timestamppb.Now(),
		Metadata:         metadataStruct,
	}

	// Publish the task
	taskReq := &pb.PublishTaskRequest{
		Task: task,
	}

	res, err := p.client.PublishTask(ctx, taskReq)
	if err != nil {
		p.logger.ErrorContext(ctx, "Error publishing task",
			slog.String("task_id", taskID),
			slog.Any("error", err),
		)
		p.traceManager.RecordError(span, err)
		p.metricsManager.IncrementEventErrors(ctx, taskType, publisherAgentID, "grpc_error")
		return err
	}

	if !res.GetSuccess() {
		err := fmt.Errorf("failed to publish task: %s", res.GetError())
		p.logger.ErrorContext(ctx, "Failed to publish task",
			slog.String("task_id", taskID),
			slog.String("error", res.GetError()),
		)
		p.traceManager.RecordError(span, err)
		p.metricsManager.IncrementEventErrors(ctx, taskType, publisherAgentID, "publish_failed")
		return err
	}

	p.logger.InfoContext(ctx, "Task published successfully",
		slog.String("task_id", taskID),
		slog.String("task_type", taskType),
	)

	// Record successful metrics
	p.metricsManager.IncrementEventsProcessed(ctx, taskType, publisherAgentID, true)
	p.metricsManager.IncrementEventsPublished(ctx, taskType, responderAgentID)
	p.traceManager.SetSpanSuccess(span)

	return nil
}

func (p *ObservablePublisher) Run(ctx context.Context) error {
	p.logger.InfoContext(ctx, "Starting observable publisher demo")

	// Start health server
	go func() {
		p.logger.Info("Starting health server on port 8081")
		if err := p.healthServer.Start(ctx); err != nil {
			p.logger.Error("Health server failed", slog.Any("error", err))
		}
	}()

	// Start metrics collection
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.metricsManager.UpdateSystemMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	p.logger.InfoContext(ctx, "Testing Agent2Agent Task Publishing via AgentHub with observability")

	// Demo Task 1: Greeting task
	if err := p.publishTask(ctx, "greeting", map[string]interface{}{
		"name": "Claude",
	}, "agent_demo_subscriber"); err != nil {
		return fmt.Errorf("failed to publish greeting task: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Demo Task 2: Math calculation
	if err := p.publishTask(ctx, "math_calculation", map[string]interface{}{
		"operation": "add",
		"a":         42.0,
		"b":         58.0,
	}, "agent_demo_subscriber"); err != nil {
		return fmt.Errorf("failed to publish math calculation task: %w", err)
	}

	time.Sleep(2 * time.Second)

	// Demo Task 3: Random number generation
	if err := p.publishTask(ctx, "random_number", map[string]interface{}{
		"seed": 12345,
	}, "agent_demo_subscriber"); err != nil {
		return fmt.Errorf("failed to publish random number task: %w", err)
	}

	time.Sleep(2 * time.Second)

	// Demo Task 4: Unknown task type (should fail)
	if err := p.publishTask(ctx, "unknown_task", map[string]interface{}{
		"data": "test",
	}, "agent_demo_subscriber"); err != nil {
		p.logger.WarnContext(ctx, "Expected failure for unknown task type",
			slog.Any("error", err),
		)
	}

	p.logger.InfoContext(ctx, "All tasks published! Check subscriber logs for results")

	return nil
}

func (p *ObservablePublisher) Shutdown(ctx context.Context) error {
	p.logger.InfoContext(ctx, "Shutting down observable publisher")

	// Shutdown observability components
	if err := p.healthServer.Shutdown(ctx); err != nil {
		p.logger.ErrorContext(ctx, "Error shutting down health server", slog.Any("error", err))
	}

	if err := p.obs.Shutdown(ctx); err != nil {
		p.logger.ErrorContext(ctx, "Publisher observability shutdown failed - likely OTLP trace export issue",
			slog.Any("error", err),
			slog.String("service", "publisher"),
			slog.String("otlp_endpoint", p.obs.Config.JaegerEndpoint),
		)
		return err
	}

	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	publisher, err := NewObservablePublisher()
	if err != nil {
		panic(fmt.Sprintf("Failed to create observable publisher: %v", err))
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := publisher.Shutdown(shutdownCtx); err != nil {
			publisher.logger.Error("Error during shutdown", slog.Any("error", err))
		}
	}()

	if err := publisher.Run(ctx); err != nil {
		publisher.logger.Error("Publisher run failed", slog.Any("error", err))
		panic(err)
	}
}
