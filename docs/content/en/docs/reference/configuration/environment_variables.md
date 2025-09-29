---
title: "Environment Variables Reference"
weight: 10
description: "Complete reference for all environment variables used by AgentHub's unified abstractions for configuration and observability."
---

# Environment Variables Reference

This reference documents all environment variables used by AgentHub's unified abstraction system. All components automatically load these variables for configuration.

## Core Configuration

### Broker Connection

| Variable | Default | Description | Used By |
|----------|---------|-------------|---------|
| `AGENTHUB_BROKER_ADDR` | `localhost` | Broker server hostname or IP address | Agents |
| `AGENTHUB_BROKER_PORT` | `50051` | Broker gRPC port number | Agents |
| `AGENTHUB_GRPC_PORT` | `:50051` | Server listen address (for broker) | Broker |

**Example:**
```bash
export AGENTHUB_BROKER_ADDR="production-broker.example.com"
export AGENTHUB_BROKER_PORT="50051"
export AGENTHUB_GRPC_PORT=":50051"
```

### Health Monitoring

| Variable | Default | Description | Used By |
|----------|---------|-------------|---------|
| `BROKER_HEALTH_PORT` | `8080` | Broker health check endpoint port | Broker |
| `PUBLISHER_HEALTH_PORT` | `8081` | Publisher health check endpoint port | Publishers |
| `SUBSCRIBER_HEALTH_PORT` | `8082` | Subscriber health check endpoint port | Subscribers |

**Health Endpoints Available:**
- `http://localhost:8080/health` - Health check
- `http://localhost:8080/metrics` - Prometheus metrics
- `http://localhost:8080/ready` - Readiness check

**Example:**
```bash
export BROKER_HEALTH_PORT="8080"
export PUBLISHER_HEALTH_PORT="8081"
export SUBSCRIBER_HEALTH_PORT="8082"
```

## Observability Configuration

### Distributed Tracing

| Variable | Default | Description | Used By |
|----------|---------|-------------|---------|
| `JAEGER_ENDPOINT` | `127.0.0.1:4317` | Jaeger OTLP endpoint for traces | All components |
| `SERVICE_NAME` | `agenthub-service` | Service name for tracing | All components |
| `SERVICE_VERSION` | `1.0.0` | Service version for telemetry | All components |

**Example:**
```bash
export JAEGER_ENDPOINT="http://jaeger.example.com:14268/api/traces"
export SERVICE_NAME="my-agenthub-app"
export SERVICE_VERSION="2.1.0"
```

**Jaeger Integration:**
- When `JAEGER_ENDPOINT` is set: Automatic tracing enabled
- When empty or unset: Tracing disabled (minimal overhead)
- Supports both gRPC (4317) and HTTP (14268) endpoints

### Metrics Collection

| Variable | Default | Description | Used By |
|----------|---------|-------------|---------|
| `PROMETHEUS_PORT` | `9090` | Prometheus server port | Observability stack |
| `GRAFANA_PORT` | `3333` | Grafana dashboard port | Observability stack |
| `ALERTMANAGER_PORT` | `9093` | AlertManager port | Observability stack |

**Example:**
```bash
export PROMETHEUS_PORT="9090"
export GRAFANA_PORT="3333"
export ALERTMANAGER_PORT="9093"
```

### OpenTelemetry Collector

| Variable | Default | Description | Used By |
|----------|---------|-------------|---------|
| `OTLP_GRPC_PORT` | `4320` | OTLP Collector gRPC port | Observability stack |
| `OTLP_HTTP_PORT` | `4321` | OTLP Collector HTTP port | Observability stack |

**Example:**
```bash
export OTLP_GRPC_PORT="4320"
export OTLP_HTTP_PORT="4321"
```

## Service Configuration

### General Settings

| Variable | Default | Description | Used By |
|----------|---------|-------------|---------|
| `ENVIRONMENT` | `development` | Deployment environment | All components |
| `LOG_LEVEL` | `INFO` | Logging level (DEBUG, INFO, WARN, ERROR) | All components |

**Example:**
```bash
export ENVIRONMENT="production"
export LOG_LEVEL="WARN"
```

## Environment-Specific Configurations

### Development Environment

```bash
# .envrc for development
export AGENTHUB_BROKER_ADDR="localhost"
export AGENTHUB_BROKER_PORT="50051"
export AGENTHUB_GRPC_PORT=":50051"

# Health ports
export BROKER_HEALTH_PORT="8080"
export PUBLISHER_HEALTH_PORT="8081"
export SUBSCRIBER_HEALTH_PORT="8082"

# Observability (local stack)
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
export PROMETHEUS_PORT="9090"
export GRAFANA_PORT="3333"

# Service metadata
export SERVICE_NAME="agenthub-dev"
export SERVICE_VERSION="dev"
export ENVIRONMENT="development"
export LOG_LEVEL="DEBUG"
```

### Staging Environment

```bash
# .envrc for staging
export AGENTHUB_BROKER_ADDR="staging-broker.example.com"
export AGENTHUB_BROKER_PORT="50051"

# Health ports (non-conflicting)
export BROKER_HEALTH_PORT="8080"
export PUBLISHER_HEALTH_PORT="8081"
export SUBSCRIBER_HEALTH_PORT="8082"

# Observability (staging stack)
export JAEGER_ENDPOINT="http://staging-jaeger.example.com:14268/api/traces"
export PROMETHEUS_PORT="9090"
export GRAFANA_PORT="3333"

# Service metadata
export SERVICE_NAME="agenthub-staging"
export SERVICE_VERSION="1.2.0-rc1"
export ENVIRONMENT="staging"
export LOG_LEVEL="INFO"
```

### Production Environment

```bash
# .envrc for production
export AGENTHUB_BROKER_ADDR="prod-broker.example.com"
export AGENTHUB_BROKER_PORT="50051"

# Health ports
export BROKER_HEALTH_PORT="8080"
export PUBLISHER_HEALTH_PORT="8081"
export SUBSCRIBER_HEALTH_PORT="8082"

# Observability (production stack)
export JAEGER_ENDPOINT="http://jaeger.prod.example.com:14268/api/traces"
export PROMETHEUS_PORT="9090"
export GRAFANA_PORT="3333"
export ALERTMANAGER_PORT="9093"

# Service metadata
export SERVICE_NAME="agenthub-prod"
export SERVICE_VERSION="1.2.0"
export ENVIRONMENT="production"
export LOG_LEVEL="WARN"
```

## Configuration Loading

### Automatic Loading by Unified Abstractions

The unified abstractions automatically load environment variables:

```go
// Automatic configuration loading
config := agenthub.NewGRPCConfig("my-component")

// Results in:
// config.BrokerAddr = "localhost:50051" (AGENTHUB_BROKER_ADDR + AGENTHUB_BROKER_PORT)
// config.ServerAddr = ":50051" (AGENTHUB_GRPC_PORT)
// config.HealthPort = "8080" (BROKER_HEALTH_PORT)
// config.ComponentName = "my-component" (from parameter)
```

### Using direnv (Recommended)

1. **Install direnv**: https://direnv.net/docs/installation.html

2. **Create .envrc file**:
   ```bash
   # Create .envrc in project root
   cat > .envrc << 'EOF'
   export AGENTHUB_BROKER_ADDR="localhost"
   export AGENTHUB_BROKER_PORT="50051"
   export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
   export SERVICE_NAME="my-agenthub-app"
   EOF
   ```

3. **Allow direnv**:
   ```bash
   direnv allow
   ```

4. **Automatic loading**: Variables load automatically when entering directory

### Manual Loading

```bash
# Source variables manually
source .envrc

# Or set individually
export AGENTHUB_BROKER_ADDR="localhost"
export JAEGER_ENDPOINT="http://localhost:14268/api/traces"
```

## Configuration Validation

### Required Variables

**Minimal configuration** (all have defaults):
- No variables are strictly required
- Defaults work for local development

**Production recommendations**:
- Set `JAEGER_ENDPOINT` for tracing
- Set `SERVICE_NAME` for identification
- Set `ENVIRONMENT` to "production"
- Configure unique health ports if running multiple services

### Configuration Verification

**Check loaded configuration**:
```go
config := agenthub.NewGRPCConfig("test")
fmt.Printf("Broker: %s\n", config.BrokerAddr)
fmt.Printf("Health: %s\n", config.HealthPort)
fmt.Printf("Component: %s\n", config.ComponentName)
```

**Verify health endpoints**:
```bash
# Check if configuration is working
curl http://localhost:8080/health
curl http://localhost:8081/health  # Publisher
curl http://localhost:8082/health  # Subscriber
```

**Verify tracing**:
- Open Jaeger UI: http://localhost:16686
- Look for traces from your service name
- Check spans are being created

## Common Patterns

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  broker:
    build: .
    command: go run broker/main.go
    environment:
      - AGENTHUB_GRPC_PORT=:50051
      - BROKER_HEALTH_PORT=8080
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
      - SERVICE_NAME=agenthub-broker
    ports:
      - "50051:50051"
      - "8080:8080"

  publisher:
    build: .
    command: go run agents/publisher/main.go
    environment:
      - AGENTHUB_BROKER_ADDR=broker
      - AGENTHUB_BROKER_PORT=50051
      - PUBLISHER_HEALTH_PORT=8081
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
      - SERVICE_NAME=agenthub-publisher
    ports:
      - "8081:8081"
```

### Kubernetes ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: agenthub-config
data:
  AGENTHUB_BROKER_ADDR: "agenthub-broker.default.svc.cluster.local"
  AGENTHUB_BROKER_PORT: "50051"
  JAEGER_ENDPOINT: "http://jaeger.observability.svc.cluster.local:14268/api/traces"
  SERVICE_NAME: "agenthub-k8s"
  SERVICE_VERSION: "1.0.0"
  ENVIRONMENT: "production"
  LOG_LEVEL: "INFO"

---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agenthub-publisher
spec:
  template:
    spec:
      containers:
      - name: publisher
        image: agenthub:latest
        envFrom:
        - configMapRef:
            name: agenthub-config
        env:
        - name: PUBLISHER_HEALTH_PORT
          value: "8080"
```

## Troubleshooting

### Common Issues

| Problem | Solution |
|---------|----------|
| Agent can't connect to broker | Check `AGENTHUB_BROKER_ADDR` and `AGENTHUB_BROKER_PORT` |
| Health endpoint not accessible | Verify `*_HEALTH_PORT` variables and port availability |
| No traces in Jaeger | Set `JAEGER_ENDPOINT` and ensure Jaeger is running |
| Port conflicts | Use different ports for each component's health endpoints |
| Configuration not loading | Ensure variables are exported, check with `printenv` |

### Debug Configuration

**Check environment variables**:
```bash
# List all AgentHub variables
printenv | grep AGENTHUB

# List all observability variables
printenv | grep -E "(JAEGER|SERVICE|PROMETHEUS|GRAFANA)"

# Check specific variable
echo $AGENTHUB_BROKER_ADDR
```

**Test configuration**:
```bash
# Quick test with temporary override
AGENTHUB_BROKER_ADDR=test-broker go run agents/publisher/main.go

# Verify health endpoint responds
curl -f http://localhost:8080/health || echo "Health check failed"
```

### Configuration Precedence

1. **Environment variables** (highest priority)
2. **Default values** (from code)

**Example**: If `AGENTHUB_BROKER_ADDR` is set, it overrides the default "localhost"

---

This environment variable reference provides comprehensive documentation for configuring AgentHub using the unified abstraction system. For practical usage examples, see the [Installation and Setup Tutorial](../../tutorials/getting-started/installation_and_setup/) and [Configuration Reference](configuration_reference/).