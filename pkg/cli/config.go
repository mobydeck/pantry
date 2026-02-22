package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"pantry/internal/config"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var configInitForce bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or manage configuration",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()
		configPath := filepath.Join(home, "config.yaml")

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Redact API keys
		cfgCopy := *cfg
		if cfgCopy.Embedding.APIKey != nil {
			redacted := "<redacted>"
			cfgCopy.Embedding.APIKey = &redacted
		}

		data, err := yaml.Marshal(&cfgCopy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("pantry_home: %s\n", home)
		fmt.Println(string(data))
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter config.yaml",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()
		configPath := filepath.Join(home, "config.yaml")

		if _, err := os.Stat(configPath); err == nil && !configInitForce {
			fmt.Printf("Config already exists at %s\n", configPath)
			fmt.Println("Use --force to overwrite.")

			return
		}

		if err := os.MkdirAll(home, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create config directory: %v\n", err)
			os.Exit(1)
		}

		template := config.GetDefaultConfigTemplate()
		if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to write config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created %s\n", configPath)
		fmt.Println("Edit the file to configure your embedding provider.")
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configInitCmd.Flags().BoolVarP(&configInitForce, "force", "f", false, "Overwrite existing config")
}
