package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaProvider implements embedding generation using Ollama.
type OllamaProvider struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama embedding provider.
func NewOllamaProvider(model string, baseURL string) *OllamaProvider {
	return &OllamaProvider{
		model:   model,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{},
	}
}

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed generates an embedding vector using Ollama.
func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	url := p.baseURL + "/api/embeddings"

	jsonData, err := json.Marshal(ollamaEmbedRequest{Model: p.model, Prompt: text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embedding := make([]float32, len(response.Embedding))
	for i, v := range response.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}
