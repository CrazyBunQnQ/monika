package memory

import (
	"reflect"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "synonyms collapse and dedup",
			in:   []string{"parallel", "concurrent"},
			want: []string{"parallel-io"},
		},
		{
			name: "case-insensitive and dedup",
			in:   []string{"Test", "TEST", "testing"},
			want: []string{"testing"},
		},
		{
			name: "unknown tags pass through lowercased",
			in:   []string{"custom-tag"},
			want: []string{"custom-tag"},
		},
		{
			name: "nil safe",
			in:   nil,
			want: []string{},
		},
		{
			name: "empty safe",
			in:   []string{},
			want: []string{},
		},
		{
			name: "mixed canonical and unknown preserves order",
			in:   []string{"bug", "react", "feature-x"},
			want: []string{"bugfix", "react-frontend", "feature-x"},
		},
		{
			name: "whitespace and case trimmed",
			in:   []string{"  Test  ", "Parallel"},
			want: []string{"testing", "parallel-io"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTags(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("normalizeTags(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
