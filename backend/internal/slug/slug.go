// Package slug provides URL-friendly slug generation.
package slug

import (
	"fmt"
	"strings"
	"time"
)

// Generate creates a URL-friendly slug from a name.
//
// Rules:
//   - Uppercase letters are lowercased.
//   - Lowercase letters and digits are kept as-is.
//   - Spaces, hyphens, underscores, dots, and tildes become hyphens.
//   - All other characters are removed.
//   - If the result is empty, a timestamp-based fallback is used.
//
// Stytch further constrains slugs to: alphanumeric plus '-', '.', '_', '~',
// with at least one alphanumeric character and length 2-128.
func Generate(name string) string {
	var slug strings.Builder
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			slug.WriteRune(r + 32) // lowercase
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			slug.WriteRune(r)
		case r == ' ', r == '-', r == '_', r == '.', r == '~':
			slug.WriteRune('-')
		}
	}
	if slug.Len() == 0 {
		fmt.Fprintf(&slug, "school-%d", time.Now().UnixNano())
	}
	return slug.String()
}

// Validate returns true if the slug meets Stytch's requirements:
// length >= 2, only alphanumeric and '-', '.', '_', '~', at least one alphanumeric.
func Validate(slug string) bool {
	if len(slug) < 2 || len(slug) > 128 {
		return false
	}
	hasAlphaNum := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			hasAlphaNum = true
		case r == '-', r == '.', r == '_', r == '~':
			// allowed
		default:
			return false
		}
	}
	return hasAlphaNum
}
