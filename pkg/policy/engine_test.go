package policy

import (
	"context"
	"testing"
	"time"

	"zavictl/pkg/events"
	"zavictl/pkg/provider"
)

func TestCELEngine_WarnViolation(t *testing.T) {
	policies := []PolicyDefinition{
		{
			Name: "test-policy",
			Rules: []PolicyRule{
				{
					Name:        "require-region",
					Description: "Region should be us-east-1",
					Expression:  `op.Parameters.region == "us-east-1"`,
					Severity:    "warn",
				},
			},
		},
	}

	engine, err := NewCELEngine(nil, policies, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	op := provider.Operation{
		Parameters: map[string]any{"region": "eu-west-1"},
	}

	dec, err := engine.Evaluate(context.Background(), op, Scope{})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}

	if !dec.Allowed {
		t.Errorf("expected allowed to be true (since it's a warn)")
	}
	if len(dec.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(dec.Violations))
	}
	if dec.Violations[0].Severity != "warn" {
		t.Errorf("expected warn severity")
	}
}

func TestCELEngine_BlockViolation(t *testing.T) {
	policies := []PolicyDefinition{
		{
			Name: "test-policy",
			Rules: []PolicyRule{
				{
					Name:        "deny-prod",
					Description: "Cannot deploy to prod without approval",
					Expression:  `scope.Environment != "prod"`,
					Severity:    "block",
				},
			},
		},
	}

	engine, err := NewCELEngine(nil, policies, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	scope := Scope{Environment: "prod"}

	dec, err := engine.Evaluate(context.Background(), provider.Operation{}, scope)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}

	if dec.Allowed {
		t.Errorf("expected allowed to be false")
	}
}

func TestCELEngine_ExceptionDowngrade(t *testing.T) {
	policies := []PolicyDefinition{
		{
			Name: "test-policy",
			Rules: []PolicyRule{
				{
					Name:        "deny-prod",
					Description: "Cannot deploy to prod",
					Expression:  `scope.Environment != "prod"`,
					Severity:    "block",
				},
			},
		},
	}

	exceptions := []ExceptionRef{
		{
			ID:        "EX-123",
			ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		},
	}

	bus := events.NewAsyncEventBus()
	engine, err := NewCELEngine(bus, policies, exceptions)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	scope := Scope{Environment: "prod"}

	dec, err := engine.Evaluate(context.Background(), provider.Operation{}, scope)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}

	if !dec.Allowed {
		t.Errorf("expected allowed to be true because of exception")
	}
	if dec.ExceptionUsed == nil {
		t.Fatalf("expected exception to be used")
	}
	if dec.ExceptionUsed.ID != "EX-123" {
		t.Errorf("expected EX-123, got %s", dec.ExceptionUsed.ID)
	}
}
