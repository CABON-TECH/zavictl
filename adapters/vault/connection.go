package vault

import (
	"context"
	"fmt"

	"zavictl/pkg/provider"

	vaultapi "github.com/hashicorp/vault/api"
)

type vaultConnection struct {
	client *vaultapi.Client
}

func (c *vaultConnection) Execute(ctx context.Context, op provider.Operation) (provider.Result, error) {
	switch op.Action {
	case "secretmanagement.read":
		return c.read(ctx, op)
	case "secretmanagement.write":
		return c.write(ctx, op)
	case "secretmanagement.list":
		return c.list(ctx, op)
	case "secretmanagement.delete":
		return c.delete(ctx, op)
	default:
		return provider.Result{}, &provider.PlatformError{
			ErrorCode: "UNSUPPORTED_OPERATION",
			Message:   fmt.Sprintf("Vault provider does not support action: %s", op.Action),
		}
	}
}

func (c *vaultConnection) Close() error {
	c.client.CloneConfig().HttpClient.CloseIdleConnections()
	return nil
}

func (c *vaultConnection) read(ctx context.Context, op provider.Operation) (provider.Result, error) {
	path, ok := op.Parameters["path"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'path'")
	}

	secret, err := c.client.Logical().Read(path)
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	if secret == nil {
		return provider.Result{}, &provider.PlatformError{
			ErrorCode: "VAULT_NOT_FOUND",
			Message:   fmt.Sprintf("Secret not found at path: %s", path),
		}
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"data": secret.Data,
		},
	}, nil
}

func (c *vaultConnection) write(ctx context.Context, op provider.Operation) (provider.Result, error) {
	path, ok := op.Parameters["path"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'path'")
	}
	data, ok := op.Parameters["data"].(map[string]interface{})
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'data'")
	}

	secret, err := c.client.Logical().Write(path, data)
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	var outputs map[string]any
	if secret != nil && secret.Data != nil {
		outputs = secret.Data
	} else {
		outputs = map[string]any{"message": "Secret written successfully"}
	}

	return provider.Result{
		Status:  "success",
		Outputs: outputs,
	}, nil
}

func (c *vaultConnection) list(ctx context.Context, op provider.Operation) (provider.Result, error) {
	path, ok := op.Parameters["path"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'path'")
	}

	secret, err := c.client.Logical().List(path)
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	if secret == nil {
		return provider.Result{
			Status: "success",
			Outputs: map[string]any{
				"keys": []string{},
			},
		}, nil
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"keys": secret.Data["keys"],
		},
	}, nil
}

func (c *vaultConnection) delete(ctx context.Context, op provider.Operation) (provider.Result, error) {
	path, ok := op.Parameters["path"].(string)
	if !ok {
		return provider.Result{}, fmt.Errorf("missing parameter 'path'")
	}

	_, err := c.client.Logical().Delete(path)
	if err != nil {
		return provider.Result{}, mapError(err)
	}

	return provider.Result{
		Status: "success",
		Outputs: map[string]any{
			"message": "Secret deleted successfully",
		},
	}, nil
}
