package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"zavictl/pkg/events"
	"zavictl/pkg/models"
	"zavictl/pkg/provider"
)

// PolicyRule defines a single rule in a policy.
type PolicyRule struct {
	Name        string
	Description string
	Expression  string // CEL expression
	Severity    string // "block" or "warn"
}

// PolicyDefinition defines a collection of rules for a domain.
type PolicyDefinition struct {
	Name  string
	Rules []PolicyRule
}

// CELEngine implements PolicyEngine using CEL.
type CELEngine struct {
	bus        events.EventBus
	policies   []PolicyDefinition
	exceptions []ExceptionRef
	env        *cel.Env
}

// NewCELEngine creates a new CEL-backed policy engine.
func NewCELEngine(bus events.EventBus, policies []PolicyDefinition, exceptions []ExceptionRef) (*CELEngine, error) {
	env, err := cel.NewEnv(
		cel.Variable("op", cel.MapType(cel.StringType, cel.AnyType)),
		cel.Variable("scope", cel.MapType(cel.StringType, cel.StringType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL env: %w", err)
	}

	return &CELEngine{
		bus:        bus,
		policies:   policies,
		exceptions: exceptions,
		env:        env,
	}, nil
}

func (e *CELEngine) Evaluate(ctx context.Context, op provider.Operation, scope Scope) (Decision, error) {
	decision := Decision{
		Allowed: true,
	}

	opVars := map[string]any{
		"Domain":         op.Domain,
		"Action":         op.Action,
		"IdempotencyKey": op.IdempotencyKey,
		"Parameters":     op.Parameters,
	}
	scopeVars := map[string]string{
		"Environment": scope.Environment,
		"Project":     scope.Project,
	}
	
	vars := map[string]any{
		"op":    opVars,
		"scope": scopeVars,
	}

	hasBlock := false

	for _, policy := range e.policies {
		for _, rule := range policy.Rules {
			ast, iss := e.env.Compile(rule.Expression)
			if iss.Err() != nil {
				return decision, fmt.Errorf("compile error in rule '%s': %w", rule.Name, iss.Err())
			}
			
			prg, err := e.env.Program(ast)
			if err != nil {
				return decision, fmt.Errorf("program error in rule '%s': %w", rule.Name, err)
			}
			
			out, _, err := prg.Eval(vars)
			if err != nil {
				return decision, fmt.Errorf("eval error in rule '%s': %w", rule.Name, err)
			}
			
			passed, ok := out.Value().(bool)
			if !ok {
				return decision, fmt.Errorf("rule '%s' did not return a boolean", rule.Name)
			}
			
			if !passed {
				decision.Violations = append(decision.Violations, Violation{
					Rule:     rule.Name,
					Message:  rule.Description,
					Severity: rule.Severity,
				})
				
				if rule.Severity == "block" {
					hasBlock = true
				}
			}
		}
	}

	if hasBlock {
		decision.Allowed = false
		now := time.Now().Unix()
		for _, ex := range e.exceptions {
			if ex.ExpiresAt > now {
				decision.Allowed = true
				exCopy := ex
				decision.ExceptionUsed = &exCopy
				break
			}
		}
	}

	if !decision.Allowed || decision.ExceptionUsed != nil || len(decision.Violations) > 0 {
		e.publishAudit(ctx, op, scope, decision)
	}

	return decision, nil
}

func (e *CELEngine) publishAudit(ctx context.Context, op provider.Operation, scope Scope, dec Decision) {
	if e.bus == nil {
		return
	}
	
	meta := map[string]any{
		"allowed":     dec.Allowed,
		"action":      op.Action,
		"environment": scope.Environment,
	}
	if dec.ExceptionUsed != nil {
		meta["exception_id"] = dec.ExceptionUsed.ID
	}
	
	ev := events.Event{
		ID:        models.GenerateID(),
		Type:      "PolicyEvaluation",
		Timestamp: time.Now(),
		Metadata:  meta,
	}
	_ = e.bus.Publish(ctx, ev)
}
