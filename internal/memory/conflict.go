package memory

import (
	"fmt"
	"strings"
)

// contradictionKeywords are word pairs that indicate opposing claims.
// When two memories share significant keyword overlap but contain opposite
// markers from this set, they are flagged as potential conflicts.
var contradictionKeywords = map[string]string{
	"always":    "never",
	"must":      "must not",
	"should":    "should not",
	"required":  "optional",
	"recommend": "avoid",
	"prefer":    "unprefer",
	"enable":    "disable",
	"use":       "unuse",
	"best":      "worst",
	"fast":      "slow",
	"increase":  "decrease",
}

// detectContradiction checks whether titles indicate potentially contradictory
// memories. It's a lightweight lexical heuristic — no LLM involved.
func detectContradiction(aTitle, aContent, bTitle string) bool {
	aLower := strings.ToLower(aTitle + " " + aContent)
	bLower := strings.ToLower(bTitle)

	aWords := toWordSet(aLower)
	bWords := toWordSet(bLower)

	// Require at least some keyword overlap to be meaningful.
	common := 0
	for w := range aWords {
		if bWords[w] {
			common++
		}
	}
	if common == 0 {
		return false
	}

	// Check for opposing markers.
	for positive, negative := range contradictionKeywords {
		aHasPos := aWords[positive]
		bHasPos := bWords[positive]
		aHasNeg := containsPhrase(aLower, negative)
		bHasNeg := containsPhrase(bLower, negative)

		// One side asserts positive, the other asserts the opposite.
		if (aHasPos && bHasNeg) || (bHasPos && aHasNeg) {
			return true
		}
	}
	return false
}

func containsPhrase(text, phrase string) bool {
	return strings.Contains(text, phrase)
}

// markConflict records a potential conflict between a newly written memory and
// an existing one via the LogEntry mechanism. This surfaces during Review.
func (s *KBStore) markConflict(scope, newTitle, existingPath string) error {
	return s.LogEntry(scope, "冲突标记",
		fmt.Sprintf("新记忆 \"%s\" 可能与现有记忆 %s 存在矛盾，请人工审查", newTitle, existingPath))
}
