package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

var environmentCmd = &cobra.Command{
	Use:   "environment",
	Short: "Manage environments",
}

var environmentCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		owner, _ := cmd.Flags().GetString("owner")

		env := models.NewEnvironment(name, desc, owner)

		if err := env.Validate(); err != nil {
			return fmt.Errorf("invalid environment: %v", err)
		}

		b, _ := json.Marshal(env)
		var data map[string]any
		json.Unmarshal(b, &data)

		rec := state.StateRecord{
			Version:   1,
			UpdatedAt: env.UpdatedAt,
			Data:      data,
		}

		if err := appCtx.Store.Put(context.Background(), env.Identity(), rec, 0); err != nil {
			return fmt.Errorf("failed to save environment: %v", err)
		}

		fmt.Printf("Environment %s created successfully.\n", env.ID)
		return nil
	},
}

var environmentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Environment")
		if err != nil {
			return fmt.Errorf("failed to list environments: %v", err)
		}
		if len(records) == 0 {
			fmt.Println("No environments found.")
			return nil
		}
		for _, rec := range records {
			fmt.Printf("ID: %-30v Name: %-20v Owner: %v\n", rec.Data["id"], rec.Data["name"], rec.Data["owner"])
		}
		return nil
	},
}

func init() {
	environmentCreateCmd.Flags().StringP("description", "d", "", "Environment description")
	environmentCreateCmd.Flags().String("owner", "", "Environment owner")

	environmentCmd.AddCommand(environmentCreateCmd)
	environmentCmd.AddCommand(environmentListCmd)

	rootCmd.AddCommand(environmentCmd)
}
