package memory

import (
	"sort"
	"strings"
)

// rerankCandidates re-sorts search results using a weighted combination of
// BM25 rank position, tag overlap, and title keyword match. This improves
// precision when multiple search channels (FTS5 + LIKE, project + global)
// return overlapping candidates. Pure Go computation — no LLM calls.
//
// Weighting rationale (based on CMU retrieval findings):
//   - BM25 rank (0.4): the underlying full-text rank already encodes term
//     frequency and document frequency; keep it as the strongest signal.
//   - Tag overlap (0.3): memories the user explicitly tagged with query-like
//     tags are almost always on-topic.
//   - Title keyword overlap (0.3): titles summarize the memory; matches here
//     indicate topical relevance even when body wording differs.
//
// Ties are broken by original order (stable sort), which preserves BM25 order.
func rerankCandidates(query string, candidates []KBFile, limit int) []KBFile {
	if len(candidates) <= limit {
		return candidates
	}

	queryTags := normalizeTags(strings.Fields(query))

	type scored struct {
		file  KBFile
		score float64
	}

	scoredList := make([]scored, len(candidates))
	n := len(candidates)
	for i, c := range candidates {
		// BM25 rank score: first result gets 1.0, last gets 0.0.
		bm25Score := 1.0
		if n > 1 {
			bm25Score = 1.0 - float64(i)/float64(n-1)
		}

		tagScore := tagOverlap(queryTags, c.Tags)
		titleScore := keywordOverlap(query, c.Title)

		scoredList[i] = scored{
			file:  c,
			score: bm25Score*0.4 + tagScore*0.3 + titleScore*0.3,
		}
	}

	// Stable so ties fall back to the original (BM25) order — deterministic.
	sort.SliceStable(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	result := make([]KBFile, 0, limit)
	for i := 0; i < limit && i < len(scoredList); i++ {
		result = append(result, scoredList[i].file)
	}
	return result
}
