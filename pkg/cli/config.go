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

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration value",
	Long: `Set embedding provider configuration and save to config.yaml.

Examples:
  pantry config set --provider ollama
  pantry config set --provider openai --model text-embedding-3-small --api-key sk-...
  pantry config set --provider openrouter --model openai/text-embedding-3-small --api-key sk-or-...
  pantry config set --api-key sk-...`,
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()
		configPath := filepath.Join(home, "config.yaml")

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		changed := false

		if cmd.Flags().Changed("provider") {
			cfg.Embedding.Provider = configSetProvider
			// Reset model to provider default when switching providers,
			// unless the user also specified a model explicitly.
			if !cmd.Flags().Changed("model") {
				switch configSetProvider {
				case "openai":
					cfg.Embedding.Model = "text-embedding-3-small"
					cfg.Embedding.BaseURL = nil
				case "openrouter":
					cfg.Embedding.Model = "openai/text-embedding-3-small"
					cfg.Embedding.BaseURL = nil
				case "ollama":
					cfg.Embedding.Model = "nomic-embed-text"
					base := "http://localhost:11434"
					cfg.Embedding.BaseURL = &base
					cfg.Embedding.APIKey = nil
				}
			}

			changed = true
		}

		if cmd.Flags().Changed("model") {
			cfg.Embedding.Model = configSetModel
			changed = true
		}

		if cmd.Flags().Changed("api-key") {
			cfg.Embedding.APIKey = &configSetAPIKey
			changed = true
		}

		if cmd.Flags().Changed("base-url") {
			cfg.Embedding.BaseURL = &configSetBaseURL
			changed = true
		}

		if !changed {
			fmt.Fprintln(os.Stderr, "No flags provided. Use --provider, --model, --api-key, or --base-url.")
			os.Exit(1)
		}

		if err := config.SaveConfig(configPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated %s\n", configPath)
		fmt.Printf("  provider: %s\n", cfg.Embedding.Provider)
		fmt.Printf("  model:    %s\n", cfg.Embedding.Model)

		if cfg.Embedding.BaseURL != nil {
			fmt.Printf("  base_url: %s\n", *cfg.Embedding.BaseURL)
		}

		if cfg.Embedding.APIKey != nil {
			fmt.Printf("  api_key:  <set>\n")
		}
	},
}

var (
	configSetProvider string
	configSetModel    string
	configSetAPIKey   string
	configSetBaseURL  string
)

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configInitCmd.Flags().BoolVarP(&configInitForce, "force", "f", false, "Overwrite existing config")
	configSetCmd.Flags().StringVar(&configSetProvider, "provider", "", "Embedding provider (ollama, openai, openrouter)")
	configSetCmd.Flags().StringVar(&configSetModel, "model", "", "Embedding model name")
	configSetCmd.Flags().StringVar(&configSetAPIKey, "api-key", "", "API key for the embedding provider")
	configSetCmd.Flags().StringVar(&configSetBaseURL, "base-url", "", "Base URL for the embedding API")
}
