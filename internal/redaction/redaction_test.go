package redaction

import "testing"

func TestRedact(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		patterns []string
		want     string
	}{
		{
			name:     "explicit redacted tags",
			input:    "My key is <redacted>secret</redacted>",
			patterns: []string{},
			want:     "My key is [REDACTED]",
		},
		{
			name:     "nested redacted tags",
			input:    "Value: <redacted>outer<redacted>inner</redacted></redacted>",
			patterns: []string{},
			want:     "Value: [REDACTED]",
		},
		{
			name:     "stripe key",
			input:    "API key: sk_live_abc123xyz",
			patterns: []string{},
			want:     "API key: [REDACTED]",
		},
		{
			name:     "github token",
			input:    "Token: ghp_abcdefghijklmnop",
			patterns: []string{},
			want:     "Token: [REDACTED]",
		},
		{
			name:     "custom pattern",
			input:    "SSN: 123-45-6789",
			patterns: []string{`\d{3}-\d{2}-\d{4}`},
			want:     "SSN: [REDACTED]",
		},
		{
			name:     "no sensitive data",
			input:    "This is a normal string with no secrets",
			patterns: []string{},
			want:     "This is a normal string with no secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Redact(tt.input, tt.patterns)
			if got != tt.want {
				t.Errorf("Redact() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadPantryIgnore(t *testing.T) {
	// Test with non-existent file
	patterns, err := LoadPantryIgnore("/nonexistent/path/.pantryignore")
	if err != nil {
		t.Errorf("LoadPantryIgnore() error = %v, want nil", err)
	}
	if len(patterns) != 0 {
		t.Errorf("LoadPantryIgnore() = %v, want empty slice", patterns)
	}
}
