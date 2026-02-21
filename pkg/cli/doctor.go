package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"pantry/internal/config"
	"pantry/internal/core"
	"pantry/internal/redaction"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check pantry health and capabilities",
	Run: func(cmd *cobra.Command, args []string) {
		ok := true
		pass := func(label, detail string) {
			fmt.Printf("  \u2713 %-28s %s\n", label, detail)
		}
		fail := func(label, detail string) {
			fmt.Printf("  \u2717 %-28s %s\n", label, detail)
			ok = false
		}
		warn := func(label, detail string) {
			fmt.Printf("  ! %-28s %s\n", label, detail)
		}

		home := config.GetPantryHome()
		fmt.Printf("\nPantry home: %s\n\n", home)

		// --- Filesystem ---
		fmt.Println("Filesystem:")

		if info, err := os.Stat(home); err != nil || !info.IsDir() {
			fail("pantry home", "directory missing — run `pantry init`")
		} else {
			pass("pantry home", home)
		}

		dbPath := filepath.Join(home, "index.db")
		if _, err := os.Stat(dbPath); err != nil {
			fail("index.db", "missing — run `pantry init`")
		} else {
			pass("index.db", dbPath)
		}

		shelvesDir := filepath.Join(home, "shelves")
		if _, err := os.Stat(shelvesDir); err != nil {
			fail("shelves/", "missing — run `pantry init`")
		} else {
			pass("shelves/", shelvesDir)
		}

		configPath := filepath.Join(home, "config.yaml")
		if _, err := os.Stat(configPath); err != nil {
			warn("config.yaml", "not found, using defaults")
		} else {
			pass("config.yaml", configPath)
		}

		ignorePath := filepath.Join(home, ".pantryignore")
		if _, err := os.Stat(ignorePath); err != nil {
			warn(".pantryignore", "not found (optional)")
		} else {
			pass(".pantryignore", ignorePath)
		}

		// --- Configuration ---
		fmt.Println("\nConfiguration:")

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			fail("load config", err.Error())
		} else {
			pass("load config", "ok")
			if err := cfg.Validate(); err != nil {
				fail("validate config", err.Error())
			} else {
				pass("validate config", "ok")
			}
			baseURL := "(default)"
			if cfg.Embedding.BaseURL != nil {
				baseURL = *cfg.Embedding.BaseURL
			}
			pass("embedding provider", fmt.Sprintf("%s / %s @ %s", cfg.Embedding.Provider, cfg.Embedding.Model, baseURL))
			pass("context.semantic", cfg.Context.Semantic)
		}

		// --- Redaction ---
		fmt.Println("\nRedaction:")
		pass("built-in patterns", fmt.Sprintf("%d patterns", len(redaction.SensitivePatterns)))
		if patterns, err := redaction.LoadPantryIgnore(ignorePath); err != nil && !os.IsNotExist(err) {
			fail(".pantryignore patterns", err.Error())
		} else {
			pass(".pantryignore patterns", fmt.Sprintf("%d custom patterns", len(patterns)))
		}

		// --- Database & search ---
		fmt.Println("\nDatabase & search:")

		svc, err := core.NewService(home)
		if err != nil {
			fail("database connection", err.Error())
			fmt.Println("\nFix the issues above and re-run `pantry doctor`.")
			os.Exit(1)
		}
		defer svc.Close()
		pass("database connection", "ok")

		total, err := svc.CountItems(nil, nil)
		if err != nil {
			fail("note count", err.Error())
		} else {
			pass("note count", fmt.Sprintf("%d notes stored", total))
		}

		pass("FTS5 search", "always available")

		if svc.VectorsAvailable() {
			pass("vector search", "available (sqlite-vec loaded, table exists)")
		} else {
			warn("vector search", "not available — run `pantry reindex` after configuring embeddings")
		}

		// --- Embedding provider live test ---
		fmt.Println("\nEmbedding provider:")

		provider, err := svc.GetEmbeddingProvider()
		if err != nil {
			fail("initialize provider", err.Error())
		} else {
			pass("initialize provider", "ok")
			embedding, err := provider.Embed(context.Background(), "pantry doctor probe")
			if err != nil {
				fail("live probe", err.Error())
				warn("", "check that your embedding service is running and reachable")
			} else {
				pass("live probe", fmt.Sprintf("ok — %d dimensions", len(embedding)))
			}
		}

		// --- Summary ---
		fmt.Println()
		if ok {
			fmt.Println("All checks passed.")
		} else {
			fmt.Println("Some checks failed. Fix the issues above.")
			os.Exit(1)
		}
	},
}
