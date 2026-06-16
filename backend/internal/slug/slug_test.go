package slug_test

import (
	"testing"

	"somotracker/backend/internal/slug"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"lowercase", "hello world", "hello-world"},
		{"uppercase", "Hello World", "hello-world"},
		{"mixed case and digits", "Somo Tracker 2025", "somo-tracker-2025"},
		{"hyphens and underscores", "school-1_alpha", "school-1-alpha"},
		{"dots and tildes", "school.one~two", "school-one-two"},
		{"special chars removed", "School №1 (test)", "school-1-test"},
		{"leading/trailing spaces", "  My School  ", "--my-school--"},
		{"only special chars", "!!! @@@ ###", "school-"},
		{"empty string", "", "school-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slug.Generate(tt.in)
			if len(got) < 2 {
				t.Errorf("Generate(%q) = %q, want at least 2 chars", tt.in, got)
			}
			// For the cases where we know the exact output
			if tt.want != "" && tt.want != "school-" {
				// For special chars and empty, the result has a timestamp suffix
				if got[:len(tt.want)] != tt.want {
					// Check prefix for timestamp-based fallbacks
					if !(len(got) >= len(tt.want) && got[:len(tt.want)] == tt.want) {
						t.Errorf("Generate(%q) = %q, want prefix %q", tt.in, got, tt.want)
					}
				}
			}
		})
	}
}

func TestGenerate_Predictable(t *testing.T) {
	// For inputs without special chars, result should be deterministic
	got := slug.Generate("My School Name")
	if got != "my-school-name" {
		t.Errorf("Generate(%q) = %q, want %q", "My School Name", got, "my-school-name")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want bool
	}{
		{"valid simple", "my-school", true},
		{"valid with dot", "school.1", true},
		{"valid with tilde", "school~one", true},
		{"valid with underscore", "school_1", true},
		{"too short", "a", false},
		{"empty", "", false},
		{"invalid chars", "school!name", false},
		{"no alphanumeric", "---", false},
		{"uppercase allowed", "School-Name", true},
		{"digits only", "school123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slug.Validate(tt.slug)
			if got != tt.want {
				t.Errorf("Validate(%q) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}
