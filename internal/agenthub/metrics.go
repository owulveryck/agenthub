package agenthub

import (
	"context"
	"time"

	"github.com/owulveryck/agenthub/internal/observability"
)

// MetricsTicker handles periodic system metrics collection
type MetricsTicker struct {
	ctx            context.Context
	metricsManager *observability.MetricsManager
	ticker         *time.Ticker
	done           chan struct{}
}

// NewMetricsTicker creates a new metrics ticker
func NewMetricsTicker(ctx context.Context, metricsManager *observability.MetricsManager) *MetricsTicker {
	return &MetricsTicker{
		ctx:            ctx,
		metricsManager: metricsManager,
		ticker:         time.NewTicker(30 * time.Second),
		done:           make(chan struct{}),
	}
}

// Start begins the metrics collection
func (m *MetricsTicker) Start() {
	go func() {
		defer m.ticker.Stop()
		for {
			select {
			case <-m.ticker.C:
				m.metricsManager.UpdateSystemMetrics(m.ctx)
			case <-m.ctx.Done():
				return
			case <-m.done:
				return
			}
		}
	}()
}

// Stop stops the metrics collection
func (m *MetricsTicker) Stop() {
	close(m.done)
}
