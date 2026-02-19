package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider implements embedding generation using OpenAI API
type OpenAIProvider struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI embedding provider
func NewOpenAIProvider(model string, apiKey string, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		model:   model,
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{},
	}
}

// Embed generates an embedding vector using OpenAI API
func (p *OpenAIProvider) Embed(text string) ([]float32, error) {
	url := fmt.Sprintf("%s/embeddings", p.baseURL)

	requestBody := map[string]interface{}{
		"model": p.model,
		"input": text,
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	// Convert []float64 to []float32
	embedding := make([]float32, len(response.Data[0].Embedding))
	for i, v := range response.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}
