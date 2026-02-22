package cli

import (
	"fmt"
	"os"

	"pantry/internal/core"

	"github.com/spf13/cobra"
)

var retrieveCmd = &cobra.Command{
	Use:   "retrieve [id]",
	Short: "Retrieve full details for a note",
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

		detail, err := svc.GetDetails(itemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if detail == nil {
			fmt.Printf("No details found for note %s\n", itemID)

			return
		}

		fmt.Println(detail.Body)
	},
}
