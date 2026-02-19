package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pantry/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Pantry MCP server (stdio transport)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := mcp.RunServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}
