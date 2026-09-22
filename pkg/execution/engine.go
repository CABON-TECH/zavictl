package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zavictl/pkg/models"
	"zavictl/pkg/policy"
	"zavictl/pkg/provider"
	"zavictl/pkg/state"
	"zavictl/pkg/workflow"
)

type engineImpl struct {
	store  state.StateStore
	runner StepRunner
	policy policy.PolicyEngine
	
	mu     sync.RWMutex
	active map[ExecutionID]context.CancelFunc
}

// NewEngine creates a new execution engine orchestrator.
func NewEngine(store state.StateStore, runner StepRunner, pe policy.PolicyEngine) ExecutionEngine {
	return &engineImpl{
		store:  store,
		runner: runner,
		policy: pe,
		active: make(map[ExecutionID]context.CancelFunc),
	}
}

func (e *engineImpl) Submit(ctx context.Context, wf workflow.WorkflowDefinition) (ExecutionID, error) {
	id := ExecutionID(models.GenerateID())
	
	execCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.active[id] = cancel
	e.mu.Unlock()

	// 1. Received
	if err := e.transition(ctx, id, StatusReceived, "Workflow received"); err != nil {
		return id, err
	}

	go e.runWorkflow(execCtx, id, wf)

	return id, nil
}

func (e *engineImpl) Status(ctx context.Context, id ExecutionID) (ExecutionStatus, error) {
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", id))
	rec, err := e.store.Get(ctx, ref)
	if err != nil {
		if err == state.ErrNotFound {
			return "", fmt.Errorf("execution not found")
		}
		return "", err
	}
	
	statusStr, _ := rec.Data["status"].(string)
	return ExecutionStatus(statusStr), nil
}

func (e *engineImpl) Cancel(ctx context.Context, id ExecutionID) error {
	e.mu.Lock()
	cancel, exists := e.active[id]
	e.mu.Unlock()
	
	if exists {
		cancel()
	}
	
	return e.transition(ctx, id, StatusCancelled, "Workflow cancelled by user")
}

func (e *engineImpl) transition(ctx context.Context, id ExecutionID, status ExecutionStatus, msg string) error {
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", id))
	rec, err := e.store.Get(ctx, ref)
	if err != nil && err != state.ErrNotFound {
		return err
	}
	
	expectedVersion := rec.Version
	if err == state.ErrNotFound {
		rec = state.StateRecord{
			Version: 1,
			Data:    make(map[string]any),
		}
		expectedVersion = 0
	}
	
	rec.Data["status"] = string(status)
	rec.Data["last_message"] = msg
	rec.UpdatedAt = time.Now()
	
	return e.store.Put(ctx, ref, rec, expectedVersion)
}

func (e *engineImpl) runWorkflow(ctx context.Context, id ExecutionID, wf workflow.WorkflowDefinition) {
	defer func() {
		e.mu.Lock()
		delete(e.active, id)
		e.mu.Unlock()
	}()

	// 2. Authorized
	if e.policy != nil {
		op := provider.Operation{
			Action: "workflow.execute",
			Parameters: map[string]any{"workflow_name": wf.Name},
		}
		dec, err := e.policy.Evaluate(ctx, op, policy.Scope{Project: "default"})
		if err != nil || !dec.Allowed {
			_ = e.transition(ctx, id, StatusFailed, "Policy authorization failed")
			return
		}
	}
	_ = e.transition(ctx, id, StatusAuthorized, "Policy authorization passed")

	// 3. Planned
	if err := workflow.Validate(&wf); err != nil {
		_ = e.transition(ctx, id, StatusFailed, fmt.Sprintf("DAG validation failed: %v", err))
		return
	}
	_ = e.transition(ctx, id, StatusPlanned, "DAG planned successfully")

	// 4. Executing
	_ = e.transition(ctx, id, StatusExecuting, "Executing steps")
	
	for _, step := range wf.Steps {
		if ctx.Err() != nil {
			return 
		}
		
		res, err := e.runner.Run(ctx, step)
		if err != nil || !res.Success {
			_ = e.transition(ctx, id, StatusFailed, fmt.Sprintf("Step %s failed", step.Name))
			return
		}
	}

	// 5. Completed
	_ = e.transition(ctx, id, StatusCompleted, "Workflow completed successfully")
}
