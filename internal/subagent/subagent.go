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

// SubAgent encapsulates the common functionality for building agents
type SubAgent struct {
	config         *Config
	client         *agenthub.AgentHubClient
	taskSubscriber *agenthub.A2ATaskSubscriber
	skills         map[string]*Skill
	agentCard      *pb.AgentCard
	running        bool
}

// New creates a new SubAgent with the given configuration
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

// AddSkill registers a new skill with the agent
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

// MustAddSkill is like AddSkill but panics on error (for cleaner initialization code)
func (s *SubAgent) MustAddSkill(name, description string, handler TaskHandler) {
	if err := s.AddSkill(name, description, handler); err != nil {
		panic(err)
	}
}

// Run starts the agent and blocks until the context is cancelled
// It handles the full lifecycle: setup, registration, subscription, and graceful shutdown
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

// GetLogger returns the agent's logger for custom logging needs
func (s *SubAgent) GetLogger() *slog.Logger {
	if s.client == nil {
		return slog.Default()
	}
	return s.client.Logger
}

// GetClient returns the underlying AgentHub client for advanced use cases
func (s *SubAgent) GetClient() *agenthub.AgentHubClient {
	return s.client
}

// GetConfig returns the agent configuration
func (s *SubAgent) GetConfig() *Config {
	return s.config
}
