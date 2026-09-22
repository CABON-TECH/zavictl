package observability

import (
	"context"
	"time"

	"zavictl/pkg/events"
)

// AuditLogger represents the sink for all platform audit events.
type AuditLogger interface {
	LogEvent(ctx context.Context, event events.Event) error
}

// MetricsClient is the interface for pushing operational metrics.
type MetricsClient interface {
	RecordExecutionTime(workflow string, d time.Duration)
	RecordStepStatus(step string, status string)
}

// AsyncMetricsClient wraps a base MetricsClient and ensures all calls are asynchronous and non-blocking.
type AsyncMetricsClient struct {
	base MetricsClient
}

// NewAsyncMetricsClient creates a non-blocking wrapper around a metrics client.
func NewAsyncMetricsClient(base MetricsClient) *AsyncMetricsClient {
	return &AsyncMetricsClient{base: base}
}

func (a *AsyncMetricsClient) RecordExecutionTime(workflow string, d time.Duration) {
	go func() {
		a.base.RecordExecutionTime(workflow, d)
	}()
}

func (a *AsyncMetricsClient) RecordStepStatus(step string, status string) {
	go func() {
		a.base.RecordStepStatus(step, status)
	}()
}
