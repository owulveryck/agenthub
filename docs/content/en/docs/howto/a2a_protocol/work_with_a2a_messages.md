---
title: "How to Work with A2A Messages"
weight: 10
description: "Learn how to create, structure, and work with Agent2Agent protocol messages including text, data, and file parts."
---

# How to Work with A2A Messages

This guide shows you how to create and work with Agent2Agent (A2A) protocol messages using AgentHub's unified abstractions. A2A messages are the foundation of all agent communication.

## Understanding A2A Message Structure

A2A messages consist of several key components:

- **Message ID**: Unique identifier for the message
- **Context ID**: Groups related messages in a conversation
- **Task ID**: Links the message to a specific task
- **Role**: Indicates if the message is from USER (requester) or AGENT (responder)
- **Content Parts**: The actual message content (text, data, or files)
- **Metadata**: Additional context for routing and processing

## Creating Basic A2A Messages

### Text Messages

Create a simple text message:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/google/uuid"
    pb "github.com/owulveryck/agenthub/events/a2a"
    "google.golang.org/protobuf/types/known/timestamppb"
)

func createTextMessage() *pb.Message {
    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: "conversation_greeting",
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Hello! Please process this greeting request.",
                },
            },
        },
        Metadata: nil, // Optional
    }
}
```

### Data Messages

Include structured data in your message:

```go
import (
    "google.golang.org/protobuf/types/known/structpb"
)

func createDataMessage() *pb.Message {
    // Create structured data
    data, err := structpb.NewStruct(map[string]interface{}{
        "operation": "calculate",
        "numbers":   []float64{10, 20, 30},
        "formula":   "sum",
        "precision": 2,
    })
    if err != nil {
        log.Fatal(err)
    }

    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: "conversation_math",
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Please perform the calculation described in the data.",
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        data,
                        Description: "Calculation parameters",
                    },
                },
            },
        },
    }
}
```

### File Reference Messages

Reference files in your messages:

```go
func createFileMessage() *pb.Message {
    // Create file metadata
    fileMetadata, _ := structpb.NewStruct(map[string]interface{}{
        "source":      "user_upload",
        "category":    "image",
        "permissions": "read-only",
    })

    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: "conversation_image_analysis",
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Please analyze the uploaded image.",
                },
            },
            {
                Part: &pb.Part_File{
                    File: &pb.FilePart{
                        FileId:   "file_abc123",
                        Filename: "analysis_target.jpg",
                        MimeType: "image/jpeg",
                        SizeBytes: 2048576, // 2MB
                        Metadata:  fileMetadata,
                    },
                },
            },
        },
    }
}
```

## Working with Mixed Content

Combine multiple part types in a single message:

```go
func createMixedContentMessage() *pb.Message {
    // Configuration data
    config, _ := structpb.NewStruct(map[string]interface{}{
        "format":     "json",
        "output_dir": "/results",
        "compress":   true,
    })

    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: "conversation_data_processing",
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "Process the dataset with the following configuration and source file.",
                },
            },
            {
                Part: &pb.Part_Data{
                    Data: &pb.DataPart{
                        Data:        config,
                        Description: "Processing configuration",
                    },
                },
            },
            {
                Part: &pb.Part_File{
                    File: &pb.FilePart{
                        FileId:   "dataset_xyz789",
                        Filename: "raw_data.csv",
                        MimeType: "text/csv",
                        SizeBytes: 5242880, // 5MB
                    },
                },
            },
        },
    }
}
```

## Publishing A2A Messages

Use AgentHub's unified abstractions to publish messages:

```go
package main

import (
    "context"
    "log"

    "github.com/owulveryck/agenthub/internal/agenthub"
    pb "github.com/owulveryck/agenthub/events/eventbus"
)

func publishA2AMessage(ctx context.Context) error {
    // Create AgentHub client
    config := agenthub.NewGRPCConfig("message_publisher")
    client, err := agenthub.NewAgentHubClient(config)
    if err != nil {
        return err
    }
    defer client.Close()

    // Create A2A message
    message := createTextMessage()

    // Publish using AgentHub client
    response, err := client.Client.PublishMessage(ctx, &pb.PublishMessageRequest{
        Message: message,
        Routing: &pb.AgentEventMetadata{
            FromAgentId: "message_publisher",
            ToAgentId:   "message_processor",
            EventType:   "a2a.message",
            Priority:    pb.Priority_PRIORITY_MEDIUM,
        },
    })

    if err != nil {
        return err
    }

    log.Printf("A2A message published: %s", response.GetEventId())
    return nil
}
```

## Processing Received A2A Messages

Handle incoming A2A messages in your agent:

```go
func processA2AMessage(ctx context.Context, message *pb.Message) (string, error) {
    var response string

    // Process each content part
    for i, part := range message.GetContent() {
        switch content := part.GetPart().(type) {
        case *pb.Part_Text:
            log.Printf("Text part %d: %s", i, content.Text)
            response += fmt.Sprintf("Processed text: %s\n", content.Text)

        case *pb.Part_Data:
            log.Printf("Data part %d: %s", i, content.Data.GetDescription())
            // Process structured data
            data := content.Data.GetData()
            response += fmt.Sprintf("Processed data: %s\n", content.Data.GetDescription())

            // Access specific fields
            if operation, ok := data.GetFields()["operation"]; ok {
                log.Printf("Operation: %s", operation.GetStringValue())
            }

        case *pb.Part_File:
            log.Printf("File part %d: %s (%s)", i, content.File.GetFilename(), content.File.GetMimeType())
            response += fmt.Sprintf("Processed file: %s\n", content.File.GetFilename())

            // Handle file processing based on MIME type
            switch content.File.GetMimeType() {
            case "image/jpeg", "image/png":
                // Process image
                response += "Image analysis completed\n"
            case "text/csv":
                // Process CSV data
                response += "CSV data parsed\n"
            }
        }
    }

    return response, nil
}
```

## Message Role Management

Properly set message roles for A2A compliance:

```go
// User message (requesting work)
func createUserMessage(content string) *pb.Message {
    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        Role:      pb.Role_USER,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{Text: content},
            },
        },
    }
}

// Agent response message
func createAgentResponse(contextId, taskId, response string) *pb.Message {
    return &pb.Message{
        MessageId: fmt.Sprintf("msg_%s", uuid.New().String()),
        ContextId: contextId,
        TaskId:    taskId,
        Role:      pb.Role_AGENT,
        Content: []*pb.Part{
            {
                Part: &pb.Part_Text{Text: response},
            },
        },
    }
}
```

## Message Validation

Validate A2A messages before publishing:

```go
func validateA2AMessage(message *pb.Message) error {
    if message.GetMessageId() == "" {
        return fmt.Errorf("message_id is required")
    }

    if message.GetRole() == pb.Role_ROLE_UNSPECIFIED {
        return fmt.Errorf("role must be specified (USER or AGENT)")
    }

    if len(message.GetContent()) == 0 {
        return fmt.Errorf("message must have at least one content part")
    }

    // Validate each part
    for i, part := range message.GetContent() {
        if part.GetPart() == nil {
            return fmt.Errorf("content part %d is empty", i)
        }
    }

    return nil
}
```

## Best Practices

### 1. Always Use Unique Message IDs
```go
messageID := fmt.Sprintf("msg_%d_%s", time.Now().Unix(), uuid.New().String())
```

### 2. Group Related Messages with Context IDs
```go
contextID := fmt.Sprintf("ctx_%s_%s", workflowType, uuid.New().String())
```

### 3. Include Descriptive Metadata for Complex Data
```go
dataPart := &pb.DataPart{
    Data:        structData,
    Description: "User preferences for recommendation engine",
}
```

### 4. Validate Messages Before Publishing
```go
if err := validateA2AMessage(message); err != nil {
    return fmt.Errorf("invalid A2A message: %w", err)
}
```

### 5. Handle All Part Types in Message Processors
```go
switch content := part.GetPart().(type) {
case *pb.Part_Text:
    // Handle text
case *pb.Part_Data:
    // Handle structured data
case *pb.Part_File:
    // Handle file references
default:
    log.Printf("Unknown part type: %T", content)
}
```

This guide covered the fundamentals of working with A2A messages. Next, learn about [A2A Conversation Context](../work_with_conversation_context/) to group related messages and maintain conversation state across multiple interactions.