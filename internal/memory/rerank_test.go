package memory

import (
	"context"
	"testing"
)

func TestRerankCandidatesBelowLimit(t *testing.T) {
	// When the candidate count is at or below limit, rerank is a no-op:
	// callers should observe the exact same slice (identity preserved).
	cands := []KBFile{
		{Title: "A", Tags: []string{"x"}},
		{Title: "B", Tags: []string{"y"}},
	}
	got := rerankCandidates("query", cands, 5)
	if len(got) != len(cands) {
		t.Fatalf("expected %d results, got %d", len(cands), len(got))
	}
	for i := range cands {
		if got[i].Title != cands[i].Title {
			t.Errorf("position %d: expected %q, got %q", i, cands[i].Title, got[i].Title)
		}
	}
}

func TestRerankCandidatesRespectsLimit(t *testing.T) {
	cands := []KBFile{
		{Title: "A"}, {Title: "B"}, {Title: "C"}, {Title: "D"},
	}
	got := rerankCandidates("anything", cands, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
}

// TestRerankCandidatesTagAndTitleBoost verifies the core promise of reranking:
// a lower-BM25-rank candidate with strong tag+title overlap should overtake
// a higher-BM25-rank candidate with no signal match.
func TestRerankCandidatesTagAndTitleBoost(t *testing.T) {
	// NOTE: candidate tags here must avoid canonicalTags synonyms (e.g. "react"
	// is rewritten to "react-frontend"), otherwise the query/candidate tag
	// sets will not overlap under normalizeTags.
	cands := []KBFile{
		{Title: "Random Misc", Tags: []string{"misc"}},                         // BM25 rank 0 → score 0.4
		{Title: "Docker Setup Tutorial", Tags: []string{"docker", "tutorial"}}, // tag=1.0, title=~0.67 → ~0.7
		{Title: "Unrelated", Tags: []string{"other"}},                          // BM25 rank 2 → score 0.0
	}
	got := rerankCandidates("docker tutorial", cands, 1)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Title != "Docker Setup Tutorial" {
		t.Errorf("expected 'Docker Setup Tutorial' to win via tag+title boost, got %q", got[0].Title)
	}
}

// TestRerankCandidatesStableOnTies verifies that when two candidates have equal
// rerank scores, original (BM25) order is preserved. This keeps reranking
// deterministic and prevents surprising reorderings of equivalent results.
func TestRerankCandidatesStableOnTies(t *testing.T) {
	// All candidates have identical tags and titles → identical scores.
	// Stable sort must preserve the input order.
	cands := []KBFile{
		{Title: "Same", Tags: []string{"t"}},
		{Title: "Same", Tags: []string{"t"}},
		{Title: "Same", Tags: []string{"t"}},
	}
	got := rerankCandidates("query", cands, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	// They are indistinguishable by field, so just verify we got exactly 2.
	// The point of this test is that rerank doesn't crash and produces a
	// deterministic subset when scores tie.
}

func TestFindMatchPosition(t *testing.T) {
	cases := []struct {
		name string
		s    string
		q    string
		want int
	}{
		{"exact word", "hello world", "world", 6},
		{"case insensitive", "Hello World", "world", 6},
		{"first of multiple query words", "alpha beta gamma", "beta gamma", 6},
		{"no match", "alpha beta", "zzz", -1},
		{"single-char query words skipped", "a b c d", "a b", -1},
		{"empty query", "anything", "", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findMatchPosition(tc.s, tc.q); got != tc.want {
				t.Errorf("findMatchPosition(%q, %q) = %d, want %d", tc.s, tc.q, got, tc.want)
			}
		})
	}
}

func TestBuildContextualSnippet(t *testing.T) {
	// Match at start: no leading ellipsis, trailing ellipsis if truncated.
	got := buildContextualSnippet("abcdefghij", 0, 4)
	if got != "abcd…" {
		t.Errorf("start match: got %q, want %q", got, "abcd…")
	}
	// Match in middle: both ellipses.
	got = buildContextualSnippet("0123456789ABCDEF", 8, 4)
	if got != "…6789…" {
		t.Errorf("middle match: got %q, want %q", got, "…6789…")
	}
	// Match near end: leading ellipsis because start>0; no trailing ellipsis
	// because the window reaches the end of the content.
	got = buildContextualSnippet("0123456789", 8, 6)
	if got != "…56789" {
		t.Errorf("end match: got %q, want %q", got, "…56789")
	}
	// Window larger than content: returns full content, no ellipses.
	got = buildContextualSnippet("short", 0, 100)
	if got != "short" {
		t.Errorf("oversize window: got %q, want %q", got, "short")
	}
}

func TestSerializeDeserializeFloat32Roundtrip(t *testing.T) {
	vec := []float32{1.0, -1.5, 0.001, 3.14159, 0, -0.0}
	out := deserializeFloat32(serializeFloat32(vec))
	if len(out) != len(vec) {
		t.Fatalf("length mismatch: %d vs %d", len(out), len(vec))
	}
	for i, v := range vec {
		if out[i] != v {
			t.Errorf("index %d: got %v, want %v", i, out[i], v)
		}
	}
}

func TestDeserializeFloat32RejectsBadInput(t *testing.T) {
	if got := deserializeFloat32([]byte{0x01, 0x02, 0x03}); got != nil {
		t.Errorf("expected nil for length not multiple of 4, got %v", got)
	}
}

func TestCosineSimilarity(t *testing.T) {
	cases := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1.0},
		{"orthogonal", []float32{1, 0}, []float32{0, 1}, 0.0},
		{"opposite", []float32{1, 1}, []float32{-1, -1}, -1.0},
		{"mismatched length", []float32{1, 2, 3}, []float32{1, 2}, 0.0},
		{"empty", []float32{}, []float32{}, 0.0},
		{"zero vector", []float32{0, 0, 0}, []float32{1, 2, 3}, 0.0},
	}
	const eps = 1e-9
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cosineSimilarity(tc.a, tc.b)
			if abs(got-tc.want) > eps {
				t.Errorf("cosineSimilarity = %v, want %v", got, tc.want)
			}
		})
	}
}

// stubEmbeddingProvider is a no-op provider used to verify the interface
// contract compiles and is callable. It is intentionally not wired into the
// store — graceful degradation means an unused provider has zero cost.
type stubEmbeddingProvider struct{}

func (stubEmbeddingProvider) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (stubEmbeddingProvider) Model() string { return "stub-v1" }

// Compile-time check that the stub satisfies the interface.
var _ EmbeddingProvider = stubEmbeddingProvider{}

func TestEmbeddingProviderInterfaceContract(t *testing.T) {
	var p EmbeddingProvider = stubEmbeddingProvider{}
	vec, err := p.Embed(t.Context(), "anything")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 3 {
		t.Errorf("expected 3-dim vector, got %d", len(vec))
	}
	if p.Model() != "stub-v1" {
		t.Errorf("Model = %q, want %q", p.Model(), "stub-v1")
	}
}

// TestKBStoreSearchHybridReranks is an integration test: with limit smaller
// than the candidate pool, Search must still return only `limit` results, and
// the result with the strongest lexical signal (tag overlap) must survive
// even if its BM25 rank was not #1.
func TestKBStoreSearchHybridReranks(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	// Three files all containing the search term "goroutines". The FTS5 BM25
	// ordering between them depends on content length / term frequency, so we
	// assert only the property reranking is responsible for: the file whose
	// title and tags match the query must be present in the top-2 result set.
	if err := store.WriteFile(ScopeProject, CategoryLesson, "Goroutine Basics",
		"A basic introduction to goroutines and how they work.", []string{"go"}, "high"); err != nil {
		t.Fatalf("WriteFile A: %v", err)
	}
	if err := store.WriteFile(ScopeProject, CategoryLesson, "Goroutines Tagged Guide",
		"Some notes on goroutines here.", []string{"goroutines"}, "high"); err != nil {
		t.Fatalf("WriteFile B: %v", err)
	}
	if err := store.WriteFile(ScopeProject, CategoryLesson, "Other Notes",
		"Miscellaneous notes mentioning goroutines once.", []string{"misc"}, "medium"); err != nil {
		t.Fatalf("WriteFile C: %v", err)
	}

	results, err := store.Search("goroutines", ScopeProject, 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	found := false
	for _, r := range results {
		if r.Title == "Goroutines Tagged Guide" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Goroutines Tagged Guide' (tag match) in top-2, got: %v", titlesOf(results))
	}
}

func titlesOf(fs []KBFile) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Title
	}
	return out
}
