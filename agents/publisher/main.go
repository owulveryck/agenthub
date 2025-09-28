package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

const (
	agentHubAddr     = "localhost:50051"      // Address of the AgentHub broker server
	publisherAgentID = "agent_demo_publisher" // This publisher's agent ID
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewEventBusClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== Testing Agent2Agent Task Publishing via AgentHub ===")

	// Demo Task 1: Greeting task
	publishTask(ctx, client, "greeting", map[string]interface{}{
		"name": "Claude",
	}, "agent_demo_subscriber")

	time.Sleep(3 * time.Second)

	// Demo Task 2: Math calculation
	publishTask(ctx, client, "math_calculation", map[string]interface{}{
		"operation": "add",
		"a":         42.0,
		"b":         58.0,
	}, "agent_demo_subscriber")

	time.Sleep(2 * time.Second)

	// Demo Task 3: Random number generation
	publishTask(ctx, client, "random_number", map[string]interface{}{
		"seed": 12345,
	}, "agent_demo_subscriber")

	time.Sleep(2 * time.Second)

	// Demo Task 4: Unknown task type (should fail)
	publishTask(ctx, client, "unknown_task", map[string]interface{}{
		"data": "test",
	}, "agent_demo_subscriber")

	fmt.Println("\n=== All tasks published! Check subscriber logs for results ===")
}

// publishTask publishes a task to the specified agent
func publishTask(ctx context.Context, client pb.EventBusClient, taskType string, params map[string]interface{}, responderAgentID string) {
	// Generate a unique task ID
	taskID := fmt.Sprintf("task_%s_%d", taskType, time.Now().Unix())

	// Convert parameters to protobuf Struct
	parametersStruct, err := structpb.NewStruct(params)
	if err != nil {
		log.Printf("Error creating parameters struct: %v", err)
		return
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
	}

	// Publish the task
	taskReq := &pb.PublishTaskRequest{
		Task: task,
	}

	fmt.Printf("Publishing task: %s (type: %s) to agent: %s\n", taskID, taskType, responderAgentID)
	res, err := client.PublishTask(ctx, taskReq)
	if err != nil {
		log.Printf("Error publishing task %s: %v", taskID, err)
	} else if !res.GetSuccess() {
		log.Printf("Failed to publish task %s: %s", taskID, res.GetError())
	} else {
		fmt.Printf("Task %s published successfully.\n", taskID)
	}
}
