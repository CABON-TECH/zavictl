package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"

	"github.com/spf13/cobra"
)

var runtimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Runtime management commands",
}

var runtimeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active services and their health states",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing runtime services (stubbed)...")
		fmt.Printf("%-20s %-15s %s\n", "SERVICE", "STATUS", "REPLICAS")
		fmt.Println("--------------------------------------------------")
		fmt.Printf("%-20s %-15s %s\n", "orders-api", "Running", "3/3")
		return nil
	},
}

var runtimeInspectCmd = &cobra.Command{
	Use:   "inspect [service]",
	Short: "Show live state from the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("runtime-inspect-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "inspect",
					Action: "kubernetes.runtime.inspect",
					Inputs: map[string]workflow.Expression{
						"service": workflow.Expression(serviceName),
					},
				},
			},
		}

		return executeRuntimeWorkflow(wf, "Inspect")
	},
}

var runtimeScaleCmd = &cobra.Command{
	Use:   "scale [service]",
	Short: "Scale the number of replicas for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		replicas, _ := cmd.Flags().GetInt("replicas")

		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("runtime-scale-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "scale",
					Action: "kubernetes.runtime.scale",
					Inputs: map[string]workflow.Expression{
						"service":  workflow.Expression(serviceName),
						"replicas": workflow.Expression(strconv.Itoa(replicas)),
					},
				},
			},
		}

		return executeRuntimeWorkflow(wf, "Scale")
	},
}

var runtimeRestartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("runtime-restart-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "restart",
					Action: "kubernetes.runtime.restart",
					Inputs: map[string]workflow.Expression{
						"service": workflow.Expression(serviceName),
					},
				},
			},
		}

		return executeRuntimeWorkflow(wf, "Restart")
	},
}

var runtimeLogsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "Stream or fetch logs for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("runtime-logs-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "logs",
					Action: "kubernetes.runtime.logs",
					Inputs: map[string]workflow.Expression{
						"service": workflow.Expression(serviceName),
					},
				},
			},
		}

		return executeRuntimeWorkflow(wf, "Logs fetch")
	},
}

var runtimeExecCmd = &cobra.Command{
	Use:   "exec [service]",
	Short: "Execute a command in a service container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Executing command in service '%s' (interactive stub)...\n", args[0])
		return nil
	},
}

var runtimeHealthCmd = &cobra.Command{
	Use:   "health [service]",
	Short: "Check the health of a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Health status for '%s': Healthy\n", args[0])
		return nil
	},
}

func executeRuntimeWorkflow(wf workflow.WorkflowDefinition, opName string) error {
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
	runtimeScaleCmd.Flags().Int("replicas", 1, "Number of replicas to scale to")

	runtimeCmd.AddCommand(runtimeListCmd)
	runtimeCmd.AddCommand(runtimeInspectCmd)
	runtimeCmd.AddCommand(runtimeScaleCmd)
	runtimeCmd.AddCommand(runtimeRestartCmd)
	runtimeCmd.AddCommand(runtimeLogsCmd)
	runtimeCmd.AddCommand(runtimeExecCmd)
	runtimeCmd.AddCommand(runtimeHealthCmd)

	rootCmd.AddCommand(runtimeCmd)
}
