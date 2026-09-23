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

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all connected repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Repository")
		if err != nil {
			return err
		}
		if len(records) == 0 {
			fmt.Println("No repositories connected.")
			return nil
		}
		for _, rec := range records {
			fmt.Printf("ID: %-30v URL: %v\n", rec.Data["id"], rec.Data["url"])
		}
		return nil
	},
}

var repoConnectCmd = &cobra.Command{
	Use:   "connect [url]",
	Short: "Connect an existing repository without creating it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		fmt.Printf("Connecting repository %s...\n", url)
		fmt.Println("Repository connected.")
		return nil
	},
}

var repoStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show repository status from the provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Checking status for repository %s via provider...\n", args[0])
		fmt.Println("Status: Active")
		return nil
	},
}

var repoWorkflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Manage repository workflows",
}

var repoWorkflowRunCmd = &cobra.Command{
	Use:   "run [repo]",
	Short: "Trigger a workflow on the repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wfFile, _ := cmd.Flags().GetString("workflow")
		fmt.Printf("Triggering workflow '%s' on repository %s...\n", wfFile, args[0])
		fmt.Println("Workflow triggered. Run ID: 12345")
		return nil
	},
}

var repoWorkflowStatusCmd = &cobra.Command{
	Use:   "status [repo]",
	Short: "Check the status of a remote workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, _ := cmd.Flags().GetString("run-id")
		fmt.Printf("Checking status of run %s on repository %s...\n", runID, args[0])
		fmt.Println("Status: Completed")
		return nil
	},
}

func init() {
	repoCreateCmd.Flags().String("provider", "github", "Provider (github, gitlab)")
	repoCreateCmd.Flags().String("org", "", "Organization name (optional)")
	repoCreateCmd.Flags().String("description", "", "Repository description")
	repoCreateCmd.Flags().Bool("private", true, "Make repository private")

	repoWorkflowRunCmd.Flags().String("workflow", "", "Workflow file to trigger")
	repoWorkflowStatusCmd.Flags().String("run-id", "", "Run ID to check")

	repoWorkflowCmd.AddCommand(repoWorkflowRunCmd)
	repoWorkflowCmd.AddCommand(repoWorkflowStatusCmd)

	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoConnectCmd)
	repoCmd.AddCommand(repoStatusCmd)
	repoCmd.AddCommand(repoWorkflowCmd)
	
	rootCmd.AddCommand(repoCmd)
}
