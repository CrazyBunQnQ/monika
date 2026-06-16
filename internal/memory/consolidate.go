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
	if top.Score >= 0.8 {
		return "update", &top.File
	} else if top.Score >= 0.4 {
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
		overlap := tagOverlap(candidate.Tags, r.Tags)
		bm25Score := 0.5
		sims = append(sims, SimilarityResult{
			File:  r,
			Score: bm25Score + float64(overlap)*0.25,
		})
	}
	return sims, nil
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
