package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetPantryHome(t *testing.T) {
	// Test default
	home := GetPantryHome()
	if home == "" {
		t.Error("GetPantryHome() should not return empty string")
	}

	// Test with environment variable
	os.Setenv("PANTRY_HOME", "/test/pantry")
	defer os.Unsetenv("PANTRY_HOME")

	home = GetPantryHome()
	if home != "/test/pantry" {
		t.Errorf("GetPantryHome() = %q, want %q", home, "/test/pantry")
	}
}

func TestLoadConfig(t *testing.T) {
	// Test with non-existent file (should return defaults)
	cfg, err := LoadConfig("/nonexistent/config.yaml")
	if err != nil {
		t.Errorf("LoadConfig() error = %v, want nil", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}
	if cfg.Embedding.Provider != "ollama" {
		t.Errorf("LoadConfig() default provider = %q, want %q", cfg.Embedding.Provider, "ollama")
	}
}

func TestGetDefaultConfigTemplate(t *testing.T) {
	template := GetDefaultConfigTemplate()
	if template == "" {
		t.Error("GetDefaultConfigTemplate() should not return empty string")
	}
	if len(template) < 100 {
		t.Error("GetDefaultConfigTemplate() should return substantial template")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Embedding: EmbeddingConfig{
			Provider: "ollama",
			Model:    "test-model",
		},
	}

	err := SaveConfig(configPath, cfg)
	if err != nil {
		t.Errorf("SaveConfig() error = %v", err)
	}

	// Verify it can be loaded back
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Errorf("LoadConfig() after SaveConfig error = %v", err)
	}
	if loaded.Embedding.Model != "test-model" {
		t.Errorf("LoadConfig() Model = %q, want %q", loaded.Embedding.Model, "test-model")
	}
}
