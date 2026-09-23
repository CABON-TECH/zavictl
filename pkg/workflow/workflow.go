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
	Required    bool   `yaml:"required" json:"required"`
	Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// RetryPolicy defines how a step handles transient failures.
type RetryPolicy struct {
	MaxAttempts int           `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	Backoff     time.Duration `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// StepDefinition represents a single node in the workflow execution graph.
type StepDefinition struct {
	Name            string                `yaml:"name" json:"name"`
	Action          string                `yaml:"action" json:"action"`
	DependsOn       []string              `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Inputs          map[string]Expression `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Approval        *ApprovalRequirement  `yaml:"approval,omitempty" json:"approval,omitempty"`
	Timeout         time.Duration         `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries         RetryPolicy           `yaml:"retries,omitempty" json:"retries,omitempty"`
	CompensationFor []string              `yaml:"compensation_for,omitempty" json:"compensation_for,omitempty"`
}

// WorkflowDefinition is the complete parsed execution plan.
type WorkflowDefinition struct {
	Name    string           `yaml:"name" json:"name"`
	Version string           `yaml:"version" json:"version"`
	Steps   []StepDefinition `yaml:"steps" json:"steps"`
}
