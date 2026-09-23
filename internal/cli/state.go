package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"zavictl/pkg/models"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage and inspect platform state",
}

var getCmd = &cobra.Command{
	Use:   "get [kind] [id]",
	Short: "Get a state record by kind and id",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := args[0]
		id := args[1]
		
		ref := models.ResourceRef(fmt.Sprintf("%s/%s", kind, id))
		rec, err := appCtx.Store.Get(context.Background(), ref)
		if err != nil {
			return fmt.Errorf("failed to get state: %v", err)
		}

		if outputFormat == "json" {
			b, _ := json.MarshalIndent(rec, "", "  ")
			fmt.Println(string(b))
		} else if outputFormat == "yaml" || outputFormat == "yml" {
			b, _ := yaml.Marshal(rec)
			fmt.Println(string(b))
		} else {
			// default to yaml for state inspect
			b, _ := yaml.Marshal(rec)
			fmt.Println(string(b))
		}
		
		return nil
	},
}

func init() {
	stateCmd.AddCommand(getCmd)
	rootCmd.AddCommand(stateCmd)
}
