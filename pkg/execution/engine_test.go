package execution

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zavictl/pkg/state/sqlite"
	"zavictl/pkg/workflow"
)

type mockRunner struct {
	runs int
}

func (m *mockRunner) Run(ctx context.Context, step workflow.StepDefinition) (StepResult, error) {
	m.runs++
	time.Sleep(10 * time.Millisecond)
	return StepResult{Success: true}, nil
}

func TestEngineLifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_exec.db")
	store, err := sqlite.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	runner := &mockRunner{}
	engine := NewEngine(store, runner, nil)

	wf := workflow.WorkflowDefinition{
		Name:    "test-wf",
		Version: "v1",
		Steps: []workflow.StepDefinition{
			{Name: "step1", Action: "test"},
		},
	}

	id, err := engine.Submit(context.Background(), wf)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	status, err := engine.Status(context.Background(), id)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status != StatusReceived {
		t.Errorf("expected Received, got %s", status)
	}

	time.Sleep(100 * time.Millisecond)

	status, _ = engine.Status(context.Background(), id)
	if status != StatusCompleted {
		t.Errorf("expected Completed, got %s", status)
	}

	if runner.runs != 1 {
		t.Errorf("expected 1 run, got %d", runner.runs)
	}
}

func TestEngineCancellation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_cancel.db")
	store, err := sqlite.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	runner := &mockRunner{}
	engine := NewEngine(store, runner, nil)

	wf := workflow.WorkflowDefinition{
		Name:    "test-cancel",
		Version: "v1",
		Steps: []workflow.StepDefinition{
			{Name: "step1", Action: "long-running"},
		},
	}

	id, err := engine.Submit(context.Background(), wf)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	err = engine.Cancel(context.Background(), id)
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	status, _ := engine.Status(context.Background(), id)
	if status != StatusCancelled {
		t.Errorf("expected Cancelled, got %s", status)
	}
}
