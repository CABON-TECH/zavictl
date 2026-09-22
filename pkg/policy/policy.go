package policy

import (
	"context"

	"zavictl/pkg/provider"
)

// Scope represents the contextual boundaries for a policy evaluation.
type Scope struct {
	Environment string
	Project     string
}

// ExceptionRef references an approved override for a policy.
type ExceptionRef struct {
	ID        string
	ExpiresAt int64
}

// Violation represents a failed policy rule.
type Violation struct {
	Rule     string
	Message  string
	Severity string // "block" or "warn"
}

// Decision represents the outcome of a policy evaluation.
type Decision struct {
	Allowed       bool
	Violations    []Violation
	ExceptionUsed *ExceptionRef
}

// PolicyEngine is the interface for evaluating operations against policies.
type PolicyEngine interface {
	Evaluate(ctx context.Context, op provider.Operation, scope Scope) (Decision, error)
}
