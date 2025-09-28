package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

type HealthCheck struct {
	Name        string       `json:"name"`
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message,omitempty"`
	LastChecked time.Time    `json:"last_checked"`
	Duration    string       `json:"duration"`
}

type HealthResponse struct {
	Status  HealthStatus  `json:"status"`
	Checks  []HealthCheck `json:"checks"`
	Version string        `json:"version"`
	Uptime  string        `json:"uptime"`
}

type HealthChecker interface {
	Check(ctx context.Context) HealthCheck
}

type HealthServer struct {
	port        string
	serviceName string
	version     string
	startTime   time.Time
	checkers    map[string]HealthChecker
	server      *http.Server
}

func NewHealthServer(port, serviceName, version string) *HealthServer {
	return &HealthServer{
		port:        port,
		serviceName: serviceName,
		version:     version,
		startTime:   time.Now(),
		checkers:    make(map[string]HealthChecker),
	}
}

func (hs *HealthServer) AddChecker(name string, checker HealthChecker) {
	hs.checkers[name] = checker
}

func (hs *HealthServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", hs.healthHandler)

	// Ready endpoint (same as health for now)
	mux.HandleFunc("/ready", hs.readyHandler)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	hs.server = &http.Server{
		Addr:    ":" + hs.port,
		Handler: mux,
	}

	return hs.server.ListenAndServe()
}

func (hs *HealthServer) Shutdown(ctx context.Context) error {
	if hs.server != nil {
		return hs.server.Shutdown(ctx)
	}
	return nil
}

func (hs *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	response := HealthResponse{
		Status:  HealthStatusHealthy,
		Version: hs.version,
		Uptime:  time.Since(hs.startTime).String(),
		Checks:  make([]HealthCheck, 0, len(hs.checkers)),
	}

	// Run all health checks
	for _, checker := range hs.checkers {
		check := checker.Check(ctx)
		response.Checks = append(response.Checks, check)

		// If any check fails, mark overall status as unhealthy
		if check.Status != HealthStatusHealthy {
			response.Status = HealthStatusUnhealthy
		}
	}

	// Set response status code
	statusCode := http.StatusOK
	if response.Status != HealthStatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (hs *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	// For now, ready is the same as health
	// In production, you might want different checks for readiness vs liveness
	hs.healthHandler(w, r)
}

// Basic health checker implementations
type BasicHealthChecker struct {
	name    string
	checkFn func(ctx context.Context) error
}

func NewBasicHealthChecker(name string, checkFn func(ctx context.Context) error) *BasicHealthChecker {
	return &BasicHealthChecker{
		name:    name,
		checkFn: checkFn,
	}
}

func (bhc *BasicHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        bhc.name,
		LastChecked: start,
	}

	if err := bhc.checkFn(ctx); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = err.Error()
	} else {
		check.Status = HealthStatusHealthy
	}

	check.Duration = time.Since(start).String()
	return check
}

// gRPC connection health checker
type GRPCHealthChecker struct {
	checkerName string
	endpoint    string
}

func NewGRPCHealthChecker(name, endpoint string) *GRPCHealthChecker {
	return &GRPCHealthChecker{
		checkerName: name,
		endpoint:    endpoint,
	}
}

func (ghc *GRPCHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()

	check := HealthCheck{
		Name:        ghc.checkerName,
		LastChecked: start,
		Status:      HealthStatusHealthy, // Simplified - would implement actual gRPC health check
	}

	check.Duration = time.Since(start).String()
	return check
}
