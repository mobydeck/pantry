package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "pantry",
	Short:   "Pantry - local notes for coding agents",
	Long:    `Pantry provides local-first note storage for coding agents. Store, search, and retrieve decisions, patterns, bugs, and context across sessions.`,
	Version: Version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(storeCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(retrieveCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(mcpCmd)
}
