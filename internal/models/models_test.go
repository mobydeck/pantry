package models

import "testing"

func TestFromRaw(t *testing.T) {
	raw := RawItemInput{
		Title: "Test Item",
		What:  "This is a test",
		Why:   stringPtr("Testing"),
		Tags:  []string{"test", "example"},
	}

	item := FromRaw(raw, "test-project", "/path/to/file.md")

	if item.ID == "" {
		t.Error("FromRaw() ID should not be empty")
	}
	if item.Title != "Test Item" {
		t.Errorf("FromRaw() Title = %q, want %q", item.Title, "Test Item")
	}
	if item.Project != "test-project" {
		t.Errorf("FromRaw() Project = %q, want %q", item.Project, "test-project")
	}
	if item.SectionAnchor == "" {
		t.Error("FromRaw() SectionAnchor should not be empty")
	}
}

func TestGenerateAnchor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Test 123", "test-123"},
		{"Special!@#Chars", "special-chars"},
		{"Multiple   Spaces", "multiple-spaces"},
	}

	for _, tt := range tests {
		raw := RawItemInput{Title: tt.input}
		item := FromRaw(raw, "test", "")
		if item.SectionAnchor != tt.want {
			t.Errorf("generateAnchor(%q) = %q, want %q", tt.input, item.SectionAnchor, tt.want)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
