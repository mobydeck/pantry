package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"pantry/internal/config"
	"pantry/internal/db"
	"pantry/internal/embeddings"
	"pantry/internal/models"
	"pantry/internal/redaction"
	"pantry/internal/search"
	"pantry/internal/storage"
)

const (
	// DedupScoreThreshold is the minimum normalized FTS score (0–1) combined
	// with an exact title match required to treat a new store as an update.
	DedupScoreThreshold = 0.7
)

// Option is a functional option for NewService.
type Option func(*Service)

// WithStore injects a custom db.Store implementation, primarily for testing.
func WithStore(s db.Store) Option {
	return func(svc *Service) { svc.db = s }
}

// Service is the main orchestrator for pantry operations
type Service struct {
	pantryHome          string
	shelvesDir            string
	dbPath              string
	configPath          string
	ignorePath          string
	config              *config.Config
	db                  db.Store
	compiledIgnore      []*regexp.Regexp // pre-compiled from .pantryignore

	// Lazy-initialized, protected by sync.Once for safety under concurrent access.
	embeddingOnce     sync.Once
	embeddingProvider embeddings.Provider
	embeddingErr      error

	vectorsOnce      sync.Once
	vectorsAvailable bool
}

// NewService creates a new pantry service. Pass Option values to override
// defaults (e.g., WithStore for testing).
func NewService(pantryHome string, opts ...Option) (*Service, error) {
	if pantryHome == "" {
		pantryHome = config.GetPantryHome()
	}

	shelvesDir := filepath.Join(pantryHome, "shelves")
	dbPath := filepath.Join(pantryHome, "index.db")
	configPath := filepath.Join(pantryHome, "config.yaml")
	ignorePath := filepath.Join(pantryHome, ".pantryignore")

	// Ensure shelves directory exists
	if err := os.MkdirAll(shelvesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create shelves directory: %w", err)
	}

	// Load and validate configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize database
	database, err := db.NewDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Load ignore patterns (.pantryignore missing is fine; other errors are surfaced)
	ignorePatterns, ignoreErr := redaction.LoadPantryIgnore(ignorePath)
	if ignoreErr != nil && !os.IsNotExist(ignoreErr) {
		fmt.Fprintf(os.Stderr, "warning: failed to load .pantryignore: %v\n", ignoreErr)
	}

	svc := &Service{
		pantryHome:     pantryHome,
		shelvesDir:       shelvesDir,
		dbPath:         dbPath,
		configPath:     configPath,
		ignorePath:     ignorePath,
		config:         cfg,
		db:             database,
		compiledIgnore: redaction.CompilePatterns(ignorePatterns),
	}

	for _, o := range opts {
		o(svc)
	}

	return svc, nil
}

// GetEmbeddingProvider returns the embedding provider, lazily initializing if needed.
// Safe for concurrent use.
func (s *Service) GetEmbeddingProvider() (embeddings.Provider, error) {
	s.embeddingOnce.Do(func() {
		s.embeddingProvider, s.embeddingErr = embeddings.NewProvider(s.config.Embedding)
	})
	return s.embeddingProvider, s.embeddingErr
}

// VectorsAvailable checks if vector operations are available.
// Safe for concurrent use.
func (s *Service) VectorsAvailable() bool {
	s.vectorsOnce.Do(func() {
		s.vectorsAvailable = s.db.HasVecTable()
	})
	return s.vectorsAvailable
}

// CountItems returns the total number of stored notes, optionally filtered.
func (s *Service) CountItems(project *string, source *string) (int64, error) {
	return s.db.CountItems(project, source)
}

// Store stores an item in the pantry
func (s *Service) Store(raw models.RawItemInput, project string) (map[string]interface{}, error) {
	if project == "" {
		project = filepath.Base(getCurrentDir())
	}

	today := time.Now().UTC().Format("2006-01-02")
	projectDir := filepath.Join(s.shelvesDir, project)

	// Ensure project directory exists
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Redact all text fields using pre-compiled patterns
	raw.What = redaction.RedactCompiled(raw.What, s.compiledIgnore)
	if raw.Why != nil {
		redacted := redaction.RedactCompiled(*raw.Why, s.compiledIgnore)
		raw.Why = &redacted
	}
	if raw.Impact != nil {
		redacted := redaction.RedactCompiled(*raw.Impact, s.compiledIgnore)
		raw.Impact = &redacted
	}
	if raw.Details != nil {
		redacted := redaction.RedactCompiled(*raw.Details, s.compiledIgnore)
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
			if normalized >= DedupScoreThreshold && titleMatch {
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

				return map[string]interface{}{
					"id":        existingID,
					"file_path": existingFilePath,
					"action":    "updated",
				}, nil
			}
		}
	}

	// Normal save path: create new item
	filePath := filepath.Join(projectDir, fmt.Sprintf("%s-notes.md", today))
	item := models.FromRaw(raw, project, filePath)

	// Write markdown file
	if _, err := storage.WriteNoteItem(projectDir, item, today, raw.Details); err != nil {
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
			}
		}
	}

	return map[string]interface{}{
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
	return search.TieredSearch(s.db, provider, query, limit, search.DefaultMinFTSResults, project, source)
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

// getCurrentDir returns the current working directory, or "unknown" if it
// cannot be determined. This prevents filepath.Base("") returning "." which
// would silently be stored as a project name.
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
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
