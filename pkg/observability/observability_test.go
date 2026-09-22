package observability

import (
	"context"
	"sync"
	"testing"
	"time"

	"zavictl/pkg/events"
)

type mockAuditLogger struct {
	mu     sync.Mutex
	events []events.Event
}

func (m *mockAuditLogger) LogEvent(ctx context.Context, event events.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

type mockMetricsClient struct {
	mu    sync.Mutex
	calls int
}

func (m *mockMetricsClient) RecordExecutionTime(workflow string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
}

func (m *mockMetricsClient) RecordStepStatus(step string, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
}

func TestAuditLoggerRedaction(t *testing.T) {
	bus := events.NewAsyncEventBus()
	logger := &mockAuditLogger{}

	_, err := StartAuditSubscription(bus, logger)
	if err != nil {
		t.Fatalf("failed to start subscription: %v", err)
	}

	ev := events.Event{
		Type: "TestEvent",
		Metadata: map[string]any{
			"public_info": "some-value",
			"secret_token": "my-super-secret",
		},
	}

	err = bus.Publish(context.Background(), ev)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	logger.mu.Lock()
	defer logger.mu.Unlock()

	if len(logger.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(logger.events))
	}

	received := logger.events[0]
	if received.Metadata["secret_token"] != "[REDACTED]" {
		t.Errorf("expected secret_token to be redacted, got %s", received.Metadata["secret_token"])
	}
	if received.Metadata["public_info"] == "[REDACTED]" {
		t.Errorf("expected public_info not to be redacted")
	}
}

func TestAsyncMetricsClient(t *testing.T) {
	mockClient := &mockMetricsClient{}
	asyncClient := NewAsyncMetricsClient(mockClient)

	asyncClient.RecordExecutionTime("wf", 100*time.Millisecond)
	asyncClient.RecordStepStatus("step1", "success")

	time.Sleep(50 * time.Millisecond)

	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()
	
	if mockClient.calls != 2 {
		t.Errorf("expected 2 metrics calls, got %d", mockClient.calls)
	}
}
