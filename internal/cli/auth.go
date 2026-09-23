package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		
		tokenStr := strings.TrimSpace(string(byteToken))
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
		
		activeFound := false
		for _, rec := range records {
			if status, ok := rec.Data["status"].(string); ok && status == "revoked" {
				continue
			}
			activeFound = true
			providerName := rec.Data["provider"].(string)
			// Simple mask: show first 4 chars, mask rest
			token := rec.Data["token"].(string)
			mask := "********"
			if len(token) > 4 {
				mask = token[:4] + mask
			}
			fmt.Printf("✓ Logged into %s (token: %s)\n", providerName, mask)
		}
		
		if !activeFound {
			fmt.Println("Not logged into any providers.")
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout [provider]",
	Short: "Log out of a provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerName := args[0]
		records, err := appCtx.Store.List(context.Background(), "Credential")
		if err != nil {
			return err
		}
		
		found := false
		for _, rec := range records {
			if rec.Data["provider"].(string) == providerName {
				rec.Data["status"] = "revoked"
				ref := models.ResourceRef(fmt.Sprintf("Credential/%s", rec.Data["id"]))
				if err := appCtx.Store.Put(context.Background(), ref, rec, rec.Version); err != nil {
					return fmt.Errorf("failed to revoke credential: %v", err)
				}
				found = true
			}
		}
		
		if found {
			fmt.Printf("Logged out of %s.\n", providerName)
		} else {
			fmt.Printf("Not logged into %s.\n", providerName)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
	
	rootCmd.AddCommand(authCmd)
}
