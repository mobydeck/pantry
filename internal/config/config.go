package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EmbeddingConfig holds embedding provider configuration
type EmbeddingConfig struct {
	Provider string  `yaml:"provider"`
	Model    string  `yaml:"model"`
	BaseURL  *string `yaml:"base_url"`
	APIKey   *string `yaml:"api_key"`
}

// ContextConfig holds context retrieval configuration
type ContextConfig struct {
	Semantic   string `yaml:"semantic"`    // auto | always | never
	TopupRecent bool  `yaml:"topup_recent"`
}

// Config holds the complete configuration
type Config struct {
	Embedding EmbeddingConfig `yaml:"embedding"`
	Context   ContextConfig   `yaml:"context"`
}

// GetPantryHome returns the pantry home directory
func GetPantryHome() string {
	if home := os.Getenv("PANTRY_HOME"); home != "" {
		return home
	}
	userHome, _ := os.UserHomeDir()
	return filepath.Join(userHome, ".pantry")
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	config := &Config{
		Embedding: EmbeddingConfig{
			Provider: "ollama",
			Model:    "nomic-embed-text",
			BaseURL:  stringPtr("http://localhost:11434"),
		},
		Context: ContextConfig{
			Semantic:   "auto",
			TopupRecent: true,
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if file doesn't exist
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure defaults are set
	if config.Embedding.Provider == "" {
		config.Embedding.Provider = "ollama"
	}
	if config.Embedding.Model == "" {
		config.Embedding.Model = "nomic-embed-text"
	}
	if config.Embedding.BaseURL == nil {
		config.Embedding.BaseURL = stringPtr("http://localhost:11434")
	}
	if config.Context.Semantic == "" {
		config.Context.Semantic = "auto"
	}

	// Environment variable overrides (take precedence over file values).
	// Useful for MCP servers launched by host applications that inject secrets
	// via the environment rather than writing them to disk.
	if v := os.Getenv("PANTRY_EMBEDDING_PROVIDER"); v != "" {
		config.Embedding.Provider = v
	}
	if v := os.Getenv("PANTRY_EMBEDDING_MODEL"); v != "" {
		config.Embedding.Model = v
	}
	if v := os.Getenv("PANTRY_EMBEDDING_API_KEY"); v != "" {
		config.Embedding.APIKey = &v
	}
	if v := os.Getenv("PANTRY_EMBEDDING_BASE_URL"); v != "" {
		config.Embedding.BaseURL = &v
	}
	if v := os.Getenv("PANTRY_CONTEXT_SEMANTIC"); v != "" {
		config.Context.Semantic = v
	}

	return config, nil
}

// Validate returns an error if the configuration contains invalid values.
// Call this after LoadConfig to surface misconfiguration at startup.
func (c *Config) Validate() error {
	validProviders := map[string]bool{"ollama": true, "openai": true, "openrouter": true}
	if !validProviders[c.Embedding.Provider] {
		return fmt.Errorf("invalid embedding.provider %q: must be one of ollama, openai, openrouter", c.Embedding.Provider)
	}
	if c.Embedding.Model == "" {
		return fmt.Errorf("embedding.model must not be empty")
	}
	validSemantic := map[string]bool{"auto": true, "always": true, "never": true}
	if !validSemantic[c.Context.Semantic] {
		return fmt.Errorf("invalid context.semantic %q: must be one of auto, always, never", c.Context.Semantic)
	}
	if c.Embedding.Provider == "openai" || c.Embedding.Provider == "openrouter" {
		if c.Embedding.APIKey == nil || *c.Embedding.APIKey == "" {
			return fmt.Errorf("embedding.api_key is required for provider %q", c.Embedding.Provider)
		}
	}
	return nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfigTemplate returns a default config template as a string
func GetDefaultConfigTemplate() string {
	return `# Pantry configuration
# Docs: https://github.com/your-org/pantry

# Embedding provider for semantic search.
# Without this, keyword search (FTS5) still works.
embedding:
  provider: ollama              # ollama | openai | openrouter
  model: nomic-embed-text
  base_url: http://localhost:11434
  # api_key: sk-...            # required for openai/openrouter

# How items are retrieved at session start.
# "auto" uses vectors when available, falls back to keywords.
context:
  semantic: auto                # auto | always | never
  topup_recent: true            # also include recent items
`
}

func stringPtr(s string) *string {
	return &s
}
