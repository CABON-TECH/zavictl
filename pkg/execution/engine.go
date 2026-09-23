package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"zavictl/pkg/models"
	"zavictl/pkg/policy"
	"zavictl/pkg/provider"
	"zavictl/pkg/state"
	"zavictl/pkg/workflow"
)

type StepState struct {
	Status  string
	Outputs map[string]any
	Error   string
}

type engineImpl struct {
	store  state.StateStore
	runner StepRunner
	policy policy.PolicyEngine
	
	mu       sync.RWMutex
	active   map[ExecutionID]context.CancelFunc
}

func NewEngine(store state.StateStore, runner StepRunner, pe policy.PolicyEngine) ExecutionEngine {
	return &engineImpl{
		store:   store,
		runner:  runner,
		policy:  pe,
		active:  make(map[ExecutionID]context.CancelFunc),
	}
}

func (e *engineImpl) Submit(ctx context.Context, wf workflow.WorkflowDefinition) (ExecutionID, error) {
	id := ExecutionID(models.GenerateID())
	
	execCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.active[id] = cancel
	e.mu.Unlock()

	if err := e.transition(ctx, id, StatusReceived, "Workflow received"); err != nil {
		return id, err
	}

	// Persist the workflow definition as part of the state for Resumability
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", id))
	rec, _ := e.store.Get(ctx, ref)
	rec.Data["workflow"] = wf
	e.store.Put(ctx, ref, rec, rec.Version)

	go e.runWorkflow(execCtx, id, wf)
	return id, nil
}

func (e *engineImpl) Resume(ctx context.Context, id ExecutionID) error {
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", id))
	rec, err := e.store.Get(ctx, ref)
	if err != nil {
		return err
	}

	statusStr, _ := rec.Data["status"].(string)
	if statusStr == string(StatusCompleted) || statusStr == string(StatusFailed) || statusStr == string(StatusCancelled) {
		return fmt.Errorf("workflow already in terminal state: %s", statusStr)
	}

	wfRaw, ok := rec.Data["workflow"]
	if !ok {
		return fmt.Errorf("workflow definition not found in state")
	}

	var wfData workflow.WorkflowDefinition
	b, err := json.Marshal(wfRaw)
	if err != nil {
		return fmt.Errorf("failed to encode workflow: %v", err)
	}
	if err := json.Unmarshal(b, &wfData); err != nil {
		return fmt.Errorf("failed to decode workflow: %v", err)
	}

	execCtx, cancel := context.WithCancel(context.Background())
	
	e.mu.Lock()
	if _, exists := e.active[id]; exists {
		e.mu.Unlock()
		cancel()
		return fmt.Errorf("workflow already running")
	}
	e.active[id] = cancel
	e.mu.Unlock()

	go e.runWorkflow(execCtx, id, wfData)
	return nil
}

func (e *engineImpl) SignalApproval(ctx context.Context, id ExecutionID, stepName string) error {
	st, exists := e.getStepState(ctx, id, stepName)
	if !exists || st.Status != "awaiting_approval" {
		return fmt.Errorf("no pending approval found for step %s", stepName)
	}

	e.saveStepState(ctx, id, stepName, StepState{Status: "approved"})
	return nil
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
	
	for i := 0; i < 5; i++ {
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
		
		err = e.store.Put(ctx, ref, rec, expectedVersion)
		if err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("failed to transition state after retries")
}

func (e *engineImpl) saveStepState(ctx context.Context, id ExecutionID, stepName string, stepState StepState) {
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", id))
	
	for i := 0; i < 5; i++ {
		rec, err := e.store.Get(ctx, ref)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		stepsRaw, ok := rec.Data["steps"]
		var steps map[string]any
		if ok {
			steps = stepsRaw.(map[string]any)
		} else {
			steps = make(map[string]any)
		}
		
		steps[stepName] = stepState
		rec.Data["steps"] = steps
		
		err = e.store.Put(ctx, ref, rec, rec.Version)
		if err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (e *engineImpl) getStepState(ctx context.Context, id ExecutionID, stepName string) (StepState, bool) {
	ref := models.ResourceRef(fmt.Sprintf("Execution/%s", id))
	rec, err := e.store.Get(ctx, ref)
	if err != nil {
		return StepState{}, false
	}
	
	stepsRaw, ok := rec.Data["steps"]
	if !ok {
		return StepState{}, false
	}
	
	steps, ok := stepsRaw.(map[string]any)
	if !ok {
		return StepState{}, false
	}
	
	stRaw, exists := steps[stepName]
	if !exists {
		return StepState{}, false
	}

	switch v := stRaw.(type) {
	case StepState:
		return v, true
	case map[string]any:
		s := StepState{}
		if stat, ok := v["Status"].(string); ok { s.Status = stat }
		if errStr, ok := v["Error"].(string); ok { s.Error = errStr }
		if out, ok := v["Outputs"].(map[string]any); ok { s.Outputs = out }
		return s, true
	}
	
	return StepState{}, false
}

func (e *engineImpl) runWorkflow(ctx context.Context, id ExecutionID, wf workflow.WorkflowDefinition) {
	defer func() {
		e.mu.Lock()
		delete(e.active, id)
		e.mu.Unlock()
	}()

	if e.policy != nil {
		// Build a representative operation using the first step's action
		// to allow rules to match on op.Action (e.g. "github.cicd.trigger_workflow")
		firstAction := "workflow.execute"
		if len(wf.Steps) > 0 {
			firstAction = wf.Steps[0].Action
		}
		op := provider.Operation{
			Action:     firstAction,
			Parameters: map[string]any{"workflow_name": wf.Name},
		}
		dec, err := e.policy.Evaluate(ctx, op, policy.Scope{Project: "default"})
		if err != nil || !dec.Allowed {
			msg := "Policy authorization failed"
			if len(dec.Violations) > 0 {
				msg = fmt.Sprintf("Policy blocked: %s", dec.Violations[0].Message)
			}
			_ = e.transition(ctx, id, StatusFailed, msg)
			return
		}
	}
	_ = e.transition(ctx, id, StatusAuthorized, "Policy authorization passed")

	if err := workflow.Validate(&wf); err != nil {
		_ = e.transition(ctx, id, StatusFailed, fmt.Sprintf("DAG validation failed: %v", err))
		return
	}
	_ = e.transition(ctx, id, StatusPlanned, "DAG planned successfully")

	_ = e.transition(ctx, id, StatusExecuting, "Executing steps")

	// Build dependency graph
	deps := make(map[string][]string)
	revDeps := make(map[string][]string)
	inDegree := make(map[string]int)
	stepsByName := make(map[string]workflow.StepDefinition)

	for _, step := range wf.Steps {
		stepsByName[step.Name] = step
		inDegree[step.Name] = len(step.DependsOn)
		deps[step.Name] = step.DependsOn
		for _, d := range step.DependsOn {
			revDeps[d] = append(revDeps[d], step.Name)
		}
	}

	// Adjust inDegree for already completed steps (resume functionality)
	for _, step := range wf.Steps {
		st, exists := e.getStepState(ctx, id, step.Name)
		if exists && st.Status == "success" {
			inDegree[step.Name] = -1 // mark completed
			for _, child := range revDeps[step.Name] {
				inDegree[child]--
			}
		}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(wf.Steps))
	readyCh := make(chan string, len(wf.Steps))

	for name, deg := range inDegree {
		if deg == 0 {
			readyCh <- name
		}
	}

	var mu sync.Mutex
	running := 0
	completed := 0
	target := len(wf.Steps)

	// Subtract already completed steps from target
	for _, deg := range inDegree {
		if deg == -1 {
			target--
		}
	}

	if target == 0 {
		_ = e.transition(ctx, id, StatusCompleted, "Workflow completed successfully")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				break
			}
			_ = e.transition(ctx, id, StatusFailed, fmt.Sprintf("Workflow failed, starting rollback: %v", err))
			e.rollback(ctx, id, wf)
			return
		case name := <-readyCh:
			running++
			wg.Add(1)
			go func(stepName string) {
				defer wg.Done()
				
				step := stepsByName[stepName]
				
				if step.Approval != nil && step.Approval.Required {
					e.saveStepState(ctx, id, stepName, StepState{Status: "awaiting_approval"})
					
					for {
						if ctx.Err() != nil {
							return
						}
						st, exists := e.getStepState(ctx, id, stepName)
						if exists && st.Status == "approved" {
							break
						}
						time.Sleep(1 * time.Second)
					}
				}
				
				e.saveStepState(ctx, id, stepName, StepState{Status: "running"})
				
				res, err := e.runner.Run(ctx, step)
				if err != nil || !res.Success {
					var errStr string
					if err != nil {
						errStr = err.Error()
					} else if res.Error != nil {
						errStr = res.Error.Error()
					} else {
						errStr = "unknown failure"
					}
					
					e.saveStepState(ctx, id, stepName, StepState{Status: "failed", Error: errStr})
					errCh <- fmt.Errorf("step %s failed: %s", stepName, errStr)
					return
				}
				
				e.saveStepState(ctx, id, stepName, StepState{Status: "success", Outputs: res.Outputs})
				
				mu.Lock()
				completed++
				for _, child := range revDeps[stepName] {
					inDegree[child]--
					if inDegree[child] == 0 {
						readyCh <- child
					}
				}
				
				isDone := completed == target
				mu.Unlock()

				if isDone {
					close(errCh)
				}
			}(name)
		}

		mu.Lock()
		isDone := completed == target
		mu.Unlock()
		if isDone {
			break
		}
	}

	wg.Wait()

	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			return
		}
	default:
	}

	_ = e.transition(ctx, id, StatusCompleted, "Workflow completed successfully")
}

func (e *engineImpl) rollback(ctx context.Context, id ExecutionID, wf workflow.WorkflowDefinition) {
	// Reverse topological sort
	// Find all completed steps that have a CompensationFor field matching their name
	var compensationSteps []workflow.StepDefinition
	
	// Quick mapping for compensations
	comps := make(map[string]workflow.StepDefinition)
	for _, step := range wf.Steps {
		for _, target := range step.CompensationFor {
			comps[target] = step
		}
	}

	// For any completed step, if it has a compensation, run it
	for _, step := range wf.Steps {
		st, exists := e.getStepState(ctx, id, step.Name)
		if exists && st.Status == "success" {
			if comp, hasComp := comps[step.Name]; hasComp {
				compensationSteps = append(compensationSteps, comp)
			}
		}
	}
	
	// Execute compensations in parallel for simplicity (ideally reverse graph order)
	var wg sync.WaitGroup
	for _, comp := range compensationSteps {
		wg.Add(1)
		go func(c workflow.StepDefinition) {
			defer wg.Done()
			_, _ = e.runner.Run(context.Background(), c)
		}(comp)
	}
	wg.Wait()
}
