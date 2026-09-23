package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Source-control integration commands",
}

var repoCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new source repository via provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		providerName, _ := cmd.Flags().GetString("provider")
		org, _ := cmd.Flags().GetString("org")
		desc, _ := cmd.Flags().GetString("description")
		private, _ := cmd.Flags().GetBool("private")

		if providerName != "github" {
			return fmt.Errorf("only github provider is currently supported for repo create")
		}

		// INTENT -> WORKFLOW Compilation
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("repo-create-%s", name),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "create_repo",
					Action: "github.sourcecontrol.create_repository",
					Inputs: map[string]workflow.Expression{
						"name":         workflow.Expression(name),
						"organization": workflow.Expression(org),
						"description":  workflow.Expression(desc),
						"private":      workflow.Expression(fmt.Sprintf("%v", private)),
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
		fmt.Println("Waiting for completion...")

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
	repoCreateCmd.Flags().String("provider", "github", "Provider (github, gitlab)")
	repoCreateCmd.Flags().String("org", "", "Organization name (optional)")
	repoCreateCmd.Flags().String("description", "", "Repository description")
	repoCreateCmd.Flags().Bool("private", true, "Make repository private")

	repoCmd.AddCommand(repoCreateCmd)
	rootCmd.AddCommand(repoCmd)
}
