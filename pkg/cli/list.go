package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"pantry/internal/core"
)

var (
	listLimit   int
	listProject bool
	listSource  string
	listQuery   string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent items",
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer svc.Close()

		var project *string
		if listProject {
			dir, _ := os.Getwd()
			projectName := filepath.Base(dir)
			project = &projectName
		}

		var source *string
		if listSource != "" {
			source = &listSource
		}

		var query *string
		if listQuery != "" {
			query = &listQuery
		}

		results, total, err := svc.GetContext(listLimit, project, source, query, "never", false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("No items found.")
			return
		}

		fmt.Printf("Available items (%d total, showing %d):\n", total, len(results))

		for _, r := range results {
			dateStr := r.CreatedAt[:10]
			dateDisplay := dateStr
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				dateDisplay = t.Format("Jan 02")
			}

			cat := ""
			if r.Category != nil {
				cat = fmt.Sprintf(" [%s]", *r.Category)
			}

			tags := ""
			if len(r.Tags) > 0 {
				tags = fmt.Sprintf(" [%s]", fmt.Sprintf("%v", r.Tags))
			}

			fmt.Printf("- [%s] %s%s%s\n", dateDisplay, r.Title, cat, tags)
		}

		fmt.Println("\nUse `pantry search <query>` for full details on any item.")
	},
}

func init() {
	listCmd.Flags().IntVar(&listLimit, "limit", 10, "Maximum number of items")
	listCmd.Flags().BoolVar(&listProject, "project", false, "Filter to current project")
	listCmd.Flags().StringVar(&listSource, "source", "", "Filter by source")
	listCmd.Flags().StringVar(&listQuery, "query", "", "Search query for filtering")
}
