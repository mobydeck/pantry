package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pantry/internal/models"
)

// WriteSessionItem writes an item to a session file
func WriteSessionItem(vaultProjectDir string, item models.Item, dateStr string, details *string) (string, error) {
	filePath := filepath.Join(vaultProjectDir, fmt.Sprintf("%s-session.md", dateStr))
	sectionContent := renderSection(item, details)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create new file
		content := createNewSessionFile(item, dateStr, sectionContent)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write session file: %w", err)
		}
	} else {
		// Append to existing file
		existingContent, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read session file: %w", err)
		}
		updatedContent := appendToSessionFile(string(existingContent), item, sectionContent)
		if err := os.WriteFile(filePath, []byte(updatedContent), 0644); err != nil {
			return "", fmt.Errorf("failed to update session file: %w", err)
		}
	}

	return filePath, nil
}

// renderSection renders a single H3 section from an Item
func renderSection(item models.Item, details *string) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("### %s", item.Title))
	lines = append(lines, fmt.Sprintf("**What:** %s", item.What))

	if item.Why != nil {
		lines = append(lines, fmt.Sprintf("**Why:** %s", *item.Why))
	}

	if item.Impact != nil {
		lines = append(lines, fmt.Sprintf("**Impact:** %s", *item.Impact))
	}

	if item.Source != nil {
		lines = append(lines, fmt.Sprintf("**Source:** %s", *item.Source))
	}

	if details != nil {
		lines = append(lines, "")
		lines = append(lines, "<details>")
		lines = append(lines, *details)
		lines = append(lines, "</details>")
	}

	return strings.Join(lines, "\n")
}

// createNewSessionFile creates a new session file with frontmatter and initial content
func createNewSessionFile(item models.Item, dateStr string, sectionContent string) string {
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
	lines = append(lines, fmt.Sprintf("project: %s", item.Project))
	if len(sources) > 0 {
		lines = append(lines, fmt.Sprintf("sources: [%s]", strings.Join(sources, ", ")))
	}
	lines = append(lines, fmt.Sprintf("created: %s", now))
	if len(tags) > 0 {
		lines = append(lines, fmt.Sprintf("tags: [%s]", strings.Join(tags, ", ")))
	}
	lines = append(lines, "---")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("# %s Session", dateStr))
	lines = append(lines, "")

	if item.Category != nil {
		categoryHeading := models.CategoryHeadings[*item.Category]
		lines = append(lines, fmt.Sprintf("## %s", categoryHeading))
		lines = append(lines, "")
	}

	lines = append(lines, sectionContent)

	return strings.Join(lines, "\n") + "\n"
}

// appendToSessionFile appends item to existing session file, updating frontmatter and structure
func appendToSessionFile(content string, item models.Item, sectionContent string) string {
	// Split frontmatter and body
	frontmatter, body := splitFrontmatter(content)

	// Update frontmatter
	updatedFrontmatter := updateFrontmatter(frontmatter, item)

	// Update body with new section
	updatedBody := insertSectionInBody(body, item, sectionContent)

	return updatedFrontmatter + "\n" + updatedBody
}

// splitFrontmatter splits content into frontmatter and body
func splitFrontmatter(content string) (string, string) {
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) >= 3 {
		frontmatter := "---\n" + parts[1] + "---"
		body := parts[2]
		return frontmatter, body
	}
	return "", content
}

// updateFrontmatter updates frontmatter with new tags and sources
func updateFrontmatter(frontmatter string, item models.Item) string {
	lines := strings.Split(frontmatter, "\n")
	var updatedLines []string

	existingTags := []string{}
	existingSources := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, "tags:") {
			// Extract existing tags
			if idx := strings.Index(line, "["); idx != -1 {
				if idx2 := strings.Index(line[idx:], "]"); idx2 != -1 {
					tagsStr := line[idx+1 : idx+idx2]
					if tagsStr != "" {
						tags := strings.Split(tagsStr, ",")
						for _, t := range tags {
							t = strings.TrimSpace(t)
							if t != "" {
								existingTags = append(existingTags, t)
							}
						}
					}
				}
			}
		} else if strings.HasPrefix(line, "sources:") {
			// Extract existing sources
			if idx := strings.Index(line, "["); idx != -1 {
				if idx2 := strings.Index(line[idx:], "]"); idx2 != -1 {
					sourcesStr := line[idx+1 : idx+idx2]
					if sourcesStr != "" {
						sources := strings.Split(sourcesStr, ",")
						for _, s := range sources {
							s = strings.TrimSpace(s)
							if s != "" {
								existingSources = append(existingSources, s)
							}
						}
					}
				}
			}
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
		found := false
		for _, s := range existingSources {
			if s == *item.Source {
				found = true
				break
			}
		}
		if !found {
			existingSources = append(existingSources, *item.Source)
		}
	}

	// Rebuild frontmatter
	for _, line := range lines {
		if strings.HasPrefix(line, "tags:") {
			if len(tagList) > 0 {
				updatedLines = append(updatedLines, fmt.Sprintf("tags: [%s]", strings.Join(tagList, ", ")))
			} else {
				updatedLines = append(updatedLines, "tags: []")
			}
		} else if strings.HasPrefix(line, "sources:") {
			if len(existingSources) > 0 {
				updatedLines = append(updatedLines, fmt.Sprintf("sources: [%s]", strings.Join(existingSources, ", ")))
			} else {
				updatedLines = append(updatedLines, "sources: []")
			}
		} else {
			updatedLines = append(updatedLines, line)
		}
	}

	return strings.Join(updatedLines, "\n")
}

// insertSectionInBody inserts section in body at correct position based on category
func insertSectionInBody(body string, item models.Item, sectionContent string) string {
	if item.Category == nil {
		// No category, just append at end
		return strings.TrimRight(body, "\n") + "\n\n" + sectionContent + "\n"
	}

	categoryHeading := models.CategoryHeadings[*item.Category]

	// Check if category heading already exists
	if strings.Contains(body, fmt.Sprintf("## %s", categoryHeading)) {
		// Append under existing heading
		return appendUnderExistingCategory(body, categoryHeading, sectionContent)
	}

	// Insert new category heading in correct order
	return insertNewCategory(body, *item.Category, categoryHeading, sectionContent)
}

// appendUnderExistingCategory appends section under existing category heading
func appendUnderExistingCategory(body string, categoryHeading string, sectionContent string) string {
	lines := strings.Split(body, "\n")
	var resultLines []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		resultLines = append(resultLines, line)

		// Found the target category heading
		if line == fmt.Sprintf("## %s", categoryHeading) {
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

// insertNewCategory inserts new category heading at correct position
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
		if strings.HasPrefix(line, "## ") {
			// Extract category from heading
			headingText := strings.TrimSpace(line[3:])
			for _, cat := range categoryOrder {
				if models.CategoryHeadings[cat] == headingText {
					catIndex := -1
					for j, c := range categoryOrder {
						if c == cat {
							catIndex = j
							break
						}
					}
					if catIndex > targetIndex {
						// Found a category that should come after ours
						insertPosition = i
						break
					}
				}
			}
			if insertPosition < len(lines) {
				break
			}
		}
	}

	// Insert new category section
	newLines := append(lines[:insertPosition],
		append([]string{fmt.Sprintf("## %s", categoryHeading), "", sectionContent, ""},
			lines[insertPosition:]...)...)

	return strings.TrimRight(strings.Join(newLines, "\n"), "\n") + "\n"
}
