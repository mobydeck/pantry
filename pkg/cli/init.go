package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"pantry/internal/config"
	"pantry/internal/core"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the pantry",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()

		shelvesDir := filepath.Join(home, "shelves")
		if err := os.MkdirAll(shelvesDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create shelves directory: %v\n", err)
			os.Exit(1)
		}

		// Create default config if missing
		configPath := filepath.Join(home, "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			cfg, _ := config.LoadConfig(configPath) // returns defaults when file missing
			if err := config.SaveConfig(configPath, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create config: %v\n", err)
			}
		}

		// Initialize database (creates index.db and runs migrations)
		if _, err := core.NewService(home); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to initialize database: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Pantry initialized at %s\n", home)
	},
}
