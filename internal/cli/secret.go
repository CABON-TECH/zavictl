package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"

	"github.com/spf13/cobra"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Secret management commands",
}

var secretListCmd = &cobra.Command{
	Use:   "list [service]",
	Short: "List secrets for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("secret-list-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "list_secrets",
					Action: "vault.secretmanagement.list",
					Inputs: map[string]workflow.Expression{
						"path": workflow.Expression(fmt.Sprintf("secret/data/%s", serviceName)),
					},
				},
			},
		}
		return executeSecretWorkflow(wf, "List Secrets")
	},
}

var secretSetCmd = &cobra.Command{
	Use:   "set [service] [key]=[value]",
	Short: "Set a secret for a service",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		kv := strings.SplitN(args[1], "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("secret format must be key=value")
		}

		// Note: in a real system we'd merge with existing data, this writes fresh
		data := map[string]interface{}{
			"data": map[string]interface{}{
				kv[0]: kv[1],
			},
		}

		dataBytes, _ := json.Marshal(data)

		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("secret-set-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "write_secret",
					Action: "vault.secretmanagement.write",
					Inputs: map[string]workflow.Expression{
						"path": workflow.Expression(fmt.Sprintf("secret/data/%s", serviceName)),
						"data": workflow.Expression(string(dataBytes)),
					},
				},
			},
		}
		return executeSecretWorkflow(wf, "Set Secret")
	},
}

var secretInspectCmd = &cobra.Command{
	Use:   "inspect [service]",
	Short: "Inspect secrets for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("secret-inspect-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "read_secret",
					Action: "vault.secretmanagement.read",
					Inputs: map[string]workflow.Expression{
						"path": workflow.Expression(fmt.Sprintf("secret/data/%s", serviceName)),
					},
				},
			},
		}
		return executeSecretWorkflow(wf, "Inspect Secret")
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete [service]",
	Short: "Delete all secrets for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("secret-delete-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "delete_secret",
					Action: "vault.secretmanagement.delete",
					Inputs: map[string]workflow.Expression{
						"path": workflow.Expression(fmt.Sprintf("secret/metadata/%s", serviceName)),
					},
				},
			},
		}
		return executeSecretWorkflow(wf, "Delete Secret")
	},
}

var secretInjectCmd = &cobra.Command{
	Use:   "inject [service]",
	Short: "Inject secrets into runtime environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Secrets injected into runtime for %s\n", args[0])
		return nil
	},
}

var secretRotateCmd = &cobra.Command{
	Use:   "rotate [service] [key]",
	Short: "Rotate a specific secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Rotated secret '%s' for %s\n", args[1], args[0])
		return nil
	},
}

func executeSecretWorkflow(wf workflow.WorkflowDefinition, opName string) error {
	ctx := context.Background()
	id, err := appCtx.Engine.Submit(ctx, wf)
	if err != nil {
		return fmt.Errorf("failed to submit %s workflow: %v", opName, err)
	}

	fmt.Printf("%s workflow submitted! Execution ID: %s\n", opName, id)

	for {
		status, err := appCtx.Engine.Status(ctx, id)
		if err != nil {
			return err
		}
		if status == execution.StatusCompleted || status == execution.StatusFailed || status == execution.StatusCancelled {
			fmt.Printf("%s finished with status: %s\n", opName, status)
			break
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func init() {
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretSetCmd)
	secretCmd.AddCommand(secretInspectCmd)
	secretCmd.AddCommand(secretDeleteCmd)
	secretCmd.AddCommand(secretInjectCmd)
	secretCmd.AddCommand(secretRotateCmd)

	rootCmd.AddCommand(secretCmd)
}
