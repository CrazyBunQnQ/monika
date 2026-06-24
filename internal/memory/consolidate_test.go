package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestKeywordOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"go concurrency patterns", "go concurrency tips", 2.0 / 3.0},
		{"Go Concurrency", "go concurrency", 1.0},
		{"hello world", "foo bar", 0},
		{"", "something", 0},
		{"a b", "a b", 0}, // single chars skipped -> empty sets
	}
	for i, tc := range cases {
		got := keywordOverlap(tc.a, tc.b)
		if abs(got-tc.want) > 1e-9 {
			t.Errorf("case %d: keywordOverlap(%q,%q) = %v, want %v", i, tc.a, tc.b, got, tc.want)
		}
	}
}

func TestTagOverlapZero(t *testing.T) {
	if got := tagOverlap(nil, []string{"a"}); got != 0 {
		t.Errorf("tagOverlap(nil, [a]) = %v, want 0", got)
	}
	if got := tagOverlap([]string{"a", "b"}, []string{"a", "b"}); got != 1.0 {
		t.Errorf("tagOverlap identical = %v, want 1.0", got)
	}
}

func TestConsolidateThresholds(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	cases := []struct {
		score  float64
		action string
	}{
		{0.80, "update"},
		{0.75, "update"},
		{0.50, "new_linked"},
		{0.35, "new_linked"},
		{0.30, "new"},
		{0.0, "new"},
	}
	for _, tc := range cases {
		action, _ := store.Consolidate(ExtractCandidate{}, []SimilarityResult{{Score: tc.score}})
		if action != tc.action {
			t.Errorf("score %.2f: got %q, want %q", tc.score, action, tc.action)
		}
	}

	action, target := store.Consolidate(ExtractCandidate{}, nil)
	if action != "new" || target != nil {
		t.Errorf("empty existing: got %q/%v, want new/nil", action, target)
	}
}

func TestComputeSimilarityScoreRange(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	if err := store.WriteFile(ScopeProject, CategoryLesson, "Go Concurrency Patterns",
		"Using goroutines and channels for concurrent programming in Go.",
		[]string{"go", "concurrency"}, "high"); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	cand := ExtractCandidate{
		Title:   "Go Concurrency Patterns",
		Content: "Patterns for goroutines and channels.",
		Tags:    []string{"go", "concurrency"},
	}
	sims, err := store.ComputeSimilarity(cand)
	if err != nil {
		t.Fatalf("ComputeSimilarity: %v", err)
	}
	if len(sims) == 0 {
		t.Fatal("expected at least one similarity result")
	}
	for _, s := range sims {
		if s.Score < 0 || s.Score > 1.0 {
			t.Errorf("score %v out of [0,1] range", s.Score)
		}
	}
	if sims[0].Score < 0.75 {
		t.Errorf("identical candidate top score = %v, want >= 0.75", sims[0].Score)
	}
}

func TestWriteFileDedupRejects(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	if err := store.WriteFile(ScopeProject, CategoryLesson, "CORS Fix",
		"How to fix CORS errors in a Go server.", []string{"go", "cors"}, "high"); err != nil {
		t.Fatalf("first WriteFile: %v", err)
	}

	err := store.WriteFile(ScopeProject, CategoryLesson, "CORS Fix Guide",
		"Resolving CORS issues.", []string{"go", "cors"}, "medium")
	if err == nil {
		t.Fatal("expected dedup rejection for near-duplicate, got nil")
	}
	if !strings.Contains(err.Error(), "similar memory already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriteFileDedupAllowsDistinct(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	if err := store.WriteFile(ScopeProject, CategoryLesson, "CORS Fix",
		"How to fix CORS errors.", []string{"go", "cors"}, "high"); err != nil {
		t.Fatalf("first WriteFile: %v", err)
	}
	if err := store.WriteFile(ScopeProject, CategoryTopic, "React State Management",
		"Using useReducer for complex state.", []string{"react-frontend"}, "medium"); err != nil {
		t.Fatalf("distinct WriteFile should succeed, got: %v", err)
	}
}

// stubLLM returns a canned response for every Chat call, recording the count.
type stubLLM struct {
	resps   []string
	calls   int
	failAt  int // 1-based; 0 = never fail
	failErr error
}

func (s *stubLLM) Chat(ctx context.Context, system, user string) (string, error) {
	s.calls++
	if s.failAt > 0 && s.calls == s.failAt {
		return "", s.failErr
	}
	if s.calls <= len(s.resps) {
		return s.resps[s.calls-1], nil
	}
	return `{"candidates":[]}`, nil
}

func TestExtractStage2NoGaps(t *testing.T) {
	stage1 := &ExtractResult{Candidates: []ExtractCandidate{
		{Title: "Already Known", Category: "lesson"},
	}}
	llm := &stubLLM{resps: []string{`{"candidates":[]}`}}
	gaps, err := extractStage2SelfQuestion(context.Background(), llm, "summary", stage1)
	if err != nil {
		t.Fatalf("extractStage2SelfQuestion: %v", err)
	}
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestExtractStage2FindsGap(t *testing.T) {
	stage1 := &ExtractResult{Candidates: []ExtractCandidate{
		{Title: "Known A", Category: "topic"},
	}}
	llm := &stubLLM{resps: []string{
		`{"candidates":[{"title":"Missed Root Cause","content":"x","category":"lesson","scope":"project","tags":["bugfix"],"confidence":"high"}]}`,
	}}
	gaps, err := extractStage2SelfQuestion(context.Background(), llm, "summary", stage1)
	if err != nil {
		t.Fatalf("extractStage2SelfQuestion: %v", err)
	}
	if len(gaps) != 1 || gaps[0].Title != "Missed Root Cause" {
		t.Errorf("unexpected gaps: %+v", gaps)
	}
}

// TestExtractMemoriesStage2FailureNonFatal: stage1 succeeds, stage2 errors —
// ExtractMemories must still return stage1's result without error.
func TestExtractMemoriesStage2FailureNonFatal(t *testing.T) {
	llm := &stubLLM{
		resps:   []string{`{"candidates":[{"title":"Known","content":"x","category":"topic","scope":"project","tags":[],"confidence":"medium"}]}`},
		failAt:  2, // second Chat call (stage2) fails
		failErr: errors.New("stage2 boom"),
	}
	res, err := ExtractMemories(context.Background(), llm, ScopeProject, "s", "summary")
	if err != nil {
		t.Fatalf("stage2 failure must be non-fatal, got err: %v", err)
	}
	if res == nil || len(res.Candidates) != 1 {
		t.Fatalf("expected stage1 result preserved, got %+v", res)
	}
	if res.Candidates[0].Title != "Known" {
		t.Errorf("stage1 candidate lost: %+v", res.Candidates[0])
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func newTestStore(t *testing.T) *KBStore {
	t.Helper()
	s, err := NewKBStore(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	return s
}
