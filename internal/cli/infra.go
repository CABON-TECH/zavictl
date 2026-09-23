package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"
)

var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Infrastructure management commands",
}

var infraPlanCmd = &cobra.Command{
	Use:   "plan [directory]",
	Short: "Generate infrastructure plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		providerName, _ := cmd.Flags().GetString("provider")

		action := fmt.Sprintf("%s.infra.plan", providerName)

		// INTENT -> WORKFLOW Compilation
		wf := workflow.WorkflowDefinition{
			Name:    "infra-plan",
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "plan",
					Action: action,
					Inputs: map[string]workflow.Expression{
						"directory": workflow.Expression(dir),
					},
				},
			},
		}

		ctx := context.Background()
		id, err := appCtx.Engine.Submit(ctx, wf)
		if err != nil {
			return fmt.Errorf("failed to submit workflow: %v", err)
		}

		fmt.Printf("Workflow submitted! Execution ID: %s\n", id)
		fmt.Println("Waiting for infrastructure plan to generate...")

		// Stream wait
		for {
			status, err := appCtx.Engine.Status(ctx, id)
			if err != nil {
				return err
			}
			if status == execution.StatusCompleted || status == execution.StatusFailed || status == execution.StatusCancelled {
				fmt.Printf("Workflow finished with status: %s\n", status)
				break
			}
			time.Sleep(1 * time.Second)
		}

		return nil
	},
}

func init() {
	infraPlanCmd.Flags().String("provider", "terraform", "Infrastructure provider (terraform, opentofu)")

	infraCmd.AddCommand(infraPlanCmd)
	rootCmd.AddCommand(infraCmd)
}
