---
title: "AgentHub Health Endpoints Reference"
weight: 30
description: "Complete documentation for AgentHub's health monitoring APIs, endpoint specifications, status codes, and integration patterns."
---

# AgentHub Health Endpoints Reference

**Technical reference**: Complete documentation for AgentHub's health monitoring APIs, endpoint specifications, status codes, and integration patterns.

## Overview

Every observable AgentHub service exposes standardized health endpoints for monitoring, load balancing, and operational management.

## Standard Endpoints

### Health Check Endpoint

#### `/health`
**Purpose**: Comprehensive service health status
**Method**: GET
**Port**: Service-specific (8080-8083)

**Response Format**:
```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "2025-09-28T21:00:00.000Z",
  "service": "agenthub-broker",
  "version": "1.0.0",
  "uptime": "2h34m12s",
  "checks": [
    {
      "name": "self",
      "status": "healthy",
      "message": "Service is running normally",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "1.2ms"
    },
    {
      "name": "database_connection",
      "status": "healthy",
      "message": "Database connection is active",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "15.6ms"
    }
  ]
}
```

**Status Codes**:
- `200 OK` - All checks healthy
- `503 Service Unavailable` - One or more checks unhealthy
- `500 Internal Server Error` - Health check system failure

### Readiness Endpoint

#### `/ready`
**Purpose**: Service readiness for traffic acceptance
**Method**: GET

**Response Format**:
```json
{
  "ready": true,
  "timestamp": "2025-09-28T21:00:00.000Z",
  "service": "agenthub-broker",
  "dependencies": [
    {
      "name": "grpc_server",
      "ready": true,
      "message": "gRPC server listening on :50051"
    },
    {
      "name": "observability",
      "ready": true,
      "message": "OpenTelemetry initialized"
    }
  ]
}
```

**Status Codes**:
- `200 OK` - Service ready for traffic
- `503 Service Unavailable` - Service not ready

### Metrics Endpoint

#### `/metrics`
**Purpose**: Prometheus metrics exposure
**Method**: GET
**Content-Type**: text/plain

**Response Format**:
```
# HELP events_processed_total Total number of events processed
# TYPE events_processed_total counter
events_processed_total{service="agenthub-broker",event_type="greeting",success="true"} 1234

# HELP system_cpu_usage_percent CPU usage percentage
# TYPE system_cpu_usage_percent gauge
system_cpu_usage_percent{service="agenthub-broker"} 23.4
```

**Status Codes**:
- `200 OK` - Metrics available
- `500 Internal Server Error` - Metrics collection failure

## Service-Specific Configurations

### Broker (Port 8080)

**Health Checks**:
- `self` - Basic service health
- `grpc_server` - gRPC server status
- `observability` - OpenTelemetry health

**Example URLs**:
- Health: http://localhost:8080/health
- Ready: http://localhost:8080/ready
- Metrics: http://localhost:8080/metrics

### Publisher (Port 8081)

**Health Checks**:
- `self` - Basic service health
- `broker_connection` - Connection to AgentHub broker
- `observability` - Tracing and metrics health

**Example URLs**:
- Health: http://localhost:8081/health
- Ready: http://localhost:8081/ready
- Metrics: http://localhost:8081/metrics

### Subscriber (Port 8082)

**Health Checks**:
- `self` - Basic service health
- `broker_connection` - Connection to AgentHub broker
- `task_processor` - Task processing capability
- `observability` - Observability stack health

**Example URLs**:
- Health: http://localhost:8082/health
- Ready: http://localhost:8082/ready
- Metrics: http://localhost:8082/metrics

### Custom Agents (Port 8083+)

**Configurable Health Checks**:
- Custom business logic checks
- External dependency checks
- Resource availability checks

## Health Check Types

### BasicHealthChecker

**Purpose**: Simple function-based health checks

**Implementation**:
```go
checker := observability.NewBasicHealthChecker("database", func(ctx context.Context) error {
    return db.Ping()
})
healthServer.AddChecker("database", checker)
```

**Use Cases**:
- Database connectivity
- File system access
- Configuration validation
- Memory/disk space checks

### GRPCHealthChecker

**Purpose**: gRPC connection health verification

**Implementation**:
```go
checker := observability.NewGRPCHealthChecker("broker_connection", "localhost:50051")
healthServer.AddChecker("broker_connection", checker)
```

**Use Cases**:
- AgentHub broker connectivity
- External gRPC service dependencies
- Service mesh health

### HTTPHealthChecker

**Purpose**: HTTP endpoint health verification

**Implementation**:
```go
checker := observability.NewHTTPHealthChecker("api_gateway", "http://gateway:8080/health")
healthServer.AddChecker("api_gateway", checker)
```

**Use Cases**:
- REST API dependencies
- Web service health
- Load balancer backends

### Custom Health Checkers

**Interface**:
```go
type HealthChecker interface {
    Check(ctx context.Context) error
    Name() string
}
```

**Custom Implementation Example**:
```go
type BusinessLogicChecker struct {
    name string
    validator func() error
}

func (c *BusinessLogicChecker) Check(ctx context.Context) error {
    return c.validator()
}

func (c *BusinessLogicChecker) Name() string {
    return c.name
}

// Usage
checker := &BusinessLogicChecker{
    name: "license_validation",
    validator: func() error {
        if time.Now().After(licenseExpiry) {
            return errors.New("license expired")
        }
        return nil
    },
}
```

## Health Check Configuration

### Check Intervals

**Default Intervals**:
- Active checks: Every 30 seconds
- On-demand checks: Per request
- Startup checks: During service initialization

**Configurable Timing**:
```go
config := observability.HealthConfig{
    CheckInterval: 15 * time.Second,
    Timeout:       5 * time.Second,
    RetryCount:    3,
    RetryDelay:    1 * time.Second,
}
```

### Timeout Configuration

**Per-Check Timeouts**:
```go
checker := observability.NewBasicHealthChecker("slow_service",
    func(ctx context.Context) error {
        // This check will timeout after 10 seconds
        return slowOperation(ctx)
    }).WithTimeout(10 * time.Second)
```

**Global Timeout**:
```go
healthServer := observability.NewHealthServer("8080", "my-service", "1.0.0")
healthServer.SetGlobalTimeout(30 * time.Second)
```

## Integration Patterns

### Kubernetes Integration

#### Liveness Probe
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

#### Readiness Probe
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

#### Startup Probe
```yaml
startupProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 30
```

### Load Balancer Integration

#### HAProxy Configuration
```
backend agentHub_brokers
    balance roundrobin
    option httpchk GET /health
    server broker1 broker1:8080 check
    server broker2 broker2:8080 check
```

#### NGINX Configuration
```nginx
upstream agenthub_backend {
    server broker1:8080;
    server broker2:8080;
}

location /health_check {
    proxy_pass http://agenthub_backend/health;
    proxy_set_header Host $host;
}
```

### Prometheus Integration

#### Service Discovery
```yaml
- job_name: 'agenthub-health'
  static_configs:
    - targets:
      - 'broker:8080'
      - 'publisher:8081'
      - 'subscriber:8082'
  metrics_path: '/metrics'
  scrape_interval: 10s
  scrape_timeout: 5s
```

#### Health Check Metrics
```promql
# Health check status (1=healthy, 0=unhealthy)
health_check_status{service="agenthub-broker",check="database"}

# Health check duration
health_check_duration_seconds{service="agenthub-broker",check="database"}

# Service uptime
service_uptime_seconds{service="agenthub-broker"}
```

## Status Definitions

### Service Status Levels

#### Healthy
**Definition**: All health checks passing
**HTTP Status**: 200 OK
**Criteria**:
- All registered checks return no error
- Service is fully operational
- All dependencies available

#### Degraded
**Definition**: Service operational but with limitations
**HTTP Status**: 200 OK (with warning indicators)
**Criteria**:
- Critical checks passing
- Non-critical checks may be failing
- Service can handle requests with reduced functionality

#### Unhealthy
**Definition**: Service cannot handle requests properly
**HTTP Status**: 503 Service Unavailable
**Criteria**:
- One or more critical checks failing
- Service should not receive new requests
- Requires intervention or automatic recovery

### Check-Level Status

#### Passing
- Check completed successfully
- No errors detected
- Within acceptable parameters

#### Warning
- Check completed with minor issues
- Service functional but attention needed
- May indicate future problems

#### Critical
- Check failed
- Service functionality compromised
- Immediate attention required

## Monitoring and Alerting

### Critical Alerts

```yaml
# Service down alert
- alert: ServiceHealthCheckFailing
  expr: health_check_status == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Service health check failing"
    description: "{{ $labels.service }} health check {{ $labels.check }} is failing"

# Service not ready alert
- alert: ServiceNotReady
  expr: up{job=~"agenthub-.*"} == 0
  for: 30s
  labels:
    severity: critical
  annotations:
    summary: "Service not responding"
    description: "{{ $labels.instance }} is not responding to health checks"
```

### Warning Alerts

```yaml
# Slow health checks
- alert: SlowHealthChecks
  expr: health_check_duration_seconds > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Health checks taking too long"
    description: "{{ $labels.service }} health check {{ $labels.check }} taking {{ $value }}s"

# Service degraded
- alert: ServiceDegraded
  expr: service_status == 1  # degraded status
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Service running in degraded mode"
    description: "{{ $labels.service }} is degraded but still operational"
```

## API Response Examples

### Healthy Service Response

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy",
  "timestamp": "2025-09-28T21:00:00.000Z",
  "service": "agenthub-broker",
  "version": "1.0.0",
  "uptime": "2h34m12s",
  "checks": [
    {
      "name": "self",
      "status": "healthy",
      "message": "Service is running normally",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "1.2ms"
    },
    {
      "name": "grpc_server",
      "status": "healthy",
      "message": "gRPC server listening on :50051",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "0.8ms"
    },
    {
      "name": "observability",
      "status": "healthy",
      "message": "OpenTelemetry exporter connected",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "12.4ms"
    }
  ]
}
```

### Unhealthy Service Response

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "unhealthy",
  "timestamp": "2025-09-28T21:00:00.000Z",
  "service": "agenthub-broker",
  "version": "1.0.0",
  "uptime": "2h34m12s",
  "checks": [
    {
      "name": "self",
      "status": "healthy",
      "message": "Service is running normally",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "1.2ms"
    },
    {
      "name": "grpc_server",
      "status": "unhealthy",
      "message": "Failed to bind to port :50051: address already in use",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "0.1ms"
    },
    {
      "name": "observability",
      "status": "healthy",
      "message": "OpenTelemetry exporter connected",
      "last_checked": "2025-09-28T21:00:00.000Z",
      "duration": "12.4ms"
    }
  ]
}
```

## Best Practices

### Health Check Design

1. **Fast Execution**: Keep checks under 5 seconds
2. **Meaningful Tests**: Test actual functionality, not just process existence
3. **Idempotent Operations**: Checks should not modify system state
4. **Appropriate Timeouts**: Set reasonable timeouts for external dependencies
5. **Clear Messages**: Provide actionable error messages

### Dependency Management

1. **Critical vs Non-Critical**: Distinguish between essential and optional dependencies
2. **Cascade Prevention**: Avoid cascading failures through dependency chains
3. **Circuit Breakers**: Implement circuit breakers for flaky dependencies
4. **Graceful Degradation**: Continue operating when non-critical dependencies fail

### Operational Considerations

1. **Monitoring**: Set up alerts for health check failures
2. **Documentation**: Document what each health check validates
3. **Testing**: Test health checks in development and staging
4. **Versioning**: Version health check APIs for compatibility

---

**🎯 Next Steps**:

**Implementation**: **[Add Observability to Your Agent](../howto/add_observability.md)**

**Monitoring**: **[Use Grafana Dashboards](../howto/use_dashboards.md)**

**Metrics**: **[Observability Metrics Reference](observability_metrics.md)**