package search

import (
	"errors"
	"testing"

	"pantry/internal/models"
)

// --- fakeStore implements db.Store for testing ---

type fakeStore struct {
	ftsResults []models.SearchResult
	ftsErr     error
	vecResults []models.SearchResult
	vecErr     error
	ftsCalled  int
	vecCalled  int
}

func (f *fakeStore) FTSSearch(_ string, _ int, _ *string, _ *string) ([]models.SearchResult, error) {
	f.ftsCalled++
	return f.ftsResults, f.ftsErr
}
func (f *fakeStore) VectorSearch(_ []float32, _ int, _ *string, _ *string) ([]models.SearchResult, error) {
	f.vecCalled++
	return f.vecResults, f.vecErr
}

// Unused interface methods — zero-value implementations.
func (f *fakeStore) InsertItem(_ models.Item, _ *string) (int64, error)           { return 0, nil }
func (f *fakeStore) InsertVector(_ int64, _ []float32) error                       { return nil }
func (f *fakeStore) GetItem(_ string) (*models.Item, bool, error)                  { return nil, false, nil }
func (f *fakeStore) GetDetails(_ string) (*models.ItemDetail, error)               { return nil, nil }
func (f *fakeStore) UpdateItem(_ string, _ *string, _ *string, _ *string, _ []string, _ *string) error {
	return nil
}
func (f *fakeStore) DeleteItem(_ string) (bool, error)                { return false, nil }
func (f *fakeStore) ListRecent(_ int, _ *string, _ *string) ([]models.SearchResult, error) {
	return nil, nil
}
func (f *fakeStore) ListAllForReindex() ([]map[string]interface{}, error) { return nil, nil }
func (f *fakeStore) CountItems(_ *string, _ *string) (int64, error)       { return 0, nil }
func (f *fakeStore) HasVecTable() bool                                    { return false }
func (f *fakeStore) EnsureVecTable(_ int) error                           { return nil }
func (f *fakeStore) SetEmbeddingDim(_ int) error                          { return nil }
func (f *fakeStore) DropVecTable() error                                  { return nil }
func (f *fakeStore) Close() error                                         { return nil }

// fakeEmbedder always returns a fixed 3-float vector.
type fakeEmbedder struct {
	called int
	err    error
}

func (e *fakeEmbedder) Embed(_ string) ([]float32, error) {
	e.called++
	if e.err != nil {
		return nil, e.err
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

// makeResult is a helper for building SearchResult values.
func makeResult(id string, score float64) models.SearchResult {
	return models.SearchResult{ID: id, Title: id, Score: score}
}

// --- normalizeScores ---

func TestNormalizeScores_Empty(t *testing.T) {
	var results []models.SearchResult
	normalizeScores(results) // should not panic
}

func TestNormalizeScores_AllZero(t *testing.T) {
	results := []models.SearchResult{makeResult("a", 0), makeResult("b", 0)}
	normalizeScores(results)
	for _, r := range results {
		if r.Score != 0 {
			t.Errorf("score should remain 0 for all-zero input, got %f", r.Score)
		}
	}
}

func TestNormalizeScores_MaxBecomesOne(t *testing.T) {
	results := []models.SearchResult{
		makeResult("a", 4.0),
		makeResult("b", 2.0),
		makeResult("c", 1.0),
	}
	normalizeScores(results)
	if results[0].Score != 1.0 {
		t.Errorf("max score should be 1.0 after normalization, got %f", results[0].Score)
	}
	if results[1].Score != 0.5 {
		t.Errorf("score[1] should be 0.5, got %f", results[1].Score)
	}
}

// --- MergeResults ---

func TestMergeResults_BothEmpty(t *testing.T) {
	result := MergeResults(nil, nil, 0.3, 0.7, 5)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d", len(result))
	}
}

func TestMergeResults_OneFTSResult(t *testing.T) {
	fts := []models.SearchResult{makeResult("a", 1.0)}
	result := MergeResults(fts, nil, 0.3, 0.7, 5)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	// After normalization: fts score = 1.0; weighted = 0.3 * 1.0 = 0.3
	if result[0].Score != 0.3 {
		t.Errorf("score = %f, want 0.3", result[0].Score)
	}
}

func TestMergeResults_DeduplicationCombinesScores(t *testing.T) {
	fts := []models.SearchResult{makeResult("shared", 2.0), makeResult("fts-only", 1.0)}
	vec := []models.SearchResult{makeResult("shared", 4.0), makeResult("vec-only", 2.0)}

	result := MergeResults(fts, vec, 0.3, 0.7, 10)

	// Find "shared" entry
	var sharedScore float64
	for _, r := range result {
		if r.ID == "shared" {
			sharedScore = r.Score
		}
	}
	// fts normalized: 2/2 = 1.0, weighted 0.3; vec normalized: 4/4 = 1.0, weighted 0.7
	// combined = 1.0
	if sharedScore != 1.0 {
		t.Errorf("shared item score = %f, want 1.0", sharedScore)
	}
}

func TestMergeResults_OrderedByScoreDesc(t *testing.T) {
	fts := []models.SearchResult{
		makeResult("low", 1.0),
		makeResult("high", 3.0),
		makeResult("mid", 2.0),
	}
	result := MergeResults(fts, nil, 1.0, 0.0, 10)

	for i := 1; i < len(result); i++ {
		if result[i].Score > result[i-1].Score {
			t.Errorf("results not ordered desc: result[%d].Score %f > result[%d].Score %f",
				i, result[i].Score, i-1, result[i-1].Score)
		}
	}
}

func TestMergeResults_LimitRespected(t *testing.T) {
	fts := []models.SearchResult{
		makeResult("a", 4.0),
		makeResult("b", 3.0),
		makeResult("c", 2.0),
		makeResult("d", 1.0),
	}
	result := MergeResults(fts, nil, 1.0, 0.0, 2)
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

// --- TieredSearch ---

func TestTieredSearch_FTSSufficient_NoEmbedCall(t *testing.T) {
	store := &fakeStore{
		ftsResults: []models.SearchResult{
			makeResult("a", 3.0),
			makeResult("b", 2.0),
			makeResult("c", 1.0),
		},
	}
	embedder := &fakeEmbedder{}

	results, err := TieredSearch(store, embedder, "query", 5, DefaultMinFTSResults, nil, nil)
	if err != nil {
		t.Fatalf("TieredSearch() error = %v", err)
	}
	if len(results) == 0 {
		t.Error("TieredSearch() should return results")
	}
	if embedder.called > 0 {
		t.Error("Embed should NOT be called when FTS results are sufficient")
	}
}

func TestTieredSearch_SparseFTS_CallsEmbed(t *testing.T) {
	store := &fakeStore{
		ftsResults: []models.SearchResult{makeResult("a", 1.0)}, // only 1, below minFTS=3
		vecResults: []models.SearchResult{makeResult("b", 0.9), makeResult("c", 0.8)},
	}
	embedder := &fakeEmbedder{}

	results, err := TieredSearch(store, embedder, "query", 5, DefaultMinFTSResults, nil, nil)
	if err != nil {
		t.Fatalf("TieredSearch() error = %v", err)
	}
	if embedder.called == 0 {
		t.Error("Embed SHOULD be called when FTS results are sparse")
	}
	_ = results
}

func TestTieredSearch_FTSError_ReturnsError(t *testing.T) {
	store := &fakeStore{ftsErr: errors.New("db failure")}
	_, err := TieredSearch(store, nil, "q", 5, 3, nil, nil)
	if err == nil {
		t.Error("TieredSearch() should propagate FTS error")
	}
}

func TestTieredSearch_NilProvider_ReturnsFTSOnly(t *testing.T) {
	store := &fakeStore{ftsResults: []models.SearchResult{makeResult("a", 1.0)}}

	results, err := TieredSearch(store, nil, "q", 5, 10, nil, nil) // minFTS=10 > 1 result
	if err != nil {
		t.Fatalf("TieredSearch() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 FTS result, got %d", len(results))
	}
	if store.vecCalled > 0 {
		t.Error("VectorSearch should not be called with nil provider")
	}
}

func TestTieredSearch_EmbedError_ReturnsFTSResults(t *testing.T) {
	store := &fakeStore{
		ftsResults: []models.SearchResult{makeResult("a", 1.0)},
	}
	embedder := &fakeEmbedder{err: errors.New("embed failed")}

	results, err := TieredSearch(store, embedder, "q", 5, 10, nil, nil)
	if err != nil {
		t.Fatalf("TieredSearch() should not error on embed failure, got: %v", err)
	}
	if len(results) == 0 {
		t.Error("TieredSearch() should return FTS results as fallback on embed error")
	}
}
