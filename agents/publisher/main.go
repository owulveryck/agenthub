package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
)

const (
	publisherAgentID = "agent_demo_publisher"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create gRPC configuration for publisher
	config := agenthub.NewGRPCConfig("publisher")
	config.HealthPort = "8081" // Different port for publisher health

	// Create AgentHub client
	client, err := agenthub.NewAgentHubClient(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create AgentHub client: %v", err))
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := client.Shutdown(shutdownCtx); err != nil {
			client.Logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
		}
	}()

	// Start the client
	if err := client.Start(ctx); err != nil {
		client.Logger.ErrorContext(ctx, "Failed to start client", "error", err)
		panic(err)
	}

	// Create A2A task publisher
	taskPublisher := &agenthub.A2ATaskPublisher{
		Client:         client.Client,
		TraceManager:   client.TraceManager,
		MetricsManager: client.MetricsManager,
		Logger:         client.Logger,
		ComponentName:  "publisher",
		AgentID:        publisherAgentID,
	}

	client.Logger.InfoContext(ctx, "Starting publisher demo")
	client.Logger.InfoContext(ctx, "Testing Agent2Agent Task Publishing via AgentHub with observability")

	// Demo Task 1: Greeting task (A2A-compliant)
	task1, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
		TaskType: "greeting",
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: "Hello! Please provide a greeting for Claude.",
				},
			},
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to publish greeting task: %v", err))
	}
	client.Logger.InfoContext(ctx, "Published greeting task", "task_id", task1.GetId())

	time.Sleep(3 * time.Second)

	// Demo Task 2: Math calculation (A2A-compliant)
	task2, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
		TaskType: "math_calculation",
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: "Please calculate 42 + 58.",
				},
			},
			{
				Part: &pb.Part_Data{
					Data: &pb.DataPart{
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"operation": structpb.NewStringValue("add"),
								"a":         structpb.NewNumberValue(42.0),
								"b":         structpb.NewNumberValue(58.0),
							},
						},
					},
				},
			},
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to publish math calculation task: %v", err))
	}
	client.Logger.InfoContext(ctx, "Published math task", "task_id", task2.GetId())

	time.Sleep(2 * time.Second)

	// Demo Task 3: Random number generation (A2A-compliant)
	task3, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
		TaskType: "random_number",
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: "Please generate a random number using seed 12345.",
				},
			},
			{
				Part: &pb.Part_Data{
					Data: &pb.DataPart{
						Data: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"seed": structpb.NewNumberValue(12345),
							},
						},
					},
				},
			},
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to publish random number task: %v", err))
	}
	client.Logger.InfoContext(ctx, "Published random number task", "task_id", task3.GetId())

	time.Sleep(2 * time.Second)

	// Demo Task 4: Unknown task type (should fail)
	task4, err := taskPublisher.PublishTask(ctx, &agenthub.A2APublishTaskRequest{
		TaskType: "unknown_task",
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: "This is an unknown task type for testing error handling.",
				},
			},
		},
		RequesterAgentID: publisherAgentID,
		ResponderAgentID: "agent_demo_subscriber",
		Priority:         pb.Priority_PRIORITY_MEDIUM,
	})
	if err != nil {
		client.Logger.InfoContext(ctx, "Expected failure for unknown task type", "error", err)
	} else {
		client.Logger.InfoContext(ctx, "Published unknown task", "task_id", task4.GetId())
	}

	client.Logger.InfoContext(ctx, "All tasks published! Check subscriber logs for results")
}
