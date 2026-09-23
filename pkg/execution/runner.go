package execution

import (
	"context"
	"fmt"
	"strings"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
	"zavictl/pkg/workflow"
)

// AdapterRunner routes workflow steps to the corresponding registered provider adapter.
type AdapterRunner struct {
	registry *provider.Registry
	resolver credentials.CredentialResolver
}

// NewAdapterRunner creates a new AdapterRunner.
func NewAdapterRunner(reg *provider.Registry, res credentials.CredentialResolver) StepRunner {
	return &AdapterRunner{
		registry: reg,
		resolver: res,
	}
}

// Run executes the step by dispatching it to the correct provider.
func (r *AdapterRunner) Run(ctx context.Context, step workflow.StepDefinition) (StepResult, error) {
	// e.g. "docker.runtime.start" -> provider="docker", operation="runtime.start"
	parts := strings.SplitN(step.Action, ".", 2)
	
	// Special internal handling for manual approvals
	if step.Action == "manual.approve" {
		return StepResult{Success: true}, nil
	}

	if len(parts) < 2 {
		return StepResult{Success: false, Error: fmt.Errorf("invalid action format: %s", step.Action)}, nil
	}

	providerName := parts[0]
	actionName := parts[1]

	p, err := r.registry.Get(providerName)
	if err != nil {
		return StepResult{Success: false, Error: err}, nil
	}

	// In a real system, the credential ref would be parameterized per environment/workflow
	credRef := credentials.CredentialRef{ID: "default", Provider: providerName}

	conn, err := p.Connect(ctx, credRef)
	if err != nil {
		return StepResult{Success: false, Error: fmt.Errorf("failed to connect to provider %s: %v", providerName, err)}, nil
	}
	defer conn.Close()

	// Convert Expressions to actual interface{} for parameters
	params := make(map[string]any)
	for k, v := range step.Inputs {
		params[k] = string(v) // Minimal expression resolving for now
	}

	op := provider.Operation{
		Action:         actionName,
		IdempotencyKey: fmt.Sprintf("%s-%s", step.Name, actionName),
		Parameters:     params,
	}

	res, err := conn.Execute(ctx, op)
	if err != nil {
		return StepResult{Success: false, Error: err}, nil
	}

	if res.Status == "Failed" {
		return StepResult{Success: false, Error: fmt.Errorf("provider execution failed"), Outputs: res.Outputs}, nil
	}

	return StepResult{Success: true, Outputs: res.Outputs}, nil
}
