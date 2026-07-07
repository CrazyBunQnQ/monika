package builtin

import (
	"strings"
	"testing"
)

func TestFindFirstBalancedBraces(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantHit  bool
		wantJSON string
	}{
		{
			name:     "simple",
			input:    `{"summary":"hi"}`,
			wantHit:  true,
			wantJSON: `{"summary":"hi"}`,
		},
		{
			name:    "no braces",
			input:   "no json here",
			wantHit: false,
		},
		{
			name:     "nested",
			input:    `{"a":{"b":1}}`,
			wantHit:  true,
			wantJSON: `{"a":{"b":1}}`,
		},
		{
			name:     "with prose and fence",
			input:    "```json\n{\"k\":\"v\"}\n```",
			wantHit:  true,
			wantJSON: "{\"k\":\"v\"}",
		},
		{
			name:     "two objects picks the first balanced one",
			input:    `{"a":1}{"summary":"second"}`,
			wantHit:  true,
			wantJSON: `{"a":1}`,
		},
		{
			name:     "string with brace inside",
			input:    `{"text":"has } inside"}`,
			wantHit:  true,
			wantJSON: `{"text":"has } inside"}`,
		},
		{
			name:     "escaped quote in string",
			input:    `{"text":"he said \"hi\""}`,
			wantHit:  true,
			wantJSON: `{"text":"he said \"hi\""}`,
		},
		{
			name:    "only opening brace",
			input:   `{"summary":"hi"`,
			wantHit: false,
		},
		{
			name:     "preceding prose",
			input:    `Here is the JSON: {"summary":"hi"} thanks`,
			wantHit:  true,
			wantJSON: `{"summary":"hi"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := findFirstBalancedBraces(tc.input)
			if !tc.wantHit {
				if start != -1 {
					t.Errorf("expected no match, got start=%d end=%d", start, end)
				}
				return
			}
			if start == -1 {
				t.Fatalf("expected a match, got none")
			}
			got := tc.input[start : end+1]
			if got != tc.wantJSON {
				t.Errorf("got %q, want %q", got, tc.wantJSON)
			}
		})
	}
}

func TestParseModelJSON(t *testing.T) {
	t.Run("well-formed whole string", func(t *testing.T) {
		out := parseModelJSON(`{"summary":"hi","timeline":[]}`)
		if out == nil {
			t.Fatal("expected non-nil")
		}
		if out["summary"] != "hi" {
			t.Errorf("summary=%v", out["summary"])
		}
	})

	t.Run("fenced json", func(t *testing.T) {
		out := parseModelJSON("```json\n{\"summary\":\"hi\"}\n```")
		if out == nil {
			t.Fatal("expected non-nil")
		}
		if out["summary"] != "hi" {
			t.Errorf("summary=%v", out["summary"])
		}
	})

	t.Run("prose + json", func(t *testing.T) {
		out := parseModelJSON(`Here you go: {"summary":"hi","key_moments":[]}`)
		if out == nil {
			t.Fatal("expected non-nil")
		}
		if out["summary"] != "hi" {
			t.Errorf("summary=%v", out["summary"])
		}
	})

	t.Run("two objects picks the first balanced one", func(t *testing.T) {
		out := parseModelJSON(`Here's an example: {"foo":1} and the real answer: {"summary":"real"}`)
		if out == nil {
			t.Fatal("expected non-nil")
		}
		// First balanced object wins. The second object (which has
		// the "real" answer) is not extracted; if the model emits
		// an example followed by the answer, the prompt must steer
		// it not to.
		if out["foo"] != float64(1) {
			t.Errorf("foo=%v", out["foo"])
		}
	})

	t.Run("unbalanced only returns nil", func(t *testing.T) {
		// Old code: substring from first '{' to last '}' would have
		// returned an unbalanced slice that failed to unmarshal.
		// New code: findFirstBalancedBraces gives up, function
		// returns nil, caller falls back to raw-text summary.
		if got := parseModelJSON(`prose {not json `); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("no json returns nil", func(t *testing.T) {
		if got := parseModelJSON("nothing here"); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("nested objects parse", func(t *testing.T) {
		out := parseModelJSON(`{"timeline":[{"t":1.5,"what":"start"}]}`)
		if out == nil {
			t.Fatal("expected non-nil")
		}
	})
}

func TestSampleTimestamps(t *testing.T) {
	cases := []struct {
		name            string
		start, end      float64
		interval        float64
		maxN            int
		wantLen         int
		wantFirstLastOK bool
		// lastMargin is the minimum gap the last sample must keep
		// from `end`. Set when the test depends on the end-backoff.
		lastMargin float64
	}{
		{"single", 0, 10, 5, 4, 3, true, 0},
		{"max-one", 0, 60, 10, 1, 1, true, 0},
		{"zero span", 5, 5, 1, 4, 0, false, 0},
		{"cap at maxN", 0, 60, 10, 4, 4, true, 0},
		// Smoke test against a real 8.2s video revealed that the
		// last sample landing at exactly end (== video duration)
		// makes ffmpeg return "Output file is empty, nothing was
		// encoded". The new behaviour backs off so the last sample
		// lands on a real frame.
		{"keeps last sample clear of end", 0, 8.196, 1, 3, 3, true, 0.05},
		{"short span still produces a sample", 0, 0.2, 1, 4, 1, true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sampleTimestamps(tc.start, tc.end, tc.interval, tc.maxN)
			if len(got) != tc.wantLen {
				t.Errorf("len=%d, want %d", len(got), tc.wantLen)
			}
			if tc.wantLen == 0 {
				return
			}
			if got[0] < tc.start-0.001 || got[0] > tc.end+0.001 {
				t.Errorf("first=%.3f out of [%.3f, %.3f]", got[0], tc.start, tc.end)
			}
			last := got[len(got)-1]
			if last > tc.end+0.001 {
				t.Errorf("last=%.3f past end=%.3f", last, tc.end)
			}
			if tc.lastMargin > 0 && tc.end-last < tc.lastMargin {
				t.Errorf("last=%.3f too close to end=%.3f (margin=%.3f)", last, tc.end, tc.end-last)
			}
		})
	}
}

func TestParseTimeline(t *testing.T) {
	in := []any{
		map[string]any{"t": 1.0, "what": "start"},
		map[string]any{"t": 5.5, "what": "middle"},
		"not a map",
	}
	out := parseTimeline(in)
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	if out[0].T != 1.0 || out[0].What != "start" {
		t.Errorf("out[0]=%+v", out[0])
	}
	if out[1].T != 5.5 || out[1].What != "middle" {
		t.Errorf("out[1]=%+v", out[1])
	}
}

func TestParseKeyMoments(t *testing.T) {
	in := []any{
		map[string]any{"t": 2.0, "title": "intro", "description": "opens"},
	}
	out := parseKeyMoments(in)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Title != "intro" || out[0].Description != "opens" {
		t.Errorf("got %+v", out[0])
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		500:                    "500 B",
		2048:                   "2.0 KB",
		5 * 1024 * 1024:        "5.0 MB",
		3 * 1024 * 1024 * 1024: "3.00 GB",
	}
	for in, want := range cases {
		if got := formatBytes(in); !strings.Contains(got, want) && got != want {
			t.Errorf("formatBytes(%d)=%q, want ~%q", in, got, want)
		}
	}
}
