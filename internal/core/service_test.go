package core

import (
	"os"
	"path/filepath"
	"testing"

	"pantry/internal/models"
)

func TestNewService(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.Close()

	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.pantryHome != tmpDir {
		t.Errorf("NewService() pantryHome = %q, want %q", svc.pantryHome, tmpDir)
	}
}

func TestService_Store(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.Close()

	raw := models.RawItemInput{
		Title: "Test Item",
		What:  "This is a test item",
		Tags:  []string{"test"},
	}

	result, err := svc.Store(raw, "test-project")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	if result["id"] == "" {
		t.Error("Store() should return item ID")
	}
	if result["action"] != "created" {
		t.Errorf("Store() action = %q, want %q", result["action"], "created")
	}
}

func TestService_Search(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.Close()

	// Store an item first
	raw := models.RawItemInput{
		Title: "Search Test",
		What:  "This is searchable content",
	}
	_, err = svc.Store(raw, "test-project")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Search for it
	results, err := svc.Search("searchable", 5, nil, nil, false)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Search() should return at least one result")
	}
}

func TestService_GetDetails(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.Close()

	// Store an item with details
	details := "Full details here"
	raw := models.RawItemInput{
		Title:   "Details Test",
		What:    "Test item",
		Details: &details,
	}
	result, err := svc.Store(raw, "test-project")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Retrieve details
	detail, err := svc.GetDetails(result["id"])
	if err != nil {
		t.Fatalf("GetDetails() error = %v", err)
	}

	if detail == nil {
		t.Fatal("GetDetails() returned nil")
	}
	if detail.Body != details {
		t.Errorf("GetDetails() Body = %q, want %q", detail.Body, details)
	}
}

func TestService_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(tmpDir)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	defer svc.Close()

	// Store an item
	raw := models.RawItemInput{
		Title: "Delete Test",
		What:  "This will be deleted",
	}
	result, err := svc.Store(raw, "test-project")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Delete it
	deleted, err := svc.Remove(result["id"])
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if !deleted {
		t.Error("Remove() should return true for existing item")
	}

	// Try to delete again (should return false)
	deleted, err = svc.Remove(result["id"])
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if deleted {
		t.Error("Remove() should return false for non-existent item")
	}
}
