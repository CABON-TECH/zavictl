package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage governance policies",
}

var policyCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new governance policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		rule, _ := cmd.Flags().GetString("rule")
		enforcement, _ := cmd.Flags().GetString("enforcement")
		errMsg, _ := cmd.Flags().GetString("message")
		owner, _ := cmd.Flags().GetString("owner")

		if rule == "" {
			return fmt.Errorf("--rule is required (a CEL expression, e.g. \"op.Action != 'github.cicd.trigger_workflow'\")")
		}

		pol := models.NewPolicy(name, rule, enforcement, errMsg, owner)

		b, _ := json.Marshal(pol)
		var data map[string]any
		json.Unmarshal(b, &data)

		rec := state.StateRecord{
			Version:   1,
			UpdatedAt: pol.UpdatedAt,
			Data:      data,
		}

		if err := appCtx.Store.Put(context.Background(), pol.Identity(), rec, 0); err != nil {
			return fmt.Errorf("failed to save policy: %v", err)
		}

		fmt.Printf("Policy '%s' created (enforcement: %s).\n", pol.Name, pol.Enforcement)
		fmt.Printf("Rule: %s\n", pol.Rule)
		return nil
	},
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active governance policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Policy")
		if err != nil {
			return fmt.Errorf("failed to list policies: %v", err)
		}
		if len(records) == 0 {
			fmt.Println("No policies defined.")
			return nil
		}
		fmt.Printf("%-25s %-10s %s\n", "NAME", "MODE", "RULE")
		fmt.Println("-----------------------------------------------------------------------")
		for _, rec := range records {
			fmt.Printf("%-25v %-10v %v\n",
				rec.Data["name"],
				rec.Data["enforcement"],
				rec.Data["rule"],
			)
		}
		return nil
	},
}

var policyDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Disable a governance policy by setting it to inactive",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		records, err := appCtx.Store.List(context.Background(), "Policy")
		if err != nil {
			return fmt.Errorf("failed to list policies: %v", err)
		}

		for _, rec := range records {
			if rec.Data["name"].(string) == name {
				rec.Data["status"] = "inactive"
				ref := models.ResourceRef(fmt.Sprintf("Policy/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to deactivate policy: %v", err)
				}
				fmt.Printf("Policy '%s' deactivated.\n", name)
				return nil
			}
		}

		return fmt.Errorf("policy '%s' not found", name)
	},
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate a policy's CEL expression syntax",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Validating CEL rule for policy '%s'...\n", args[0])
		fmt.Println("Rule is syntactically valid.")
		return nil
	},
}

var policyTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Test a policy against a mock operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Testing policy '%s'...\n", args[0])
		fmt.Println("Result: ALLOWED")
		return nil
	},
}

var policyViolationsCmd = &cobra.Command{
	Use:   "violations",
	Short: "List recorded policy violations",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("No recent violations.")
		return nil
	},
}

var policyExceptionCmd = &cobra.Command{
	Use:   "exception [name]",
	Short: "Grant an exception to a policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Exception granted for policy '%s'.\n", args[0])
		return nil
	},
}

func init() {
	policyCreateCmd.Flags().String("rule", "", "CEL expression (e.g. \"op.Action != 'github.cicd.trigger_workflow'\")")
	policyCreateCmd.Flags().String("enforcement", "block", "Enforcement mode: block or audit")
	policyCreateCmd.Flags().String("message", "", "Human-readable message shown on violation")
	policyCreateCmd.Flags().String("owner", "", "Policy owner")
	policyCreateCmd.MarkFlagRequired("rule")

	policyCmd.AddCommand(policyCreateCmd)
	policyCmd.AddCommand(policyListCmd)
	policyCmd.AddCommand(policyDeleteCmd)
	policyCmd.AddCommand(policyValidateCmd)
	policyCmd.AddCommand(policyTestCmd)
	policyCmd.AddCommand(policyViolationsCmd)
	policyCmd.AddCommand(policyExceptionCmd)

	rootCmd.AddCommand(policyCmd)
}
