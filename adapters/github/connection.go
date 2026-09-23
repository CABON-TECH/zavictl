package github

import (
	"context"
	"fmt"

	"zavictl/pkg/provider"

	githubapi "github.com/google/go-github/v64/github"
)

type githubConnection struct {
	client *githubapi.Client
}

func (c *githubConnection) Execute(ctx context.Context, op provider.Operation) (provider.Result, error) {
	switch op.Action {
	case "sourcecontrol.read_repository":
		return c.readRepository(ctx, op)
	case "cicd.trigger_workflow":
		return c.triggerWorkflow(ctx, op)
	default:
		return provider.Result{}, &provider.PlatformError{
			ErrorCode: "UNSUPPORTED_OPERATION",
			Message:   fmt.Sprintf("GitHub provider does not support action: %s", op.Action),
		}
	}
}

func (c *githubConnection) Close() error {
	return nil
}

func (c *githubConnection) readRepository(ctx context.Context, op provider.Operation) (provider.Result, error) {
	owner, ok := op.Parameters["owner"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'owner'")
	}
	repo, ok := op.Parameters["repo"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'repo'")
	}

	r, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"id":          r.GetID(),
			"name":        r.GetName(),
			"full_name":   r.GetFullName(),
			"description": r.GetDescription(),
			"url":         r.GetHTMLURL(),
			"private":     r.GetPrivate(),
		},
	}, nil
}

func (c *githubConnection) triggerWorkflow(ctx context.Context, op provider.Operation) (provider.Result, error) {
	owner, ok := op.Parameters["owner"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'owner'")
	}
	repo, ok := op.Parameters["repo"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'repo'")
	}
	workflowID, ok := op.Parameters["workflow_id"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'workflow_id'")
	}
	ref, ok := op.Parameters["ref"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'ref'")
	}

	event := githubapi.CreateWorkflowDispatchEventRequest{
		Ref: ref,
	}
	
	if inputs, ok := op.Parameters["inputs"].(map[string]interface{}); ok {
		event.Inputs = inputs
	}

	_, err := c.client.Actions.CreateWorkflowDispatchEventByFileName(ctx, owner, repo, workflowID, event)
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"message": "Workflow dispatched successfully",
		},
	}, nil
}
