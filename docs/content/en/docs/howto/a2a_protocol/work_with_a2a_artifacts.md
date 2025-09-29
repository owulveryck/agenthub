---
title: "How to Work with A2A Artifacts"
weight: 30
description: "Learn how to create, structure, and deliver Agent2Agent protocol artifacts as structured outputs from completed tasks."
---

# How to Work with A2A Artifacts

This guide shows you how to create and work with Agent2Agent (A2A) protocol artifacts, which are structured outputs delivered when tasks are completed. Artifacts provide rich, typed results that can include text reports, data files, structured data, and more.

## Understanding A2A Artifacts

A2A artifacts are structured containers for task outputs that include:

- **Artifact ID**: Unique identifier for the artifact
- **Name**: Human-readable name for the artifact
- **Description**: Explanation of what the artifact contains
- **Parts**: The actual content (text, data, files)
- **Metadata**: Additional context about the artifact

Artifacts are typically generated when tasks reach `TASK_STATE_COMPLETED` status.

## Creating Basic Artifacts

### Text Report Artifacts

Create simple text-based results:

```go
package main

import (
    "fmt"
    "github.com/google/uuid"
    pb "github.com/owulveryck/agenthub/events/a2a"
    "google.golang.org/protobuf/types/known/structpb"
)

func createTextReportArtifact(taskID, reportContent string) *pb.Artifact {
    return &pb.Artifact{
        ArtifactId:  fmt.Sprintf("artifact_%s_%s", taskID, uuid.New().String()),
        Name:        "Analysis Report",
        Description: "Detailed analysis results and recommendations",
        Parts: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: reportContent,
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "artifact_type": structpb.NewStringValue("report"),
                "format":        structpb.NewStringValue("text"),
                "task_id":       structpb.NewStringValue(taskID),
                "generated_at":  structpb.NewStringValue(time.Now().Format(time.RFC3339)),
            },
        },
    }
}
```

### Data Analysis Artifacts

Create artifacts with structured analysis results:

```go
func createDataAnalysisArtifact(taskID string, results map[string]interface{}) *pb.Artifact {
    // Convert results to structured data
    resultsData, err := structpb.NewStruct(results)
    if err != nil {
        log.Printf("Error creating results data: %v", err)
        resultsData = &structpb.Struct{}
    }

    // Create summary statistics
    summary := map[string]interface{}{
        "total_records":    results["record_count"],
        "processing_time":  results["duration_ms"],
        "success_rate":     results["success_percentage"],
        "anomalies_found":  results["anomaly_count"],
    }
    summaryData, _ := structpb.NewStruct(summary)

    return &pb.Artifact{
        ArtifactId:  fmt.Sprintf("artifact_analysis_%s", uuid.New().String()),
        Name:        "Data Analysis Results",
        Description: "Complete analysis results with statistics and insights",
        Parts: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Data analysis completed successfully. See attached results for detailed findings.",
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        resultsData,
                        Description: "Complete analysis results",
                    },
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        summaryData,
                        Description: "Summary statistics",
                    },
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "artifact_type":   structpb.NewStringValue("analysis"),
                "analysis_type":   structpb.NewStringValue("statistical"),
                "data_source":     structpb.NewStringValue(results["source"].(string)),
                "record_count":    structpb.NewNumberValue(results["record_count"].(float64)),
                "processing_time": structpb.NewNumberValue(results["duration_ms"].(float64)),
            },
        },
    }
}
```

### File-Based Artifacts

Create artifacts that reference generated files:

```go
func createFileArtifact(taskID, fileID, filename, mimeType string, sizeBytes int64) *pb.Artifact {
    // File metadata
    fileMetadata, _ := structpb.NewStruct(map[string]interface{}{
        "generated_by":   "data_processor_v1.2",
        "file_version":   "1.0",
        "encoding":       "utf-8",
        "compression":    "gzip",
        "checksum_sha256": "abc123...", // Calculate actual checksum
    })

    return &pb.Artifact{
        ArtifactId:  fmt.Sprintf("artifact_file_%s", uuid.New().String()),
        Name:        "Processed Dataset",
        Description: "Cleaned and processed dataset ready for analysis",
        Parts: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: fmt.Sprintf("Dataset processing completed. Generated file: %s", filename),
                },
            },
            {
                Part: &pb.Part_File{
                    File: &pb.FilePart{
                        FileId:   fileID,
                        Filename: filename,
                        MimeType: mimeType,
                        SizeBytes: sizeBytes,
                        Metadata:  fileMetadata,
                    },
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "artifact_type":    structpb.NewStringValue("file"),
                "file_type":        structpb.NewStringValue("dataset"),
                "original_task":    structpb.NewStringValue(taskID),
                "processing_stage": structpb.NewStringValue("cleaned"),
            },
        },
    }
}
```

## Complex Multi-Part Artifacts

### Complete Analysis Package

Create comprehensive artifacts with multiple content types:

```go
func createCompleteAnalysisArtifact(taskID string, analysisResults map[string]interface{}) *pb.Artifact {
    // Executive summary
    summary := fmt.Sprintf(`
Analysis Complete: %s

Key Findings:
- Processed %v records
- Found %v anomalies
- Success rate: %v%%
- Processing time: %v ms

Recommendations:
%s
`,
        analysisResults["dataset_name"],
        analysisResults["record_count"],
        analysisResults["anomaly_count"],
        analysisResults["success_percentage"],
        analysisResults["duration_ms"],
        analysisResults["recommendations"],
    )

    // Detailed results data
    detailedResults, _ := structpb.NewStruct(analysisResults)

    // Configuration used
    configData, _ := structpb.NewStruct(map[string]interface{}{
        "algorithm":         "statistical_analysis_v2",
        "confidence_level":  0.95,
        "outlier_threshold": 2.5,
        "normalization":     "z-score",
    })

    return &pb.Artifact{
        ArtifactId:  fmt.Sprintf("artifact_complete_%s", uuid.New().String()),
        Name:        "Complete Analysis Package",
        Description: "Full analysis results including summary, data, configuration, and generated files",
        Parts: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: summary,
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        detailedResults,
                        Description: "Detailed analysis results and metrics",
                    },
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        configData,
                        Description: "Analysis configuration parameters",
                    },
                },
            },
            {
                Part: &pb.Part_File{
                    File: &pb.FilePart{
                        FileId:   "results_visualization_123",
                        Filename: "analysis_charts.png",
                        MimeType: "image/png",
                        SizeBytes: 1024000,
                    },
                },
            },
            {
                Part: &pb.Part_File{
                    File: &pb.FilePart{
                        FileId:   "results_dataset_456",
                        Filename: "processed_data.csv",
                        MimeType: "text/csv",
                        SizeBytes: 5120000,
                    },
                },
            },
        },
        Metadata: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "artifact_type":     structpb.NewStringValue("complete_package"),
                "analysis_type":     structpb.NewStringValue("comprehensive"),
                "includes_files":    structpb.NewBoolValue(true),
                "includes_data":     structpb.NewBoolValue(true),
                "includes_summary":  structpb.NewBoolValue(true),
                "file_count":        structpb.NewNumberValue(2),
                "total_size_bytes":  structpb.NewNumberValue(6144000),
            },
        },
    }
}
```

## Publishing Artifacts

### Using A2A Task Completion

Publish artifacts when completing tasks:

```go
import (
    "context"
    "github.com/owulveryck/agenthub/internal/agenthub"
    eventbus "github.com/owulveryck/agenthub/events/eventbus"
)

func completeTaskWithArtifact(ctx context.Context, client eventbus.AgentHubClient, task *pb.Task, artifact *pb.Artifact) error {
    // Update task status to completed
    task.Status = &pb.TaskStatus{
        State: pb.TaskState_TASK_STATE_COMPLETED,
        Update: &pb.Message{
            MessageId: fmt.Sprintf("msg_completion_%s", uuid.New().String()),
            ContextId: task.GetContextId(),
            TaskId:    task.GetId(),
            Role:      pb.Role_AGENT,
            Content: []*pb.Part{
                {
                    Part: &pb.Part_Text{
                        Text: "Task completed successfully. Artifact has been generated.",
                    },
                },
            },
        },
        Timestamp: timestamppb.Now(),
    }

    // Add artifact to task
    task.Artifacts = append(task.Artifacts, artifact)

    // Publish task completion
    _, err := client.PublishTaskUpdate(ctx, &eventbus.PublishTaskUpdateRequest{
        Task: task,
        Routing: &eventbus.AgentEventMetadata{
            FromAgentId: "processing_agent",
            ToAgentId:   "", // Broadcast completion
            EventType:   "task.completed",
            Priority:    eventbus.Priority_PRIORITY_MEDIUM,
        },
    })

    if err != nil {
        return fmt.Errorf("failed to publish task completion: %w", err)
    }

    // Separately publish artifact update
    return publishArtifactUpdate(ctx, client, task.GetId(), artifact)
}

func publishArtifactUpdate(ctx context.Context, client eventbus.AgentHubClient, taskID string, artifact *pb.Artifact) error {
    _, err := client.PublishTaskArtifact(ctx, &eventbus.PublishTaskArtifactRequest{
        TaskId:   taskID,
        Artifact: artifact,
        Routing: &eventbus.AgentEventMetadata{
            FromAgentId: "processing_agent",
            ToAgentId:   "", // Broadcast to interested parties
            EventType:   "artifact.created",
            Priority:    eventbus.Priority_PRIORITY_LOW,
        },
    })

    return err
}
```

### Using A2A Abstractions

Use AgentHub's simplified artifact publishing:

```go
func completeTaskWithA2AArtifact(ctx context.Context, subscriber *agenthub.A2ATaskSubscriber, task *pb.Task, artifact *pb.Artifact) error {
    return subscriber.CompleteA2ATaskWithArtifact(ctx, task, artifact)
}
```

## Processing Received Artifacts

### Artifact Event Handling

Handle incoming artifact notifications:

```go
func handleArtifactEvents(ctx context.Context, client eventbus.AgentHubClient, agentID string) error {
    stream, err := client.SubscribeToAgentEvents(ctx, &eventbus.SubscribeToAgentEventsRequest{
        AgentId: agentID,
        EventTypes: []string{"artifact.created", "task.completed"},
    })
    if err != nil {
        return err
    }

    for {
        event, err := stream.Recv()
        if err != nil {
            return err
        }

        switch payload := event.GetPayload().(type) {
        case *eventbus.AgentEvent_ArtifactUpdate:
            artifactEvent := payload.ArtifactUpdate
            log.Printf("Received artifact: %s for task: %s",
                artifactEvent.GetArtifact().GetArtifactId(),
                artifactEvent.GetTaskId())

            // Process the artifact
            err := processArtifact(ctx, artifactEvent.GetArtifact())
            if err != nil {
                log.Printf("Error processing artifact: %v", err)
            }

        case *eventbus.AgentEvent_Task:
            task := payload.Task
            if task.GetStatus().GetState() == pb.TaskState_TASK_STATE_COMPLETED {
                // Process completed task artifacts
                for _, artifact := range task.GetArtifacts() {
                    err := processArtifact(ctx, artifact)
                    if err != nil {
                        log.Printf("Error processing task artifact: %v", err)
                    }
                }
            }
        }
    }
}
```

### Artifact Content Processing

Process different types of artifact content:

```go
func processArtifact(ctx context.Context, artifact *pb.Artifact) error {
    log.Printf("Processing artifact: %s - %s", artifact.GetName(), artifact.GetDescription())

    for i, part := range artifact.GetParts() {
        switch content := part.GetPart().(type) {
        case *pb.Part_Text:
            log.Printf("Text part %d: Processing text content (%d chars)", i, len(content.Text))
            // Process text content
            err := processTextArtifact(content.Text)
            if err != nil {
                return fmt.Errorf("failed to process text part: %w", err)
            }

        case *pb.Part_Data:
            log.Printf("Data part %d: Processing structured data (%s)", i, content.Data.GetDescription())
            // Process structured data
            err := processDataArtifact(content.Data.GetData())
            if err != nil {
                return fmt.Errorf("failed to process data part: %w", err)
            }

        case *pb.Part_File:
            log.Printf("File part %d: Processing file %s (%s, %d bytes)",
                i, content.File.GetFilename(), content.File.GetMimeType(), content.File.GetSizeBytes())
            // Process file reference
            err := processFileArtifact(ctx, content.File)
            if err != nil {
                return fmt.Errorf("failed to process file part: %w", err)
            }
        }
    }

    return nil
}

func processTextArtifact(text string) error {
    // Extract insights, save to database, etc.
    log.Printf("Extracting insights from text artifact...")
    return nil
}

func processDataArtifact(data *structpb.Struct) error {
    // Parse structured data, update metrics, etc.
    log.Printf("Processing structured data artifact...")

    // Access specific fields
    if recordCount, ok := data.GetFields()["record_count"]; ok {
        log.Printf("Records processed: %v", recordCount.GetNumberValue())
    }

    return nil
}

func processFileArtifact(ctx context.Context, file *pb.FilePart) error {
    // Download file, process content, etc.
    log.Printf("Processing file artifact: %s", file.GetFileId())

    // Handle different file types
    switch file.GetMimeType() {
    case "text/csv":
        return processCSVFile(ctx, file.GetFileId())
    case "image/png", "image/jpeg":
        return processImageFile(ctx, file.GetFileId())
    case "application/json":
        return processJSONFile(ctx, file.GetFileId())
    default:
        log.Printf("Unknown file type: %s", file.GetMimeType())
    }

    return nil
}
```

## Artifact Chaining

### Using Artifacts as Inputs

Use artifacts from one task as inputs to another:

```go
func chainArtifactProcessing(ctx context.Context, client eventbus.AgentHubClient, inputArtifact *pb.Artifact) error {
    // Create a new task using the artifact as input
    contextID := fmt.Sprintf("ctx_chained_%s", uuid.New().String())

    chainedTask := &pb.Task{
        Id:        fmt.Sprintf("task_chained_%s", uuid.New().String()),
        ContextId: contextID,
        Status: &pb.TaskStatus{
            State: pb.TaskState_TASK_STATE_SUBMITTED,
            Update: &pb.Message{
                MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
                ContextId: contextID,
                Role:      pb.Role_USER,
                Content: []*pb.Part{
                    {
                        Part: &pb.Part_Text{
                            Text: "Please process the results from the previous analysis task.",
                        },
                    },
                    {
                        Part: &pb.Part_Data{
                            Data: &pb.DataPart{
                                Data: &structpb.Struct{
                                    Fields: map[string]*structpb.Value{
                                        "input_artifact_id": structpb.NewStringValue(inputArtifact.GetArtifactId()),
                                        "processing_type":   structpb.NewStringValue("enhancement"),
                                    },
                                },
                                Description: "Processing parameters with input artifact reference",
                            },
                        },
                    },
                },
            },
            Timestamp: timestamppb.Now(),
        },
    }

    // Publish the chained task
    _, err := client.PublishTaskUpdate(ctx, &eventbus.PublishTaskUpdateRequest{
        Task: chainedTask,
        Routing: &eventbus.AgentEventMetadata{
            FromAgentId: "workflow_coordinator",
            ToAgentId:   "enhancement_processor",
            EventType:   "task.chained",
            Priority:    eventbus.Priority_PRIORITY_MEDIUM,
        },
    })

    return err
}
```

## Best Practices

### 1. Use Descriptive Artifact Names and Descriptions
```go
artifact := &pb.Artifact{
    Name:        "Customer Segmentation Analysis Results",
    Description: "Complete customer segmentation with demographics, behavior patterns, and actionable insights",
    // ...
}
```

### 2. Include Rich Metadata for Discovery
```go
metadata := map[string]interface{}{
    "artifact_type":    "analysis",
    "domain":          "customer_analytics",
    "data_source":     "customer_transactions_2024",
    "algorithm":       "k_means_clustering",
    "confidence":      0.94,
    "generated_by":    "analytics_engine_v2.1",
    "valid_until":     time.Now().Add(30*24*time.Hour).Format(time.RFC3339),
}
```

### 3. Structure Multi-Part Artifacts Logically
```go
// Order parts from most important to least important
parts := []*pb.Part{
    textSummaryPart,      // Human-readable summary first
    structuredDataPart,   // Machine-readable data second
    configurationPart,    // Configuration details third
    fileReferencePart,    // File references last
}
```

### 4. Validate Artifacts Before Publishing
```go
func validateArtifact(artifact *pb.Artifact) error {
    if artifact.GetArtifactId() == "" {
        return fmt.Errorf("artifact_id is required")
    }
    if len(artifact.GetParts()) == 0 {
        return fmt.Errorf("artifact must have at least one part")
    }
    return nil
}
```

### 5. Handle Large Artifacts Appropriately
```go
// For large data, use file references instead of inline data
if len(dataBytes) > 1024*1024 { // 1MB threshold
    // Save to file storage and reference
    fileID := saveToFileStorage(dataBytes)
    part = createFileReferencePart(fileID, filename, mimeType)
} else {
    // Include data inline
    part = createInlineDataPart(data)
}
```

This guide covered creating and working with A2A artifacts. Next, learn about [A2A Task Lifecycle Management](../work_with_task_lifecycle/) to understand how to properly manage task states and coordinate complex workflows.