package observability

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MetricsManager struct {
	meter metric.Meter

	// Event metrics
	eventsProcessedTotal    metric.Int64Counter
	eventProcessingDuration metric.Float64Histogram
	eventErrorsTotal        metric.Int64Counter
	eventsPublishedTotal    metric.Int64Counter

	// System metrics
	processCPUSecondsTotal     metric.Float64Counter
	processResidentMemoryBytes metric.Int64UpDownCounter
	goGoroutines               metric.Int64UpDownCounter
	goMemstatsAllocBytes       metric.Int64UpDownCounter

	// Message broker metrics
	messageBrokerPublishDuration  metric.Float64Histogram
	messageBrokerConsumeDuration  metric.Float64Histogram
	messageBrokerConnectionErrors metric.Int64Counter
}

func NewMetricsManager(meter metric.Meter) (*MetricsManager, error) {
	mm := &MetricsManager{meter: meter}

	var err error

	// Event metrics
	mm.eventsProcessedTotal, err = meter.Int64Counter(
		"events_processed_total",
		metric.WithDescription("Total number of events processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	mm.eventProcessingDuration, err = meter.Float64Histogram(
		"event_processing_duration_seconds",
		metric.WithDescription("Event processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	mm.eventErrorsTotal, err = meter.Int64Counter(
		"event_errors_total",
		metric.WithDescription("Total number of event processing errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	mm.eventsPublishedTotal, err = meter.Int64Counter(
		"events_published_total",
		metric.WithDescription("Total number of events published"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// System metrics
	mm.processCPUSecondsTotal, err = meter.Float64Counter(
		"process_cpu_seconds_total",
		metric.WithDescription("Total user and system CPU time spent in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	mm.processResidentMemoryBytes, err = meter.Int64UpDownCounter(
		"process_resident_memory_bytes",
		metric.WithDescription("Resident memory size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	mm.goGoroutines, err = meter.Int64UpDownCounter(
		"go_goroutines",
		metric.WithDescription("Number of goroutines that currently exist"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	mm.goMemstatsAllocBytes, err = meter.Int64UpDownCounter(
		"go_memstats_alloc_bytes",
		metric.WithDescription("Number of bytes allocated and still in use"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	// Message broker metrics
	mm.messageBrokerPublishDuration, err = meter.Float64Histogram(
		"message_broker_publish_duration_seconds",
		metric.WithDescription("Message broker publish duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	mm.messageBrokerConsumeDuration, err = meter.Float64Histogram(
		"message_broker_consume_duration_seconds",
		metric.WithDescription("Message broker consume duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	mm.messageBrokerConnectionErrors, err = meter.Int64Counter(
		"message_broker_connection_errors_total",
		metric.WithDescription("Total number of message broker connection errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return mm, nil
}

// Event metrics methods
func (mm *MetricsManager) IncrementEventsProcessed(ctx context.Context, eventType, source string, success bool) {
	mm.eventsProcessedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("source", source),
		attribute.Bool("success", success),
	))
}

func (mm *MetricsManager) RecordEventProcessingDuration(ctx context.Context, eventType, source string, duration time.Duration) {
	mm.eventProcessingDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("source", source),
	))
}

func (mm *MetricsManager) IncrementEventErrors(ctx context.Context, eventType, source, errorType string) {
	mm.eventErrorsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("source", source),
		attribute.String("error", errorType),
	))
}

func (mm *MetricsManager) IncrementEventsPublished(ctx context.Context, eventType, destination string) {
	mm.eventsPublishedTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("destination", destination),
	))
}

// System metrics methods
func (mm *MetricsManager) UpdateSystemMetrics(ctx context.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mm.goGoroutines.Add(ctx, int64(runtime.NumGoroutine()))
	mm.goMemstatsAllocBytes.Add(ctx, int64(m.Alloc))
	mm.processResidentMemoryBytes.Add(ctx, int64(m.Sys))
}

// Message broker metrics methods
func (mm *MetricsManager) RecordBrokerPublishDuration(ctx context.Context, topic string, duration time.Duration) {
	mm.messageBrokerPublishDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("topic", topic),
	))
}

func (mm *MetricsManager) RecordBrokerConsumeDuration(ctx context.Context, topic string, duration time.Duration) {
	mm.messageBrokerConsumeDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("topic", topic),
	))
}

func (mm *MetricsManager) IncrementBrokerConnectionErrors(ctx context.Context) {
	mm.messageBrokerConnectionErrors.Add(ctx, 1)
}

// Helper method to start timing an operation
func (mm *MetricsManager) StartTimer() func(ctx context.Context, eventType, source string) {
	start := time.Now()
	return func(ctx context.Context, eventType, source string) {
		duration := time.Since(start)
		mm.RecordEventProcessingDuration(ctx, eventType, source, duration)
	}
}
