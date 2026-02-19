package embeddings

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIProvider implements embedding generation using the OpenAI SDK.
// Also works with OpenRouter and other OpenAI-compatible APIs via base_url.
type OpenAIProvider struct {
	model  string
	client openai.Client
}

// NewOpenAIProvider creates a new OpenAI embedding provider.
// baseURL is optional; defaults to https://api.openai.com/v1.
func NewOpenAIProvider(model string, apiKey string, baseURL string) *OpenAIProvider {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(strings.TrimSuffix(baseURL, "/")))
	}
	return &OpenAIProvider{
		model:  model,
		client: openai.NewClient(opts...),
	}
}

// Embed generates an embedding vector using the OpenAI embeddings API.
func (p *OpenAIProvider) Embed(text string) ([]float32, error) {
	resp, err := p.client.Embeddings.New(context.Background(), openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(p.model),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI embedding request failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	raw := resp.Data[0].Embedding
	result := make([]float32, len(raw))
	for i, v := range raw {
		result[i] = float32(v)
	}
	return result, nil
}
