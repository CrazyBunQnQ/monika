package memory

import (
	"strings"
)

type SimilarityResult struct {
	File  KBFile
	Score float64
}

func (s *KBStore) Consolidate(candidate ExtractCandidate, existing []SimilarityResult) (action string, targetFile *KBFile) {
	if len(existing) == 0 {
		return "new", nil
	}

	top := existing[0]
	if top.Score >= 0.75 {
		return "update", &top.File
	} else if top.Score >= 0.35 {
		return "new_linked", &top.File
	}
	return "new", nil
}

func (s *KBStore) ComputeSimilarity(candidate ExtractCandidate) ([]SimilarityResult, error) {
	results, err := s.Search(candidate.Title+" "+candidate.Content, "", 3)
	if err != nil {
		return nil, err
	}
	var sims []SimilarityResult
	for _, r := range results {
		tagSim := tagOverlap(candidate.Tags, r.Tags)
		titleSim := keywordOverlap(candidate.Title, r.Title)
		candidateLen := len(strings.Fields(candidate.Content))
		existingLen := r.CharCount / 5
		lenRatio := 0.5
		if candidateLen > 0 && existingLen > 0 {
			lenRatio = float64(min(candidateLen, existingLen)) / float64(max(candidateLen, existingLen))
		}
		score := tagSim*0.5 + titleSim*0.3 + lenRatio*0.2
		sims = append(sims, SimilarityResult{File: r, Score: score})
	}
	return sims, nil
}

// keywordOverlap returns the Jaccard-like overlap of word sets in a and b,
// normalized to 0-1 against the larger set.
func keywordOverlap(a, b string) float64 {
	aWords := toWordSet(a)
	bWords := toWordSet(b)
	if len(aWords) == 0 || len(bWords) == 0 {
		return 0
	}
	common := 0
	for w := range aWords {
		if bWords[w] {
			common++
		}
	}
	return float64(common) / float64(max(len(aWords), len(bWords)))
}

// toWordSet splits s into a set of lowercase words, skipping single chars.
func toWordSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(s)) {
		if len(w) > 1 {
			set[w] = true
		}
	}
	return set
}

func tagOverlap(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[t] = true
	}
	common := 0
	for _, t := range b {
		if set[t] {
			common++
		}
	}
	return float64(common) / float64(max(len(a), len(b)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MergeFileContent(existingBody, newContent, candidateTitle string) string {
	if strings.Contains(existingBody, candidateTitle) {
		return existingBody + "\n\n## 更新\n" + newContent
	}
	return existingBody + "\n\n## " + candidateTitle + "\n" + newContent
}
