// Package config provides centralized configuration management for AgentHub services
// through environment variables with sensible defaults.
//
// # Overview
//
// The config package loads application configuration from environment variables,
// providing a single source of truth for all AgentHub services including:
//   - Broker connection settings
//   - Observability stack endpoints (Jaeger, Prometheus, Grafana)
//   - Health check ports for each service
//   - OpenTelemetry Collector configuration
//   - Service metadata (name, version, environment)
//
// All configuration values have sensible defaults, so services can run without
// any environment variable configuration.
//
// # Quick Start
//
// Load configuration in your service:
//
//	config := config.Load()
//	fmt.Printf("Broker: %s\n", config.GetBrokerAddress())
//	fmt.Printf("Jaeger: %s\n", config.JaegerEndpoint)
//	fmt.Printf("Environment: %s\n", config.Environment)
//
// # Configuration Fields
//
// **Broker Configuration**:
//   - AGENTHUB_BROKER_ADDR: Broker hostname (default: "localhost")
//   - AGENTHUB_BROKER_PORT: Broker port (default: "50051")
//
// **Observability Stack**:
//   - JAEGER_ENDPOINT: Jaeger OTLP endpoint (default: "127.0.0.1:4317")
//   - PROMETHEUS_PORT: Prometheus port (default: "9090")
//   - GRAFANA_PORT: Grafana port (default: "3333")
//   - ALERTMANAGER_PORT: AlertManager port (default: "9093")
//
// **Health Check Ports**:
//   - BROKER_HEALTH_PORT: Broker health endpoint (default: "8080")
//   - PUBLISHER_HEALTH_PORT: Publisher health endpoint (default: "8081")
//   - SUBSCRIBER_HEALTH_PORT: Subscriber health endpoint (default: "8082")
//
// **OpenTelemetry Collector**:
//   - OTLP_GRPC_PORT: OTLP gRPC receiver port (default: "4320")
//   - OTLP_HTTP_PORT: OTLP HTTP receiver port (default: "4321")
//
// **Service Metadata**:
//   - SERVICE_NAME: Service name for observability (default: "agenthub-service")
//   - SERVICE_VERSION: Service version (default: "1.0.0")
//   - ENVIRONMENT: Deployment environment (default: "development")
//   - LOG_LEVEL: Logging level - DEBUG, INFO, WARN, ERROR (default: "INFO")
//
// # Usage Examples
//
// **Basic Configuration**:
//
//	config := config.Load()
//	brokerAddr := config.GetBrokerAddress()  // "localhost:50051"
//
// **Custom Environment**:
//
//	// Set environment variables
//	os.Setenv("AGENTHUB_BROKER_ADDR", "broker.prod.example.com")
//	os.Setenv("AGENTHUB_BROKER_PORT", "443")
//	os.Setenv("ENVIRONMENT", "production")
//	os.Setenv("LOG_LEVEL", "WARN")
//
//	config := config.Load()
//	// Uses production values
//
// **Service-Specific Health Ports**:
//
//	config := config.Load()
//	brokerPort := config.GetHealthPort("broker")      // "8080"
//	publisherPort := config.GetHealthPort("publisher")  // "8081"
//	subscriberPort := config.GetHealthPort("subscriber") // "8082"
//
// **Observability URLs**:
//
//	config := config.Load()
//	jaegerUI := config.GetJaegerWebURL()     // "http://localhost:16686"
//	grafana := config.GetGrafanaURL()        // "http://localhost:3333"
//	prometheus := config.GetPrometheusURL()  // "http://localhost:9090"
//	alertMgr := config.GetAlertManagerURL()  // "http://localhost:9093"
//
// # Configuration Precedence
//
// Configuration is loaded in this order:
//  1. Environment variables (if set)
//  2. Default values (if not set)
//
// # Development vs Production
//
// **Development (defaults)**:
//
//	ENVIRONMENT=development
//	AGENTHUB_BROKER_ADDR=localhost
//	LOG_LEVEL=INFO
//
// **Production (recommended)**:
//
//	ENVIRONMENT=production
//	AGENTHUB_BROKER_ADDR=broker.prod.internal
//	LOG_LEVEL=WARN
//	SERVICE_VERSION=1.2.3
//
// # Integration with Other Packages
//
// The config package is used by:
//
// **observability.DefaultConfig()**:
//
//	func DefaultConfig(serviceName string) observability.Config {
//	    appConfig := config.Load()
//	    return observability.Config{
//	        ServiceName:    serviceName,
//	        ServiceVersion: appConfig.ServiceVersion,
//	        JaegerEndpoint: appConfig.JaegerEndpoint,
//	        // ...
//	    }
//	}
//
// **agenthub.NewGRPCConfig()**:
//
//	// Uses AGENTHUB_BROKER_ADDR and AGENTHUB_BROKER_PORT from environment
//	config := agenthub.NewGRPCConfig("my_service")
//
// # Docker Compose Integration
//
// When running with docker-compose.yml, environment variables are typically
// defined in the compose file or .env file:
//
//	services:
//	  broker:
//	    environment:
//	      - AGENTHUB_BROKER_ADDR=0.0.0.0
//	      - AGENTHUB_BROKER_PORT=50051
//	      - JAEGER_ENDPOINT=jaeger:4317
//	      - ENVIRONMENT=staging
//
// # Best Practices
//
// **Use Load() once per service**:
//
//	// In main.go
//	config := config.Load()
//	// Pass to components that need it
//
// **Don't mutate AppConfig**:
//
//	// AppConfig is a read-only snapshot of environment at startup
//	config := config.Load()
//	// Don't modify config fields after loading
//
// **Use helper methods**:
//
//	addr := config.GetBrokerAddress()  // Prefer this
//	// Over: addr := config.BrokerAddr + ":" + config.BrokerPort
//
// # Thread Safety
//
// AppConfig is safe to read from multiple goroutines once loaded.
// Do not modify AppConfig fields after calling Load().
package config
