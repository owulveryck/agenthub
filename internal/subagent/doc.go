// Package subagent provides a high-level library for building A2A-compliant agents
// with minimal boilerplate code.
//
// # Overview
//
// The SubAgent library encapsulates common agent functionality including:
//   - gRPC client setup and connection management
//   - A2A-compliant AgentCard creation and registration
//   - Task subscription and skill-based routing
//   - Automatic distributed tracing and structured logging
//   - Graceful shutdown and lifecycle management
//   - Health check endpoints
//
// # Quick Start
//
// Creating an agent with the SubAgent library requires only three steps:
//
//  1. Configure your agent
//  2. Register skills with handlers
//  3. Run the agent
//
// Example:
//
//	config := &subagent.Config{
//	    AgentID:     "my_agent",
//	    Name:        "My Agent",
//	    Description: "Does something useful",
//	}
//
//	agent, err := subagent.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	agent.MustAddSkill("My Skill", "Description", myHandler)
//
//	if err := agent.Run(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// # Handler Functions
//
// Skills are implemented as handler functions with the signature:
//
//	func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string)
//
// Handler functions receive:
//   - ctx: Context for cancellation and tracing
//   - task: The complete A2A task object
//   - message: The initial message that triggered the task
//
// Handler functions return:
//   - artifact: Result data (nil if failed)
//   - state: Task state (COMPLETED, FAILED, etc.)
//   - errorMsg: Error message (empty string if successful)
//
// Example handler:
//
//	func echoHandler(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
//	    // Extract input
//	    text := message.Content[0].GetText()
//	    if text == "" {
//	        return nil, pb.TaskState_TASK_STATE_FAILED, "no input provided"
//	    }
//
//	    // Process
//	    result := fmt.Sprintf("Echo: %s", text)
//
//	    // Create artifact
//	    artifact := &pb.Artifact{
//	        ArtifactId: fmt.Sprintf("echo_%d", time.Now().Unix()),
//	        Name:       "echo_result",
//	        Parts:      []*pb.Part{{Part: &pb.Part_Text{Text: result}}},
//	    }
//
//	    return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
//	}
//
// # Configuration
//
// Config specifies agent properties and connection details. Required fields are:
//   - AgentID: Unique identifier for the agent
//   - Name: Human-readable agent name
//   - Description: What the agent does
//
// Optional fields with defaults:
//   - ServiceName: gRPC service name (defaults to AgentID)
//   - Version: Agent version (defaults to "1.0.0")
//   - HealthPort: Health check port (defaults to "8080")
//   - BrokerAddr: Broker address (defaults to env AGENTHUB_BROKER_ADDR)
//   - BrokerPort: Broker port (defaults to env AGENTHUB_GRPC_PORT)
//
// # Automatic Features
//
// When you call agent.Run(), the library automatically:
//
//  1. Validates configuration and applies defaults
//  2. Connects to the broker
//  3. Creates an A2A-compliant AgentCard with all registered skills
//  4. Registers the agent with the broker (triggers Cortex discovery)
//  5. Subscribes to tasks
//  6. Routes incoming tasks to the appropriate skill handler
//  7. Wraps each task execution with:
//     - Distributed tracing (automatic span creation with A2A attributes)
//     - Structured logging (task receipt, processing, completion, errors)
//     - Error handling (captures and reports failures)
//  8. Publishes task results back through the broker
//  9. Handles SIGINT/SIGTERM for graceful shutdown
//  10. Ensures proper resource cleanup
//
// # Observability
//
// Every task execution gets automatic observability:
//
// Tracing:
//   - Span created for each task with operation name "agent.{agentID}.handle_task"
//   - A2A task attributes added (task ID, skill name, context ID, etc.)
//   - Component attribute identifies which agent processed the task
//   - Success/failure status recorded
//   - Errors captured with full details
//
// Logging:
//   - Structured logs using log/slog
//   - Task receipt logged with task ID and skill name
//   - Completion logged with success/failure status
//   - Errors logged with full context
//
// Access the logger for custom logging:
//
//	logger := agent.GetLogger()
//	logger.InfoContext(ctx, "Custom log message", "key", "value")
//
// # AgentCard Generation
//
// The library automatically creates A2A-compliant AgentCards with:
//   - Protocol version: "0.2.9"
//   - Capabilities: {Streaming: false, PushNotifications: false}
//   - Skills: One AgentSkill per registered skill with:
//   - Unique ID (skill_0, skill_1, ...)
//   - Name and description from MustAddSkill()
//   - Tags using skill name for routing
//   - InputModes and OutputModes set to "text/plain"
//
// # Error Handling
//
// The library defines common errors:
//   - ErrMissingAgentID: AgentID not provided in config
//   - ErrMissingName: Name not provided in config
//   - ErrMissingDescription: Description not provided in config
//   - ErrNoSkills: No skills registered before Run()
//   - ErrDuplicateSkill: Skill name already registered
//   - ErrAgentNotStarted: Operation attempted before Run()
//   - ErrAgentAlreadyRunning: Run() called multiple times
//
// Configuration errors are caught early in New() or Run().
// Runtime errors are logged and reported through task states.
//
// # Advanced Usage
//
// Access underlying components for advanced needs:
//
//	client := agent.GetClient()        // AgentHub client for custom operations
//	logger := agent.GetLogger()        // Logger for custom logging
//	config := agent.GetConfig()        // Current configuration
//
// Multiple skills per agent:
//
//	agent.MustAddSkill("Skill A", "Does A", handlerA)
//	agent.MustAddSkill("Skill B", "Does B", handlerB)
//	agent.MustAddSkill("Skill C", "Does C", handlerC)
//
// The library routes each task to the appropriate handler based on task type.
//
// # Performance Considerations
//
// The SubAgent library is designed for production use:
//   - Asynchronous task processing (doesn't block on task execution)
//   - Efficient task routing (O(1) lookup by skill name)
//   - Minimal overhead (wrapping adds <1ms per task)
//   - Graceful shutdown (waits for in-flight tasks with timeout)
//   - Resource cleanup (all connections properly closed)
//
// # Examples
//
// See agents/echo_agent for a complete working example (82 lines total).
//
// Translation agent example:
//
//	func main() {
//	    config := &subagent.Config{
//	        AgentID:     "agent_translator",
//	        Name:        "Translation Agent",
//	        Description: "Translates text between languages",
//	    }
//
//	    agent, _ := subagent.New(config)
//	    agent.MustAddSkill("Translate", "Translates text", translateHandler)
//	    agent.Run(context.Background())
//	}
//
//	func translateHandler(ctx context.Context, task *pb.Task, msg *pb.Message) (*pb.Artifact, pb.TaskState, string) {
//	    input := msg.Content[0].GetText()
//	    result := performTranslation(input)
//
//	    artifact := &pb.Artifact{
//	        ArtifactId: fmt.Sprintf("trans_%d", time.Now().Unix()),
//	        Name:       "translation",
//	        Parts:      []*pb.Part{{Part: &pb.Part_Text{Text: result}}},
//	    }
//	    return artifact, pb.TaskState_TASK_STATE_COMPLETED, ""
//	}
package subagent
