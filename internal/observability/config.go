package observability

import (
	"context"
	"log/slog"

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
	)
	if err != nil {
		return nil, err
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

	// Create observability handler
	handler, err := NewObservabilityHandler(tracer, meter, config.ServiceName)
	if err != nil {
		return nil, err
	}

	// Create logger with observability handler
	logger := slog.New(handler)

	obs := &Observability{
		Config:  config,
		Tracer:  tracer,
		Meter:   meter,
		Logger:  logger,
		Handler: handler,
		shutdown: func(ctx context.Context) error {
			if err := tracerProvider.Shutdown(ctx); err != nil {
				return err
			}
			return meterProvider.Shutdown(ctx)
		},
	}

	return obs, nil
}

func (o *Observability) Shutdown(ctx context.Context) error {
	return o.shutdown(ctx)
}

func DefaultConfig(serviceName string) Config {
	return Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		JaegerEndpoint: "127.0.0.1:4317",
		PrometheusPort: "9090",
		Environment:    "development",
	}
}
