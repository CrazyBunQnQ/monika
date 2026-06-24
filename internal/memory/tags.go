package memory

import "strings"

// canonicalTags maps common tag variants to their canonical form.
// Writing and searching both call normalizeTags to ensure consistency.
var canonicalTags = map[string]string{
	// I/O
	"parallel": "parallel-io", "concurrent": "parallel-io",
	"concurrent-write": "parallel-io", "batch-write": "batch-io",
	"batch-save": "batch-io",
	// Testing
	"test": "testing", "tests": "testing", "unit-test": "testing",
	// Error handling
	"bug": "bugfix", "fix": "bugfix", "error": "error-handling",
	// Frontend
	"frontend": "ui", "react": "react-frontend",
}

// normalizeTags converts free-form tags to canonical form, deduplicates, and lowercases.
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := make(map[string]bool)
	var result []string
	for _, t := range tags {
		canonical := canonicalTags[strings.ToLower(strings.TrimSpace(t))]
		if canonical == "" {
			canonical = strings.ToLower(strings.TrimSpace(t))
		}
		if !seen[canonical] {
			seen[canonical] = true
			result = append(result, canonical)
		}
	}
	return result
}
