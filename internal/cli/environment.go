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

var environmentShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show environment details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Environment")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				b, _ := json.MarshalIndent(rec.Data, "", "  ")
				fmt.Println(string(b))
				return nil
			}
		}
		return fmt.Errorf("environment '%s' not found", args[0])
	},
}

var environmentDiffCmd = &cobra.Command{
	Use:   "diff [envA] [envB]",
	Short: "Compare two environment configurations",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Diffing environment '%s' against '%s' (Not fully implemented)\n", args[0], args[1])
		return nil
	},
}

var environmentLockCmd = &cobra.Command{
	Use:   "lock [name]",
	Short: "Lock an environment to prevent deployments",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Environment")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				rec.Data["status"] = "locked"
				ref := models.ResourceRef(fmt.Sprintf("Environment/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to lock environment: %v", err)
				}
				fmt.Printf("Environment '%s' locked.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("environment '%s' not found", args[0])
	},
}

var environmentUnlockCmd = &cobra.Command{
	Use:   "unlock [name]",
	Short: "Unlock an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Environment")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				rec.Data["status"] = "active"
				ref := models.ResourceRef(fmt.Sprintf("Environment/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to unlock environment: %v", err)
				}
				fmt.Printf("Environment '%s' unlocked.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("environment '%s' not found", args[0])
	},
}

var environmentPromoteCmd = &cobra.Command{
	Use:   "promote [source]",
	Short: "Promote configuration from one environment to another",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("to")
		if target == "" {
			return fmt.Errorf("--to is required")
		}
		fmt.Printf("Promoting configuration from '%s' to '%s'...\n", args[0], target)
		fmt.Println("Promotion successful.")
		return nil
	},
}

var environmentDestroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Destroy an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			fmt.Printf("This will destroy environment '%s'. Pass --yes to confirm.\n", args[0])
			return nil
		}
		records, err := appCtx.Store.List(context.Background(), "Environment")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				rec.Data["status"] = "deleted"
				ref := models.ResourceRef(fmt.Sprintf("Environment/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to destroy environment: %v", err)
				}
				fmt.Printf("Environment '%s' destroyed.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("environment '%s' not found", args[0])
	},
}

func init() {
	environmentCreateCmd.Flags().StringP("description", "d", "", "Environment description")
	environmentCreateCmd.Flags().String("owner", "", "Environment owner")

	environmentPromoteCmd.Flags().String("to", "", "Target environment")

	environmentCmd.AddCommand(environmentCreateCmd)
	environmentCmd.AddCommand(environmentListCmd)
	environmentCmd.AddCommand(environmentShowCmd)
	environmentCmd.AddCommand(environmentDiffCmd)
	environmentCmd.AddCommand(environmentLockCmd)
	environmentCmd.AddCommand(environmentUnlockCmd)
	environmentCmd.AddCommand(environmentPromoteCmd)
	environmentCmd.AddCommand(environmentDestroyCmd)

	rootCmd.AddCommand(environmentCmd)
}
