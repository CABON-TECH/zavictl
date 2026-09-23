package docker

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

type dockerConnection struct {
	cred credentials.Credential
}

func (c *dockerConnection) Execute(ctx context.Context, op provider.Operation) (provider.Result, error) {
	switch op.Action {
	case "runtime.list":
		return c.runCommand(ctx, op, "ps", "-a", "--format", "{{json .}}")
	case "runtime.start":
		id, ok := op.Parameters["container_id"].(string)
		if !ok {
			return provider.Result{}, fmt.Errorf("missing parameter 'container_id'")
		}
		return c.runCommand(ctx, op, "start", id)
	case "runtime.stop":
		id, ok := op.Parameters["container_id"].(string)
		if !ok {
			return provider.Result{}, fmt.Errorf("missing parameter 'container_id'")
		}
		return c.runCommand(ctx, op, "stop", id)
	case "runtime.inspect":
		id, ok := op.Parameters["container_id"].(string)
		if !ok {
			return provider.Result{}, fmt.Errorf("missing parameter 'container_id'")
		}
		return c.runCommand(ctx, op, "inspect", id)
	default:
		return provider.Result{}, &provider.PlatformError{
			ErrorCode: "UNSUPPORTED_OPERATION",
			Message:   fmt.Sprintf("Docker provider does not support action: %s", op.Action),
		}
	}
}

func (c *dockerConnection) Close() error {
	return nil
}

func (c *dockerConnection) checkBinary(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "--version")
	return cmd.Run()
}

func (c *dockerConnection) runCommand(ctx context.Context, op provider.Operation, args ...string) (provider.Result, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = os.Environ()

	if c.cred != nil {
		req, _ := http.NewRequest("GET", "http://dummy", nil)
		providerReq := &credentials.ProviderRequest{Header: req.Header}

		err := c.cred.Apply(providerReq)
		if err != nil {
			return provider.Result{}, &provider.PlatformError{
				ErrorCode:       "CREDENTIAL_APPLY_FAILED",
				Message:         "Failed to apply credentials for Docker execution",
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
