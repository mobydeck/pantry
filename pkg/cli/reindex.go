package cli

import (
	"fmt"
	"os"

	"pantry/internal/core"

	"github.com/spf13/cobra"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild vector index with current embedding provider",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		defer func() { _ = svc.Close() }()

		// Check if there are any notes
		// Simplified - would need to get count from service
		fmt.Println("Reindexing notes...")

		progressCallback := func(current, total int) {
			fmt.Printf("  %d/%d\r", current, total)

			if current == total {
				fmt.Println()
			}
		}

		result, err := svc.Reindex(progressCallback)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Re-indexed %v notes with %v (%v dims)\n",
			result["count"], result["model"], result["dim"])
	},
}
