---
title: "Building Multi-Agent Workflows"
weight: 10
description: "Learn to create complex workflows involving multiple specialized agents working together to accomplish sophisticated tasks. Build a real document processing pipeline with multiple agents handling different stages."
---

# Building Multi-Agent Workflows

This advanced tutorial teaches you to create complex workflows involving multiple specialized agents working together to accomplish sophisticated tasks. You'll build a real document processing pipeline with multiple agents handling different stages.

## What You'll Build

By the end of this tutorial, you'll have an A2A-compliant multi-agent system that:

1. **Ingests documents** through an A2A Document Intake Agent
2. **Validates content** using an A2A Validation Agent
3. **Extracts metadata** with an A2A Metadata Extraction Agent
4. **Processes text** through an A2A Text Processing Agent
5. **Generates summaries** using an A2A Summary Agent
6. **Orchestrates the workflow** with an A2A Workflow Coordinator Agent

This demonstrates real-world A2A agent collaboration patterns with conversation context, structured message content, and artifact-based results used in production systems.

## Prerequisites

- Complete the [Installation and Setup](installation_and_setup.md) tutorial
- Complete the [Running the Demo](run_demo.md) tutorial
- Familiarity with Go programming
- Understanding of basic agent concepts

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  A2A Workflow   │    │   AgentHub      │    │  A2A Specialized│
│  Coordinator    │    │  A2A Broker     │    │    Agents       │
│                 │    │                 │    │                 │
│ • A2A context   │◄──►│ • Routes A2A    │◄──►│ • Document      │
│   management    │    │   messages      │    │   Intake        │
│ • Conversation  │    │ • Tracks A2A    │    │ • Validation    │
│   threading     │    │   conversations │    │ • Metadata      │
│ • Artifact      │    │ • Manages A2A   │    │ • Text Proc     │
│   aggregation   │    │   state         │    │ • Summary       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Step 1: Create the Workflow Coordinator

First, let's create the main coordinator that manages the document processing pipeline.

Create the coordinator agent:

```bash
mkdir -p agents/coordinator
```

Create `agents/coordinator/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	a2a "github.com/owulveryck/agenthub/events/a2a"
	pb "github.com/owulveryck/agenthub/events/eventbus"
)

const (
	agentHubAddr = "localhost:50051"
	agentID      = "a2a_workflow_coordinator"
)

type A2ADocumentWorkflow struct {
	DocumentID    string
	ContextID     string                 // A2A conversation context
	Status        string
	CurrentStage  string
	TaskHistory   []*a2a.Task           // Complete A2A task history
	Artifacts     []*a2a.Artifact       // Collected artifacts from stages
	StartTime     time.Time
	client        pb.AgentHubClient      // A2A-compliant client
}

func main() {
	conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAgentHubClient(conn)
	coordinator := &A2AWorkflowCoordinator{
		client:        client,
		workflows:     make(map[string]*A2ADocumentWorkflow),
	}

	ctx := context.Background()

	// Start listening for A2A task events
	go coordinator.subscribeToA2AEvents(ctx)

	// Start processing documents with A2A workflow
	coordinator.startA2ADocumentProcessing(ctx)

	// Keep running
	select {}
}

type A2AWorkflowCoordinator struct {
	client    pb.AgentHubClient
	workflows map[string]*A2ADocumentWorkflow
}

func (wc *A2AWorkflowCoordinator) startA2ADocumentProcessing(ctx context.Context) {
	// Simulate document arrival with A2A structured content
	documents := []map[string]interface{}{
		{
			"document_id": "doc_001",
			"content":     "This is a sample business document about quarterly results.",
			"filename":    "q3_results.txt",
			"source":      "email_attachment",
			"doc_type":    "business_report",
		},
		{
			"document_id": "doc_002",
			"content":     "Technical specification for the new API endpoints and authentication mechanisms.",
			"filename":    "api_spec.txt",
			"source":      "file_upload",
			"doc_type":    "technical_spec",
		},
	}

	for _, doc := range documents {
		wc.processA2ADocument(ctx, doc)
		time.Sleep(5 * time.Second)
	}
}

func (wc *A2AWorkflowCoordinator) processA2ADocument(ctx context.Context, document map[string]interface{}) {
	documentID := document["document_id"].(string)
	contextID := fmt.Sprintf("doc_workflow_%s_%s", documentID, uuid.New().String())

	workflow := &A2ADocumentWorkflow{
		DocumentID:   documentID,
		ContextID:    contextID,
		Status:       "started",
		CurrentStage: "intake",
		TaskHistory:  make([]*a2a.Task, 0),
		Artifacts:    make([]*a2a.Artifact, 0),
		StartTime:    time.Now(),
		client:       wc.client,
	}

	wc.workflows[documentID] = workflow

	log.Printf("Starting A2A document processing workflow for %s with context %s", documentID, contextID)

	// Stage 1: A2A Document Intake
	wc.publishA2ATask(ctx, "document_intake", document, "a2a_document_intake_agent", workflow)
}

func (wc *A2AWorkflowCoordinator) publishA2ATask(ctx context.Context, taskDescription string, params map[string]interface{}, targetAgent string, workflow *A2ADocumentWorkflow) {
	taskID := fmt.Sprintf("task_%s_%s", taskDescription, uuid.New().String())
	messageID := fmt.Sprintf("msg_%d_%s", time.Now().Unix(), uuid.New().String())

	// Create A2A structured content
	paramsData, err := structpb.NewStruct(params)
	if err != nil {
		log.Printf("Error creating parameters: %v", err)
		return
	}

	// Create A2A message with structured parts
	requestMessage := &a2a.Message{
		MessageId: messageID,
		ContextId: workflow.ContextID,
		TaskId:    taskID,
		Role:      a2a.Role_USER,
		Content: []*a2a.Part{
			{
				Part: &a2a.Part_Text{
					Text: fmt.Sprintf("Please process %s for document %s", taskDescription, workflow.DocumentID),
				},
			},
			{
				Part: &a2a.Part_Data{
					Data: &a2a.DataPart{
						Data:        paramsData,
						Description: fmt.Sprintf("%s parameters", taskDescription),
					},
				},
			},
		},
	}

	// Create A2A task
	task := &a2a.Task{
		Id:        taskID,
		ContextId: workflow.ContextID,
		Status: &a2a.TaskStatus{
			State:     a2a.TaskState_TASK_STATE_SUBMITTED,
			Update:    requestMessage,
			Timestamp: timestamppb.Now(),
		},
		History: []*a2a.Message{requestMessage},
		Metadata: paramsData,
	}

	// Store in workflow history
	workflow.TaskHistory = append(workflow.TaskHistory, task)

	// Publish A2A task update
	req := &pb.PublishTaskUpdateRequest{
		Task: task,
		Routing: &pb.AgentEventMetadata{
			FromAgentId: agentID,
			ToAgentId:   targetAgent,
			EventType:   "task.submitted",
			Priority:    pb.Priority_PRIORITY_MEDIUM,
		},
	}

	log.Printf("Publishing A2A %s task for workflow %s in context %s", taskDescription, workflow.DocumentID, workflow.ContextID)
	_, err = wc.client.PublishTaskUpdate(ctx, req)
	if err != nil {
		log.Printf("Error publishing A2A task: %v", err)
	}
}

func (wc *WorkflowCoordinator) subscribeToResults(ctx context.Context) {
	req := &pb.SubscribeToTaskResultsRequest{
		RequesterAgentId: agentID,
	}

	stream, err := wc.client.SubscribeToTaskResults(ctx, req)
	if err != nil {
		log.Printf("Error subscribing to results: %v", err)
		return
	}

	for {
		result, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving result: %v", err)
			return
		}

		wc.handleTaskResult(ctx, result)
	}
}

func (wc *WorkflowCoordinator) handleTaskResult(ctx context.Context, result *pb.TaskResult) {
	params := result.GetResult().AsMap()
	workflowID := params["workflow_id"].(string)
	stage := params["stage"].(string)

	workflow, exists := wc.workflows[workflowID]
	if !exists {
		log.Printf("Unknown workflow ID: %s", workflowID)
		return
	}

	log.Printf("Received result for workflow %s, stage %s: %s",
		workflowID, stage, result.GetStatus().String())

	if result.GetStatus() == pb.TaskStatus_TASK_STATUS_FAILED {
		workflow.Status = "failed"
		log.Printf("Workflow %s failed at stage %s: %s",
			workflowID, stage, result.GetErrorMessage())
		return
	}

	// Store stage results
	workflow.Results[stage] = params

	// Advance to next stage
	wc.advanceWorkflow(ctx, workflow, stage)
}

func (wc *WorkflowCoordinator) advanceWorkflow(ctx context.Context, workflow *DocumentWorkflow, completedStage string) {
	switch completedStage {
	case "document_intake":
		// Move to validation
		workflow.CurrentStage = "validation"
		data := workflow.Results["document_intake"]
		wc.publishTask(ctx, "document_validation", data.(map[string]interface{}), "validation_agent", workflow.DocumentID)

	case "document_validation":
		// Move to metadata extraction
		workflow.CurrentStage = "metadata_extraction"
		data := workflow.Results["document_validation"]
		wc.publishTask(ctx, "metadata_extraction", data.(map[string]interface{}), "metadata_agent", workflow.DocumentID)

	case "metadata_extraction":
		// Move to text processing
		workflow.CurrentStage = "text_processing"
		data := workflow.Results["metadata_extraction"]
		wc.publishTask(ctx, "text_processing", data.(map[string]interface{}), "text_processor_agent", workflow.DocumentID)

	case "text_processing":
		// Move to summary generation
		workflow.CurrentStage = "summary_generation"
		data := workflow.Results["text_processing"]
		wc.publishTask(ctx, "summary_generation", data.(map[string]interface{}), "summary_agent", workflow.DocumentID)

	case "summary_generation":
		// Workflow complete
		workflow.Status = "completed"
		workflow.CurrentStage = "finished"
		duration := time.Since(workflow.StartTime)

		log.Printf("Workflow %s completed successfully in %v", workflow.DocumentID, duration)
		wc.printWorkflowSummary(workflow)
	}
}

func (wc *WorkflowCoordinator) printWorkflowSummary(workflow *DocumentWorkflow) {
	fmt.Printf("\n=== WORKFLOW SUMMARY ===\n")
	fmt.Printf("Document ID: %s\n", workflow.DocumentID)
	fmt.Printf("Status: %s\n", workflow.Status)
	fmt.Printf("Duration: %v\n", time.Since(workflow.StartTime))
	fmt.Printf("Stages completed:\n")

	for stage, result := range workflow.Results {
		fmt.Printf("  - %s: %v\n", stage, result)
	}
	fmt.Printf("=======================\n\n")
}
```

## Step 2: Create Specialized Agents

Now let's create each specialized agent that handles specific stages of the pipeline.

### Document Intake Agent

Create `agents/document_intake/main.go`:

```go
package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

const (
	agentHubAddr = "localhost:50051"
	agentID      = "document_intake_agent"
)

func main() {
	conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEventBusClient(conn)
	agent := &DocumentIntakeAgent{client: client}

	ctx := context.Background()
	agent.start(ctx)
}

type DocumentIntakeAgent struct {
	client pb.EventBusClient
}

func (dia *DocumentIntakeAgent) start(ctx context.Context) {
	log.Printf("Document Intake Agent %s starting...", agentID)

	req := &pb.SubscribeToTasksRequest{
		AgentId:   agentID,
		TaskTypes: []string{"document_intake"},
	}

	stream, err := dia.client.SubscribeToTasks(ctx, req)
	if err != nil {
		log.Fatalf("Error subscribing: %v", err)
	}

	log.Printf("Subscribed to document intake tasks")

	for {
		task, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Error receiving task: %v", err)
			return
		}

		go dia.processTask(ctx, task)
	}
}

func (dia *DocumentIntakeAgent) processTask(ctx context.Context, task *pb.TaskMessage) {
	log.Printf("Processing document intake task: %s", task.GetTaskId())

	params := task.GetParameters().AsMap()

	// Simulate document intake processing
	time.Sleep(2 * time.Second)

	// Generate document hash
	content := params["content"].(string)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))

	// Extract basic metadata
	wordCount := len(strings.Fields(content))
	charCount := len(content)

	result := map[string]interface{}{
		"document_id":   params["document_id"],
		"workflow_id":   params["workflow_id"],
		"stage":         "document_intake",
		"content":       content,
		"filename":      params["filename"],
		"source":        params["source"],
		"document_hash": hash,
		"word_count":    wordCount,
		"char_count":    charCount,
		"intake_timestamp": time.Now().Format(time.RFC3339),
		"status":        "intake_complete",
	}

	dia.publishResult(ctx, task, result, pb.TaskStatus_TASK_STATUS_COMPLETED, "")
}

func (dia *DocumentIntakeAgent) publishResult(ctx context.Context, originalTask *pb.TaskMessage, result map[string]interface{}, status pb.TaskStatus, errorMsg string) {
	resultStruct, err := structpb.NewStruct(result)
	if err != nil {
		log.Printf("Error creating result struct: %v", err)
		return
	}

	taskResult := &pb.TaskResult{
		TaskId:          originalTask.GetTaskId(),
		Status:          status,
		Result:          resultStruct,
		ErrorMessage:    errorMsg,
		ExecutorAgentId: agentID,
		CompletedAt:     timestamppb.Now(),
	}

	req := &pb.PublishTaskResultRequest{Result: taskResult}

	_, err = dia.client.PublishTaskResult(ctx, req)
	if err != nil {
		log.Printf("Error publishing result: %v", err)
	} else {
		log.Printf("Published result for task %s", originalTask.GetTaskId())
	}
}
```

### Validation Agent

Create `agents/validation/main.go`:

```go
package main

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/owulveryck/agenthub/events/a2a"
)

const (
	agentHubAddr = "localhost:50051"
	agentID      = "validation_agent"
)

func main() {
	conn, err := grpc.Dial(agentHubAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewEventBusClient(conn)
	agent := &ValidationAgent{client: client}

	ctx := context.Background()
	agent.start(ctx)
}

type ValidationAgent struct {
	client pb.EventBusClient
}

func (va *ValidationAgent) start(ctx context.Context) {
	log.Printf("Validation Agent %s starting...", agentID)

	req := &pb.SubscribeToTasksRequest{
		AgentId:   agentID,
		TaskTypes: []string{"document_validation"},
	}

	stream, err := va.client.SubscribeToTasks(ctx, req)
	if err != nil {
		log.Fatalf("Error subscribing: %v", err)
	}

	log.Printf("Subscribed to document validation tasks")

	for {
		task, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Error receiving task: %v", err)
			return
		}

		go va.processTask(ctx, task)
	}
}

func (va *ValidationAgent) processTask(ctx context.Context, task *pb.TaskMessage) {
	log.Printf("Processing validation task: %s", task.GetTaskId())

	params := task.GetParameters().AsMap()

	// Simulate validation processing
	time.Sleep(1500 * time.Millisecond)

	content := params["content"].(string)

	// Perform validation checks
	validationResults := va.validateDocument(content)

	result := map[string]interface{}{
		"document_id":       params["document_id"],
		"workflow_id":       params["workflow_id"],
		"stage":             "document_validation",
		"content":           content,
		"filename":          params["filename"],
		"source":            params["source"],
		"document_hash":     params["document_hash"],
		"word_count":        params["word_count"],
		"char_count":        params["char_count"],
		"intake_timestamp":  params["intake_timestamp"],
		"validation_results": validationResults,
		"validation_timestamp": time.Now().Format(time.RFC3339),
		"status":            "validation_complete",
	}

	var status pb.TaskStatus
	var errorMsg string

	if validationResults["is_valid"].(bool) {
		status = pb.TaskStatus_TASK_STATUS_COMPLETED
	} else {
		status = pb.TaskStatus_TASK_STATUS_FAILED
		errorMsg = "Document validation failed: " + validationResults["errors"].(string)
	}

	va.publishResult(ctx, task, result, status, errorMsg)
}

func (va *ValidationAgent) validateDocument(content string) map[string]interface{} {
	// Simple validation rules
	isValid := true
	var errors []string

	// Check minimum length
	if len(content) < 10 {
		isValid = false
		errors = append(errors, "content too short")
	}

	// Check for suspicious content
	suspiciousTerms := []string{"malware", "virus", "hack"}
	for _, term := range suspiciousTerms {
		if strings.Contains(strings.ToLower(content), term) {
			isValid = false
			errors = append(errors, "suspicious content detected")
			break
		}
	}

	// Check language (simple heuristic)
	isEnglish := va.isEnglishContent(content)

	return map[string]interface{}{
		"is_valid":    isValid,
		"is_english":  isEnglish,
		"errors":      strings.Join(errors, "; "),
		"length_ok":   len(content) >= 10,
		"safe_content": !strings.Contains(strings.ToLower(content), "malware"),
	}
}

func (va *ValidationAgent) isEnglishContent(content string) bool {
	// Simple heuristic: check for common English words
	commonWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"}
	lowerContent := strings.ToLower(content)

	matches := 0
	for _, word := range commonWords {
		if strings.Contains(lowerContent, " "+word+" ") {
			matches++
		}
	}

	return matches >= 2
}

func (va *ValidationAgent) publishResult(ctx context.Context, originalTask *pb.TaskMessage, result map[string]interface{}, status pb.TaskStatus, errorMsg string) {
	resultStruct, err := structpb.NewStruct(result)
	if err != nil {
		log.Printf("Error creating result struct: %v", err)
		return
	}

	taskResult := &pb.TaskResult{
		TaskId:          originalTask.GetTaskId(),
		Status:          status,
		Result:          resultStruct,
		ErrorMessage:    errorMsg,
		ExecutorAgentId: agentID,
		CompletedAt:     timestamppb.Now(),
	}

	req := &pb.PublishTaskResultRequest{Result: taskResult}

	_, err = va.client.PublishTaskResult(ctx, req)
	if err != nil {
		log.Printf("Error publishing result: %v", err)
	} else {
		log.Printf("Published result for task %s", originalTask.GetTaskId())
	}
}
```

## Step 3: Build and Test the Multi-Agent System

Update the Makefile to include the new agents:

```bash
# Add to Makefile build target
build: proto
	@echo "Building server binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SERVER_BINARY) broker/main.go

	@echo "Building coordinator binary..."
	go build $(GO_BUILD_FLAGS) -o bin/coordinator agents/coordinator/main.go

	@echo "Building document intake agent..."
	go build $(GO_BUILD_FLAGS) -o bin/document-intake agents/document_intake/main.go

	@echo "Building validation agent..."
	go build $(GO_BUILD_FLAGS) -o bin/validation agents/validation/main.go

	@echo "Building publisher binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(PUBLISHER_BINARY) agents/publisher/main.go

	@echo "Building subscriber binary..."
	go build $(GO_BUILD_FLAGS) -o bin/$(SUBSCRIBER_BINARY) agents/subscriber/main.go

	@echo "Build complete. Binaries are in the 'bin/' directory."
```

Build all components:

```bash
make build
```

## Step 4: Run the Multi-Agent Workflow

Now let's run the complete multi-agent system:

**Terminal 1 - Start the broker:**
```bash
make run-server
```

**Terminal 2 - Start the document intake agent:**
```bash
./bin/document-intake
```

**Terminal 3 - Start the validation agent:**
```bash
./bin/validation
```

**Terminal 4 - Start the workflow coordinator:**
```bash
./bin/coordinator
```

## Step 5: Observe the Workflow

You'll see the workflow coordinator processing documents through multiple stages:

1. **Document Intake**: Receives and processes raw documents
2. **Validation**: Checks content for safety and validity
3. **Metadata Extraction**: Extracts structured metadata
4. **Text Processing**: Processes and analyzes text content
5. **Summary Generation**: Creates document summaries

Each agent processes its stage and passes results to the next stage via the AgentHub broker.

## Understanding the Multi-Agent Pattern

This tutorial demonstrates several key patterns:

### 1. Workflow Orchestration
The coordinator agent manages the overall workflow, determining which stage comes next and handling failures.

### 2. Specialized Agents
Each agent has a specific responsibility and can be developed, deployed, and scaled independently.

### 3. Asynchronous Processing
Agents work asynchronously, allowing for better resource utilization and scalability.

### 4. Error Handling
The system handles failures gracefully, with the coordinator managing workflow state.

### 5. Data Flow
Structured data flows between agents, with each stage adding value to the processing pipeline.

## Next Steps

Now that you understand multi-agent workflows:

1. **Add more agents**: Create metadata extraction, text processing, and summary agents
2. **Implement error recovery**: Add retry mechanisms and failure handling
3. **Add monitoring**: Create a dashboard agent that tracks workflow progress
4. **Scale the system**: Run multiple instances of each agent type
5. **Add persistence**: Store workflow state in a database for recovery

This pattern scales to handle complex business processes, data pipelines, and automated workflows in production systems.

## Common Patterns and Best Practices

### Workflow State Management
- Store workflow state persistently for recovery
- Use unique workflow IDs for tracking
- Implement timeouts for stuck workflows

### Agent Communication
- Use structured messages with clear schemas
- Include metadata for routing and tracking
- Implement progress reporting for long-running tasks

### Error Handling
- Design for partial failures
- Implement retry mechanisms with backoff
- Provide clear error messages and recovery paths

### Monitoring and Observability
- Log all state transitions
- Track workflow performance metrics
- Implement health checks for agents

You now have the foundation for building sophisticated multi-agent systems that can handle complex, real-world workflows!