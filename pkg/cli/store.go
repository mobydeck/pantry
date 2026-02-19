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
	Short: "Store an item in the pantry",
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

		fmt.Printf("Stored: %s (id: %s)\n", storeTitle, result["id"])
		fmt.Printf("File: %s\n", result["file_path"])
	},
}

func init() {
	storeCmd.Flags().StringVar(&storeTitle, "title", "", "Title of the item (required)")
	storeCmd.Flags().StringVar(&storeWhat, "what", "", "What happened or was learned (required)")
	storeCmd.Flags().StringVar(&storeWhy, "why", "", "Why it matters")
	storeCmd.Flags().StringVar(&storeImpact, "impact", "", "Impact or consequences")
	storeCmd.Flags().StringVar(&storeTags, "tags", "", "Comma-separated tags")
	storeCmd.Flags().StringVar(&storeCategory, "category", "", "Category (decision, pattern, bug, context, learning)")
	storeCmd.Flags().StringVar(&storeRelatedFiles, "related-files", "", "Comma-separated file paths")
	storeCmd.Flags().StringVar(&storeDetails, "details", "", "Extended details or context")
	storeCmd.Flags().StringVar(&storeSource, "source", "", "Source of the item")
	storeCmd.Flags().StringVar(&storeProject, "project", "", "Project name (defaults to current directory)")
}
