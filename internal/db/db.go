package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	// used to import sqlite vec bindings.
	_ "github.com/asg017/sqlite-vec-go-bindings/ncruces"
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"pantry/internal/models"
)

// Compile-time check that *DB satisfies the Store interface.
var _ Store = (*DB)(nil)

// DB wraps the database connection and provides methods for pantry operations.
type DB struct {
	db *gorm.DB
}

// NewDB creates a new database connection.
func NewDB(dbPath string) (*DB, error) {
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)"

	gormDB, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm connection: %w", err)
	}

	db := &DB{db: gormDB}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// HasVecTable checks if the vector table exists.
func (d *DB) HasVecTable() bool {
	var count int64

	d.db.Raw(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='items_vec'
	`).Scan(&count)

	return count > 0
}

// DropVecTable drops the vector table.
func (d *DB) DropVecTable() error {
	return d.db.Exec("DROP TABLE IF EXISTS items_vec").Error
}

// SetEmbeddingDim stores the embedding dimension in meta table.
func (d *DB) SetEmbeddingDim(dim int) error {
	meta := MetaModel{
		Key:   "embedding_dim",
		Value: strconv.Itoa(dim),
	}

	return d.db.Save(&meta).Error
}

// EnsureVecTable ensures the vector table exists with the correct dimension.
func (d *DB) EnsureVecTable(dim int) error {
	storedDim := d.getEmbeddingDim()
	if storedDim == nil {
		if err := d.SetEmbeddingDim(dim); err != nil {
			return err
		}

		return d.createVecTable(dim)
	} else if *storedDim != dim {
		return fmt.Errorf("%w: database has %d, provider returned %d. Run 'pantry reindex' to rebuild", ErrDimensionMismatch, *storedDim, dim)
	}

	return nil
}

// InsertItem inserts an item into the database using GORM.
func (d *DB) InsertItem(item models.Item, details *string) (int64, error) {
	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tags: %w", err)
	}

	relatedFilesJSON, err := json.Marshal(item.RelatedFiles)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal related_files: %w", err)
	}

	itemModel := ItemModel{}
	itemModel.FromItem(item, string(tagsJSON), string(relatedFilesJSON))

	if err := d.db.Create(&itemModel).Error; err != nil {
		return 0, err
	}

	// Get rowid
	var rowid int64
	if err := d.db.Raw("SELECT rowid FROM items WHERE id = ?", item.ID).Scan(&rowid).Error; err != nil {
		return 0, err
	}

	// Insert details if provided
	if details != nil {
		detailModel := ItemDetailModel{
			ItemID: item.ID,
			Body:   *details,
		}
		if err := d.db.Create(&detailModel).Error; err != nil {
			return 0, err
		}
	}

	return rowid, nil
}

// InsertVector inserts an embedding vector for an item.
func (d *DB) InsertVector(rowid int64, embedding []float32) error {
	if !d.HasVecTable() {
		return nil
	}

	embeddingBytes, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding: %w", err)
	}

	return d.db.Exec(`
		INSERT INTO items_vec (rowid, embedding)
		VALUES (?, ?)
	`, rowid, embeddingBytes).Error
}

// GetItem gets an item by ID using GORM.
func (d *DB) GetItem(itemID string) (*models.Item, bool, error) {
	var itemModel ItemModel
	if err := d.db.Where("id = ?", itemID).First(&itemModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	// Check if details exist
	var hasDetails bool

	d.db.Model(&ItemDetailModel{}).Select("COUNT(*) > 0").Where("item_id = ?", itemID).Scan(&hasDetails)

	item := itemModel.ToItem()

	// Parse tags and related files; ignore errors on malformed JSON (fields stay nil)
	_ = json.Unmarshal([]byte(itemModel.Tags), &item.Tags)
	_ = json.Unmarshal([]byte(itemModel.RelatedFiles), &item.RelatedFiles)

	return &item, hasDetails, nil
}

// GetDetails gets full details for an item using GORM.
func (d *DB) GetDetails(itemID string) (*models.ItemDetail, error) {
	var detailModel ItemDetailModel
	if err := d.db.Where("item_id LIKE ?", itemID+"%").First(&detailModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil //nolint:nilnil
		}

		return nil, err
	}

	return &models.ItemDetail{
		ItemID: detailModel.ItemID,
		Body:   detailModel.Body,
	}, nil
}

// UpdateItem updates an existing item's fields using GORM.
func (d *DB) UpdateItem(itemID string, what *string, why *string, impact *string, tags []string, detailsAppend *string) error {
	// Resolve full ID from prefix
	var itemModel ItemModel
	if err := d.db.Where("id LIKE ?", itemID+"%").First(&itemModel).Error; err != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, itemID)
	}

	fullID := itemModel.ID

	// Build updates
	updates := map[string]any{
		"updated_count": gorm.Expr("updated_count + 1"),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	}

	if what != nil {
		updates["what"] = *what
	}

	if why != nil {
		updates["why"] = *why
	}

	if impact != nil {
		updates["impact"] = *impact
	}

	if tags != nil {
		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}

		updates["tags"] = string(tagsJSON)
	}

	if err := d.db.Model(&ItemModel{}).Where("id = ?", fullID).Updates(updates).Error; err != nil {
		return err
	}

	// Handle details append
	if detailsAppend != nil {
		var detailModel ItemDetailModel
		if err := d.db.Where("item_id = ?", fullID).First(&detailModel).Error; err == nil {
			detailModel.Body = detailModel.Body + "\n\n" + *detailsAppend
			d.db.Save(&detailModel)
		} else {
			detailModel = ItemDetailModel{
				ItemID: fullID,
				Body:   *detailsAppend,
			}
			d.db.Create(&detailModel)
		}
	}

	return nil
}

// DeleteItem deletes an item by ID or prefix using GORM.
func (d *DB) DeleteItem(itemID string) (bool, error) {
	// Resolve full ID from prefix
	var itemModel ItemModel
	if err := d.db.Where("id LIKE ?", itemID+"%").First(&itemModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}

		return false, err
	}

	fullID := itemModel.ID

	// Delete details first
	d.db.Where("item_id = ?", fullID).Delete(&ItemDetailModel{})

	// Delete item
	result := d.db.Where("id = ?", fullID).Delete(&ItemModel{})

	return result.RowsAffected > 0, result.Error
}

// FTSSearch searches items using FTS5 (must use raw SQL for FTS).
func (d *DB) FTSSearch(query string, limit int, project *string, source *string) ([]models.SearchResult, error) {
	// Build prefix matching query
	terms := splitQuery(query)
	ftsQuery := ""

	var ftsQuerySb315 strings.Builder

	for i, term := range terms {
		if i > 0 {
			ftsQuerySb315.WriteString(" OR ")
		}

		ftsQuerySb315.WriteString(fmt.Sprintf(`"%s"*`, term))
	}

	ftsQuery += ftsQuerySb315.String()

	whereClause := ""
	args := []any{ftsQuery}

	if project != nil {
		whereClause += " AND m.project = ?" //nolint:goconst

		args = append(args, *project)
	}

	if source != nil {
		whereClause += " AND m.source = ?" //nolint:goconst

		args = append(args, *source)
	}

	args = append(args, limit)

	var rows []struct {
		ID         string
		Title      string
		What       string
		Why        sql.NullString
		Impact     sql.NullString
		Category   sql.NullString
		Tags       string
		Project    string
		Source     sql.NullString
		FilePath   string
		CreatedAt  string
		Score      float64
		HasDetails bool
	}

	err := d.db.Raw(fmt.Sprintf(`
		SELECT m.id, m.title, m.what, m.why, m.impact, m.category, m.tags,
		       m.project, m.source, m.file_path, m.created_at,
		       -fts.rank as score,
		       EXISTS(SELECT 1 FROM item_details WHERE item_id = m.id) as has_details
		FROM items_fts fts
		JOIN items m ON m.rowid = fts.rowid
		WHERE fts.items_fts MATCH ?
		%s
		ORDER BY fts.rank
		LIMIT ?
	`, whereClause), args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]models.SearchResult, len(rows))

	for i, row := range rows {
		result := models.SearchResult{
			ID:         row.ID,
			Title:      row.Title,
			What:       row.What,
			Project:    row.Project,
			FilePath:   row.FilePath,
			CreatedAt:  row.CreatedAt,
			Score:      row.Score,
			HasDetails: row.HasDetails,
		}

		if row.Why.Valid {
			result.Why = &row.Why.String
		}

		if row.Impact.Valid {
			result.Impact = &row.Impact.String
		}

		if row.Category.Valid {
			result.Category = &row.Category.String
		}

		if row.Source.Valid {
			result.Source = &row.Source.String
		}

		if err := json.Unmarshal([]byte(row.Tags), &result.Tags); err != nil {
			result.Tags = []string{}
		}

		results[i] = result
	}

	return results, nil
}

// VectorSearch searches items using vector similarity (must use raw SQL for vec).
func (d *DB) VectorSearch(queryEmbedding []float32, limit int, project *string, source *string) ([]models.SearchResult, error) {
	if !d.HasVecTable() {
		return []models.SearchResult{}, nil
	}

	embeddingBytes, err := json.Marshal(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query embedding: %w", err)
	}

	var rows []struct {
		ID         string
		Title      string
		What       string
		Why        sql.NullString
		Impact     sql.NullString
		Category   sql.NullString
		Tags       string
		Project    string
		Source     sql.NullString
		FilePath   string
		CreatedAt  string
		Distance   float64
		HasDetails bool
	}

	whereClause := ""
	args := []any{embeddingBytes, limit}

	if project != nil {
		whereClause += " AND m.project = ?"

		args = append(args, *project)
	}

	if source != nil {
		whereClause += " AND m.source = ?"

		args = append(args, *source)
	}

	err = d.db.Raw(fmt.Sprintf(`
		SELECT m.id, m.title, m.what, m.why, m.impact, m.category, m.tags,
		       m.project, m.source, m.file_path, m.created_at,
		       v.distance,
		       EXISTS(SELECT 1 FROM item_details WHERE item_id = m.id) as has_details
		FROM items_vec v
		JOIN items m ON m.rowid = v.rowid
		WHERE v.embedding MATCH ?
		AND k = ?
		%s
		ORDER BY v.distance
	`, whereClause), args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]models.SearchResult, len(rows))

	for i, row := range rows {
		result := models.SearchResult{
			ID:         row.ID,
			Title:      row.Title,
			What:       row.What,
			Project:    row.Project,
			FilePath:   row.FilePath,
			CreatedAt:  row.CreatedAt,
			Score:      1.0 - row.Distance,
			HasDetails: row.HasDetails,
		}

		if row.Why.Valid {
			result.Why = &row.Why.String
		}

		if row.Impact.Valid {
			result.Impact = &row.Impact.String
		}

		if row.Category.Valid {
			result.Category = &row.Category.String
		}

		if row.Source.Valid {
			result.Source = &row.Source.String
		}

		if err := json.Unmarshal([]byte(row.Tags), &result.Tags); err != nil {
			result.Tags = []string{}
		}

		results[i] = result
	}

	return results, nil
}

// ListRecent lists recent items ordered by creation date descending.
// Uses a single raw SQL query with an EXISTS subquery to avoid N+1 queries.
func (d *DB) ListRecent(limit int, project *string, source *string) ([]models.SearchResult, error) {
	whereClause := "1=1"
	args := []any{}

	if project != nil {
		whereClause += " AND m.project = ?"

		args = append(args, *project)
	}

	if source != nil {
		whereClause += " AND m.source = ?"

		args = append(args, *source)
	}

	args = append(args, limit)

	var rows []struct {
		ID         string
		Title      string
		What       string
		Why        sql.NullString
		Impact     sql.NullString
		Category   sql.NullString
		Tags       string
		Project    string
		Source     sql.NullString
		FilePath   string
		CreatedAt  string
		HasDetails bool
	}

	err := d.db.Raw(fmt.Sprintf(`
		SELECT m.id, m.title, m.what, m.why, m.impact, m.category, m.tags,
		       m.project, m.source, m.file_path, m.created_at,
		       EXISTS(SELECT 1 FROM item_details WHERE item_id = m.id) AS has_details
		FROM items m
		WHERE %s
		ORDER BY m.created_at DESC
		LIMIT ?
	`, whereClause), args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]models.SearchResult, len(rows))

	for i, row := range rows {
		result := models.SearchResult{
			ID:         row.ID,
			Title:      row.Title,
			What:       row.What,
			Project:    row.Project,
			FilePath:   row.FilePath,
			CreatedAt:  row.CreatedAt,
			HasDetails: row.HasDetails,
		}
		if row.Why.Valid {
			result.Why = &row.Why.String
		}

		if row.Impact.Valid {
			result.Impact = &row.Impact.String
		}

		if row.Category.Valid {
			result.Category = &row.Category.String
		}

		if row.Source.Valid {
			result.Source = &row.Source.String
		}

		if err := json.Unmarshal([]byte(row.Tags), &result.Tags); err != nil {
			result.Tags = []string{}
		}

		results[i] = result
	}

	return results, nil
}

// ListAllForReindex lists all items with fields needed for re-embedding using GORM.
func (d *DB) ListAllForReindex() ([]map[string]any, error) {
	var itemModels []ItemModel
	if err := d.db.Order("rowid").Find(&itemModels).Error; err != nil {
		return nil, err
	}

	results := make([]map[string]any, len(itemModels))

	for i, im := range itemModels {
		// Get rowid
		var rowid int64

		d.db.Raw("SELECT rowid FROM items WHERE id = ?", im.ID).Scan(&rowid)

		result := map[string]any{
			"rowid": rowid,
			"title": im.Title,
			"what":  im.What,
		}
		if im.Why != nil {
			result["why"] = *im.Why
		}

		if im.Impact != nil {
			result["impact"] = *im.Impact
		}

		var tags []string
		if err := json.Unmarshal([]byte(im.Tags), &tags); err != nil {
			tags = []string{}
		}

		result["tags"] = tags
		results[i] = result
	}

	return results, nil
}

// CountItems counts total items with optional filters using GORM.
func (d *DB) CountItems(project *string, source *string) (int64, error) {
	var count int64

	query := d.db.Model(&ItemModel{})

	if project != nil {
		query = query.Where("project = ?", *project)
	}

	if source != nil {
		query = query.Where("source = ?", *source)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// migrate runs database migrations using GORM AutoMigrate.
func (d *DB) migrate() error {
	// Auto-migrate GORM models
	if err := d.db.AutoMigrate(&ItemModel{}, &ItemDetailModel{}, &MetaModel{}); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	// Create FTS5 virtual table (must use raw SQL)
	if err := d.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
			title, what, why, impact, tags, category, project, source,
			content='items', content_rowid='rowid',
			tokenize='porter unicode61'
		)
	`).Error; err != nil {
		return err
	}

	// Create FTS5 triggers (must use raw SQL)
	if err := d.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
			INSERT INTO items_fts(rowid, title, what, why, impact, tags, category, project, source)
			VALUES (new.rowid, new.title, new.what, new.why, new.impact, new.tags, new.category, new.project, new.source);
		END
	`).Error; err != nil {
		return err
	}

	if err := d.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
			INSERT INTO items_fts(items_fts, rowid, title, what, why, impact, tags, category, project, source)
			VALUES ('delete', old.rowid, old.title, old.what, old.why, old.impact, old.tags, old.category, old.project, old.source);
			INSERT INTO items_fts(rowid, title, what, why, impact, tags, category, project, source)
			VALUES (new.rowid, new.title, new.what, new.why, new.impact, new.tags, new.category, new.project, new.source);
		END
	`).Error; err != nil {
		return err
	}

	// Create vec table if dimension is known
	dim := d.getEmbeddingDim()
	if dim != nil {
		if err := d.createVecTable(*dim); err != nil {
			return err
		}
	}

	return nil
}

// createVecTable creates the vector table with the given dimension.
func (d *DB) createVecTable(dim int) error {
	query := fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS items_vec USING vec0(
			rowid INTEGER PRIMARY KEY,
			embedding float[%d]
		)
	`, dim)

	return d.db.Exec(query).Error
}

// getEmbeddingDim gets the stored embedding dimension from meta table.
func (d *DB) getEmbeddingDim() *int {
	var meta MetaModel
	if err := d.db.Where("key = ?", "embedding_dim").First(&meta).Error; err != nil {
		return nil
	}

	var dim int
	if _, err := fmt.Sscanf(meta.Value, "%d", &dim); err != nil {
		return nil
	}

	return &dim
}

// Helper functions.
func splitQuery(query string) []string {
	terms := []string{}
	current := ""

	for _, r := range query {
		if r == ' ' || r == '\t' || r == '\n' {
			if current != "" {
				terms = append(terms, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		terms = append(terms, current)
	}

	return terms
}
