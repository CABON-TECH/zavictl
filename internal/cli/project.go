package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		owner, _ := cmd.Flags().GetString("owner")

		proj := models.NewProject(name, desc, owner)

		if err := proj.Validate(); err != nil {
			return fmt.Errorf("invalid project: %v", err)
		}

		b, _ := json.Marshal(proj)
		var data map[string]any
		json.Unmarshal(b, &data)

		rec := state.StateRecord{
			Version:   1,
			UpdatedAt: proj.UpdatedAt,
			Data:      data,
		}

		if err := appCtx.Store.Put(context.Background(), proj.Identity(), rec, 0); err != nil {
			return fmt.Errorf("failed to save project: %v", err)
		}

		fmt.Printf("Project %s created successfully.\n", proj.ID)
		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Project")
		if err != nil {
			return fmt.Errorf("failed to list projects: %v", err)
		}
		
		if len(records) == 0 {
			fmt.Println("No projects found.")
			return nil
		}
		
		for _, rec := range records {
			fmt.Printf("ID: %-30v Name: %-20v Owner: %v\n", rec.Data["id"], rec.Data["name"], rec.Data["owner"])
		}
		return nil
	},
}

var projectShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := models.ResourceRef(fmt.Sprintf("Project/%s", args[0]))
		rec, err := appCtx.Store.Get(context.Background(), ref)
		if err != nil {
			return fmt.Errorf("project not found: %v", err)
		}
		
		b, _ := json.MarshalIndent(rec.Data, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

var projectValidateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate a project's configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Project")
		if err != nil {
			return fmt.Errorf("failed to list projects: %v", err)
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				proj := &models.Project{}
				b, _ := json.Marshal(rec.Data)
				json.Unmarshal(b, proj)
				if err := proj.Validate(); err != nil {
					return fmt.Errorf("validation failed: %v", err)
				}
				fmt.Printf("Project '%s' is valid.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("project '%s' not found", args[0])
	},
}

var projectArchiveCmd = &cobra.Command{
	Use:   "archive [name]",
	Short: "Archive a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Project")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				rec.Data["status"] = "archived"
				ref := models.ResourceRef(fmt.Sprintf("Project/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to archive project: %v", err)
				}
				fmt.Printf("Project '%s' archived.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("project '%s' not found", args[0])
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a project permanently",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			fmt.Printf("This will permanently delete project '%s'. Pass --yes to confirm.\n", args[0])
			return nil
		}
		records, err := appCtx.Store.List(context.Background(), "Project")
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.Data["name"].(string) == args[0] {
				rec.Data["status"] = "deleted"
				ref := models.ResourceRef(fmt.Sprintf("Project/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to delete project: %v", err)
				}
				fmt.Printf("Project '%s' deleted.\n", args[0])
				return nil
			}
		}
		return fmt.Errorf("project '%s' not found", args[0])
	},
}

func init() {
	projectCreateCmd.Flags().StringP("description", "d", "", "Project description")
	projectCreateCmd.Flags().String("owner", "", "Project owner")

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectValidateCmd)
	projectCmd.AddCommand(projectArchiveCmd)
	projectCmd.AddCommand(projectDeleteCmd)

	rootCmd.AddCommand(projectCmd)
}
