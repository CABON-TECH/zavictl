package execution

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"zavictl/pkg/models"
	"zavictl/pkg/state/sqlite"
	"zavictl/pkg/workflow"
)

type mockRunner struct {
	mu sync.Mutex

	runs int
}

func (m *mockRunner) Run(ctx context.Context, step workflow.StepDefinition) (StepResult, error) {
	m.mu.Lock()
	m.runs++
	m.mu.Unlock()
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
		rec, _ := store.Get(context.Background(), models.ResourceRef("Execution/" + id)); t.Errorf("expected Completed, got %s. Record: %+v", status, rec.Data)
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

func TestEngineDAG(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_dag.db")
	store, _ := sqlite.NewSQLiteStore(dbPath)
	runner := &mockRunner{}
	engine := NewEngine(store, runner, nil)

	wf := workflow.WorkflowDefinition{
		Name:    "test-dag",
		Version: "v1",
		Steps: []workflow.StepDefinition{
			{Name: "stepA", Action: "test"},
			{Name: "stepB", Action: "test", DependsOn: []string{"stepA"}},
			{Name: "stepC", Action: "test", DependsOn: []string{"stepA"}},
			{Name: "stepD", Action: "test", DependsOn: []string{"stepB", "stepC"}},
		},
	}

	id, _ := engine.Submit(context.Background(), wf)
	
	time.Sleep(200 * time.Millisecond)

	status, _ := engine.Status(context.Background(), id)
	if status != StatusCompleted {
		t.Logf("Runner runs: %d", runner.runs); t.Errorf("expected Completed, got %s", status)
	}

	if runner.runs != 4 {
		t.Errorf("expected 4 runs, got %d", runner.runs)
	}
}

func TestEngineApproval(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_approval.db")
	store, _ := sqlite.NewSQLiteStore(dbPath)
	runner := &mockRunner{}
	engine := NewEngine(store, runner, nil)

	wf := workflow.WorkflowDefinition{
		Name:    "test-approval",
		Version: "v1",
		Steps: []workflow.StepDefinition{
			{Name: "step1", Action: "test", Approval: &workflow.ApprovalRequirement{Required: true}},
		},
	}

	id, _ := engine.Submit(context.Background(), wf)
	
	time.Sleep(50 * time.Millisecond)
	status, _ := engine.Status(context.Background(), id)
	if status != StatusExecuting {
		t.Errorf("expected Executing (blocked), got %s", status)
	}
	
	if runner.runs != 0 {
		t.Errorf("expected 0 runs before approval, got %d", runner.runs)
	}

	engine.SignalApproval(context.Background(), id, "step1")
	
	time.Sleep(50 * time.Millisecond)
	
	status, _ = engine.Status(context.Background(), id)
	if status != StatusCompleted {
		t.Logf("Runner runs: %d", runner.runs); t.Errorf("expected Completed, got %s", status)
	}

	if runner.runs != 1 {
		t.Errorf("expected 1 run after approval, got %d", runner.runs)
	}
}

func TestEngineResume(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_resume.db")
	store, _ := sqlite.NewSQLiteStore(dbPath)
	runner := &mockRunner{}
	engine := NewEngine(store, runner, nil)

	wf := workflow.WorkflowDefinition{
		Name:    "test-resume",
		Version: "v1",
		Steps: []workflow.StepDefinition{
			{Name: "stepA", Action: "test"},
			{Name: "stepB", Action: "test", DependsOn: []string{"stepA"}},
		},
	}

	id, _ := engine.Submit(context.Background(), wf)
	time.Sleep(10 * time.Millisecond) // Let step A start
	_ = engine.Cancel(context.Background(), id)
	time.Sleep(50 * time.Millisecond)
	
	// Create a new runner and engine to simulate process restart
	store2, _ := sqlite.NewSQLiteStore(dbPath)
	runner2 := &mockRunner{}
	engine2 := NewEngine(store2, runner2, nil)

	// Inject fake state for stepA as success to simulate it finished before crash
	ref := models.ResourceRef("Execution/" + id)
	rec, _ := store2.Get(context.Background(), ref)
	steps := map[string]any{"stepA": StepState{Status: "success"}}
	rec.Data["steps"] = steps
	rec.Data["status"] = string(StatusExecuting)
	_ = store2.Put(context.Background(), ref, rec, rec.Version)

	err := engine2.Resume(context.Background(), id)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	status, _ := engine2.Status(context.Background(), id)
	if status != StatusCompleted {
		t.Errorf("expected Completed, got %s", status)
	}

	if runner2.runs != 1 {
		t.Errorf("expected 1 run (stepB only), got %d", runner2.runs)
	}
}
