package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"pantry/internal/models"
)

// WriteNoteItem writes an item to a daily notes file.
func WriteNoteItem(projectDir string, item models.Item, dateStr string, details *string) (string, error) {
	filePath := filepath.Join(projectDir, dateStr+"-notes.md")
	sectionContent := renderSection(item, details)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create new file
		content := createNewNotesFile(item, dateStr, sectionContent)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write notes file: %w", err)
		}
	} else {
		// Append to existing file
		existingContent, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read notes file: %w", err)
		}

		updatedContent := appendToNotesFile(string(existingContent), item, sectionContent)
		if err := os.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
			return "", fmt.Errorf("failed to update notes file: %w", err)
		}
	}

	return filePath, nil
}

// renderSection renders a single H3 section from an Item.
func renderSection(item models.Item, details *string) string {
	var lines []string

	lines = append(lines, "### "+item.Title)
	lines = append(lines, "**What:** "+item.What)

	if item.Why != nil {
		lines = append(lines, "**Why:** "+*item.Why)
	}

	if item.Impact != nil {
		lines = append(lines, "**Impact:** "+*item.Impact)
	}

	if item.Source != nil {
		lines = append(lines, "**Source:** "+*item.Source)
	}

	if details != nil {
		lines = append(lines, "")
		lines = append(lines, "<details>")
		lines = append(lines, *details)
		lines = append(lines, "</details>")
	}

	return strings.Join(lines, "\n")
}

// createNewNotesFile creates a new notes file with frontmatter and initial content.
func createNewNotesFile(item models.Item, dateStr string, sectionContent string) string {
	now := time.Now().UTC().Format(time.RFC3339)

	sources := []string{}
	if item.Source != nil {
		sources = append(sources, *item.Source)
	}

	tags := make([]string, len(item.Tags))
	copy(tags, item.Tags)
	sort.Strings(tags)

	var lines []string

	lines = append(lines, "---")

	lines = append(lines, "project: "+item.Project)

	if len(sources) > 0 {
		lines = append(lines, fmt.Sprintf("sources: [%s]", strings.Join(sources, ", ")))
	}

	lines = append(lines, "created: "+now)
	if len(tags) > 0 {
		lines = append(lines, fmt.Sprintf("tags: [%s]", strings.Join(tags, ", ")))
	}

	lines = append(lines, "---")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("# %s Notes", dateStr))
	lines = append(lines, "")

	if item.Category != nil {
		categoryHeading := models.CategoryHeadings[*item.Category]
		lines = append(lines, "## "+categoryHeading)
		lines = append(lines, "")
	}

	lines = append(lines, sectionContent)

	return strings.Join(lines, "\n") + "\n"
}

// appendToNotesFile appends item to existing notes file, updating frontmatter and structure.
func appendToNotesFile(content string, item models.Item, sectionContent string) string {
	// Split frontmatter and body
	frontmatter, body := splitFrontmatter(content)

	// Update frontmatter
	updatedFrontmatter := updateFrontmatter(frontmatter, item)

	// Update body with new section
	updatedBody := insertSectionInBody(body, item, sectionContent)

	return updatedFrontmatter + "\n" + updatedBody
}

// splitFrontmatter splits content into frontmatter and body.
func splitFrontmatter(content string) (string, string) {
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) >= 3 {
		frontmatter := "---\n" + parts[1] + "---"
		body := parts[2]

		return frontmatter, body
	}

	return "", content
}

// updateFrontmatter updates frontmatter with new tags and sources.
// parseBracketedList extracts trimmed non-empty values from a "[a, b, c]" frontmatter line value.
func parseBracketedList(line string) []string {
	idx := strings.Index(line, "[")
	if idx == -1 {
		return nil
	}

	idx2 := strings.Index(line[idx:], "]")
	if idx2 == -1 {
		return nil
	}

	raw := line[idx+1 : idx+idx2]
	if raw == "" {
		return nil
	}

	var result []string

	for s := range strings.SplitSeq(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, s)
		}
	}

	return result
}

func updateFrontmatter(frontmatter string, item models.Item) string {
	lines := strings.Split(frontmatter, "\n")

	var updatedLines []string

	var existingTags, existingSources []string

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "tags:"):
			existingTags = parseBracketedList(line)
		case strings.HasPrefix(line, "sources:"):
			existingSources = parseBracketedList(line)
		}
	}

	// Merge and deduplicate tags
	allTags := make(map[string]bool)
	for _, t := range existingTags {
		allTags[strings.ToLower(t)] = true
	}

	for _, t := range item.Tags {
		allTags[strings.ToLower(t)] = true
	}

	tagList := make([]string, 0, len(allTags))
	for t := range allTags {
		tagList = append(tagList, t)
	}

	sort.Strings(tagList)

	// Merge sources
	if item.Source != nil {
		found := slices.Contains(existingSources, *item.Source)

		if !found {
			existingSources = append(existingSources, *item.Source)
		}
	}

	// Rebuild frontmatter
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "tags:"):
			if len(tagList) > 0 {
				updatedLines = append(updatedLines, fmt.Sprintf("tags: [%s]", strings.Join(tagList, ", ")))
			} else {
				updatedLines = append(updatedLines, "tags: []")
			}
		case strings.HasPrefix(line, "sources:"):
			if len(existingSources) > 0 {
				updatedLines = append(updatedLines, fmt.Sprintf("sources: [%s]", strings.Join(existingSources, ", ")))
			} else {
				updatedLines = append(updatedLines, "sources: []")
			}
		default:
			updatedLines = append(updatedLines, line)
		}
	}

	return strings.Join(updatedLines, "\n")
}

// insertSectionInBody inserts section in body at correct position based on category.
func insertSectionInBody(body string, item models.Item, sectionContent string) string {
	if item.Category == nil {
		// No category, just append at end
		return strings.TrimRight(body, "\n") + "\n\n" + sectionContent + "\n"
	}

	categoryHeading := models.CategoryHeadings[*item.Category]

	// Check if category heading already exists
	if strings.Contains(body, "## "+categoryHeading) {
		// Append under existing heading
		return appendUnderExistingCategory(body, categoryHeading, sectionContent)
	}

	// Insert new category heading in correct order
	return insertNewCategory(body, *item.Category, categoryHeading, sectionContent)
}

// appendUnderExistingCategory appends section under existing category heading.
func appendUnderExistingCategory(body string, categoryHeading string, sectionContent string) string {
	lines := strings.Split(body, "\n")

	var resultLines []string

	i := 0

	for i < len(lines) {
		line := lines[i]
		resultLines = append(resultLines, line)

		// Found the target category heading
		if line == "## "+categoryHeading {
			// Skip blank lines after heading
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				resultLines = append(resultLines, lines[i])
				i++
			}

			// Collect all H3 sections under this category
			for i < len(lines) && !strings.HasPrefix(lines[i], "## ") {
				resultLines = append(resultLines, lines[i])
				i++
			}

			// Insert new section before next H2 or end
			resultLines = append(resultLines, "")
			resultLines = append(resultLines, sectionContent)

			continue
		}

		i++
	}

	return strings.Join(resultLines, "\n") + "\n"
}

// insertNewCategory inserts new category heading at correct position.
func insertNewCategory(body string, category string, categoryHeading string, sectionContent string) string {
	// Get category order
	categoryOrder := models.ValidCategories
	targetIndex := -1

	for i, cat := range categoryOrder {
		if cat == category {
			targetIndex = i

			break
		}
	}

	if targetIndex == -1 {
		// Unknown category, append at end
		return strings.TrimRight(body, "\n") + "\n\n" + sectionContent + "\n"
	}

	lines := strings.Split(body, "\n")
	insertPosition := len(lines)

	// Find where to insert based on category order
	for i, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}

		headingText := strings.TrimSpace(line[3:])

		for _, cat := range categoryOrder {
			if models.CategoryHeadings[cat] != headingText {
				continue
			}

			if catIndex := slices.Index(categoryOrder, cat); catIndex > targetIndex {
				insertPosition = i
			}

			break
		}

		if insertPosition < len(lines) {
			break
		}
	}

	// Insert new category section
	newLines := append(lines[:insertPosition], //nolint:gocritic
		append([]string{"## " + categoryHeading, "", sectionContent, ""},
			lines[insertPosition:]...)...)

	return strings.TrimRight(strings.Join(newLines, "\n"), "\n") + "\n"
}
