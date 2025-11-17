package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/subagent"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	// Create agent configuration
	config := &subagent.Config{
		AgentID:     "agent_echo",
		Name:        "Echo Agent",
		Description: "A simple agent that echoes messages back to demonstrate task delegation",
		Version:     "1.0.0",
		HealthPort:  "8085",
	}

	// Create the subagent
	agent, err := subagent.New(config)
	if err != nil {
		log.Fatal(err)
	}

	// Register the echo skill with its handler
	agent.MustAddSkill(
		"Echo Messages",
		"Echoes the input text back to the sender",
		echoHandler,
	)

	// Run the agent (blocks until shutdown signal)
	if err := agent.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// echoHandler implements the echo logic
func echoHandler(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
	// Extract input from message
	var input string
	for _, part := range message.Content {
		if text := part.GetText(); text != "" {
			input = text
			break
		}
	}

	if input == "" {
		return nil, pb.TaskState_TASK_STATE_FAILED, "No input text provided"
	}

	// Create echo response
	echoText := fmt.Sprintf("Echo: %s", input)

	// Create artifact with the echo response
	artifact := &pb.Artifact{
		ArtifactId:  fmt.Sprintf("echo_%s_%d", task.GetId(), time.Now().Unix()),
		Name:        "echo_response",
		Description: "Echoed message",
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: echoText,
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"original_input": structpb.NewStringValue(input),
				"processed_at":   structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
}
