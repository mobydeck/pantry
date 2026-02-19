package redaction

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SensitivePatterns contains regex patterns for known sensitive data formats
var SensitivePatterns = []string{
	`sk_live_[a-zA-Z0-9]+`,                      // Stripe live keys
	`sk_test_[a-zA-Z0-9]+`,                     // Stripe test keys
	`ghp_[a-zA-Z0-9]+`,                         // GitHub personal access tokens
	`AKIA[0-9A-Z]{16}`,                         // AWS access key IDs
	`xoxb-[a-zA-Z0-9-]+`,                       // Slack bot tokens
	`-----BEGIN (?:RSA )?PRIVATE KEY-----`,      // Private keys (RSA and generic)
	`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+`,    // JWT tokens
	`password\s*[:=]\s*["']?.+`,                // Password fields
	`secret\s*[:=]\s*["']?.+`,                  // Secret fields
	`api[_-]?key\s*[:=]\s*["']?.+`,             // API key fields
}

// Redact applies three-layer redaction to text
func Redact(text string, extraPatterns []string) string {
	// Layer 1: Explicit <redacted> tags
	// Handle nested tags by repeatedly substituting until no more matches found
	redactedTagPattern := regexp.MustCompile(`<redacted>.*?</redacted>`)
	for {
		prevText := text
		text = redactedTagPattern.ReplaceAllString(text, "[REDACTED]")
		if prevText == text {
			break
		}
	}

	// Clean up any remaining orphaned tags
	text = strings.ReplaceAll(text, "<redacted>", "")
	text = strings.ReplaceAll(text, "</redacted>", "")

	// Layer 2 & 3: Automatic pattern detection (built-in + custom)
	allPatterns := append(SensitivePatterns, extraPatterns...)
	for _, pattern := range allPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		text = re.ReplaceAllString(text, "[REDACTED]")
	}

	return text
}

// LoadPantryIgnore loads custom redaction patterns from a .pantryignore file
func LoadPantryIgnore(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open .pantryignore: %w", err)
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read .pantryignore: %w", err)
	}

	return patterns, nil
}
