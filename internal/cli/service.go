package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
}

var serviceCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		owner, _ := cmd.Flags().GetString("owner")
		repoURL, _ := cmd.Flags().GetString("repo")
		envRef, _ := cmd.Flags().GetString("env")

		svc := models.NewService(name, desc, owner, repoURL, envRef)

		if err := svc.Validate(); err != nil {
			return fmt.Errorf("invalid service: %v", err)
		}

		b, _ := json.Marshal(svc)
		var data map[string]any
		json.Unmarshal(b, &data)

		rec := state.StateRecord{
			Version:   1,
			UpdatedAt: svc.UpdatedAt,
			Data:      data,
		}

		if err := appCtx.Store.Put(context.Background(), svc.Identity(), rec, 0); err != nil {
			return fmt.Errorf("failed to save service: %v", err)
		}

		fmt.Printf("Service %s created successfully.\n", svc.ID)
		return nil
	},
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Service")
		if err != nil {
			return fmt.Errorf("failed to list services: %v", err)
		}
		if len(records) == 0 {
			fmt.Println("No services found.")
			return nil
		}
		for _, rec := range records {
			fmt.Printf("ID: %-30v Name: %-20v Repo: %v\n", rec.Data["id"], rec.Data["name"], rec.Data["repository_url"])
		}
		return nil
	},
}

func init() {
	serviceCreateCmd.Flags().StringP("description", "d", "", "Service description")
	serviceCreateCmd.Flags().String("owner", "", "Service owner")
	serviceCreateCmd.Flags().String("repo", "", "Repository URL")
	serviceCreateCmd.Flags().String("env", "", "Environment Ref ID")
	serviceCreateCmd.MarkFlagRequired("repo")

	serviceCmd.AddCommand(serviceCreateCmd)
	serviceCmd.AddCommand(serviceListCmd)

	rootCmd.AddCommand(serviceCmd)
}
