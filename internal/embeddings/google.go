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

// GoogleProvider implements embedding generation using Google's Gemini API.
type GoogleProvider struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGoogleProvider creates a new Google Gemini embedding provider.
func NewGoogleProvider(model string, apiKey string, baseURL string) *GoogleProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &GoogleProvider{
		model:   model,
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client:  &http.Client{},
	}
}

type googleEmbedRequest struct {
	Model   string             `json:"model"`
	Content googleEmbedContent `json:"content"`
}

type googleEmbedContent struct {
	Parts []googleEmbedPart `json:"parts"`
}

type googleEmbedPart struct {
	Text string `json:"text"`
}

type googleEmbedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// Embed generates an embedding vector using Google Gemini API.
func (p *GoogleProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// The Gemini API requires the model name to be prefixed with "models/"
	modelPath := p.model
	if !strings.HasPrefix(modelPath, "models/") {
		modelPath = "models/" + modelPath
	}

	url := fmt.Sprintf("%s/%s:embedContent", p.baseURL, modelPath)

	reqData := googleEmbedRequest{
		Model: modelPath,
		Content: googleEmbedContent{
			Parts: []googleEmbedPart{{Text: text}},
		},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Google API: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response googleEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding values returned by google api")
	}

	return response.Embedding.Values, nil
}
