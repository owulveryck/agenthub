---
title: "Configuration Reference"
weight: 20
description: "Comprehensive reference for configuring AgentHub components using the unified abstraction library with environment-based configuration."
---

# Configuration Reference

This document provides comprehensive reference for configuring AgentHub components using the unified abstraction library with environment-based configuration.

## Unified Abstraction Configuration

AgentHub uses environment variables for all configuration with the unified abstraction library providing automatic configuration setup.

### Core Environment Variables

#### gRPC Connection Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTHUB_BROKER_ADDR` | `localhost` | Broker server hostname or IP address |
| `AGENTHUB_BROKER_PORT` | `50051` | Broker gRPC port number |
| `AGENTHUB_GRPC_PORT` | `:50051` | Server listen address (for broker) |

**Note:** The unified abstraction automatically combines `AGENTHUB_BROKER_ADDR` and `AGENTHUB_BROKER_PORT` into a complete broker address (e.g., `localhost:50051`).

#### Health Monitoring Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `BROKER_HEALTH_PORT` | `8080` | Broker health check endpoint port |
| `PUBLISHER_HEALTH_PORT` | `8081` | Publisher health check endpoint port |
| `SUBSCRIBER_HEALTH_PORT` | `8082` | Subscriber health check endpoint port |

#### Observability Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `JAEGER_ENDPOINT` | `127.0.0.1:4317` | Jaeger OTLP endpoint for distributed tracing |
| `PROMETHEUS_PORT` | `9090` | Prometheus metrics collection port |
| `GRAFANA_PORT` | `3333` | Grafana dashboard web interface port |
| `ALERTMANAGER_PORT` | `9093` | AlertManager web interface port |
| `OTLP_GRPC_PORT` | `4320` | OpenTelemetry Collector gRPC port |
| `OTLP_HTTP_PORT` | `4321` | OpenTelemetry Collector HTTP port |

#### Service Metadata

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_VERSION` | `1.0.0` | Service version for telemetry and observability |
| `ENVIRONMENT` | `development` | Deployment environment (development, staging, production) |

## Unified Abstraction Usage

### Using Configuration with the Unified Abstraction

The unified abstraction library automatically loads configuration from environment variables:

```go
// Create configuration from environment variables
config := agenthub.NewGRPCConfig("my-component")

// Configuration is automatically populated:
// - config.BrokerAddr: "localhost:50051" (combined from AGENTHUB_BROKER_ADDR + AGENTHUB_BROKER_PORT)
// - config.ServerAddr: ":50051" (from AGENTHUB_GRPC_PORT)
// - config.HealthPort: "8080" (from BROKER_HEALTH_PORT)
// - config.ComponentName: "my-component" (from parameter)
```

### Environment Variable Loading

The recommended way to load environment variables:

**Option 1: Using direnv (recommended)**
```bash
# Place variables in .envrc file
direnv allow
```

**Option 2: Source manually**
```bash
source .envrc
```

**Option 3: Set individual variables**
```bash
export AGENTHUB_BROKER_ADDR=localhost
export AGENTHUB_BROKER_PORT=50051
export JAEGER_ENDPOINT=127.0.0.1:4317
```

### Configuration Override Examples

You can override defaults by setting environment variables before running:

```bash
# Use different broker address
export AGENTHUB_BROKER_ADDR=remote-broker.example.com
export AGENTHUB_BROKER_PORT=9090
go run broker/main.go

# Use different health ports to avoid conflicts
export BROKER_HEALTH_PORT=8083
export PUBLISHER_HEALTH_PORT=8084
export SUBSCRIBER_HEALTH_PORT=8085
go run agents/publisher/main.go

# Use custom observability endpoints
export JAEGER_ENDPOINT=jaeger.example.com:4317
export PROMETHEUS_PORT=9091
go run broker/main.go
```

### Configuration Best Practices

1. **Use .envrc for Development**: Keep all environment variables in `.envrc` for consistent development experience
2. **Override Selectively**: Only override specific variables when needed, use defaults otherwise
3. **Environment-Specific Configs**: Use different variable values for development, staging, and production
4. **Health Port Management**: Use different health ports for each component to avoid conflicts
5. **Observability Integration**: Always configure observability endpoints for production deployments

### Legacy Configuration Migration

If migrating from previous versions of AgentHub:

**Old Configuration Pattern:**
```go
// Manual server setup (deprecated)
lis, err := net.Listen("tcp", ":50051")
server := grpc.NewServer()
// ... extensive setup code
```

**New Unified Abstraction Pattern:**
```go
// Automatic configuration from environment
config := agenthub.NewGRPCConfig("broker")
server, err := agenthub.NewAgentHubServer(config)
service := agenthub.NewEventBusService(server)
pb.RegisterEventBusServer(server.Server, service)
server.Start(ctx)
```

## Command-Line Usage

### Basic Commands

The unified abstraction provides simplified command execution:

```bash
eventbus-server [OPTIONS]

Options:
  -port int
        Server port (default 50051)
  -host string
        Server host (default "0.0.0.0")
  -config string
        Configuration file path
  -log-level string
        Log level: debug, info, warn, error (default "info")
  -log-file string
        Log file path (default: stdout)
  -max-connections int
        Maximum concurrent connections (default 1000)
  -channel-buffer-size int
        Channel buffer size (default 10)
  -help
        Show help message
  -version
        Show version information
```

### Configuration File

The broker can also be configured using a YAML configuration file:

```yaml
# agenthub.yaml
server:
  host: "0.0.0.0"
  port: 50051
  max_connections: 1000
  timeout: "30s"

logging:
  level: "info"
  format: "json"
  file: "/var/log/agenthub/broker.log"

performance:
  channel_buffer_size: 10
  max_message_size: "4MB"
  keepalive_time: "30s"
  keepalive_timeout: "5s"

limits:
  max_agents: 10000
  max_tasks_per_agent: 100
  memory_limit: "1GB"

security:
  tls_enabled: false
  cert_file: ""
  key_file: ""
  ca_file: ""
```

**Loading Configuration:**
```bash
eventbus-server -config /path/to/agenthub.yaml
```

## Agent Configuration

### Environment Variables

Agents can be configured using environment variables:

#### Connection Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTHUB_BROKER_ADDRESS` | `localhost:50051` | Broker server address |
| `AGENTHUB_AGENT_ID` | Generated | Unique agent identifier |
| `AGENTHUB_CONNECTION_TIMEOUT` | `10s` | Connection timeout |
| `AGENTHUB_RETRY_ATTEMPTS` | `3` | Connection retry attempts |
| `AGENTHUB_RETRY_DELAY` | `1s` | Delay between retries |

#### Task Processing Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTHUB_MAX_CONCURRENT_TASKS` | `5` | Maximum concurrent task processing |
| `AGENTHUB_TASK_TIMEOUT` | `300s` | Default task timeout |
| `AGENTHUB_PROGRESS_INTERVAL` | `5s` | Progress reporting interval |
| `AGENTHUB_TASK_TYPES` | `""` | Comma-separated list of supported task types |

#### Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTHUB_AGENT_LOG_LEVEL` | `info` | Agent logging level |
| `AGENTHUB_AGENT_LOG_FORMAT` | `text` | Agent log format |
| `AGENTHUB_AGENT_LOG_FILE` | `""` | Agent log file path |

### Agent Configuration Examples

#### Publisher Configuration

```go
package main

import (
    "os"
    "strconv"
    "time"
)

type PublisherConfig struct {
    BrokerAddress    string
    AgentID          string
    ConnectionTimeout time.Duration
    RetryAttempts    int
    RetryDelay       time.Duration
    LogLevel         string
}

func LoadPublisherConfig() *PublisherConfig {
    config := &PublisherConfig{
        BrokerAddress:    getEnv("AGENTHUB_BROKER_ADDRESS", "localhost:50051"),
        AgentID:          getEnv("AGENTHUB_AGENT_ID", generateAgentID()),
        ConnectionTimeout: getDuration("AGENTHUB_CONNECTION_TIMEOUT", "10s"),
        RetryAttempts:    getInt("AGENTHUB_RETRY_ATTEMPTS", 3),
        RetryDelay:       getDuration("AGENTHUB_RETRY_DELAY", "1s"),
        LogLevel:         getEnv("AGENTHUB_AGENT_LOG_LEVEL", "info"),
    }

    return config
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}

func getDuration(key string, defaultValue string) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    d, _ := time.ParseDuration(defaultValue)
    return d
}
```

#### Subscriber Configuration

```go
type SubscriberConfig struct {
    BrokerAddress      string
    AgentID            string
    MaxConcurrentTasks int
    TaskTimeout        time.Duration
    ProgressInterval   time.Duration
    SupportedTaskTypes []string
    LogLevel           string
}

func LoadSubscriberConfig() *SubscriberConfig {
    taskTypesStr := getEnv("AGENTHUB_TASK_TYPES", "")
    var taskTypes []string
    if taskTypesStr != "" {
        taskTypes = strings.Split(taskTypesStr, ",")
        for i, taskType := range taskTypes {
            taskTypes[i] = strings.TrimSpace(taskType)
        }
    }

    config := &SubscriberConfig{
        BrokerAddress:      getEnv("AGENTHUB_BROKER_ADDRESS", "localhost:50051"),
        AgentID:            getEnv("AGENTHUB_AGENT_ID", generateAgentID()),
        MaxConcurrentTasks: getInt("AGENTHUB_MAX_CONCURRENT_TASKS", 5),
        TaskTimeout:        getDuration("AGENTHUB_TASK_TIMEOUT", "300s"),
        ProgressInterval:   getDuration("AGENTHUB_PROGRESS_INTERVAL", "5s"),
        SupportedTaskTypes: taskTypes,
        LogLevel:           getEnv("AGENTHUB_AGENT_LOG_LEVEL", "info"),
    }

    return config
}
```

### Agent Configuration File

Agents can also use configuration files:

```yaml
# agent.yaml
agent:
  id: "data_processor_001"
  broker_address: "broker.example.com:50051"
  connection_timeout: "10s"
  retry_attempts: 3
  retry_delay: "1s"

task_processing:
  max_concurrent_tasks: 5
  task_timeout: "300s"
  progress_interval: "5s"
  supported_task_types:
    - "data_analysis"
    - "data_transformation"
    - "data_validation"

logging:
  level: "info"
  format: "json"
  file: "/var/log/agenthub/agent.log"

health:
  port: 8080
  endpoint: "/health"
  check_interval: "30s"
```

## Security Configuration

### TLS Configuration

#### Broker TLS Setup

```yaml
# broker configuration
security:
  tls_enabled: true
  cert_file: "/etc/agenthub/certs/server.crt"
  key_file: "/etc/agenthub/certs/server.key"
  ca_file: "/etc/agenthub/certs/ca.crt"
  client_auth: "require_and_verify"
```

#### Agent TLS Setup

```go
// Agent TLS connection
func createTLSConnection(address string) (*grpc.ClientConn, error) {
    config := &tls.Config{
        ServerName: "agenthub-broker",
        // Load client certificates if needed
    }

    creds := credentials.NewTLS(config)

    conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
    if err != nil {
        return nil, fmt.Errorf("failed to connect with TLS: %v", err)
    }

    return conn, nil
}
```

### Authentication Configuration

#### JWT Authentication

```yaml
# broker configuration
security:
  auth_enabled: true
  auth_method: "jwt"
  jwt_secret: "your-secret-key"
  jwt_issuer: "agenthub-broker"
  jwt_expiry: "24h"
```

```go
// Agent authentication
type AuthenticatedAgent struct {
    client   pb.EventBusClient
    token    string
    agentID  string
}

func (a *AuthenticatedAgent) authenticate() error {
    // Add authentication token to context
    ctx := metadata.AppendToOutgoingContext(context.Background(),
        "authorization", "Bearer "+a.token)

    // Use authenticated context for requests
    _, err := a.client.PublishTask(ctx, request)
    return err
}
```

## Production Configuration Examples

### High-Performance Broker Configuration

```yaml
# production-broker.yaml
server:
  host: "0.0.0.0"
  port: 50051
  max_connections: 5000
  timeout: "60s"

performance:
  channel_buffer_size: 50
  max_message_size: "16MB"
  keepalive_time: "10s"
  keepalive_timeout: "3s"

limits:
  max_agents: 50000
  max_tasks_per_agent: 500
  memory_limit: "8GB"

logging:
  level: "warn"
  format: "json"
  file: "/var/log/agenthub/broker.log"

security:
  tls_enabled: true
  cert_file: "/etc/ssl/certs/agenthub.crt"
  key_file: "/etc/ssl/private/agenthub.key"
```

### Cluster Agent Configuration

```yaml
# cluster-agent.yaml
agent:
  id: "${HOSTNAME}_${POD_ID}"
  broker_address: "agenthub-broker.agenthub.svc.cluster.local:50051"
  connection_timeout: "15s"
  retry_attempts: 5
  retry_delay: "2s"

task_processing:
  max_concurrent_tasks: 10
  task_timeout: "1800s"  # 30 minutes
  progress_interval: "10s"

logging:
  level: "info"
  format: "json"
  file: "stdout"

health:
  port: 8080
  endpoint: "/health"
  check_interval: "30s"

metrics:
  enabled: true
  port: 9090
  endpoint: "/metrics"
```

## Environment-Specific Configurations

### Development Environment

```bash
# .env.development
AGENTHUB_PORT=50051
AGENTHUB_LOG_LEVEL=debug
AGENTHUB_LOG_FORMAT=text
AGENTHUB_MAX_CONNECTIONS=100
AGENTHUB_CHANNEL_BUFFER_SIZE=5

# Agent settings
AGENTHUB_BROKER_ADDRESS=localhost:50051
AGENTHUB_MAX_CONCURRENT_TASKS=2
AGENTHUB_TASK_TIMEOUT=60s
AGENTHUB_AGENT_LOG_LEVEL=debug
```

### Staging Environment

```bash
# .env.staging
AGENTHUB_PORT=50051
AGENTHUB_LOG_LEVEL=info
AGENTHUB_LOG_FORMAT=json
AGENTHUB_MAX_CONNECTIONS=1000
AGENTHUB_CHANNEL_BUFFER_SIZE=20

# Security
AGENTHUB_TLS_ENABLED=true
AGENTHUB_CERT_FILE=/etc/certs/staging.crt
AGENTHUB_KEY_FILE=/etc/certs/staging.key

# Agent settings
AGENTHUB_BROKER_ADDRESS=staging-broker.example.com:50051
AGENTHUB_MAX_CONCURRENT_TASKS=5
AGENTHUB_TASK_TIMEOUT=300s
```

### Production Environment

```bash
# .env.production
AGENTHUB_PORT=50051
AGENTHUB_LOG_LEVEL=warn
AGENTHUB_LOG_FORMAT=json
AGENTHUB_LOG_FILE=/var/log/agenthub/broker.log
AGENTHUB_MAX_CONNECTIONS=5000
AGENTHUB_CHANNEL_BUFFER_SIZE=50

# Security
AGENTHUB_TLS_ENABLED=true
AGENTHUB_CERT_FILE=/etc/ssl/certs/agenthub.crt
AGENTHUB_KEY_FILE=/etc/ssl/private/agenthub.key
AGENTHUB_CA_FILE=/etc/ssl/certs/ca.crt

# Performance
AGENTHUB_MAX_MESSAGE_SIZE=16MB
AGENTHUB_KEEPALIVE_TIME=10s
AGENTHUB_MEMORY_LIMIT=8GB

# Agent settings
AGENTHUB_BROKER_ADDRESS=agenthub-prod.example.com:50051
AGENTHUB_MAX_CONCURRENT_TASKS=10
AGENTHUB_TASK_TIMEOUT=1800s
AGENTHUB_CONNECTION_TIMEOUT=15s
AGENTHUB_RETRY_ATTEMPTS=5
```

## Configuration Validation

### Broker Configuration Validation

```go
type BrokerConfig struct {
    Port             int           `yaml:"port" validate:"min=1,max=65535"`
    Host             string        `yaml:"host" validate:"required"`
    MaxConnections   int           `yaml:"max_connections" validate:"min=1"`
    Timeout          time.Duration `yaml:"timeout" validate:"min=1s"`
    ChannelBufferSize int          `yaml:"channel_buffer_size" validate:"min=1"`
}

func (c *BrokerConfig) Validate() error {
    validate := validator.New()
    return validate.Struct(c)
}
```

### Agent Configuration Validation

```go
type AgentConfig struct {
    BrokerAddress      string        `yaml:"broker_address" validate:"required"`
    AgentID            string        `yaml:"agent_id" validate:"required,min=1,max=64"`
    MaxConcurrentTasks int           `yaml:"max_concurrent_tasks" validate:"min=1,max=100"`
    TaskTimeout        time.Duration `yaml:"task_timeout" validate:"min=1s"`
}

func (c *AgentConfig) Validate() error {
    validate := validator.New()
    if err := validate.Struct(c); err != nil {
        return err
    }

    // Custom validation
    if !strings.Contains(c.BrokerAddress, ":") {
        return errors.New("broker_address must include port")
    }

    return nil
}
```

This comprehensive configuration reference covers all aspects of configuring AgentHub for different environments and use cases.