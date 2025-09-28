package observability

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/owulveryck/agenthub/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	JaegerEndpoint string
	PrometheusPort string
	Environment    string
	LogLevel       string
}

type Observability struct {
	Config   Config
	Tracer   trace.Tracer
	Meter    metric.Meter
	Logger   *slog.Logger
	Handler  *ObservabilityHandler
	shutdown func(context.Context) error
}

func NewObservability(config Config) (*Observability, error) {
	ctx := context.Background()

	// Set up OpenTelemetry error handler with service context
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Printf("[%s] OpenTelemetry error (OTLP endpoint: %s): %v",
			config.ServiceName, config.JaegerEndpoint, err)
	}))

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Setup tracing with OTLP exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(config.JaegerEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithTimeout(time.Second*10), // Add explicit timeout
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: time.Second,
			MaxInterval:     time.Second * 5,
			MaxElapsedTime:  time.Second * 30,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter for service %s (endpoint: %s): %w", config.ServiceName, config.JaegerEndpoint, err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tracerProvider)

	// Configure text map propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := otel.Tracer(config.ServiceName)

	// Setup metrics
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExporter),
	)

	otel.SetMeterProvider(meterProvider)
	meter := otel.Meter(config.ServiceName)

	// Parse log level
	var logLevel slog.Level
	switch strings.ToUpper(config.LogLevel) {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN", "WARNING":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create observability handler with log level
	handlerOpts := HandlerOptions{
		Level: logLevel,
	}

	// If DEBUG level, also log to stdout
	var logger *slog.Logger
	if logLevel == slog.LevelDebug {
		// Create a multi-writer: observability handler + stdout
		stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})

		handler, err := NewObservabilityHandlerWithOptions(tracer, meter, config.ServiceName, handlerOpts)
		if err != nil {
			return nil, err
		}

		// Create a combined handler that writes to both
		logger = slog.New(&CombinedHandler{
			handlers: []slog.Handler{handler, stdoutHandler},
		})
	} else {
		// Only use observability handler
		handler, err := NewObservabilityHandlerWithOptions(tracer, meter, config.ServiceName, handlerOpts)
		if err != nil {
			return nil, err
		}
		logger = slog.New(handler)
	}

	obs := &Observability{
		Config:  config,
		Tracer:  tracer,
		Meter:   meter,
		Logger:  logger,
		Handler: nil, // Will be set below
		shutdown: func(ctx context.Context) error {
			if err := tracerProvider.Shutdown(ctx); err != nil {
				return fmt.Errorf("failed to shutdown trace provider for service %s (OTLP endpoint: %s): %w", config.ServiceName, config.JaegerEndpoint, err)
			}
			if err := meterProvider.Shutdown(ctx); err != nil {
				return fmt.Errorf("failed to shutdown meter provider for service %s: %w", config.ServiceName, err)
			}
			return nil
		},
	}

	return obs, nil
}

func (o *Observability) Shutdown(ctx context.Context) error {
	return o.shutdown(ctx)
}

func DefaultConfig(serviceName string) Config {
	appConfig := config.Load()
	return Config{
		ServiceName:    serviceName,
		ServiceVersion: appConfig.ServiceVersion,
		JaegerEndpoint: appConfig.JaegerEndpoint,
		PrometheusPort: appConfig.PrometheusPort,
		Environment:    appConfig.Environment,
		LogLevel:       appConfig.LogLevel,
	}
}

// CombinedHandler implements slog.Handler and forwards to multiple handlers
type CombinedHandler struct {
	handlers []slog.Handler
}

func (h *CombinedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *CombinedHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record); err != nil {
				// Continue to other handlers even if one fails
				continue
			}
		}
	}
	return nil
}

func (h *CombinedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &CombinedHandler{handlers: newHandlers}
}

func (h *CombinedHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &CombinedHandler{handlers: newHandlers}
}
