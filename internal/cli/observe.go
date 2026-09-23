package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"zavictl/pkg/state/sqlite"
)

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Observability and audit commands",
}

var observeAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Show the full platform audit trail",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		return printEvents("", limit)
	},
}

var observeEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show platform events, optionally filtered by kind",
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, _ := cmd.Flags().GetString("kind")
		limit, _ := cmd.Flags().GetInt("limit")
		return printEvents(kind, limit)
	},
}

var observeExecutionsCmd = &cobra.Command{
	Use:   "executions",
	Short: "Show workflow execution history",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, ok := appCtx.Store.(*sqlite.SQLiteStore)
		if !ok {
			return fmt.Errorf("requires SQLite store backend")
		}

		execs, err := store.ListExecutions(context.Background())
		if err != nil {
			return fmt.Errorf("failed to list executions: %v", err)
		}
		if len(execs) == 0 {
			fmt.Println("No executions recorded.")
			return nil
		}

		fmt.Printf("%-28s %-12s %-34s %s\n", "EXECUTION ID", "STATUS", "WORKFLOW", "LAST MESSAGE")
		fmt.Println("------------------------------------------------------------------------------------------------------------")
		for _, ex := range execs {
			msg := ex.LastMessage
			wfName := ex.WorkflowName
			if len(msg) > 45 {
				msg = msg[:45] + "..."
			}
			if len(wfName) > 30 {
				wfName = wfName[:30]
			}
			fmt.Printf("%-28s %-12s %-34s %s\n", ex.ID, ex.Status, wfName, msg)
		}
		return nil
	},
}

func printEvents(kind string, limit int) error {
	store, ok := appCtx.Store.(*sqlite.SQLiteStore)
	if !ok {
		return fmt.Errorf("audit log requires the SQLite store backend")
	}

	evs, err := store.ListEvents(context.Background(), kind, limit)
	if err != nil {
		return fmt.Errorf("failed to list events: %v", err)
	}
	if len(evs) == 0 {
		fmt.Println("No events recorded yet.")
		return nil
	}

	fmt.Printf("%-30s %-30s %-10s\n", "TIME", "TYPE", "STATUS")
	fmt.Println("------------------------------------------------------------------------")
	for _, ev := range evs {
		fmt.Printf("%-30s %-30s %-10s\n",
			ev.Timestamp.Local().Format("2006-01-02 15:04:05"),
			ev.Type,
			ev.Status,
		)
	}
	return nil
}

func init() {
	observeAuditCmd.Flags().Int("limit", 50, "Max number of events to show")
	observeEventsCmd.Flags().String("kind", "", "Filter by event kind (e.g. PolicyEvaluation, CredentialResolved)")
	observeEventsCmd.Flags().Int("limit", 50, "Max number of events to show")

	observeCmd.AddCommand(observeAuditCmd)
	observeCmd.AddCommand(observeEventsCmd)
	observeCmd.AddCommand(observeExecutionsCmd)

	rootCmd.AddCommand(observeCmd)
}
