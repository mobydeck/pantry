package db

import (
	"pantry/internal/models"
)

// ItemModel represents the items table in the database
type ItemModel struct {
	ID            string `gorm:"primaryKey;type:text"`
	Title         string `gorm:"type:text;not null"`
	What          string `gorm:"type:text;not null"`
	Why           *string `gorm:"type:text"`
	Impact        *string `gorm:"type:text"`
	Tags          string `gorm:"type:text"` // JSON encoded
	Category      *string `gorm:"type:text"`
	Project       string `gorm:"type:text;not null"`
	Source        *string `gorm:"type:text"`
	RelatedFiles  string `gorm:"type:text"` // JSON encoded
	FilePath      string `gorm:"type:text;not null"`
	SectionAnchor string `gorm:"type:text"`
	CreatedAt     string `gorm:"type:text;not null"`
	UpdatedAt     string `gorm:"type:text;not null"`
	UpdatedCount  int    `gorm:"default:0"`
}

// TableName specifies the table name for GORM
func (ItemModel) TableName() string {
	return "items"
}

// ItemDetailModel represents the item_details table
type ItemDetailModel struct {
	ItemID string `gorm:"primaryKey;type:text"`
	Body   string `gorm:"type:text;not null"`
}

// TableName specifies the table name for GORM
func (ItemDetailModel) TableName() string {
	return "item_details"
}

// MetaModel represents the meta table
type MetaModel struct {
	Key   string `gorm:"primaryKey;type:text"`
	Value string `gorm:"type:text;not null"`
}

// TableName specifies the table name for GORM
func (MetaModel) TableName() string {
	return "meta"
}

// ToItem converts ItemModel to models.Item
func (im *ItemModel) ToItem() models.Item {
	item := models.Item{
		ID:            im.ID,
		Title:         im.Title,
		What:          im.What,
		FilePath:      im.FilePath,
		SectionAnchor: im.SectionAnchor,
		Project:       im.Project,
		CreatedAt:     im.CreatedAt,
		UpdatedAt:     im.UpdatedAt,
	}

	if im.Why != nil {
		item.Why = im.Why
	}
	if im.Impact != nil {
		item.Impact = im.Impact
	}
	if im.Category != nil {
		item.Category = im.Category
	}
	if im.Source != nil {
		item.Source = im.Source
	}

	return item
}

// FromItem converts models.Item to ItemModel
func (im *ItemModel) FromItem(item models.Item, tagsJSON, relatedFilesJSON string) {
	im.ID = item.ID
	im.Title = item.Title
	im.What = item.What
	im.Why = item.Why
	im.Impact = item.Impact
	im.Tags = tagsJSON
	im.Category = item.Category
	im.Project = item.Project
	im.Source = item.Source
	im.RelatedFiles = relatedFilesJSON
	im.FilePath = item.FilePath
	im.SectionAnchor = item.SectionAnchor
	im.CreatedAt = item.CreatedAt
	im.UpdatedAt = item.UpdatedAt
}
