package cli

import (
	"fmt"
	"os"

	"pantry/internal/core"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove a note from the pantry",
	Args:  cobra.ExactArgs(1),
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		itemID := args[0]

		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		defer func() { _ = svc.Close() }()

		deleted, err := svc.Remove(itemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if deleted {
			fmt.Printf("Removed note %s\n", itemID)
		} else {
			fmt.Printf("No note found for %s\n", itemID)
		}
	},
}
