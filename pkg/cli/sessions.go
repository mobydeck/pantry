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
	logLimit   int
	logProject string
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "List daily note logs",
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()
		shelvesDir := filepath.Join(home, "shelves")

		logFiles := []struct {
			project string
			fname   string
		}{}

		entries, err := os.ReadDir(shelvesDir)
		if err != nil {
			fmt.Println("No logs found.")
			return
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			projDir := filepath.Join(shelvesDir, entry.Name())
			if logProject != "" && entry.Name() != logProject {
				continue
			}

			files, err := os.ReadDir(projDir)
			if err != nil {
				continue
			}

			for _, f := range files {
				if strings.HasSuffix(f.Name(), "-notes.md") {
					logFiles = append(logFiles, struct {
						project string
						fname   string
					}{entry.Name(), f.Name()})
				}
			}
		}

		if len(logFiles) == 0 {
			fmt.Println("No logs found.")
			return
		}

		// Sort by filename (date) descending
		sort.Slice(logFiles, func(i, j int) bool {
			return logFiles[i].fname > logFiles[j].fname
		})

		fmt.Println("\nLogs:")
		for i, lf := range logFiles {
			if i >= logLimit {
				break
			}
			dateStr := strings.Replace(lf.fname, "-notes.md", "", 1)
			fmt.Printf("  %s | %s\n", dateStr, lf.project)
		}
	},
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 10, "Maximum number of logs to show")
	logCmd.Flags().StringVarP(&logProject, "project", "p", "", "Filter by project name")
}
