package config

import (
	"os"
	"strconv"
)

// AppConfig holds all application configuration
type AppConfig struct {
	// AgentHub Core Configuration
	BrokerAddr string
	BrokerPort string

	// Observability Configuration
	JaegerEndpoint   string
	PrometheusPort   string
	GrafanaPort      string
	AlertManagerPort string

	// Health Check Ports
	BrokerHealthPort     string
	PublisherHealthPort  string
	SubscriberHealthPort string

	// OpenTelemetry Collector Ports
	OTLPGRPCPort string
	OTLPHTTPPort string

	// Service Configuration
	ServiceName    string
	ServiceVersion string
	Environment    string
	LogLevel       string
}

// Load loads configuration from environment variables with defaults
func Load() *AppConfig {
	return &AppConfig{
		// AgentHub Core
		BrokerAddr: getEnv("AGENTHUB_BROKER_ADDR", "localhost"),
		BrokerPort: getEnv("AGENTHUB_BROKER_PORT", "50051"),

		// Observability Stack
		JaegerEndpoint:   getEnv("JAEGER_ENDPOINT", "127.0.0.1:4317"),
		PrometheusPort:   getEnv("PROMETHEUS_PORT", "9090"),
		GrafanaPort:      getEnv("GRAFANA_PORT", "3333"),
		AlertManagerPort: getEnv("ALERTMANAGER_PORT", "9093"),

		// Health Check Ports
		BrokerHealthPort:     getEnv("BROKER_HEALTH_PORT", "8080"),
		PublisherHealthPort:  getEnv("PUBLISHER_HEALTH_PORT", "8081"),
		SubscriberHealthPort: getEnv("SUBSCRIBER_HEALTH_PORT", "8082"),

		// OpenTelemetry Collector Ports
		OTLPGRPCPort: getEnv("OTLP_GRPC_PORT", "4320"),
		OTLPHTTPPort: getEnv("OTLP_HTTP_PORT", "4321"),

		// Service Configuration
		ServiceName:    getEnv("SERVICE_NAME", "agenthub-service"),
		ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		LogLevel:       getEnv("LOG_LEVEL", "INFO"),
	}
}

// GetBrokerAddress returns the full broker address
func (c *AppConfig) GetBrokerAddress() string {
	return c.BrokerAddr + ":" + c.BrokerPort
}

// GetHealthPort returns the health port for a given service type
func (c *AppConfig) GetHealthPort(serviceType string) string {
	switch serviceType {
	case "broker":
		return c.BrokerHealthPort
	case "publisher":
		return c.PublisherHealthPort
	case "subscriber":
		return c.SubscriberHealthPort
	default:
		return "8080"
	}
}

// GetJaegerWebURL returns the Jaeger web interface URL
func (c *AppConfig) GetJaegerWebURL() string {
	return "http://localhost:16686"
}

// GetGrafanaURL returns the Grafana web interface URL
func (c *AppConfig) GetGrafanaURL() string {
	return "http://localhost:" + c.GrafanaPort
}

// GetPrometheusURL returns the Prometheus web interface URL
func (c *AppConfig) GetPrometheusURL() string {
	return "http://localhost:" + c.PrometheusPort
}

// GetAlertManagerURL returns the AlertManager web interface URL
func (c *AppConfig) GetAlertManagerURL() string {
	return "http://localhost:" + c.AlertManagerPort
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer with a default fallback
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as boolean with a default fallback
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
