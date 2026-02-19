package search

import (
	"pantry/internal/db"
	"pantry/internal/embeddings"
	"pantry/internal/models"
)

// MergeResults merges FTS5 and vector search results with weighted scoring
func MergeResults(ftsResults []models.SearchResult, vecResults []models.SearchResult, ftsWeight float64, vecWeight float64, limit int) []models.SearchResult {
	// Normalize FTS scores to 0-1
	if len(ftsResults) > 0 {
		maxFTS := ftsResults[0].Score
		for _, r := range ftsResults {
			if r.Score > maxFTS {
				maxFTS = r.Score
			}
		}
		if maxFTS > 0 {
			for i := range ftsResults {
				ftsResults[i].Score = ftsResults[i].Score / maxFTS
			}
		}
	}

	// Normalize vec scores to 0-1
	if len(vecResults) > 0 {
		maxVec := vecResults[0].Score
		for _, r := range vecResults {
			if r.Score > maxVec {
				maxVec = r.Score
			}
		}
		if maxVec > 0 {
			for i := range vecResults {
				vecResults[i].Score = vecResults[i].Score / maxVec
			}
		}
	}

	// Combine with weighted scoring, dedup by id
	scores := make(map[string]*models.SearchResult)
	for _, r := range ftsResults {
		rid := r.ID
		result := r
		result.Score = ftsWeight * r.Score
		scores[rid] = &result
	}
	for _, r := range vecResults {
		rid := r.ID
		if existing, ok := scores[rid]; ok {
			existing.Score += vecWeight * r.Score
		} else {
			result := r
			result.Score = vecWeight * r.Score
			scores[rid] = &result
		}
	}

	// Sort by score descending
	ranked := make([]models.SearchResult, 0, len(scores))
	for _, r := range scores {
		ranked = append(ranked, *r)
	}

	// Simple sort by score (descending)
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[i].Score < ranked[j].Score {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	if len(ranked) > limit {
		return ranked[:limit]
	}
	return ranked
}

// TieredSearch performs FTS-first tiered search that only calls embed when FTS results are sparse
func TieredSearch(db *db.DB, embeddingProvider embeddings.Provider, query string, limit int, minFTSResults int, project *string, source *string) ([]models.SearchResult, error) {
	ftsResults, err := db.FTSSearch(query, limit*2, project, source)
	if err != nil {
		return nil, err
	}

	// Normalize FTS scores to 0-1
	if len(ftsResults) > 0 {
		maxScore := ftsResults[0].Score
		for _, r := range ftsResults {
			if r.Score > maxScore {
				maxScore = r.Score
			}
		}
		if maxScore > 0 {
			for i := range ftsResults {
				ftsResults[i].Score = ftsResults[i].Score / maxScore
			}
		}
	}

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
	queryVec, err := embeddingProvider.Embed(query)
	if err != nil {
		// On any embedding error, return whatever FTS found
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	vecResults, err := db.VectorSearch(queryVec, limit*2, project, source)
	if err != nil {
		// On vector search error, return FTS results
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	return MergeResults(ftsResults, vecResults, 0.3, 0.7, limit), nil
}

// HybridSearch runs FTS5 and optionally vector search, merges results
func HybridSearch(db *db.DB, embeddingProvider embeddings.Provider, query string, limit int, project *string, source *string) ([]models.SearchResult, error) {
	ftsResults, err := db.FTSSearch(query, limit*2, project, source)
	if err != nil {
		return nil, err
	}

	// Normalize FTS scores
	if len(ftsResults) > 0 {
		maxScore := ftsResults[0].Score
		for _, r := range ftsResults {
			if r.Score > maxScore {
				maxScore = r.Score
			}
		}
		if maxScore > 0 {
			for i := range ftsResults {
				ftsResults[i].Score = ftsResults[i].Score / maxScore
			}
		}
	}

	if embeddingProvider == nil {
		// FTS-only mode: return directly
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	queryVec, err := embeddingProvider.Embed(query)
	if err != nil {
		// On embedding error, return FTS results
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	vecResults, err := db.VectorSearch(queryVec, limit*2, project, source)
	if err != nil {
		// On vector search error, return FTS results
		if len(ftsResults) > limit {
			return ftsResults[:limit], nil
		}
		return ftsResults, nil
	}

	return MergeResults(ftsResults, vecResults, 0.3, 0.7, limit), nil
}
