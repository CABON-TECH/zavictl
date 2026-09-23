package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage and execute workflows",
}

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Run a workflow from a YAML definition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read workflow file: %v", err)
		}

		var wf workflow.WorkflowDefinition
		if err := yaml.Unmarshal(data, &wf); err != nil {
			return fmt.Errorf("failed to parse workflow: %v", err)
		}

		ctx := context.Background()
		id, err := appCtx.Engine.Submit(ctx, wf)
		if err != nil {
			return fmt.Errorf("failed to submit workflow: %v", err)
		}

		fmt.Printf("Workflow submitted successfully.\nExecution ID: %s\n", id)
		
		wait, _ := cmd.Flags().GetBool("wait")
		if wait {
			fmt.Println("Waiting for workflow to complete...")
			for {
				status, err := appCtx.Engine.Status(ctx, id)
				if err != nil {
					return err
				}
				if status == execution.StatusCompleted || status == execution.StatusFailed || status == execution.StatusCancelled {
					fmt.Printf("Workflow finished with status: %s\n", status)
					if status != execution.StatusCompleted {
						os.Exit(1)
					}
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}

		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [execution-id]",
	Short: "Check the status of a workflow execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := execution.ExecutionID(args[0])
		status, err := appCtx.Engine.Status(context.Background(), id)
		if err != nil {
			return fmt.Errorf("failed to get status: %v", err)
		}
		fmt.Printf("Execution %s is %s\n", id, status)
		return nil
	},
}

var approveCmd = &cobra.Command{
	Use:   "approve [execution-id] [step-name]",
	Short: "Approve a paused workflow step",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := execution.ExecutionID(args[0])
		stepName := args[1]
		err := appCtx.Engine.SignalApproval(context.Background(), id, stepName)
		if err != nil {
			return fmt.Errorf("failed to approve: %v", err)
		}
		fmt.Printf("Approved step %s for execution %s\n", stepName, id)
		return nil
	},
}

func init() {
	runCmd.Flags().BoolP("wait", "w", false, "Wait for the workflow to complete")
	
	workflowCmd.AddCommand(runCmd)
	workflowCmd.AddCommand(statusCmd)
	workflowCmd.AddCommand(approveCmd)
	
	rootCmd.AddCommand(workflowCmd)
}
