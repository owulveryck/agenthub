package subagent

// Config holds the configuration for a SubAgent
type Config struct {
	// AgentID is the unique identifier for this agent
	AgentID string

	// Name is the human-readable name of the agent
	Name string

	// Description is a brief description of what the agent does
	Description string

	// Version is the agent version (optional, defaults to "1.0.0")
	Version string

	// HealthPort is the port for the health check server (optional, defaults to "8080")
	HealthPort string

	// BrokerAddr is the address of the broker (optional, uses env AGENTHUB_BROKER_ADDR)
	BrokerAddr string

	// BrokerPort is the gRPC port of the broker (optional, uses env AGENTHUB_GRPC_PORT)
	BrokerPort string
}

// WithDefaults returns a new Config with default values applied for optional fields
func (c *Config) WithDefaults() *Config {
	config := *c

	if config.Version == "" {
		config.Version = "1.0.0"
	}

	if config.HealthPort == "" {
		config.HealthPort = "8080"
	}

	return &config
}

// Validate checks if the required configuration fields are set
func (c *Config) Validate() error {
	if c.AgentID == "" {
		return ErrMissingAgentID
	}

	if c.Name == "" {
		return ErrMissingName
	}

	if c.Description == "" {
		return ErrMissingDescription
	}

	return nil
}
