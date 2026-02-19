package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"pantry/internal/config"
)

var (
	sessionsLimit   int
	sessionsProject string
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List recent sessions",
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()
		vaultDir := filepath.Join(home, "shelf")

		sessionFiles := []struct {
			project string
			fname   string
		}{}

		entries, err := os.ReadDir(vaultDir)
		if err != nil {
			fmt.Println("No sessions found.")
			return
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			projDir := filepath.Join(vaultDir, entry.Name())
			if sessionsProject != "" && entry.Name() != sessionsProject {
				continue
			}

			files, err := os.ReadDir(projDir)
			if err != nil {
				continue
			}

			for _, f := range files {
				if strings.HasSuffix(f.Name(), "-session.md") {
					sessionFiles = append(sessionFiles, struct {
						project string
						fname   string
					}{entry.Name(), f.Name()})
				}
			}
		}

		if len(sessionFiles) == 0 {
			fmt.Println("No sessions found.")
			return
		}

		// Sort by filename (date) descending
		sort.Slice(sessionFiles, func(i, j int) bool {
			return sessionFiles[i].fname > sessionFiles[j].fname
		})

		fmt.Println("\nSessions:")
		for i, sf := range sessionFiles {
			if i >= sessionsLimit {
				break
			}
			dateStr := strings.Replace(sf.fname, "-session.md", "", 1)
			fmt.Printf("  %s | %s\n", dateStr, sf.project)
		}
	},
}

func init() {
	sessionsCmd.Flags().IntVar(&sessionsLimit, "limit", 10, "Maximum number of sessions to show")
	sessionsCmd.Flags().StringVar(&sessionsProject, "project", "", "Filter by project name")
}
