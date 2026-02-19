package embeddings

// Provider defines the interface for embedding providers
type Provider interface {
	// Embed generates an embedding vector for the given text
	Embed(text string) ([]float32, error)
}
