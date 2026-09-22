package provider

import (
	"context"
	"fmt"

	"zavictl/pkg/credentials"
)

// Capability describes what a provider can do.
type Capability struct {
	Domain     string   // e.g. "sourcecontrol", "infrastructure", "runtime"
	Operations []string // e.g. "plan", "apply", "destroy"
}

// Provider represents an external tool integration.
type Provider interface {
	Name() string
	Version() string
	Capabilities() []Capability
	Connect(ctx context.Context, creds credentials.CredentialRef) (Connection, error)
	HealthCheck(ctx context.Context, conn Connection) (HealthStatus, error)
}

// Operation describes an action to be executed by a Connection.
type Operation struct {
	Domain         string
	Action         string
	IdempotencyKey string
	Parameters     map[string]any
}

// Result describes the outcome of an Operation.
type Result struct {
	Status  string
	Outputs map[string]any
}

// HealthStatus represents the health of a provider connection.
type HealthStatus struct {
	Status  string // e.g. "healthy", "unhealthy"
	Message string
}

// Connection represents an authenticated session with a provider.
type Connection interface {
	Execute(ctx context.Context, op Operation) (Result, error)
	Close() error
}

// PlatformError represents the standard error model per Section 22.
// It ensures provider-specific errors do not leak past the Connection boundary.
type PlatformError struct {
	ErrorCode        string
	Message          string
	UnderlyingCause  error
	Resource         string
	Operation        string
	CorrelationID    string
	RecoveryGuidance string
	IsRetryable      bool
}

func (e *PlatformError) Error() string {
	if e.UnderlyingCause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.ErrorCode, e.Message, e.UnderlyingCause)
	}
	return fmt.Sprintf("[%s] %s", e.ErrorCode, e.Message)
}
