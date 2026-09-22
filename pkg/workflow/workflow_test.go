package workflow

import (
	"strings"
	"testing"
)

func TestValidate_ValidDAG(t *testing.T) {
	wf := &WorkflowDefinition{
		Name:    "deploy-service",
		Version: "v1",
		Steps: []StepDefinition{
			{Name: "validate", Action: "project.validate"},
			{Name: "build", Action: "artifact.build", DependsOn: []string{"validate"}},
			{Name: "security_scan", Action: "security.scan", DependsOn: []string{"build"}},
			{Name: "deploy", Action: "runtime.deploy", DependsOn: []string{"security_scan"}},
			{Name: "verify", Action: "health.check", DependsOn: []string{"deploy"}},
			{Name: "rollback", Action: "deployment.rollback", CompensationFor: []string{"deploy"}},
		},
	}

	err := Validate(wf)
	if err != nil {
		t.Fatalf("expected valid workflow, got error: %v", err)
	}
}

func TestValidate_InvalidVersion(t *testing.T) {
	wf := &WorkflowDefinition{
		Name:    "deploy",
		Version: "v2",
		Steps:   []StepDefinition{{Name: "step1"}},
	}
	err := Validate(wf)
	if err == nil || !strings.Contains(err.Error(), "unsupported workflow version") {
		t.Fatalf("expected unsupported version error, got: %v", err)
	}
}

func TestValidate_CycleDetection(t *testing.T) {
	wf := &WorkflowDefinition{
		Name:    "cycle-test",
		Version: "v1",
		Steps: []StepDefinition{
			{Name: "stepA", DependsOn: []string{"stepB"}},
			{Name: "stepB", DependsOn: []string{"stepC"}},
			{Name: "stepC", DependsOn: []string{"stepA"}},
		},
	}

	err := Validate(wf)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle detected message, got: %v", err)
	}
}

func TestValidate_MissingDependency(t *testing.T) {
	wf := &WorkflowDefinition{
		Name:    "missing-dep-test",
		Version: "v1",
		Steps: []StepDefinition{
			{Name: "step1", DependsOn: []string{"missing-step"}},
		},
	}

	err := Validate(wf)
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "depends on unknown step") {
		t.Fatalf("expected unknown step message, got: %v", err)
	}
}

func TestValidate_ExpressionDependency(t *testing.T) {
	wf := &WorkflowDefinition{
		Name:    "expr-test",
		Version: "v1",
		Steps: []StepDefinition{
			{Name: "producer", Action: "generate"},
			{
				Name: "consumer",
				Inputs: map[string]Expression{
					"data": "${{ steps.producer.outputs.data }}",
				},
			},
		},
	}

	err := Validate(wf)
	if err != nil {
		t.Fatalf("expected valid workflow, got error: %v", err)
	}

	wfCycle := &WorkflowDefinition{
		Name:    "expr-cycle",
		Version: "v1",
		Steps: []StepDefinition{
			{
				Name: "step1",
				Inputs: map[string]Expression{
					"val": "${{ steps.step2.outputs.x }}",
				},
			},
			{
				Name: "step2",
				Inputs: map[string]Expression{
					"val": "${{ steps.step1.outputs.y }}",
				},
			},
		},
	}
	
	err = Validate(wfCycle)
	if err == nil {
		t.Fatal("expected expression cycle detection error, got nil")
	}
}
