package terraform

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"zavictl/pkg/credentials"
	"zavictl/pkg/provider"
)

type terraformConnection struct {
	cred credentials.Credential
}

func (c *terraformConnection) Execute(ctx context.Context, op provider.Operation) (provider.Result, error) {
	switch op.Action {
	case "infrastructure.init":
		return c.runCommand(ctx, op, "init", "-no-color")
	case "infrastructure.plan":
		return c.runCommand(ctx, op, "plan", "-no-color", "-out=tfplan")
	case "infrastructure.apply":
		return c.runCommand(ctx, op, "apply", "-no-color", "-auto-approve")
	case "infrastructure.destroy":
		return c.runCommand(ctx, op, "destroy", "-no-color", "-auto-approve")
	default:
		return provider.Result{}, &provider.PlatformError{
			ErrorCode: "UNSUPPORTED_OPERATION",
			Message:   fmt.Sprintf("Terraform provider does not support action: %s", op.Action),
		}
	}
}

func (c *terraformConnection) Close() error {
	return nil
}

func (c *terraformConnection) checkBinary(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "terraform", "--version")
	return cmd.Run()
}

func (c *terraformConnection) runCommand(ctx context.Context, op provider.Operation, args ...string) (provider.Result, error) {
	dir, ok := op.Parameters["directory"].(string)
	if !ok || dir == "" {
		dir = "."
	}

	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = dir

	cmd.Env = os.Environ()
	
	if c.cred != nil {
		req, _ := http.NewRequest("GET", "http://dummy", nil)
		providerReq := &credentials.ProviderRequest{Header: req.Header}
		
		err := c.cred.Apply(providerReq)
		if err != nil {
			return provider.Result{}, &provider.PlatformError{
				ErrorCode: "CREDENTIAL_APPLY_FAILED",
				Message:   "Failed to apply credentials for Terraform execution",
				UnderlyingCause: err,
			}
		}

		for k, v := range req.Header {
			if len(v) > 0 {
				envKey := strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, v[0]))
			}
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return provider.Result{}, mapExecutionError(err, stderr.String())
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
		},
	}, nil
}
