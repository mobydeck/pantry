package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pantry/internal/config"
	"pantry/internal/db"
	"pantry/internal/embeddings"
	"pantry/internal/models"
	"pantry/internal/redaction"
	"pantry/internal/search"
	"pantry/internal/storage"
)

// Service is the main orchestrator for pantry operations
type Service struct {
	pantryHome   string
	vaultDir     string
	dbPath       string
	configPath   string
	ignorePath   string
	config       *config.Config
	db           *db.DB
	embeddingProvider embeddings.Provider
	ignorePatterns    []string
	vectorsAvailable  *bool
}

// NewService creates a new pantry service
func NewService(pantryHome string) (*Service, error) {
	if pantryHome == "" {
		pantryHome = config.GetPantryHome()
	}

	vaultDir := filepath.Join(pantryHome, "shelf")
	dbPath := filepath.Join(pantryHome, "index.db")
	configPath := filepath.Join(pantryHome, "config.yaml")
	ignorePath := filepath.Join(pantryHome, ".pantryignore")

	// Ensure vault directory exists
	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize database
	database, err := db.NewDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Load ignore patterns
	ignorePatterns, _ := redaction.LoadPantryIgnore(ignorePath)

	return &Service{
		pantryHome:     pantryHome,
		vaultDir:       vaultDir,
		dbPath:         dbPath,
		configPath:     configPath,
		ignorePath:     ignorePath,
		config:         cfg,
		db:            database,
		ignorePatterns: ignorePatterns,
	}, nil
}

// GetEmbeddingProvider returns the embedding provider, lazily initializing if needed
func (s *Service) GetEmbeddingProvider() (embeddings.Provider, error) {
	if s.embeddingProvider == nil {
		provider, err := embeddings.NewProvider(s.config.Embedding)
		if err != nil {
			return nil, err
		}
		s.embeddingProvider = provider
	}
	return s.embeddingProvider, nil
}

// VectorsAvailable checks if vector operations are available
func (s *Service) VectorsAvailable() bool {
	if s.vectorsAvailable == nil {
		available := s.db.HasVecTable()
		s.vectorsAvailable = &available
	}
	return *s.vectorsAvailable
}

// Store stores an item in the pantry
func (s *Service) Store(raw models.RawItemInput, project string) (map[string]string, error) {
	if project == "" {
		project = filepath.Base(getCurrentDir())
	}

	today := time.Now().UTC().Format("2006-01-02")
	vaultProjectDir := filepath.Join(s.vaultDir, project)

	// Ensure project directory exists
	if err := os.MkdirAll(vaultProjectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Redact all text fields
	raw.What = redaction.Redact(raw.What, s.ignorePatterns)
	if raw.Why != nil {
		redacted := redaction.Redact(*raw.Why, s.ignorePatterns)
		raw.Why = &redacted
	}
	if raw.Impact != nil {
		redacted := redaction.Redact(*raw.Impact, s.ignorePatterns)
		raw.Impact = &redacted
	}
	if raw.Details != nil {
		redacted := redaction.Redact(*raw.Details, s.ignorePatterns)
		raw.Details = &redacted
	}

	// Dedup check: look for similar existing item in same project
	dedupQuery := fmt.Sprintf("%s %s", raw.Title, raw.What)
	candidates, err := s.db.FTSSearch(dedupQuery, 5, &project, nil)
	if err == nil && len(candidates) > 0 {
		// Normalize score
		broad, _ := s.db.FTSSearch(dedupQuery, 5, nil, nil)
		maxScore := 0.0
		if len(broad) > 0 {
			maxScore = broad[0].Score
		}
		if len(candidates) > 0 {
			top := candidates[0]
			normalized := 0.0
			if maxScore > 0 {
				normalized = top.Score / maxScore
			}
			titleMatch := strings.EqualFold(strings.TrimSpace(raw.Title), strings.TrimSpace(top.Title))
			if normalized >= 0.7 && titleMatch {
				// Update existing item
				existingID := top.ID
				existingFilePath := top.FilePath

				// Merge tags
				mergedTags := mergeTags(top.Tags, raw.Tags)

				detailsAppend := ""
				if raw.Details != nil {
					detailsAppend = fmt.Sprintf("--- updated %s ---\n%s", today, *raw.Details)
				}

				err := s.db.UpdateItem(existingID, &raw.What, raw.Why, raw.Impact, mergedTags, &detailsAppend)
				if err != nil {
					return nil, fmt.Errorf("failed to update item: %w", err)
				}

				// Re-embed the updated item (if needed)
				// Note: Re-embedding on update would require rowid lookup
				// For now, skip re-embedding on update to avoid complexity
				_, _ = s.GetEmbeddingProvider() // Ensure provider is initialized

				return map[string]string{
					"id":        existingID,
					"file_path": existingFilePath,
					"action":    "updated",
				}, nil
			}
		}
	}

	// Normal save path: create new item
	filePath := filepath.Join(vaultProjectDir, fmt.Sprintf("%s-session.md", today))
	item := models.FromRaw(raw, project, filePath)

	// Write markdown file
	if _, err := storage.WriteSessionItem(vaultProjectDir, item, today, raw.Details); err != nil {
		return nil, fmt.Errorf("failed to write session file: %w", err)
	}

	// Insert into database
	rowid, err := s.db.InsertItem(item, raw.Details)
	if err != nil {
		return nil, fmt.Errorf("failed to insert item: %w", err)
	}

	// Generate and store embedding
	provider, err := s.GetEmbeddingProvider()
	if err == nil {
		embedText := fmt.Sprintf("%s %s %s %s %s", item.Title, item.What, getString(item.Why), getString(item.Impact), strings.Join(item.Tags, " "))
		embedding, err := provider.Embed(embedText)
		if err == nil {
			if err := s.db.EnsureVecTable(len(embedding)); err == nil {
				s.db.InsertVector(rowid, embedding)
				if s.vectorsAvailable != nil {
					*s.vectorsAvailable = true
				}
			}
		}
	}

	return map[string]string{
		"id":        item.ID,
		"file_path": filePath,
		"action":    "created",
	}, nil
}

// Search searches items using hybrid FTS + vector search
func (s *Service) Search(query string, limit int, project *string, source *string, useVectors bool) ([]models.SearchResult, error) {
	provider, err := s.GetEmbeddingProvider()
	if err != nil || !useVectors || !s.VectorsAvailable() {
		// FTS-only path
		return s.db.FTSSearch(query, limit, project, source)
	}

	// Use tiered search: FTS first, embed only if sparse results
	return search.TieredSearch(s.db, provider, query, limit, 3, project, source)
}

// GetContext gets item pointers for context injection
func (s *Service) GetContext(limit int, project *string, source *string, query *string, semanticMode string, topupRecent bool) ([]models.SearchResult, int64, error) {
	total, err := s.db.CountItems(project, source)
	if err != nil {
		return nil, 0, err
	}

	var results []models.SearchResult
	if query != nil {
		useVectors := false
		if semanticMode == "always" {
			useVectors = true
		} else if semanticMode == "auto" && s.VectorsAvailable() {
			useVectors = true
		}

		results, err = s.Search(*query, limit, project, source, useVectors)
		if err != nil {
			return nil, 0, err
		}

		if topupRecent && len(results) < limit {
			recent, err := s.db.ListRecent(limit, project, source)
			if err == nil {
				seen := make(map[string]bool)
				for _, r := range results {
					seen[r.ID] = true
				}
				for _, r := range recent {
					if !seen[r.ID] {
						results = append(results, r)
						if len(results) >= limit {
							break
						}
					}
				}
			}
		}
	} else {
		results, err = s.db.ListRecent(limit, project, source)
		if err != nil {
			return nil, 0, err
		}
	}

	return results, total, nil
}

// GetDetails gets full details for an item
func (s *Service) GetDetails(itemID string) (*models.ItemDetail, error) {
	return s.db.GetDetails(itemID)
}

// Remove removes an item from pantry
func (s *Service) Remove(itemID string) (bool, error) {
	return s.db.DeleteItem(itemID)
}

// Reindex rebuilds the vector table with current embedding provider
func (s *Service) Reindex(progressCallback func(current, total int)) (map[string]interface{}, error) {
	provider, err := s.GetEmbeddingProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding provider: %w", err)
	}

	// Detect dimension from provider
	probe, err := provider.Embed("dimension probe")
	if err != nil {
		return nil, fmt.Errorf("failed to probe embedding dimension: %w", err)
	}
	dim := len(probe)

	// Drop and recreate vec table
	s.db.DropVecTable()
	if err := s.db.SetEmbeddingDim(dim); err != nil {
		return nil, err
	}
	if err := s.db.EnsureVecTable(dim); err != nil {
		return nil, err
	}

	// Re-embed all items
	items, err := s.db.ListAllForReindex()
	if err != nil {
		return nil, err
	}
	total := len(items)

	for i, item := range items {
		tags := ""
		if tagsVal, ok := item["tags"].([]string); ok {
			tags = strings.Join(tagsVal, " ")
		}

		embedText := fmt.Sprintf("%s %s %s %s %s",
			getStringFromMap(item, "title"),
			getStringFromMap(item, "what"),
			getStringFromMap(item, "why"),
			getStringFromMap(item, "impact"),
			tags)

		embedding, err := provider.Embed(embedText)
		if err != nil {
			continue
		}

		rowid := item["rowid"].(int64)
		s.db.InsertVector(rowid, embedding)

		if progressCallback != nil {
			progressCallback(i+1, total)
		}
	}

	if s.vectorsAvailable != nil {
		*s.vectorsAvailable = true
	}

	return map[string]interface{}{
		"count": total,
		"dim":   dim,
		"model": s.config.Embedding.Model,
	}, nil
}

// Close closes the service and cleans up resources
func (s *Service) Close() error {
	return s.db.Close()
}

// Helper functions
func getCurrentDir() string {
	dir, _ := os.Getwd()
	return dir
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func mergeTags(existing []string, extra []string) []string {
	combined := make([]string, len(existing))
	copy(combined, existing)
	existingNorm := make(map[string]bool)
	for _, t := range existing {
		existingNorm[strings.ToLower(t)] = true
	}
	for _, tag := range extra {
		if !existingNorm[strings.ToLower(tag)] {
			combined = append(combined, tag)
			existingNorm[strings.ToLower(tag)] = true
		}
	}
	return combined
}
