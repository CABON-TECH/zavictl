package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"zavictl/pkg/models"
	"zavictl/pkg/state"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with providers",
}

var authLoginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Authenticate with a specific provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := args[0]
		
		fmt.Printf("Enter token for %s: ", providerName)
		byteToken, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		fmt.Println()
		
		tokenStr := string(byteToken)
		if tokenStr == "" {
			return fmt.Errorf("token cannot be empty")
		}

		cred := models.NewCredential(providerName, tokenStr, "local-user")
		
		b, _ := json.Marshal(cred)
		var data map[string]any
		json.Unmarshal(b, &data)

		rec := state.StateRecord{
			Version:   1,
			UpdatedAt: cred.UpdatedAt,
			Data:      data,
		}

		// Delete existing credential for this provider if it exists
		// In a real app we'd update it or scope it properly, but for now we'll just overwrite
		if err := appCtx.Store.Put(context.Background(), cred.Identity(), rec, 0); err != nil {
			return fmt.Errorf("failed to save credential: %v", err)
		}

		fmt.Printf("Successfully logged into %s.\n", providerName)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "View authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := appCtx.Store.List(context.Background(), "Credential")
		if err != nil {
			return fmt.Errorf("failed to list credentials: %v", err)
		}
		
		if len(records) == 0 {
			fmt.Println("Not logged into any providers.")
			return nil
		}
		
		for _, rec := range records {
			providerName := rec.Data["provider"].(string)
			fmt.Printf("✓ Logged into %s\n", providerName)
		}
		
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	
	rootCmd.AddCommand(authCmd)
}
