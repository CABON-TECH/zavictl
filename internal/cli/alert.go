package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Alert management commands",
}

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active alerts",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Active Alerts:")
		fmt.Printf("%-15s %-15s %-40s %s\n", "ID", "SEVERITY", "NAME", "STATUS")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("%-15s %-15s %-40s %s\n", "ALT-9001", "CRITICAL", "High CPU Load - orders-api", "FIRING")
		fmt.Printf("%-15s %-15s %-40s %s\n", "ALT-9002", "WARNING", "High Memory Usage - search-api", "FIRING")
		return nil
	},
}

var alertAckCmd = &cobra.Command{
	Use:   "acknowledge [id]",
	Short: "Acknowledge an active alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Alert %s acknowledged. Notifications silenced.\n", args[0])
		return nil
	},
}

var alertResolveCmd = &cobra.Command{
	Use:   "resolve [id]",
	Short: "Mark an alert as resolved",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Alert %s marked as resolved.\n", args[0])
		return nil
	},
}

func init() {
	alertCmd.AddCommand(alertListCmd)
	alertCmd.AddCommand(alertAckCmd)
	alertCmd.AddCommand(alertResolveCmd)

	rootCmd.AddCommand(alertCmd)
}
