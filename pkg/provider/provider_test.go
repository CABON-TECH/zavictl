package provider

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockConnection struct {
	mu           sync.Mutex
	executeCount int
	errToReturn  error
}

func (m *mockConnection) Execute(ctx context.Context, op Operation) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCount++
	
	if m.errToReturn != nil {
		return Result{}, m.errToReturn
	}

	return Result{Status: "success"}, nil
}

func (m *mockConnection) Close() error { return nil }

func TestManagedConnection_Success(t *testing.T) {
	raw := &mockConnection{}
	managed := NewManagedConnection(raw, 1000, 3)

	res, err := managed.Execute(context.Background(), Operation{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "success" {
		t.Errorf("expected success, got %s", res.Status)
	}
	if raw.executeCount != 1 {
		t.Errorf("expected 1 execution, got %d", raw.executeCount)
	}
}

func TestManagedConnection_RetryableError(t *testing.T) {
	retryableErr := &PlatformError{
		ErrorCode:   "RATE_LIMIT",
		IsRetryable: true,
	}
	
	raw := &mockConnection{errToReturn: retryableErr}
	managed := NewManagedConnection(raw, 1000, 2)

	start := time.Now()
	_, err := managed.Execute(context.Background(), Operation{})
	duration := time.Since(start)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	
	var platErr *PlatformError
	if !errors.As(err, &platErr) {
		t.Fatalf("expected PlatformError, got %T", err)
	}
	if platErr.ErrorCode != "RATE_LIMIT" {
		t.Errorf("expected RATE_LIMIT, got %s", platErr.ErrorCode)
	}
	
	if raw.executeCount != 3 {
		t.Errorf("expected 3 executions, got %d", raw.executeCount)
	}

	if duration < 250*time.Millisecond {
		t.Errorf("expected duration to be at least ~300ms, got %v", duration)
	}
}

func TestManagedConnection_NonRetryableError(t *testing.T) {
	nonRetryableErr := errors.New("raw provider error")
	
	raw := &mockConnection{errToReturn: nonRetryableErr}
	managed := NewManagedConnection(raw, 1000, 3)

	_, err := managed.Execute(context.Background(), Operation{})
	
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	
	var platErr *PlatformError
	if !errors.As(err, &platErr) {
		t.Fatalf("expected PlatformError, got %T", err)
	}
	
	if platErr.ErrorCode != "PROVIDER_EXEC_ERR" {
		t.Errorf("expected wrapped PROVIDER_EXEC_ERR, got %s", platErr.ErrorCode)
	}
	
	if raw.executeCount != 1 {
		t.Errorf("expected 1 execution (no retries), got %d", raw.executeCount)
	}
}
