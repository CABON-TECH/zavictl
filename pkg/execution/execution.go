package execution

import (
	"context"

	"zavictl/pkg/workflow"
)

type ExecutionID string

type ExecutionStatus string

const (
	StatusReceived   ExecutionStatus = "Received"
	StatusAuthorized ExecutionStatus = "Authorized"
	StatusPlanned    ExecutionStatus = "Planned"
	StatusExecuting  ExecutionStatus = "Executing"
	StatusCompleted  ExecutionStatus = "Completed"
	StatusFailed     ExecutionStatus = "Failed"
	StatusCancelled  ExecutionStatus = "Cancelled"
)

// ExecutionEngine handles workflow lifecycle and scheduling.
type ExecutionEngine interface {
	Submit(ctx context.Context, wf workflow.WorkflowDefinition) (ExecutionID, error)
	Cancel(ctx context.Context, id ExecutionID) error
	Status(ctx context.Context, id ExecutionID) (ExecutionStatus, error)
}

// StepResult represents the outcome of executing a step.
type StepResult struct {
	Success bool
	Outputs map[string]any
	Error   error
}

// StepRunner executes an individual workflow step in isolation.
type StepRunner interface {
	Run(ctx context.Context, step workflow.StepDefinition) (StepResult, error)
}
