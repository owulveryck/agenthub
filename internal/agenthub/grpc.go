package agenthub

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/owulveryck/agenthub/internal/grpc"
	"github.com/owulveryck/agenthub/internal/observability"
)

const (
	DefaultGRPCPort   = ":50051"
	DefaultHealthPort = "8080"
)

// GRPCConfig holds configuration for gRPC client/server
type GRPCConfig struct {
	// ServerAddr is the address the gRPC server will listen on (e.g., ":50051")
	ServerAddr string
	// BrokerAddr is the address to connect to the broker (e.g., "localhost:50051")
	BrokerAddr string
	// HealthPort is the port for health/metrics endpoints
	HealthPort string
	// ComponentName identifies the component (broker, publisher, subscriber)
	ComponentName string
}

// NewGRPCConfig creates a new gRPC configuration from environment variables
func NewGRPCConfig(componentName string) *GRPCConfig {
	config := &GRPCConfig{
		ComponentName: componentName,
		ServerAddr:    getEnvWithDefault("AGENTHUB_GRPC_PORT", DefaultGRPCPort),
		BrokerAddr:    getEnvWithDefault("AGENTHUB_BROKER_ADDR", "localhost:50051"),
		HealthPort:    getEnvWithDefault("BROKER_HEALTH_PORT", DefaultHealthPort),
	}

	// For broker, use ServerAddr as listen address
	// For agents, use BrokerAddr as connection address
	return config
}

// AgentHubServer wraps the gRPC server with observability
type AgentHubServer struct {
	Server         *grpc.Server
	Listener       net.Listener
	Observability  *observability.Observability
	TraceManager   *observability.TraceManager
	MetricsManager *observability.MetricsManager
	HealthServer   *observability.HealthServer
	Logger         *slog.Logger
	Config         *GRPCConfig
}

// NewAgentHubServer creates a new gRPC server with observability
func NewAgentHubServer(config *GRPCConfig) (*AgentHubServer, error) {
	// Initialize observability
	obsConfig := observability.DefaultConfig("agenthub")
	obs, err := observability.NewObservability(obsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}

	// Initialize metrics manager
	metricsManager, err := observability.NewMetricsManager(obs.Meter)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics manager: %w", err)
	}

	// Initialize trace manager
	traceManager := observability.NewTraceManager(obsConfig.ServiceName)

	// Initialize health server
	healthServer := observability.NewHealthServer(config.HealthPort, obsConfig.ServiceName, obsConfig.ServiceVersion)

	// Add basic health check
	healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
		return nil
	}))

	// Create listener
	lis, err := net.Listen("tcp", config.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", config.ServerAddr, err)
	}

	// Create gRPC server with OpenTelemetry instrumentation
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	return &AgentHubServer{
		Server:         grpcServer,
		Listener:       lis,
		Observability:  obs,
		TraceManager:   traceManager,
		MetricsManager: metricsManager,
		HealthServer:   healthServer,
		Logger:         obs.Logger,
		Config:         config,
	}, nil
}

// Start starts the gRPC server and health server
func (s *AgentHubServer) Start(ctx context.Context) error {
	// Start health server
	go func() {
		s.Logger.Info("Starting health server", slog.String("port", s.Config.HealthPort))
		if err := s.HealthServer.Start(ctx); err != nil {
			s.Logger.Error("Health server failed", slog.Any("error", err))
		}
	}()

	// Start metrics collection
	go func() {
		ticker := NewMetricsTicker(ctx, s.MetricsManager)
		ticker.Start()
	}()

	s.Logger.Info("AgentHub gRPC server with observability listening",
		slog.String("address", s.Listener.Addr().String()),
		slog.String("health_endpoint", fmt.Sprintf("http://localhost:%s/health", s.Config.HealthPort)),
		slog.String("metrics_endpoint", fmt.Sprintf("http://localhost:%s/metrics", s.Config.HealthPort)),
		slog.String("component", s.Config.ComponentName),
	)

	return s.Server.Serve(s.Listener)
}

// Shutdown gracefully shuts down the server
func (s *AgentHubServer) Shutdown(ctx context.Context) error {
	s.Logger.InfoContext(ctx, "Shutting down AgentHub server")

	// Graceful shutdown of gRPC server
	s.Server.GracefulStop()

	// Shutdown observability components
	if err := s.HealthServer.Shutdown(ctx); err != nil {
		s.Logger.ErrorContext(ctx, "Error shutting down health server", slog.Any("error", err))
	}

	if err := s.Observability.Shutdown(ctx); err != nil {
		s.Logger.ErrorContext(ctx, "Observability shutdown failed - likely OTLP trace export issue",
			slog.Any("error", err),
			slog.String("service", s.Config.ComponentName),
			slog.String("otlp_endpoint", s.Observability.Config.JaegerEndpoint),
		)
		return err
	}

	return nil
}

// AgentHubClient wraps the gRPC client with observability
type AgentHubClient struct {
	Client         pb.EventBusClient
	Connection     *grpc.ClientConn
	Observability  *observability.Observability
	TraceManager   *observability.TraceManager
	MetricsManager *observability.MetricsManager
	HealthServer   *observability.HealthServer
	Logger         *slog.Logger
	Config         *GRPCConfig
}

// NewAgentHubClient creates a new gRPC client with observability
func NewAgentHubClient(config *GRPCConfig) (*AgentHubClient, error) {
	// Initialize observability
	obsConfig := observability.DefaultConfig("agenthub")
	obs, err := observability.NewObservability(obsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}

	// Initialize metrics manager
	metricsManager, err := observability.NewMetricsManager(obs.Meter)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics manager: %w", err)
	}

	// Initialize trace manager
	traceManager := observability.NewTraceManager(obsConfig.ServiceName)

	// Initialize health server
	healthServer := observability.NewHealthServer(config.HealthPort, obsConfig.ServiceName, obsConfig.ServiceVersion)

	// Add basic health check
	healthServer.AddChecker("self", observability.NewBasicHealthChecker("self", func(ctx context.Context) error {
		return nil
	}))

	// Set up gRPC connection with OpenTelemetry instrumentation
	conn, err := grpc.Dial(config.BrokerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to broker at %s: %w", config.BrokerAddr, err)
	}

	client := pb.NewEventBusClient(conn)

	// Add gRPC connection health check
	healthServer.AddChecker("agenthub_connection", observability.NewGRPCHealthChecker("agenthub_connection", config.BrokerAddr))

	return &AgentHubClient{
		Client:         client,
		Connection:     conn,
		Observability:  obs,
		TraceManager:   traceManager,
		MetricsManager: metricsManager,
		HealthServer:   healthServer,
		Logger:         obs.Logger,
		Config:         config,
	}, nil
}

// Start starts the client's health server and metrics collection
func (c *AgentHubClient) Start(ctx context.Context) error {
	// Start health server
	go func() {
		c.Logger.Info("Starting health server", slog.String("port", c.Config.HealthPort))
		if err := c.HealthServer.Start(ctx); err != nil {
			c.Logger.Error("Health server failed", slog.Any("error", err))
		}
	}()

	// Start metrics collection
	go func() {
		ticker := NewMetricsTicker(ctx, c.MetricsManager)
		ticker.Start()
	}()

	c.Logger.InfoContext(ctx, "AgentHub client started with observability",
		slog.String("broker_addr", c.Config.BrokerAddr),
		slog.String("component", c.Config.ComponentName),
	)

	return nil
}

// Shutdown gracefully shuts down the client
func (c *AgentHubClient) Shutdown(ctx context.Context) error {
	c.Logger.InfoContext(ctx, "Shutting down AgentHub client")

	// Close gRPC connection
	if err := c.Connection.Close(); err != nil {
		c.Logger.ErrorContext(ctx, "Error closing gRPC connection", slog.Any("error", err))
	}

	// Shutdown observability components
	if err := c.HealthServer.Shutdown(ctx); err != nil {
		c.Logger.ErrorContext(ctx, "Error shutting down health server", slog.Any("error", err))
	}

	if err := c.Observability.Shutdown(ctx); err != nil {
		c.Logger.ErrorContext(ctx, "Observability shutdown failed - likely OTLP trace export issue",
			slog.Any("error", err),
			slog.String("service", c.Config.ComponentName),
			slog.String("otlp_endpoint", c.Observability.Config.JaegerEndpoint),
		)
		return err
	}

	return nil
}

// Helper function to get environment variable with default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
