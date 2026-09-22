package provider

import (
	"context"
	"errors"
	"time"

	"golang.org/x/time/rate"
)

// ManagedConnection wraps a raw Connection to provide rate limiting, backoff, and error translation.
type ManagedConnection struct {
	raw        Connection
	limiter    *rate.Limiter
	maxRetries int
}

// NewManagedConnection creates a wrapped connection that enforces rate limits and exponential backoff.
func NewManagedConnection(raw Connection, requestsPerSecond float64, maxRetries int) *ManagedConnection {
	return &ManagedConnection{
		raw:        raw,
		limiter:    rate.NewLimiter(rate.Limit(requestsPerSecond), 1), // burst of 1
		maxRetries: maxRetries,
	}
}

func (m *ManagedConnection) Execute(ctx context.Context, op Operation) (Result, error) {
	var lastErr error
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		if err := m.limiter.Wait(ctx); err != nil {
			return Result{}, wrapError(err, op, false)
		}

		res, err := m.raw.Execute(ctx, op)
		if err == nil {
			return res, nil
		}

		var platErr *PlatformError
		isRetryable := false
		if errors.As(err, &platErr) {
			isRetryable = platErr.IsRetryable
		} else {
			platErr = wrapError(err, op, false)
		}

		if !isRetryable || attempt == m.maxRetries {
			lastErr = platErr
			break
		}

		select {
		case <-ctx.Done():
			return Result{}, wrapError(ctx.Err(), op, false)
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return Result{}, lastErr
}

func (m *ManagedConnection) Close() error {
	return m.raw.Close()
}

func wrapError(err error, op Operation, retryable bool) *PlatformError {
	var platErr *PlatformError
	if errors.As(err, &platErr) {
		return platErr
	}

	return &PlatformError{
		ErrorCode:        "PROVIDER_EXEC_ERR",
		Message:          "provider execution failed",
		UnderlyingCause:  err,
		Operation:        op.Action,
		IsRetryable:      retryable,
		RecoveryGuidance: "Check provider configuration and connection status",
	}
}
