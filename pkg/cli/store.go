package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"pantry/internal/core"
	"pantry/internal/models"
)

var (
	storeTitle        string
	storeWhat         string
	storeWhy          string
	storeImpact       string
	storeTags         string
	storeCategory     string
	storeRelatedFiles string
	storeDetails      string
	storeSource       string
	storeProject      string
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Store a note in the pantry",
	Run: func(cmd *cobra.Command, args []string) {
		if storeTitle == "" || storeWhat == "" {
			fmt.Fprintf(os.Stderr, "Error: --title and --what are required\n")
			os.Exit(1)
		}

		raw := models.RawItemInput{
			Title: storeTitle,
			What:  storeWhat,
		}

		if storeWhy != "" {
			raw.Why = &storeWhy
		}
		if storeImpact != "" {
			raw.Impact = &storeImpact
		}
		if storeCategory != "" {
			raw.Category = &storeCategory
		}
		if storeSource != "" {
			raw.Source = &storeSource
		}
		if storeDetails != "" {
			raw.Details = &storeDetails
		}

		if storeTags != "" {
			tags := strings.Split(storeTags, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
			raw.Tags = tags
		}

		if storeRelatedFiles != "" {
			files := strings.Split(storeRelatedFiles, ",")
			for i := range files {
				files[i] = strings.TrimSpace(files[i])
			}
			raw.RelatedFiles = files
		}

		svc, err := core.NewService("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer svc.Close()

		result, err := svc.Store(raw, storeProject)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Stored: %s (id: %s)\n", storeTitle, result["id"].(string))
		fmt.Printf("File: %s\n", result["file_path"].(string))
	},
}

func init() {
	storeCmd.Flags().StringVarP(&storeTitle, "title", "t", "", "Title of the note (required)")
	storeCmd.Flags().StringVarP(&storeWhat, "what", "w", "", "What happened or was learned (required)")
	storeCmd.Flags().StringVarP(&storeWhy, "why", "y", "", "Why it matters")
	storeCmd.Flags().StringVarP(&storeImpact, "impact", "i", "", "Impact or consequences")
	storeCmd.Flags().StringVarP(&storeTags, "tags", "g", "", "Comma-separated tags")
	storeCmd.Flags().StringVarP(&storeCategory, "category", "c", "", "Category (decision, pattern, bug, context, learning)")
	storeCmd.Flags().StringVar(&storeRelatedFiles, "related-files", "", "Comma-separated file paths")
	storeCmd.Flags().StringVarP(&storeDetails, "details", "d", "", "Extended details or context")
	storeCmd.Flags().StringVarP(&storeSource, "source", "s", "", "Source of the note")
	storeCmd.Flags().StringVarP(&storeProject, "project", "p", "", "Project name (defaults to current directory)")
}
