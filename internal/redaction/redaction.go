package redaction

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SensitivePatterns contains regex patterns for known sensitive data formats.
var SensitivePatterns = []string{
	`sk_live_[a-zA-Z0-9]+`,                 // Stripe live keys
	`sk_test_[a-zA-Z0-9]+`,                 // Stripe test keys
	`ghp_[a-zA-Z0-9]+`,                     // GitHub personal access tokens
	`AKIA[0-9A-Z]{16}`,                     // AWS access key IDs
	`xoxb-[a-zA-Z0-9-]+`,                   // Slack bot tokens
	`-----BEGIN (?:RSA )?PRIVATE KEY-----`, // Private keys (RSA and generic)
	`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+`, // JWT tokens
	`password\s*[:=]\s*["']?.+`,            // Password fields
	`secret\s*[:=]\s*["']?.+`,              // Secret fields
	`api[_-]?key\s*[:=]\s*["']?.+`,         // API key fields
}

// compiledBuiltins holds pre-compiled versions of SensitivePatterns.
// Compiled once at package init; zero runtime cost per Redact call.
var (
	compiledBuiltins []*regexp.Regexp
	redactedTagRe    = regexp.MustCompile(`<redacted>.*?</redacted>`)
)

func init() {
	compiledBuiltins = make([]*regexp.Regexp, 0, len(SensitivePatterns))
	for _, p := range SensitivePatterns {
		// All built-in patterns are hardcoded and valid; panic immediately if not.
		compiledBuiltins = append(compiledBuiltins, regexp.MustCompile(p))
	}
}

// CompilePatterns compiles a slice of regex strings into []*regexp.Regexp.
// Invalid patterns are skipped. Intended for pre-compiling custom .pantryignore
// patterns once at service startup.
func CompilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))

	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}

	return compiled
}

// Redact applies three-layer redaction to text.
//   - Layer 1: explicit <redacted>…</redacted> tags
//   - Layer 2: built-in sensitive patterns (pre-compiled)
//   - Layer 3: caller-supplied extra patterns as raw strings (compiled on first use)
//
// Use RedactCompiled for hot paths where extra patterns are already compiled.
func Redact(text string, extraPatterns []string) string {
	return RedactCompiled(text, CompilePatterns(extraPatterns))
}

// RedactCompiled is the same as Redact but accepts pre-compiled extra patterns,
// avoiding regexp.Compile overhead on repeated calls.
func RedactCompiled(text string, extra []*regexp.Regexp) string {
	// Layer 1: Explicit <redacted> tags
	for {
		prev := text

		text = redactedTagRe.ReplaceAllString(text, "[REDACTED]")

		if prev == text {
			break
		}
	}

	text = strings.ReplaceAll(text, "<redacted>", "")
	text = strings.ReplaceAll(text, "</redacted>", "")

	// Layer 2: Built-in patterns (pre-compiled, zero cost)
	for _, re := range compiledBuiltins {
		text = re.ReplaceAllString(text, "[REDACTED]")
	}

	// Layer 3: Custom patterns
	for _, re := range extra {
		text = re.ReplaceAllString(text, "[REDACTED]")
	}

	return text
}

// LoadPantryIgnore loads custom redaction patterns from a .pantryignore file.
func LoadPantryIgnore(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, fmt.Errorf("failed to open .pantryignore: %w", err)
	}

	defer func() { _ = file.Close() }()

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
