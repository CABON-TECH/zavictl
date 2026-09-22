package workflow

import (
	"time"
)

// SupportedVersions lists the workflow schema versions we can process.
var SupportedVersions = []string{"v1"}

// Expression represents a string that might contain expression syntax (e.g. ${{ steps.build.outputs.image }}).
type Expression string

// ApprovalRequirement models an external gate that must be cleared before a step executes.
type ApprovalRequirement struct {
	Required    bool
	Environment string
}

// RetryPolicy defines how a step handles transient failures.
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

// StepDefinition represents a single node in the workflow execution graph.
type StepDefinition struct {
	Name            string
	Action          string
	DependsOn       []string
	Inputs          map[string]Expression
	Approval        *ApprovalRequirement
	Timeout         time.Duration
	Retries         RetryPolicy
	CompensationFor []string
}

// WorkflowDefinition is the complete parsed execution plan.
type WorkflowDefinition struct {
	Name    string
	Version string
	Steps   []StepDefinition
}
