package search

import (
	"context"
	"sort"

	"pantry/internal/db"
	"pantry/internal/embeddings"
	"pantry/internal/models"
)

const (
	DefaultFTSWeight     = 0.3
	DefaultVecWeight     = 0.7
	DefaultMinFTSResults = 3
)

// normalizeScores scales all scores so the maximum score becomes 1.0.
// It mutates the slice in place. If the slice is empty or all scores are
// zero the slice is returned unchanged.
func normalizeScores(results []models.SearchResult) {
	if len(results) == 0 {
		return
	}
	max := results[0].Score
	for _, r := range results {
		if r.Score > max {
			max = r.Score
		}
	}
	if max <= 0 {
		return
	}
	for i := range results {
		results[i].Score /= max
	}
}

// MergeResults merges FTS5 and vector search results with weighted scoring.
// Both input slices are normalized to 0–1 before weighting.
func MergeResults(ftsResults []models.SearchResult, vecResults []models.SearchResult, ftsWeight float64, vecWeight float64, limit int) []models.SearchResult {
	normalizeScores(ftsResults)
	normalizeScores(vecResults)

	// Combine with weighted scoring, dedup by ID
	scores := make(map[string]*models.SearchResult)
	for _, r := range ftsResults {
		result := r
		result.Score = ftsWeight * r.Score
		scores[r.ID] = &result
	}
	for _, r := range vecResults {
		if existing, ok := scores[r.ID]; ok {
			existing.Score += vecWeight * r.Score
		} else {
			result := r
			result.Score = vecWeight * r.Score
			scores[r.ID] = &result
		}
	}

	ranked := make([]models.SearchResult, 0, len(scores))
	for _, r := range scores {
		ranked = append(ranked, *r)
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > limit {
		return ranked[:limit]
	}
	return ranked
}

// TieredSearch performs FTS-first tiered search that only calls embed when FTS results are sparse
func TieredSearch(ctx context.Context, store db.Store, embeddingProvider embeddings.Provider, query string, limit int, minFTSResults int, project *string, source *string) ([]models.SearchResult, error) {
	ftsResults, err := store.FTSSearch(query, limit*2, project, source)
	if err != nil {
		return nil, err
	}

	normalizeScores(ftsResults)

	// If FTS has enough results, return without calling embed
	if len(ftsResults) >= minFTSResults {
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	// If no embedding provider, return FTS-only
	if embeddingProvider == nil {
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	// FTS results are sparse — fall back to hybrid (embed + vector search + merge)
	queryVec, err := embeddingProvider.Embed(ctx, query)
	if err != nil {
		// On any embedding error, return whatever FTS found
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	vecResults, err := store.VectorSearch(queryVec, limit*2, project, source)
	if err != nil {
		// On vector search error, return FTS results
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	return MergeResults(ftsResults, vecResults, DefaultFTSWeight, DefaultVecWeight, limit), nil
}

// HybridSearch runs FTS5 and optionally vector search, merges results
func HybridSearch(ctx context.Context, store db.Store, embeddingProvider embeddings.Provider, query string, limit int, project *string, source *string) ([]models.SearchResult, error) {
	ftsResults, err := store.FTSSearch(query, limit*2, project, source)
	if err != nil {
		return nil, err
	}

	normalizeScores(ftsResults)

	if embeddingProvider == nil {
		// FTS-only mode: return directly
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	queryVec, err := embeddingProvider.Embed(ctx, query)
	if err != nil {
		// On embedding error, return FTS results
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	vecResults, err := store.VectorSearch(queryVec, limit*2, project, source)
	if err != nil {
		// On vector search error, return FTS results
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	return MergeResults(ftsResults, vecResults, DefaultFTSWeight, DefaultVecWeight, limit), nil
}
