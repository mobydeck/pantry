package models

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ValidCategories defines the allowed categories for items.
var ValidCategories = []string{"decision", "pattern", "bug", "context", "learning"}

// CategoryHeadings maps category values to display headings.
var CategoryHeadings = map[string]string{
	"decision": "Decisions",
	"pattern":  "Patterns",
	"bug":      "Bugs Fixed",
	"context":  "Context",
	"learning": "Learnings",
}

// RawItemInput represents raw input for creating an item before processing.
type RawItemInput struct {
	Title        string
	What         string
	Why          *string
	Impact       *string
	Tags         []string
	Category     *string
	RelatedFiles []string
	Details      *string
	Source       *string
}

// Item represents a stored item in the pantry.
type Item struct {
	ID            string
	Title         string
	What          string
	Why           *string
	Impact        *string
	Tags          []string
	Category      *string
	Project       string
	Source        *string
	RelatedFiles  []string
	FilePath      string
	SectionAnchor string
	CreatedAt     string
	UpdatedAt     string
}

// FromRaw creates an Item from RawItemInput with generated fields.
func FromRaw(raw RawItemInput, project string, filePath string) Item {
	now := time.Now().UTC().Format(time.RFC3339)
	anchor := generateAnchor(raw.Title)

	return Item{
		ID:            uuid.New().String(),
		Title:         raw.Title,
		What:          raw.What,
		Why:           raw.Why,
		Impact:        raw.Impact,
		Tags:          raw.Tags,
		Category:      raw.Category,
		Project:       project,
		Source:        raw.Source,
		RelatedFiles:  raw.RelatedFiles,
		FilePath:      filePath,
		SectionAnchor: anchor,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// generateAnchor creates a URL-friendly anchor from a title.
func generateAnchor(title string) string {
	// Convert to lowercase and replace non-alphanumeric with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	anchor := strings.ToLower(title)
	anchor = re.ReplaceAllString(anchor, "-")
	anchor = strings.Trim(anchor, "-")

	return anchor
}

// ItemDetail represents full details/body content for an item.
type ItemDetail struct {
	ItemID string
	Body   string
}

// SearchResult represents a search result with score and metadata.
type SearchResult struct {
	ID         string
	Title      string
	What       string
	Why        *string
	Impact     *string
	Category   *string
	Tags       []string
	Project    string
	Source     *string
	Score      float64
	HasDetails bool
	FilePath   string
	CreatedAt  string
}
