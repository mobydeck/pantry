package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"pantry/internal/config"

	"github.com/spf13/cobra"
)

var (
	notesLimit   int
	notesProject string
)

var notesCmd = &cobra.Command{
	Use:     "notes",
	Aliases: []string{"log"},
	Short:   "List daily note files",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		home := config.GetPantryHome()
		shelvesDir := filepath.Join(home, "shelves")

		type noteFile struct {
			project string
			fname   string
		}

		var noteFiles []noteFile

		entries, err := os.ReadDir(shelvesDir)
		if err != nil {
			fmt.Println("No notes found.")

			return
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			projDir := filepath.Join(shelvesDir, entry.Name())
			if notesProject != "" && entry.Name() != notesProject {
				continue
			}

			files, err := os.ReadDir(projDir)
			if err != nil {
				continue
			}

			for _, f := range files {
				if strings.HasSuffix(f.Name(), "-notes.md") {
					noteFiles = append(noteFiles, noteFile{entry.Name(), f.Name()})
				}
			}
		}

		if len(noteFiles) == 0 {
			fmt.Println("No notes found.")

			return
		}

		// Sort by filename (date) descending
		sort.Slice(noteFiles, func(i, j int) bool {
			return noteFiles[i].fname > noteFiles[j].fname
		})

		fmt.Println("\nNotes:")

		maxProject := 0

		for i, nf := range noteFiles {
			if i >= notesLimit {
				break
			}

			if len(nf.project) > maxProject {
				maxProject = len(nf.project)
			}
		}

		for i, nf := range noteFiles {
			if i >= notesLimit {
				break
			}

			dateStr := strings.Replace(nf.fname, "-notes.md", "", 1)
			fullPath := filepath.Join(shelvesDir, nf.project, nf.fname)
			fmt.Printf("  %s | %-*s | %s\n", dateStr, maxProject, nf.project, fullPath)
		}
	},
}

func init() {
	notesCmd.Flags().IntVarP(&notesLimit, "limit", "n", 10, "Maximum number of files to show")
	notesCmd.Flags().StringVarP(&notesProject, "project", "p", "", "Filter by project name")
}
