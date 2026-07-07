package agent

import "testing"

func TestStripThinkTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"closed", "<think>reasoning here</think>actual content", "actual content"},
		{"multiline", "<think>\nline 1\nline 2\n</think>\n\nHello", "Hello"},
		{"attrs", `<think foo="bar">reasoning</think>text`, "text"},
		{"unclosed", "<think>still thinking...", ""},
		{"no_tags", "just regular text", "just regular text"},
		{"empty_think", "<think></think>after", "after"},
		{"multiple", "<think>a</think>mid<think>b</think>end", "midend"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripThinkTags(tt.input)
			if got != tt.expected {
				t.Errorf("stripThinkTags(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestThinkStreamFilterBasic(t *testing.T) {
	var f thinkStreamFilter
	text, thinking := f.Write("<think>secret</think>public")
	if thinking != "secret" {
		t.Errorf("thinking = %q, want %q", thinking, "secret")
	}
	if text != "public" {
		t.Errorf("text = %q, want %q", text, "public")
	}
}

func TestThinkStreamFilterSplitTag(t *testing.T) {
	var f thinkStreamFilter
	var allText, allThink string

	chunks := []string{"<th", "ink>sec", "ret</thi", "nk>pub", "lic"}
	for _, c := range chunks {
		text, thinking := f.Write(c)
		allText += text
		allThink += thinking
	}
	ft, ft2 := f.Flush()
	allText += ft
	allThink += ft2

	if allThink != "secret" {
		t.Errorf("thinking = %q, want %q", allThink, "secret")
	}
	if allText != "public" {
		t.Errorf("text = %q, want %q", allText, "public")
	}
}

func TestThinkStreamFilterNoTags(t *testing.T) {
	var f thinkStreamFilter
	text, thinking := f.Write("just text")
	if thinking != "" {
		t.Errorf("thinking = %q, want empty", thinking)
	}
	if text != "just text" {
		t.Errorf("text = %q, want %q", text, "just text")
	}
}

func TestThinkStreamFilterUnclosed(t *testing.T) {
	var f thinkStreamFilter
	text, thinking := f.Write("<think>still going")
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
	if thinking != "still going" {
		// The content after <think> is buffered as thinking
		t.Errorf("thinking = %q, want %q", thinking, "still going")
	}

	// Flush should return remaining as thinking
	ft, ft2 := f.Flush()
	if ft != "" {
		t.Errorf("flush text = %q, want empty", ft)
	}
	_ = ft2 // remaining buffer
}

func TestThinkStreamFilterMixed(t *testing.T) {
	var f thinkStreamFilter
	text, thinking := f.Write("before<think>hidden</think>after")
	if text != "beforeafter" {
		t.Errorf("text = %q, want %q", text, "beforeafter")
	}
	if thinking != "hidden" {
		t.Errorf("thinking = %q, want %q", thinking, "hidden")
	}
}

func TestSafeFlushLen(t *testing.T) {
	tests := []struct {
		s        string
		tag      string
		expected int
	}{
		{"hello", "<think", 5},      // no overlap
		{"hello<thi", "<think", 5},  // "<thi" is prefix of "<think"
		{"hello<thin", "<think", 5}, // "<thin" is prefix
		{"<think", "<think", 0},     // exact match, flush nothing
		{"hello<", "<think", 5},     // "<" is prefix
	}
	for _, tt := range tests {
		got := safeFlushLen(tt.s, tt.tag)
		if got != tt.expected {
			t.Errorf("safeFlushLen(%q, %q) = %d, want %d", tt.s, tt.tag, got, tt.expected)
		}
	}
}
