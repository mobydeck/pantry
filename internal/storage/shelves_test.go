package storage

import (
	"os"
	"path/filepath"
	"testing"

	"pantry/internal/models"
)

func TestWriteNoteItem(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	os.MkdirAll(projectDir, 0755)

	item := models.Item{
		ID:            "test-id",
		Title:         "Test Item",
		What:          "This is a test",
		Project:       "test-project",
		FilePath:      filepath.Join(projectDir, "2026-01-01-notes.md"),
		SectionAnchor: "test-item",
		CreatedAt:     "2026-01-01T00:00:00Z",
		UpdatedAt:     "2026-01-01T00:00:00Z",
	}

	details := "Full details here"
	filePath, err := WriteNoteItem(projectDir, item, "2026-01-01", &details)
	if err != nil {
		t.Fatalf("WriteNoteItem() error = %v", err)
	}

	if filePath == "" {
		t.Error("WriteNoteItem() should return file path")
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("WriteNoteItem() file does not exist: %s", filePath)
	}

	// Verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "Test Item") {
		t.Error("File should contain item title")
	}
	if !contains(contentStr, "This is a test") {
		t.Error("File should contain item what")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
