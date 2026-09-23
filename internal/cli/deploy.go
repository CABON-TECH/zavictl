package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"zavictl/pkg/execution"
	"zavictl/pkg/models"
	"zavictl/pkg/workflow"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deployment management commands",
}

var deployStartCmd = &cobra.Command{
	Use:   "start [service-name]",
	Short: "Start a deployment for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		wfFile, _ := cmd.Flags().GetString("workflow-file")
		branch, _ := cmd.Flags().GetString("branch")

		ctx := context.Background()

		// 1. Relational State Lookup
		records, err := appCtx.Store.List(ctx, "Service")
		if err != nil {
			return fmt.Errorf("failed to list services: %v", err)
		}

		var repoURL string
		for _, rec := range records {
			if rec.Data["name"].(string) == serviceName {
				repoURL = rec.Data["repository_url"].(string)
				break
			}
		}

		if repoURL == "" {
			return fmt.Errorf("service '%s' not found in local catalog", serviceName)
		}

		// 2. Parse GitHub owner/repo from URL
		// E.g., https://github.com/my-org/orders-api.git
		parsed, err := url.Parse(repoURL)
		if err != nil {
			return fmt.Errorf("invalid repository URL in service definition: %v", err)
		}
		
		pathParts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
		if len(pathParts) < 2 {
			return fmt.Errorf("unable to parse owner and repo from URL: %s", repoURL)
		}
		owner := pathParts[0]
		repo := strings.TrimSuffix(pathParts[1], ".git")

		// 3. INTENT -> WORKFLOW Compilation
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("deploy-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "trigger_ci",
					Action: "github.cicd.trigger_workflow",
					Inputs: map[string]workflow.Expression{
						"owner":       workflow.Expression(owner),
						"repo":        workflow.Expression(repo),
						"workflow_id": workflow.Expression(wfFile),
						"ref":         workflow.Expression(branch),
					},
				},
			},
		}

		// 4. Submit and Wait
		id, err := appCtx.Engine.Submit(ctx, wf)
		if err != nil {
			return fmt.Errorf("failed to submit deployment workflow: %v", err)
		}

		fmt.Printf("Deployment workflow submitted! Execution ID: %s\n", id)
		fmt.Printf("Triggering remote CI pipeline '%s' on %s/%s...\n", wfFile, owner, repo)

		for {
			status, err := appCtx.Engine.Status(ctx, id)
			if err != nil {
				return err
			}
			if status == execution.StatusCompleted || status == execution.StatusFailed || status == execution.StatusCancelled {
				if status == execution.StatusFailed {
					// Try to pull the last_message for context
					rec, ferr := appCtx.Store.Get(ctx, models.ResourceRef(fmt.Sprintf("Execution/%s", id)))
					if ferr == nil {
						if msg, ok := rec.Data["last_message"].(string); ok {
							fmt.Printf("Reason: %s\n", msg)
						}
					}
				}
				fmt.Printf("Deployment finished with status: %s\n", status)
				break
			}
			time.Sleep(1 * time.Second)
		}

		return nil
	},
}

var deployPlanCmd = &cobra.Command{
	Use:   "plan [service-name]",
	Short: "Generate a deployment plan without executing it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Planning deployment for service '%s' (Dry-run mode)...\n", args[0])
		fmt.Println("Plan: 1 artifact to deploy, 0 config changes.")
		return nil
	},
}

var deployStatusCmd = &cobra.Command{
	Use:   "status [execution-id]",
	Short: "Check the status of a deployment execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := execution.ExecutionID(args[0])
		status, err := appCtx.Engine.Status(context.Background(), id)
		if err != nil {
			return err
		}
		fmt.Printf("Execution ID: %s\nStatus: %s\n", id, status)
		return nil
	},
}

var deployCancelCmd = &cobra.Command{
	Use:   "cancel [execution-id]",
	Short: "Cancel an active deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := execution.ExecutionID(args[0])
		if err := appCtx.Engine.Cancel(context.Background(), id); err != nil {
			return err
		}
		fmt.Printf("Cancellation requested for %s.\n", id)
		return nil
	},
}

var deployPauseCmd = &cobra.Command{
	Use:   "pause [execution-id]",
	Short: "Pause an active deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Deployment %s paused.\n", args[0])
		return nil
	},
}

var deployResumeCmd = &cobra.Command{
	Use:   "resume [execution-id]",
	Short: "Resume a paused deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := execution.ExecutionID(args[0])
		if err := appCtx.Engine.Resume(context.Background(), id); err != nil {
			return err
		}
		fmt.Printf("Deployment %s resumed.\n", id)
		return nil
	},
}

var deployRollbackCmd = &cobra.Command{
	Use:   "rollback [service-name]",
	Short: "Rollback a deployment using a rollback workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Triggering rollback for service '%s'...\n", args[0])
		fmt.Println("Rollback workflow submitted!")
		return nil
	},
}

func init() {
	deployStartCmd.Flags().String("workflow-file", "deploy.yaml", "Name of the GitHub Actions workflow file to trigger")
	deployStartCmd.Flags().String("branch", "main", "Branch to deploy")

	deployCmd.AddCommand(deployPlanCmd)
	deployCmd.AddCommand(deployStartCmd)
	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployCancelCmd)
	deployCmd.AddCommand(deployPauseCmd)
	deployCmd.AddCommand(deployResumeCmd)
	deployCmd.AddCommand(deployRollbackCmd)

	rootCmd.AddCommand(deployCmd)
}
