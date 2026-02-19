package embeddings

import (
	"fmt"

	"pantry/internal/config"
)

// NewProvider creates a new embedding provider based on configuration
func NewProvider(cfg config.EmbeddingConfig) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		baseURL := "http://localhost:11434"
		if cfg.BaseURL != nil {
			baseURL = *cfg.BaseURL
		}
		return NewOllamaProvider(cfg.Model, baseURL), nil

	case "openai":
		if cfg.APIKey == nil || *cfg.APIKey == "" {
			return nil, fmt.Errorf("API key required for OpenAI provider")
		}
		baseURL := ""
		if cfg.BaseURL != nil {
			baseURL = *cfg.BaseURL
		}
		return NewOpenAIProvider(cfg.Model, *cfg.APIKey, baseURL), nil

	case "openrouter":
		// OpenRouter uses OpenAI-compatible API
		if cfg.APIKey == nil || *cfg.APIKey == "" {
			return nil, fmt.Errorf("API key required for OpenRouter provider")
		}
		baseURL := "https://openrouter.ai/api/v1"
		if cfg.BaseURL != nil {
			baseURL = *cfg.BaseURL
		}
		return NewOpenAIProvider(cfg.Model, *cfg.APIKey, baseURL), nil

	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", cfg.Provider)
	}
}
