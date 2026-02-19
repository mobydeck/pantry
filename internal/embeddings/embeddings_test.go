package embeddings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pantry/internal/config"
)

// --- OllamaProvider tests ---

func TestOllamaProvider_Embed_Success(t *testing.T) {
	// Fake Ollama server that returns a fixed embedding
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Verify request body has "model" and "prompt"
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["model"] == nil {
			t.Error("request body missing 'model'")
		}
		if body["prompt"] == nil {
			t.Error("request body missing 'prompt'")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": []float64{0.1, 0.2, 0.3},
		})
	}))
	defer srv.Close()

	p := NewOllamaProvider("nomic-embed-text", srv.URL)
	embedding, err := p.Embed("hello world")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embedding) != 3 {
		t.Errorf("embedding len = %d, want 3", len(embedding))
	}
	if embedding[0] != float32(0.1) {
		t.Errorf("embedding[0] = %f, want 0.1", embedding[0])
	}
}

func TestOllamaProvider_Embed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOllamaProvider("model", srv.URL)
	_, err := p.Embed("text")
	if err == nil {
		t.Fatal("Embed() should return error on non-200 response")
	}
}

func TestOllamaProvider_Embed_ConnectionRefused(t *testing.T) {
	// Point at a port that isn't listening
	p := NewOllamaProvider("model", "http://127.0.0.1:1")
	_, err := p.Embed("text")
	if err == nil {
		t.Fatal("Embed() should return error when connection is refused")
	}
}

// --- OpenAIProvider tests ---

func TestOpenAIProvider_Embed_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-key")
		}
		if r.URL.Path != "/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float64{0.4, 0.5, 0.6}},
			},
		})
	}))
	defer srv.Close()

	p := NewOpenAIProvider("text-embedding-3-small", "test-key", srv.URL)
	embedding, err := p.Embed("hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(embedding) != 3 {
		t.Errorf("embedding len = %d, want 3", len(embedding))
	}
}

func TestOpenAIProvider_Embed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := NewOpenAIProvider("model", "bad-key", srv.URL)
	_, err := p.Embed("text")
	if err == nil {
		t.Fatal("Embed() should return error on non-200 response")
	}
}

func TestOpenAIProvider_Embed_EmptyDataArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer srv.Close()

	p := NewOpenAIProvider("model", "key", srv.URL)
	_, err := p.Embed("text")
	if err == nil {
		t.Fatal("Embed() should return error when data array is empty")
	}
}

// --- Factory tests ---

func TestNewProvider_Ollama(t *testing.T) {
	cfg := config.EmbeddingConfig{
		Provider: "ollama",
		Model:    "nomic-embed-text",
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider(ollama) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider(ollama) returned nil")
	}
}

func TestNewProvider_OpenAI_RequiresAPIKey(t *testing.T) {
	cfg := config.EmbeddingConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider(openai) without API key should return error")
	}
}

func TestNewProvider_OpenAI_WithAPIKey(t *testing.T) {
	key := "sk-test"
	cfg := config.EmbeddingConfig{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		APIKey:   &key,
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider(openai) error = %v", err)
	}
	if p == nil {
		t.Fatal("NewProvider(openai) returned nil")
	}
}

func TestNewProvider_OpenRouter_RequiresAPIKey(t *testing.T) {
	cfg := config.EmbeddingConfig{
		Provider: "openrouter",
		Model:    "some-model",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider(openrouter) without API key should return error")
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	cfg := config.EmbeddingConfig{
		Provider: "bogus",
		Model:    "model",
	}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider(bogus) should return error for unknown provider")
	}
}
