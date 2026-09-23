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

var serviceShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show service details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Service")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				fmt.Printf("Name:           %v\n", rec.Data["name"])
				fmt.Printf("ID:             %v\n", rec.Data["id"])
				fmt.Printf("Owner:          %v\n", rec.Data["owner"])
				fmt.Printf("Repository URL: %v\n", rec.Data["repository_url"])
				fmt.Printf("Status:         %v\n", rec.Data["status"])
				fmt.Printf("Created:        %v\n", rec.Data["created_at"])
				return nil
			}
		}
		return fmt.Errorf("service '%s' not found", args[0])
	},
}

var serviceValidateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate a service's registration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Service")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				if rec.Data["repository_url"] == nil || rec.Data["repository_url"].(string) == "" {
					return fmt.Errorf("validation failed: service has no repository_url")
				}
				fmt.Printf("Service '%s' is valid.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("service '%s' not found", args[0])
	},
}

var serviceDepsCmd = &cobra.Command{
	Use:   "dependencies [name]",
	Short: "List declared dependencies of a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Service")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				deps, _ := rec.Data["dependencies"].([]any)
				if len(deps) == 0 {
					fmt.Printf("Service '%s' has no declared dependencies.\n", args[0])
					return nil
				}
				fmt.Printf("Dependencies of '%s':\n", args[0])
				for _, d := range deps {
					fmt.Printf("  - %v\n", d)
				}
				return nil
			}
		}
		return fmt.Errorf("service '%s' not found", args[0])
	},
}

var serviceOwnerCmd = &cobra.Command{
	Use:   "owner [name]",
	Short: "Get or set the owner of a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		set, _ := cmd.Flags().GetString("set")
		records, err := appCtx.Store.List(context.Background(), "Service")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				if set == "" {
					fmt.Printf("Owner of '%s': %v\n", args[0], rec.Data["owner"])
					return nil
				}
				rec.Data["owner"] = set
				ref := models.ResourceRef(fmt.Sprintf("Service/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to update owner: %v", err)
				}
				fmt.Printf("Owner of '%s' updated to '%s'.\n", args[0], set)
				return nil
			}
		}
		return fmt.Errorf("service '%s' not found", args[0])
	},
}

func init() {
	serviceCreateCmd.Flags().StringP("description", "d", "", "Service description")
	serviceCreateCmd.Flags().String("owner", "", "Service owner")
	serviceCreateCmd.Flags().String("repo", "", "Repository URL")
	serviceCreateCmd.Flags().String("env", "", "Environment Ref ID")
	serviceCreateCmd.MarkFlagRequired("repo")

	serviceOwnerCmd.Flags().String("set", "", "New owner value")

	serviceCmd.AddCommand(serviceCreateCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceShowCmd)
	serviceCmd.AddCommand(serviceValidateCmd)
	serviceCmd.AddCommand(serviceDepsCmd)
	serviceCmd.AddCommand(serviceOwnerCmd)

	rootCmd.AddCommand(serviceCmd)
}
