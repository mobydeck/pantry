package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"pantry/internal/core"

	"github.com/spf13/cobra"
)

var (
	searchLimit   int
	searchProject bool
	searchSource  string
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search pantry items",
	Args:  cobra.ExactArgs(1),
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		defer func() { _ = svc.Close() }()

		var project *string

		if searchProject {
			dir, _ := os.Getwd()
			projectName := filepath.Base(dir)
			project = &projectName
		}

		var source *string
		if searchSource != "" {
			source = &searchSource
		}

		results, err := svc.Search(query, searchLimit, project, source, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("No results found.")

			return
		}

		fmt.Printf("\n Results (%d found) \n\n", len(results))

		for i, r := range results {
			cat := ""
			if r.Category != nil {
				cat = *r.Category
			}

			src := ""
			if r.Source != nil {
				src = *r.Source
			}

			fmt.Printf(" [%d] %s (score: %.2f)\n", i+1, r.Title, r.Score)
			fmt.Printf("     %s | %s | %s", cat, r.CreatedAt[:10], r.Project)

			if src != "" {
				fmt.Printf(" | %s", src)
			}

			fmt.Println()
			fmt.Printf("     What: %s\n", r.What)

			if r.Why != nil {
				fmt.Printf("     Why: %s\n", *r.Why)
			}

			if r.Impact != nil {
				fmt.Printf("     Impact: %s\n", *r.Impact)
			}

			if r.HasDetails {
				fmt.Printf("     Details: available (use `pantry retrieve %s`)\n", r.ID[:12])
			}

			fmt.Println()
		}
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "Maximum number of results")
	searchCmd.Flags().BoolVarP(&searchProject, "project", "p", false, "Filter to current project")
	searchCmd.Flags().StringVarP(&searchSource, "source", "s", "", "Filter by source")
}
