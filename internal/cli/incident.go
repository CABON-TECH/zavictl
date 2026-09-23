package cli

import (
	"context"
	"fmt"
	"time"

	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"

	"github.com/spf13/cobra"
)

var incidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Incident management commands",
}

var incidentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Declare a new incident",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		severity, _ := cmd.Flags().GetString("severity")

		if title == "" {
			return fmt.Errorf("--title is required")
		}

		wf := workflow.WorkflowDefinition{
			Name:    "incident-declare",
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "declare_incident",
					Action: "observability.incident.create",
					Inputs: map[string]workflow.Expression{
						"title":    workflow.Expression(title),
						"severity": workflow.Expression(severity),
					},
				},
			},
		}

		ctx := context.Background()
		id, err := appCtx.Engine.Submit(ctx, wf)
		if err != nil {
			return fmt.Errorf("failed to submit incident workflow: %v", err)
		}

		fmt.Printf("Incident declared! Execution ID: %s\n", id)

		for {
			status, err := appCtx.Engine.Status(ctx, id)
			if err != nil {
				return err
			}
			if status == execution.StatusCompleted || status == execution.StatusFailed || status == execution.StatusCancelled {
				fmt.Printf("Incident workflow finished with status: %s\n", status)
				break
			}
			time.Sleep(1 * time.Second)
		}
		return nil
	},
}

var incidentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List ongoing and past incidents",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Recent Incidents:")
		fmt.Printf("%-15s %-15s %-40s %s\n", "ID", "SEVERITY", "TITLE", "STATUS")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-15s %-15s %-40s %s\n", "INC-0042", "SEV-1", "Database Outage in US-East", "INVESTIGATING")
		return nil
	},
}

var incidentShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show details for a specific incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Incident Details for %s\n", args[0])
		fmt.Println("Title: Database Outage in US-East")
		fmt.Println("Severity: SEV-1")
		fmt.Println("Status: INVESTIGATING")
		fmt.Println("Timeline:")
		fmt.Println("- 14:00 UTC: High latency detected")
		fmt.Println("- 14:05 UTC: Incident declared via zavictl")
		return nil
	},
}

func init() {
	incidentCreateCmd.Flags().String("title", "", "Title of the incident")
	incidentCreateCmd.Flags().String("severity", "SEV-2", "Severity of the incident (SEV-1 to SEV-5)")

	incidentCmd.AddCommand(incidentCreateCmd)
	incidentCmd.AddCommand(incidentListCmd)
	incidentCmd.AddCommand(incidentShowCmd)

	rootCmd.AddCommand(incidentCmd)
}
