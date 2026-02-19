package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaProvider implements embedding generation using Ollama
type OllamaProvider struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama embedding provider
func NewOllamaProvider(model string, baseURL string) *OllamaProvider {
	return &OllamaProvider{
		model:   model,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{},
	}
}

// Embed generates an embedding vector using Ollama
func (p *OllamaProvider) Embed(text string) ([]float32, error) {
	url := fmt.Sprintf("%s/api/embeddings", p.baseURL)

	requestBody := map[string]interface{}{
		"model": p.model,
		"prompt": text,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Embedding []float64 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert []float64 to []float32
	embedding := make([]float32, len(response.Embedding))
	for i, v := range response.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}
