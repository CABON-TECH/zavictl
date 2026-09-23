package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"zavictl/pkg/execution"
	"zavictl/pkg/workflow"
)

var databaseCmd = &cobra.Command{
	Use:     "database",
	Aliases: []string{"db"},
	Short:   "Database lifecycle and operations commands",
}

var databaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered databases",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing active databases (stubbed)...")
		fmt.Printf("%-20s %-15s %-15s %s\n", "NAME", "ENGINE", "STATUS", "ENDPOINT")
		fmt.Println("-------------------------------------------------------------------------")
		fmt.Printf("%-20s %-15s %-15s %s\n", "orders-db-prod", "postgres-15", "Available", "orders-db.internal:5432")
		return nil
	},
}

var databaseProvisionCmd = &cobra.Command{
	Use:   "provision [name]",
	Short: "Provision a new database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		engine, _ := cmd.Flags().GetString("engine")

		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("db-provision-%s", dbName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "provision",
					Action: "infrastructure.database.provision",
					Inputs: map[string]workflow.Expression{
						"name":   workflow.Expression(dbName),
						"engine": workflow.Expression(engine),
					},
				},
			},
		}

		return executeDatabaseWorkflow(wf, "Provision")
	},
}

var databaseMigrateCmd = &cobra.Command{
	Use:   "migrate [name]",
	Short: "Apply schema migrations to a database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		dir, _ := cmd.Flags().GetString("dir")

		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("db-migrate-%s", dbName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "migrate",
					Action: "application.database.migrate",
					Inputs: map[string]workflow.Expression{
						"database":  workflow.Expression(dbName),
						"directory": workflow.Expression(dir),
					},
				},
			},
		}

		return executeDatabaseWorkflow(wf, "Migration")
	},
}

var databaseBackupCmd = &cobra.Command{
	Use:   "backup [name]",
	Short: "Trigger a database backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("db-backup-%s", dbName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "backup",
					Action: "infrastructure.database.backup",
					Inputs: map[string]workflow.Expression{
						"database": workflow.Expression(dbName),
					},
				},
			},
		}

		return executeDatabaseWorkflow(wf, "Backup")
	},
}

var databaseRestoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "Restore a database from a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		backupID, _ := cmd.Flags().GetString("from")

		if backupID == "" {
			return fmt.Errorf("--from flag is required to specify backup ID")
		}

		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("db-restore-%s", dbName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "restore",
					Action: "infrastructure.database.restore",
					Inputs: map[string]workflow.Expression{
						"database":  workflow.Expression(dbName),
						"backup_id": workflow.Expression(backupID),
					},
				},
			},
		}

		return executeDatabaseWorkflow(wf, "Restore")
	},
}

var databaseFailoverCmd = &cobra.Command{
	Use:   "failover [name]",
	Short: "Initiate failover to a standby database replica",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		wf := workflow.WorkflowDefinition{
			Name:    fmt.Sprintf("db-failover-%s", dbName),
			Version: "v1",
			Steps: []workflow.StepDefinition{
				{
					Name:   "failover",
					Action: "infrastructure.database.failover",
					Inputs: map[string]workflow.Expression{
						"database": workflow.Expression(dbName),
					},
				},
			},
		}

		return executeDatabaseWorkflow(wf, "Failover")
	},
}

func executeDatabaseWorkflow(wf workflow.WorkflowDefinition, opName string) error {
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
	databaseProvisionCmd.Flags().String("engine", "postgres", "Database engine (e.g. postgres, mysql)")
	databaseMigrateCmd.Flags().String("dir", "./migrations", "Directory containing migration scripts")
	databaseRestoreCmd.Flags().String("from", "", "Backup ID to restore from")

	databaseCmd.AddCommand(databaseListCmd)
	databaseCmd.AddCommand(databaseProvisionCmd)
	databaseCmd.AddCommand(databaseMigrateCmd)
	databaseCmd.AddCommand(databaseBackupCmd)
	databaseCmd.AddCommand(databaseRestoreCmd)
	databaseCmd.AddCommand(databaseFailoverCmd)

	rootCmd.AddCommand(databaseCmd)
}
