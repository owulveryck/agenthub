package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type ObservabilityHandler struct {
	opts        HandlerOptions
	tracer      trace.Tracer
	meter       metric.Meter
	serviceName string

	// Metrics
	eventCounter  metric.Int64Counter
	eventDuration metric.Float64Histogram
	eventErrors   metric.Int64Counter
	logCounter    metric.Int64Counter

	// Event posting
	postEvent func(event EventData) error

	// Buffering
	buffer   chan logEntry
	mu       sync.RWMutex
	shutdown chan struct{}
	wg       sync.WaitGroup
}

type HandlerOptions struct {
	Level       slog.Level
	Writer      io.Writer
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
	BufferSize  int
}

type logEntry struct {
	time  time.Time
	level slog.Level
	msg   string
	attrs []slog.Attr
	ctx   context.Context
}

type EventData struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Source  string            `json:"source"`
	Subject string            `json:"subject"`
	Time    time.Time         `json:"time"`
	Data    interface{}       `json:"data"`
	Headers map[string]string `json:"headers"`
	TraceID string            `json:"trace_id"`
	SpanID  string            `json:"span_id"`
}

func NewObservabilityHandler(tracer trace.Tracer, meter metric.Meter, serviceName string) (*ObservabilityHandler, error) {
	return NewObservabilityHandlerWithOptions(tracer, meter, serviceName, HandlerOptions{
		Level:      slog.LevelInfo,
		BufferSize: 1000,
	})
}

func NewObservabilityHandlerWithOptions(tracer trace.Tracer, meter metric.Meter, serviceName string, opts HandlerOptions) (*ObservabilityHandler, error) {
	if opts.BufferSize <= 0 {
		opts.BufferSize = 1000
	}

	// Initialize metrics
	eventCounter, err := meter.Int64Counter(
		"events_processed_total",
		metric.WithDescription("Total number of events processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	eventDuration, err := meter.Float64Histogram(
		"event_processing_duration_seconds",
		metric.WithDescription("Event processing duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	eventErrors, err := meter.Int64Counter(
		"event_errors_total",
		metric.WithDescription("Total number of event processing errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	logCounter, err := meter.Int64Counter(
		"logs_total",
		metric.WithDescription("Total number of log entries"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	h := &ObservabilityHandler{
		opts:          opts,
		tracer:        tracer,
		meter:         meter,
		serviceName:   serviceName,
		eventCounter:  eventCounter,
		eventDuration: eventDuration,
		eventErrors:   eventErrors,
		logCounter:    logCounter,
		buffer:        make(chan logEntry, opts.BufferSize),
		shutdown:      make(chan struct{}),
	}

	// Start background processor
	h.wg.Add(1)
	go h.processLogs()

	return h, nil
}

func (h *ObservabilityHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

func (h *ObservabilityHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	// Extract attributes
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	// Add trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add service information
	attrs = append(attrs,
		slog.String("service", h.serviceName),
		slog.String("source", getSource()),
	)

	entry := logEntry{
		time:  r.Time,
		level: r.Level,
		msg:   r.Message,
		attrs: attrs,
		ctx:   ctx,
	}

	// Non-blocking send to buffer
	select {
	case h.buffer <- entry:
	default:
		// Buffer full, drop the log entry to prevent blocking
		h.eventErrors.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error", "log_buffer_full"),
			attribute.String("service", h.serviceName),
		))
	}

	return nil
}

func (h *ObservabilityHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, return a new handler with the same configuration
	// In a production implementation, you'd want to preserve the attributes
	newHandler, _ := NewObservabilityHandlerWithOptions(h.tracer, h.meter, h.serviceName, h.opts)
	return newHandler
}

func (h *ObservabilityHandler) WithGroup(name string) slog.Handler {
	// For simplicity, return the same handler
	// In a production implementation, you'd want to handle grouping
	return h
}

func (h *ObservabilityHandler) processLogs() {
	defer h.wg.Done()

	for {
		select {
		case entry := <-h.buffer:
			h.processLogEntry(entry)
		case <-h.shutdown:
			// Process remaining logs
			for {
				select {
				case entry := <-h.buffer:
					h.processLogEntry(entry)
				default:
					return
				}
			}
		}
	}
}

func (h *ObservabilityHandler) processLogEntry(entry logEntry) {
	// Update metrics
	h.logCounter.Add(entry.ctx, 1, metric.WithAttributes(
		attribute.String("level", entry.level.String()),
		attribute.String("service", h.serviceName),
	))

	// Convert to structured format for output
	logData := map[string]interface{}{
		"time":    entry.time.Format(time.RFC3339),
		"level":   entry.level.String(),
		"msg":     entry.msg,
		"service": h.serviceName,
	}

	// Add attributes
	for _, attr := range entry.attrs {
		logData[attr.Key] = attr.Value.Any()
	}

	// Output to writer if configured
	if h.opts.Writer != nil {
		// Simple JSON-like output for demonstration
		fmt.Fprintf(h.opts.Writer, "%v\n", logData)
	}

	// Post event if handler is configured
	if h.postEvent != nil {
		event := EventData{
			ID:      fmt.Sprintf("log_%d", time.Now().UnixNano()),
			Type:    "log.entry",
			Source:  h.serviceName,
			Subject: entry.msg,
			Time:    entry.time,
			Data:    logData,
			Headers: make(map[string]string),
		}

		// Add trace context to headers
		for _, attr := range entry.attrs {
			if attr.Key == "trace_id" || attr.Key == "span_id" {
				event.Headers[attr.Key] = attr.Value.String()
			}
		}

		go func() {
			if err := h.postEvent(event); err != nil {
				h.eventErrors.Add(context.Background(), 1, metric.WithAttributes(
					attribute.String("error", "post_event_failed"),
					attribute.String("service", h.serviceName),
				))
			}
		}()
	}
}

func (h *ObservabilityHandler) SetEventPoster(poster func(event EventData) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.postEvent = poster
}

func (h *ObservabilityHandler) IncrementEventCounter(ctx context.Context, eventType, source string, success bool) {
	h.eventCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("source", source),
		attribute.Bool("success", success),
	))
}

func (h *ObservabilityHandler) RecordEventDuration(ctx context.Context, eventType, source string, duration time.Duration) {
	h.eventDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("source", source),
	))
}

func (h *ObservabilityHandler) RecordEventError(ctx context.Context, eventType, source, errorType string) {
	h.eventErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("source", source),
		attribute.String("error", errorType),
	))
}

func (h *ObservabilityHandler) Shutdown(ctx context.Context) error {
	close(h.shutdown)

	// Wait for background processor to finish
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func getSource() string {
	_, file, line, ok := runtime.Caller(4) // Adjust caller depth as needed
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
