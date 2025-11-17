package subagent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/owulveryck/agenthub/events/a2a"
	"github.com/owulveryck/agenthub/internal/agenthub"
)

// SubAgent encapsulates the common functionality for building A2A-compliant agents.
//
// It handles all infrastructure concerns including client setup, registration,
// task subscription, observability, and lifecycle management. Developers only
// need to implement business logic in handler functions.
//
// A SubAgent is created with New(), skills are registered with AddSkill() or
// MustAddSkill(), and then Run() is called to start the agent. All setup,
// registration, and cleanup is handled automatically.
//
// SubAgent is not thread-safe during configuration (before Run()) but is safe
// for concurrent task processing after Run() is called.
type SubAgent struct {
	config         *Config
	client         *agenthub.AgentHubClient
	taskSubscriber *agenthub.A2ATaskSubscriber
	skills         map[string]*Skill
	agentCard      *pb.AgentCard
	running        bool
}

// New creates a new SubAgent with the given configuration.
//
// The configuration is validated and defaults are applied for optional fields.
// Required configuration fields are: AgentID, Name, and Description.
//
// Returns an error if configuration is invalid (missing required fields).
//
// Example:
//
//	config := &subagent.Config{
//	    AgentID:     "my_agent",
//	    Name:        "My Agent",
//	    Description: "Does something useful",
//	}
//	agent, err := subagent.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func New(config *Config) (*SubAgent, error) {
	// Apply defaults and validate
	config = config.WithDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &SubAgent{
		config: config,
		skills: make(map[string]*Skill),
	}, nil
}

// AddSkill registers a new skill with the agent.
//
// Skills define capabilities that the agent provides. Each skill has a name,
// description, and handler function. The name is used for task routing and
// should be unique within the agent. The description helps LLMs understand
// when to delegate tasks to this agent.
//
// Returns ErrDuplicateSkill if a skill with the same name is already registered.
//
// Skills must be registered before calling Run(). Attempting to add skills
// after Run() has been called will result in undefined behavior.
//
// Example:
//
//	err := agent.AddSkill(
//	    "Language Translation",
//	    "Translates text from one language to another",
//	    translateHandler,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (s *SubAgent) AddSkill(name, description string, handler TaskHandler) error {
	if _, exists := s.skills[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateSkill, name)
	}

	s.skills[name] = &Skill{
		Name:        name,
		Description: description,
		Handler:     handler,
	}

	return nil
}

// MustAddSkill is like AddSkill but panics on error.
//
// This is useful for cleaner initialization code when you want the program to
// fail fast during setup rather than handling errors. Suitable for agent main
// functions where skill registration errors are unrecoverable.
//
// Panics if a skill with the same name is already registered.
//
// Example:
//
//	agent.MustAddSkill("Echo", "Echoes messages", echoHandler)
//	agent.MustAddSkill("Translate", "Translates text", translateHandler)
//	agent.Run(ctx) // Will fail fast if skills couldn't be added
func (s *SubAgent) MustAddSkill(name, description string, handler TaskHandler) {
	if err := s.AddSkill(name, description, handler); err != nil {
		panic(err)
	}
}

// Run starts the agent and blocks until shutdown.
//
// This method handles the complete agent lifecycle:
//  1. Setup: Validates configuration, connects to broker
//  2. Registration: Creates and registers A2A-compliant AgentCard
//  3. Subscription: Subscribes to tasks from broker
//  4. Processing: Routes tasks to skill handlers with automatic observability
//  5. Shutdown: Handles SIGINT/SIGTERM, waits for in-flight tasks, cleanup
//
// Run blocks until:
//   - The context is cancelled
//   - A SIGINT or SIGTERM signal is received
//   - A fatal error occurs during initialization
//
// Returns an error if:
//   - Agent is already running (ErrAgentAlreadyRunning)
//   - No skills have been registered (ErrNoSkills)
//   - Initialization fails (client setup, registration, subscription)
//
// All resources are automatically cleaned up on shutdown, including:
//   - Broker connection closed
//   - Health check server stopped
//   - Task subscriptions cancelled
//
// Example:
//
//	agent := setupAgent() // Create and configure agent
//	if err := agent.Run(context.Background()); err != nil {
//	    log.Fatalf("Agent failed: %v", err)
//	}
func (s *SubAgent) Run(ctx context.Context) error {
	if s.running {
		return ErrAgentAlreadyRunning
	}

	if len(s.skills) == 0 {
		return ErrNoSkills
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Initialize the agent
	if err := s.initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	s.running = true
	defer func() {
		s.running = false
	}()

	// Ensure cleanup happens
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := s.client.Shutdown(shutdownCtx); err != nil {
			s.client.Logger.ErrorContext(shutdownCtx, "Error during shutdown", "error", err)
		}
	}()

	s.client.Logger.InfoContext(ctx, "Agent started successfully",
		"agent_id", s.config.AgentID,
		"name", s.config.Name,
		"skills", len(s.skills),
	)

	// Wait for shutdown signal
	<-ctx.Done()

	s.client.Logger.InfoContext(context.Background(), "Agent shutting down gracefully",
		"agent_id", s.config.AgentID,
	)

	return nil
}

// initialize sets up the AgentHub client, registers the agent card, and starts task subscription
func (s *SubAgent) initialize(ctx context.Context) error {
	// Create gRPC configuration using ServiceName (defaults to AgentID)
	grpcConfig := agenthub.NewGRPCConfig(s.config.ServiceName)
	grpcConfig.HealthPort = s.config.HealthPort

	if s.config.BrokerAddr != "" {
		if err := os.Setenv("AGENTHUB_BROKER_ADDR", s.config.BrokerAddr); err != nil {
			return fmt.Errorf("failed to set broker address: %w", err)
		}
	}

	if s.config.BrokerPort != "" {
		if err := os.Setenv("AGENTHUB_GRPC_PORT", s.config.BrokerPort); err != nil {
			return fmt.Errorf("failed to set broker port: %w", err)
		}
	}

	// Create AgentHub client
	client, err := agenthub.NewAgentHubClient(grpcConfig)
	if err != nil {
		return fmt.Errorf("failed to create AgentHub client: %w", err)
	}
	s.client = client

	// Start the client
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	// Build and register agent card
	if err := s.buildAndRegisterAgentCard(ctx); err != nil {
		return fmt.Errorf("failed to register agent card: %w", err)
	}

	// Setup task subscription with handlers
	if err := s.setupTaskSubscription(ctx); err != nil {
		return fmt.Errorf("failed to setup task subscription: %w", err)
	}

	return nil
}

// buildAndRegisterAgentCard creates the agent card from registered skills and publishes it
func (s *SubAgent) buildAndRegisterAgentCard(ctx context.Context) error {
	// Build skills for agent card
	cardSkills := make([]*pb.AgentSkill, 0, len(s.skills))
	skillIndex := 0
	for skillName, skill := range s.skills {
		cardSkills = append(cardSkills, &pb.AgentSkill{
			Id:          fmt.Sprintf("skill_%d", skillIndex),
			Name:        skill.Name,
			Description: skill.Description,
			Tags:        []string{skillName}, // Use skill name as tag for routing
			InputModes:  []string{"text/plain"},
			OutputModes: []string{"text/plain"},
		})
		skillIndex++
	}

	// Create agent card with required A2A fields
	s.agentCard = &pb.AgentCard{
		ProtocolVersion: "0.2.9",
		Name:            s.config.AgentID,
		Description:     s.config.Description,
		Version:         s.config.Version,
		Skills:          cardSkills,
		Capabilities: &pb.AgentCapabilities{
			Streaming:         false,
			PushNotifications: false,
		},
	}

	// Register agent card with broker
	_, err := s.client.Client.RegisterAgent(ctx, &pb.RegisterAgentRequest{
		AgentCard: s.agentCard,
	})

	if err != nil {
		return fmt.Errorf("failed to register agent with broker: %w", err)
	}

	s.client.Logger.InfoContext(ctx, "Agent card registered",
		"agent_id", s.config.AgentID,
		"name", s.config.Name,
		"skills", len(cardSkills),
	)

	return nil
}

// setupTaskSubscription creates the task subscriber and registers all skill handlers
func (s *SubAgent) setupTaskSubscription(ctx context.Context) error {
	// Create task subscriber
	s.taskSubscriber = agenthub.NewA2ATaskSubscriber(s.client, s.config.AgentID)

	// Register handlers for each skill
	for skillName, skill := range s.skills {
		// Capture variables for closure
		handlerName := skillName
		handlerFunc := skill.Handler

		// Wrap the handler with observability
		wrappedHandler := s.wrapHandlerWithObservability(handlerName, handlerFunc)

		// Register with task subscriber
		s.taskSubscriber.RegisterTaskHandler(handlerName, wrappedHandler)

		s.client.Logger.DebugContext(ctx, "Registered task handler",
			"skill", handlerName,
		)
	}

	// Start task subscription in goroutine
	go func() {
		s.client.Logger.InfoContext(ctx, "Starting task subscription",
			"agent_id", s.config.AgentID,
		)

		if err := s.taskSubscriber.SubscribeToTasks(ctx); err != nil {
			s.client.Logger.ErrorContext(ctx, "Task subscription ended",
				"agent_id", s.config.AgentID,
				"error", err,
			)
		}
	}()

	return nil
}

// wrapHandlerWithObservability wraps a task handler with automatic tracing and logging
func (s *SubAgent) wrapHandlerWithObservability(skillName string, handler TaskHandler) agenthub.A2ATaskHandler {
	return func(ctx context.Context, task *pb.Task, message *pb.Message) (*pb.Artifact, pb.TaskState, string) {
		// Start tracing for task processing
		taskCtx, taskSpan := s.client.TraceManager.StartSpan(ctx, fmt.Sprintf("agent.%s.handle_task", s.config.AgentID)) // No additional attributes needed here; will be added by AddA2ATaskAttributes

		defer taskSpan.End()

		// Add A2A task attributes for observability
		s.client.TraceManager.AddA2ATaskAttributes(
			taskSpan,
			task.GetId(),
			skillName,
			task.GetContextId(),
			len(task.GetHistory()),
			len(task.GetArtifacts()),
		)
		s.client.TraceManager.AddComponentAttribute(taskSpan, s.config.AgentID)

		// Log task processing start
		s.client.Logger.InfoContext(taskCtx, "Processing task",
			"task_id", task.GetId(),
			"skill", skillName,
			"context_id", task.GetContextId(),
		)

		// Call the actual handler
		artifact, state, errorMsg := handler(taskCtx, task, message)

		// Record results in trace
		if state == pb.TaskState_TASK_STATE_COMPLETED {
			s.client.TraceManager.SetSpanSuccess(taskSpan)
			s.client.Logger.InfoContext(taskCtx, "Task completed successfully",
				"task_id", task.GetId(),
				"skill", skillName,
				"has_artifact", artifact != nil,
			)
		} else {
			if errorMsg != "" {
				err := fmt.Errorf("task failed: %s", errorMsg)
				s.client.TraceManager.RecordError(taskSpan, err)
			}
			s.client.Logger.ErrorContext(taskCtx, "Task failed",
				"task_id", task.GetId(),
				"skill", skillName,
				"state", state.String(),
				"error", errorMsg,
			)
		}

		return artifact, state, errorMsg
	}
}

// GetLogger returns the agent's structured logger for custom logging.
//
// The logger is configured with the agent's component name and provides
// structured logging using log/slog. Use this for custom log messages in
// your handler functions.
//
// Returns slog.Default() if called before Run() (client not yet initialized).
//
// Example:
//
//	logger := agent.GetLogger()
//	logger.InfoContext(ctx, "Processing started", "input_size", len(data))
//	logger.ErrorContext(ctx, "Processing failed", "error", err)
func (s *SubAgent) GetLogger() *slog.Logger {
	if s.client == nil {
		return slog.Default()
	}
	return s.client.Logger
}

// GetClient returns the underlying AgentHub client for advanced use cases.
//
// This provides access to the low-level AgentHub client for operations not
// covered by the SubAgent abstraction, such as:
//   - Custom message publishing
//   - Direct task management
//   - Advanced tracing and metrics
//
// Returns nil if called before Run() (client not yet initialized).
//
// Example:
//
//	client := agent.GetClient()
//	_, err := client.Client.PublishMessage(ctx, customMsg, routing)
func (s *SubAgent) GetClient() *agenthub.AgentHubClient {
	return s.client
}

// GetConfig returns the agent's configuration.
//
// The returned configuration includes all defaults that were applied during
// initialization. Modifying the returned Config will not affect the running
// agent.
//
// Example:
//
//	config := agent.GetConfig()
//	fmt.Printf("Agent ID: %s\n", config.AgentID)
//	fmt.Printf("Health Port: %s\n", config.HealthPort)
func (s *SubAgent) GetConfig() *Config {
	return s.config
}
