package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/subagent"
	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	config := &subagent.Config{
		AgentID:     "agent_mp3",
		ServiceName: "mp3_agent",
		Name:        "Audio Analyzer",
		Description: "Analyzes MP3 and M4A audio files: extracts metadata and optionally transcribes via Gemini",
		Version:     "1.1.0",
		HealthPort:  "8086",
	}

	agent, err := subagent.New(config)
	if err != nil {
		log.Fatal(err)
	}

	agent.MustAddSkill(
		"Analyze Audio",
		"Reads an MP3 or M4A file, extracts metadata, and optionally transcribes audio via Gemini",
		mp3Handler,
	)

	if err := agent.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func mp3Handler(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
	// Log received message for debugging
	var msgText string
	for _, part := range message.Content {
		if text := part.GetText(); text != "" {
			msgText = text
			break
		}
	}
	log.Printf("[mp3_agent] Received task %s, message content: %q", task.GetId(), msgText)

	// Extract file path from message content
	var filePath string
	for _, part := range message.Content {
		if text := part.GetText(); text != "" {
			filePath = extractFilePath(text)
			break
		}
	}

	if filePath == "" {
		return nil, pb.TaskState_TASK_STATE_FAILED, "No file path provided in message"
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, pb.TaskState_TASK_STATE_FAILED, fmt.Sprintf("Failed to open file: %v", err)
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, pb.TaskState_TASK_STATE_FAILED, fmt.Sprintf("Failed to read tags: %v", err)
	}

	// Format metadata
	var sb strings.Builder
	sb.WriteString("Audio Metadata Analysis\n")
	sb.WriteString("=======================\n")
	writeField(&sb, "Title", m.Title())
	writeField(&sb, "Artist", m.Artist())
	writeField(&sb, "Album", m.Album())
	writeField(&sb, "Year", fmt.Sprintf("%d", m.Year()))
	writeField(&sb, "Genre", m.Genre())
	writeField(&sb, "Format", string(m.Format()))
	writeField(&sb, "File Type", string(m.FileType()))

	trackNum, trackTotal := m.Track()
	if trackTotal > 0 {
		writeField(&sb, "Track", fmt.Sprintf("%d/%d", trackNum, trackTotal))
	} else if trackNum > 0 {
		writeField(&sb, "Track", fmt.Sprintf("%d", trackNum))
	}

	discNum, discTotal := m.Disc()
	if discTotal > 0 {
		writeField(&sb, "Disc", fmt.Sprintf("%d/%d", discNum, discTotal))
	} else if discNum > 0 {
		writeField(&sb, "Disc", fmt.Sprintf("%d", discNum))
	}

	if m.Comment() != "" {
		writeField(&sb, "Comment", m.Comment())
	}

	if m.Picture() != nil {
		writeField(&sb, "Cover Art", fmt.Sprintf("Yes (%s, %d bytes)", m.Picture().MIMEType, len(m.Picture().Data)))
	}

	// Optional Gemini transcription
	var hasTranscription bool
	gcpProject := os.Getenv("GCP_PROJECT")
	if gcpProject != "" {
		transcript, err := transcribeAudio(ctx, filePath, gcpProject)
		if err != nil {
			sb.WriteString(fmt.Sprintf("\nTranscription: [failed - %v]\n", err))
		} else {
			sb.WriteString("\nTranscription\n")
			sb.WriteString("-------------\n")
			sb.WriteString(transcript)
			sb.WriteString("\n")
			hasTranscription = true
		}
	}

	resultText := sb.String()

	artifactName := "audio_metadata"
	artifactDesc := "Audio file metadata analysis"
	metadataFields := map[string]*structpb.Value{
		"file_path":    structpb.NewStringValue(filePath),
		"processed_at": structpb.NewStringValue(time.Now().Format(time.RFC3339)),
	}
	if hasTranscription {
		artifactName = "transcription"
		artifactDesc = "Audio transcription with metadata"
		metadataFields["content_type"] = structpb.NewStringValue("transcription")
	}

	artifact := &pb.Artifact{
		ArtifactId:  fmt.Sprintf("audio_%s_%d", task.GetId(), time.Now().Unix()),
		Name:        artifactName,
		Description: artifactDesc,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: resultText,
				},
			},
		},
		Metadata: &structpb.Struct{
			Fields: metadataFields,
		},
	}

	return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
}

// extractFilePath finds a file path in the message text.
// Looks for patterns like "/tmp/xxx.mp3", "/path/to/file.m4a".
func extractFilePath(text string) string {
	for _, word := range strings.Fields(text) {
		lower := strings.ToLower(word)
		if strings.HasPrefix(word, "/") && (strings.HasSuffix(lower, ".mp3") || strings.HasSuffix(lower, ".m4a")) {
			return word
		}
	}
	return ""
}

func transcribeAudio(ctx context.Context, filePath, gcpProject string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := "audio/mpeg"
	if ext == ".m4a" {
		mimeType = "audio/mp4"
	}

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

	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromBytes(data, mimeType),
			genai.NewPartFromText("Transcribe the audio content. Return only the transcription text."),
		}, genai.RoleUser),
	}

	result, err := client.Models.GenerateContent(ctx, model, contents, nil)
	if err != nil {
		return "", fmt.Errorf("generate content: %w", err)
	}

	if len(result.Candidates) == 0 || result.Candidates[0].Content == nil || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func writeField(sb *strings.Builder, label, value string) {
	if value != "" && value != "0" {
		fmt.Fprintf(sb, "%-12s: %s\n", label, value)
	}
}
