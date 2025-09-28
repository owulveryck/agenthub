package observability

import (
	"context"

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
