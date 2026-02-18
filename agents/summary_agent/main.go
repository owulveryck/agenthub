package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/subagent"
	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	config := &subagent.Config{
		AgentID:     "agent_summary",
		ServiceName: "summary_agent",
		Name:        "Summary Generator",
		Description: "Generates structured meeting summaries from transcriptions using generative AI",
		Version:     "1.0.0",
		HealthPort:  "8087",
	}

	agent, err := subagent.New(config)
	if err != nil {
		log.Fatal(err)
	}

	agent.MustAddSkill(
		"Generate Summary",
		"Takes a transcription text and produces a structured meeting summary with key topics, decisions, and action items",
		summaryHandler,
	)

	if err := agent.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func summaryHandler(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
	// Extract text content from the incoming message
	var inputText string
	for _, part := range message.Content {
		if text := part.GetText(); text != "" {
			inputText = text
			break
		}
	}

	if inputText == "" {
		return nil, pb.TaskState_TASK_STATE_FAILED, "No input text provided"
	}

	log.Printf("[summary_agent] Received task %s, input length: %d", task.GetId(), len(inputText))

	gcpProject := os.Getenv("GCP_PROJECT")
	if gcpProject == "" {
		return nil, pb.TaskState_TASK_STATE_FAILED, "GCP_PROJECT environment variable is required for summary generation"
	}

	summary, err := generateSummary(ctx, inputText, gcpProject)
	if err != nil {
		return nil, pb.TaskState_TASK_STATE_FAILED, fmt.Sprintf("Failed to generate summary: %v", err)
	}

	artifact := &pb.Artifact{
		ArtifactId:  fmt.Sprintf("summary_%s_%d", task.GetId(), time.Now().Unix()),
		Name:        "meeting_summary",
		Description: "AI-generated meeting summary",
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: summary,
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"content_type": structpb.NewStringValue("summary"),
				"processed_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
			},
		},
	}

	return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
}

func generateSummary(ctx context.Context, transcription, gcpProject string) (string, error) {
	location := os.Getenv("GCP_LOCATION")
	if location == "" {
		location = "us-central1"
	}
	model := os.Getenv("VERTEX_AI_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  gcpProject,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return "", fmt.Errorf("create genai client: %w", err)
	}

	prompt := fmt.Sprintf(`From the following transcription, create a structured meeting summary with:
- Main topics discussed
- Key decisions made
- Action items identified
- A brief synthesis

Transcription:
%s`, transcription)

	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromText(prompt),
		}, genai.RoleUser),
	}

	result, err := client.Models.GenerateContent(ctx, model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	if len(result.Candidates) == 0 || result.Candidates[0].Content == nil || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	// Collect all text parts from the response
	var parts []string
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}

	return strings.Join(parts, "\n"), nil
}
