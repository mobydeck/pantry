package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pantry/internal/core"
)

var retrieveCmd = &cobra.Command{
	Use:   "retrieve [id]",
	Short: "Retrieve full details for an item",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		itemID := args[0]

		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer svc.Close()

		detail, err := svc.GetDetails(itemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if detail == nil {
			fmt.Printf("No details found for item %s\n", itemID)
			return
		}

		fmt.Println(detail.Body)
	},
}
