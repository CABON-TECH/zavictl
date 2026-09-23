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

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build management commands",
}

var buildStartCmd = &cobra.Command{
	Use:   "start [service-name]",
	Short: "Start a build for a service",
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
			Name:    fmt.Sprintf("build-%s", serviceName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "trigger_build",
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
			return fmt.Errorf("failed to submit build workflow: %v", err)
		}

		fmt.Printf("Build workflow submitted! Execution ID: %s\n", id)
		fmt.Printf("Triggering remote CI pipeline '%s' on %s/%s...\n", wfFile, owner, repo)

		for {
			status, err := appCtx.Engine.Status(ctx, id)
			if err != nil {
				return err
			}
			if status == execution.StatusCompleted || status == execution.StatusFailed || status == execution.StatusCancelled {
				if status == execution.StatusFailed {
					rec, ferr := appCtx.Store.Get(ctx, models.ResourceRef(fmt.Sprintf("Execution/%s", id)))
					if ferr == nil {
						if msg, ok := rec.Data["last_message"].(string); ok {
							fmt.Printf("Reason: %s\n", msg)
						}
					}
				}
				fmt.Printf("Build finished with status: %s\n", status)
				break
			}
			time.Sleep(1 * time.Second)
		}

		return nil
	},
}

var buildListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent builds",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing recent builds (stubbed)...")
		return nil
	},
}

var buildShowCmd = &cobra.Command{
	Use:   "show [execution-id]",
	Short: "Show details of a specific build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Showing details for build execution %s...\n", args[0])
		return nil
	},
}

var buildLogsCmd = &cobra.Command{
	Use:   "logs [execution-id]",
	Short: "View logs for a build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Fetching logs for build %s...\n", args[0])
		return nil
	},
}

var buildCancelCmd = &cobra.Command{
	Use:   "cancel [execution-id]",
	Short: "Cancel an active build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := execution.ExecutionID(args[0])
		if err := appCtx.Engine.Cancel(context.Background(), id); err != nil {
			return err
		}
		fmt.Printf("Cancellation requested for build %s.\n", id)
		return nil
	},
}

var buildRetryCmd = &cobra.Command{
	Use:   "retry [execution-id]",
	Short: "Retry a failed build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Retrying build execution %s...\n", args[0])
		return nil
	},
}

var buildArtifactsCmd = &cobra.Command{
	Use:   "artifacts [execution-id]",
	Short: "List artifacts produced by a build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Listing artifacts for build %s...\n", args[0])
		return nil
	},
}

func init() {
	buildStartCmd.Flags().String("workflow-file", "build.yaml", "Name of the GitHub Actions workflow file to trigger")
	buildStartCmd.Flags().String("branch", "main", "Branch to build")

	buildCmd.AddCommand(buildStartCmd)
	buildCmd.AddCommand(buildListCmd)
	buildCmd.AddCommand(buildShowCmd)
	buildCmd.AddCommand(buildLogsCmd)
	buildCmd.AddCommand(buildCancelCmd)
	buildCmd.AddCommand(buildRetryCmd)
	buildCmd.AddCommand(buildArtifactsCmd)

	rootCmd.AddCommand(buildCmd)
}
