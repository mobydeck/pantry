package cli

import (
	"fmt"
	"os"

	"pantry/internal/mcp"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Pantry MCP server (stdio transport)",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		if err := mcp.RunServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}
