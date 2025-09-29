package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type TraceManager struct {
	tracer trace.Tracer
}

func NewTraceManager(serviceName string) *TraceManager {
	return &TraceManager{
		tracer: otel.Tracer(serviceName),
	}
}

func (tm *TraceManager) StartSpan(ctx context.Context, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, operationName, trace.WithAttributes(attrs...))
}

func (tm *TraceManager) InjectTraceContext(ctx context.Context, headers map[string]string) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))
}

func (tm *TraceManager) ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
}

func (tm *TraceManager) StartEventProcessingSpan(ctx context.Context, eventID, eventType, source, subject string) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, "process_event", trace.WithAttributes(
		attribute.String("event.id", eventID),
		attribute.String("event.type", eventType),
		attribute.String("event.source", source),
		attribute.String("event.subject", subject),
	))
}

func (tm *TraceManager) StartPublishSpan(ctx context.Context, destination, eventType string) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, "publish_event", trace.WithAttributes(
		attribute.String("messaging.system", "grpc"),
		attribute.String("messaging.destination", destination),
		attribute.String("messaging.operation", "publish"),
		attribute.String("event.type", eventType),
	))
}

func (tm *TraceManager) StartConsumeSpan(ctx context.Context, source, eventType string) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, "consume_event", trace.WithAttributes(
		attribute.String("messaging.system", "grpc"),
		attribute.String("messaging.source", source),
		attribute.String("messaging.operation", "receive"),
		attribute.String("event.type", eventType),
	))
}

func (tm *TraceManager) RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(1, err.Error()) // Error status
	}
}

func (tm *TraceManager) SetSpanSuccess(span trace.Span) {
	span.SetStatus(2, "") // OK status
}

// AddTaskAttributes adds rich task information to a span
func (tm *TraceManager) AddTaskAttributes(span trace.Span, taskID, taskType string, parameters map[string]interface{}) {
	span.SetAttributes(
		attribute.String("task.id", taskID),
		attribute.String("task.type", taskType),
	)

	// Add task parameters as span attributes
	for key, value := range parameters {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String("task.param."+key, v))
		case float64:
			span.SetAttributes(attribute.Float64("task.param."+key, v))
		case int:
			span.SetAttributes(attribute.Int("task.param."+key, v))
		case bool:
			span.SetAttributes(attribute.Bool("task.param."+key, v))
		default:
			span.SetAttributes(attribute.String("task.param."+key, fmt.Sprintf("%v", v)))
		}
	}
}

// AddTaskResult adds task execution result to a span
func (tm *TraceManager) AddTaskResult(span trace.Span, status string, result map[string]interface{}, errorMessage string) {
	span.SetAttributes(attribute.String("task.status", status))

	if errorMessage != "" {
		span.SetAttributes(attribute.String("task.error", errorMessage))
	}

	// Add result data as span attributes
	for key, value := range result {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String("task.result."+key, v))
		case float64:
			span.SetAttributes(attribute.Float64("task.result."+key, v))
		case int:
			span.SetAttributes(attribute.Int("task.result."+key, v))
		case bool:
			span.SetAttributes(attribute.Bool("task.result."+key, v))
		default:
			span.SetAttributes(attribute.String("task.result."+key, fmt.Sprintf("%v", v)))
		}
	}
}

// AddSpanEvent adds a timestamped event to a span for tracking processing steps
func (tm *TraceManager) AddSpanEvent(span trace.Span, eventName string, attributes ...attribute.KeyValue) {
	span.AddEvent(eventName, trace.WithAttributes(attributes...))
}

// AddComponentAttribute adds a component identifier to a span
func (tm *TraceManager) AddComponentAttribute(span trace.Span, component string) {
	span.SetAttributes(attribute.String("agenthub.component", component))
}
