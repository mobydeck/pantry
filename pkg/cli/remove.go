package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pantry/internal/core"
)

var removeCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove an item from pantry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		itemID := args[0]

		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer svc.Close()

		deleted, err := svc.Remove(itemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if deleted {
			fmt.Printf("Removed item %s\n", itemID)
		} else {
			fmt.Printf("No item found for %s\n", itemID)
		}
	},
}
