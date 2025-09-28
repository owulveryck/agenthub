package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/internal/grpc"
)

const (
	agentHubAddr = "localhost:50051"       // Address of the AgentHub broker server
	agentID      = "agent_demo_subscriber" // This agent's ID
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewEventBusClient(conn)

	// --- Task Subscribers for Agent2Agent Protocol ---
	// Task subscriber: Listen for tasks assigned to this agent
	taskCtx, taskCancel := context.WithCancel(context.Background())
	go func() {
		subscribeToTasks(taskCtx, client)
		taskCancel() // Signal main to stop if this subscriber exits
	}()

	// Task result subscriber: Listen for results of tasks this agent requested
	resultCtx, resultCancel := context.WithCancel(context.Background())
	go func() {
		subscribeToTaskResults(resultCtx, client)
		resultCancel() // Signal main to stop if this subscriber exits
	}()

	// Keep the main goroutine alive to allow subscribers to run.
	fmt.Println("Agent started. Listening for events and tasks. Press Enter to stop.")
	fmt.Scanln() // Wait for user input to stop

	fmt.Println("Stopping agent...")
	taskCancel()
	resultCancel()

	// Give subscribers a moment to clean up
	time.Sleep(1 * time.Second)
	fmt.Println("Agent stopped.")
}

// displayNotification shows a macOS notification using osascript
func displayNotification(title, subtitle, text string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "%s" sound name "Submarine"`, text, title, subtitle)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// subscribeToTasks subscribes to tasks assigned to this agent and processes them
func subscribeToTasks(ctx context.Context, client pb.EventBusClient) {
	log.Printf("Agent %s subscribing to tasks...", agentID)

	subscribeReq := &pb.SubscribeToTasksRequest{
		AgentId: agentID,
		// TaskTypes can be specified to filter, leaving empty to receive all tasks
	}

	stream, err := client.SubscribeToTasks(ctx, subscribeReq)
	if err != nil {
		log.Printf("Error subscribing to tasks: %v", err)
		return
	}

	log.Printf("Successfully subscribed to tasks for agent %s. Waiting for tasks...", agentID)

	for {
		task, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Task subscription stream closed by server.")
			return
		}
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("Task subscription context cancelled: %v", ctx.Err())
				return
			}
			log.Printf("Error receiving task: %v", err)
			return
		}

		log.Printf("Received task: %s (type: %s) from agent: %s",
			task.GetTaskId(), task.GetTaskType(), task.GetRequesterAgentId())

		// Process the task asynchronously
		go processTask(ctx, task, client)
	}
}

// subscribeToTaskResults subscribes to results of tasks this agent requested
func subscribeToTaskResults(ctx context.Context, client pb.EventBusClient) {
	log.Printf("Agent %s subscribing to task results...", agentID)

	subscribeReq := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: agentID,
		// TaskIds can be specified to filter, leaving empty to receive all results
	}

	stream, err := client.SubscribeToTaskResults(ctx, subscribeReq)
	if err != nil {
		log.Printf("Error subscribing to task results: %v", err)
		return
	}

	log.Printf("Successfully subscribed to task results for agent %s.", agentID)

	for {
		result, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Task result subscription stream closed by server.")
			return
		}
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("Task result subscription context cancelled: %v", ctx.Err())
				return
			}
			log.Printf("Error receiving task result: %v", err)
			return
		}

		log.Printf("Received task result for task: %s (status: %s) from agent: %s",
			result.GetTaskId(), result.GetStatus().String(), result.GetExecutorAgentId())

		// Process the result
		processTaskResult(result)
	}
}

// processTask simulates processing a task and sends back a result
func processTask(ctx context.Context, task *pb.TaskMessage, client pb.EventBusClient) {
	log.Printf("Processing task %s of type '%s'", task.GetTaskId(), task.GetTaskType())

	// Send progress update
	sendTaskProgress(ctx, task, 25, "Starting task processing", client)

	// Simulate different types of task processing
	var result *structpb.Struct
	var status pb.TaskStatus
	var errorMsg string

	switch task.GetTaskType() {
	case "greeting":
		// Simple greeting task
		time.Sleep(2 * time.Second) // Simulate processing time
		sendTaskProgress(ctx, task, 75, "Generating greeting", client)

		result, _ = structpb.NewStruct(map[string]interface{}{
			"message":   fmt.Sprintf("Hello from agent %s! Task %s completed successfully.", agentID, task.GetTaskId()),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		status = pb.TaskStatus_TASK_STATUS_COMPLETED

	case "math_calculation":
		// Simple math calculation
		time.Sleep(1 * time.Second)
		sendTaskProgress(ctx, task, 50, "Performing calculation", client)

		// Extract parameters
		params := task.GetParameters().AsMap()
		if operation, ok := params["operation"].(string); ok {
			if a, ok := params["a"].(float64); ok {
				if b, ok := params["b"].(float64); ok {
					var calcResult float64
					switch operation {
					case "add":
						calcResult = a + b
					case "subtract":
						calcResult = a - b
					case "multiply":
						calcResult = a * b
					case "divide":
						if b != 0 {
							calcResult = a / b
						} else {
							status = pb.TaskStatus_TASK_STATUS_FAILED
							errorMsg = "Division by zero"
						}
					default:
						status = pb.TaskStatus_TASK_STATUS_FAILED
						errorMsg = "Unknown operation: " + operation
					}

					if status != pb.TaskStatus_TASK_STATUS_FAILED {
						result, _ = structpb.NewStruct(map[string]interface{}{
							"operation": operation,
							"operand_a": a,
							"operand_b": b,
							"result":    calcResult,
						})
						status = pb.TaskStatus_TASK_STATUS_COMPLETED
					}
				}
			}
		}

		if status == pb.TaskStatus_TASK_STATUS_UNSPECIFIED {
			status = pb.TaskStatus_TASK_STATUS_FAILED
			errorMsg = "Invalid math calculation parameters"
		}

	case "random_number":
		// Generate random number
		time.Sleep(500 * time.Millisecond)
		sendTaskProgress(ctx, task, 90, "Generating random number", client)

		randomNum := rand.Intn(1000)
		result, _ = structpb.NewStruct(map[string]interface{}{
			"random_number": randomNum,
			"range":         "0-999",
		})
		status = pb.TaskStatus_TASK_STATUS_COMPLETED

	default:
		// Unknown task type
		status = pb.TaskStatus_TASK_STATUS_FAILED
		errorMsg = fmt.Sprintf("Unknown task type: %s", task.GetTaskType())
	}

	// Send final progress update
	sendTaskProgress(ctx, task, 100, "Task completed", client)

	// Send result
	taskResult := &pb.TaskResult{
		TaskId:            task.GetTaskId(),
		Status:            status,
		Result:            result,
		ErrorMessage:      errorMsg,
		ExecutorAgentId:   agentID,
		CompletedAt:       timestamppb.Now(),
		ExecutionMetadata: nil, // Could include execution details
	}

	resultReq := &pb.PublishTaskResultRequest{
		Result: taskResult,
	}

	if _, err := client.PublishTaskResult(ctx, resultReq); err != nil {
		log.Printf("Error publishing task result for %s: %v", task.GetTaskId(), err)
	} else {
		log.Printf("Published result for task %s with status %s", task.GetTaskId(), status.String())
	}

	// Display notification
	title := fmt.Sprintf("Task Completed: %s", task.GetTaskType())
	subtitle := fmt.Sprintf("Agent %s", agentID)
	text := fmt.Sprintf("Task %s completed with status: %s", task.GetTaskId(), status.String())

	if err := displayNotification(title, subtitle, text); err != nil {
		log.Printf("Error displaying notification: %v", err)
	}
}

// sendTaskProgress sends a progress update for a task
func sendTaskProgress(ctx context.Context, task *pb.TaskMessage, percentage int32, message string, client pb.EventBusClient) {
	progress := &pb.TaskProgress{
		TaskId:             task.GetTaskId(),
		Status:             pb.TaskStatus_TASK_STATUS_IN_PROGRESS,
		ProgressMessage:    message,
		ProgressPercentage: percentage,
		ExecutorAgentId:    agentID,
		UpdatedAt:          timestamppb.Now(),
	}

	progressReq := &pb.PublishTaskProgressRequest{
		Progress: progress,
	}

	if _, err := client.PublishTaskProgress(ctx, progressReq); err != nil {
		log.Printf("Error publishing task progress for %s: %v", task.GetTaskId(), err)
	} else {
		log.Printf("Published progress for task %s: %d%% - %s", task.GetTaskId(), percentage, message)
	}
}

// processTaskResult handles received task results
func processTaskResult(result *pb.TaskResult) {
	log.Printf("Task %s completed with status: %s", result.GetTaskId(), result.GetStatus().String())

	if result.GetStatus() == pb.TaskStatus_TASK_STATUS_COMPLETED {
		if result.GetResult() != nil {
			log.Printf("Task result data: %+v", result.GetResult().AsMap())
		}
	} else if result.GetStatus() == pb.TaskStatus_TASK_STATUS_FAILED {
		log.Printf("Task failed with error: %s", result.GetErrorMessage())
	}

	// Display notification
	title := fmt.Sprintf("Task Result: %s", result.GetTaskId())
	subtitle := "Task Response"
	text := fmt.Sprintf("Status: %s", result.GetStatus().String())

	if err := displayNotification(title, subtitle, text); err != nil {
		log.Printf("Error displaying notification: %v", err)
	}
}
