package agent

import (
	"regexp"
	"strings"
)

var thinkTagRe = regexp.MustCompile(`(?s)<think\b[^>]*>.*?</think\s*>`)

// stripThinkTags removes <think>...</think> reasoning blocks from a complete
// string. Handles closed tags via regex, then strips any residual unclosed
// <think> tag (from tag start to end of string).
func stripThinkTags(s string) string {
	s = thinkTagRe.ReplaceAllString(s, "")
	if idx := strings.Index(s, "<think"); idx >= 0 {
		closeIdx := strings.Index(s[idx:], "</think")
		if closeIdx >= 0 {
			s = s[:idx] + s[idx+closeIdx+len("</think"):]
		} else {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}

// thinkStreamFilter separates <think>...</think> reasoning blocks from
// regular content in a streaming text channel. It correctly handles tags
// split across multiple Write calls.
type thinkStreamFilter struct {
	pending string // unprocessed text that may contain partial tag boundaries
	inThink bool   // currently inside a <think> block
}

// Write processes a chunk of streaming text and returns the regular content
// and thinking content extracted from it. Some text may be buffered
// internally if it could be part of a tag that spans the next chunk.
func (f *thinkStreamFilter) Write(chunk string) (text, thinking string) {
	f.pending += chunk
	var textBuf, thinkBuf strings.Builder

	for {
		if !f.inThink {
			idx := strings.Index(f.pending, "<think")
			if idx < 0 {
				safe := safeFlushLen(f.pending, "<think")
				if safe > 0 {
					textBuf.WriteString(f.pending[:safe])
					f.pending = f.pending[safe:]
				}
				break
			}
			if idx > 0 {
				textBuf.WriteString(f.pending[:idx])
			}
			f.pending = f.pending[idx:]
			// Skip past the '>' that ends the opening <think ...> tag
			gtIdx := strings.Index(f.pending, ">")
			if gtIdx < 0 {
				break // opening tag incomplete, wait for more data
			}
			f.pending = f.pending[gtIdx+1:]
			f.inThink = true
		}

		// Inside think block — look for closing </think
		closeIdx := strings.Index(f.pending, "</think")
		if closeIdx < 0 {
			safe := safeFlushLen(f.pending, "</think")
			if safe > 0 {
				thinkBuf.WriteString(f.pending[:safe])
				f.pending = f.pending[safe:]
			}
			break
		}
		thinkBuf.WriteString(f.pending[:closeIdx])
		gtIdx := strings.Index(f.pending[closeIdx:], ">")
		if gtIdx >= 0 {
			f.pending = f.pending[closeIdx+gtIdx+1:]
		} else {
			f.pending = f.pending[closeIdx:]
		}
		f.inThink = false
	}

	return textBuf.String(), thinkBuf.String()
}

// Flush returns any remaining buffered content. Call once after the last Write.
func (f *thinkStreamFilter) Flush() (text, thinking string) {
	remaining := f.pending
	f.pending = ""
	if f.inThink {
		return "", remaining
	}
	return remaining, ""
}

// safeFlushLen returns the number of bytes that can be safely emitted
// without cutting a potential partial tag at the end of s. For example,
// if s ends with "<thi" and tag is "<think", only len(s)-3 bytes are safe.
func safeFlushLen(s, tag string) int {
	n := len(s)
	maxOverlap := len(tag)
	if maxOverlap > n {
		maxOverlap = n
	}
	for i := maxOverlap; i > 0; i-- {
		if strings.HasPrefix(tag, s[n-i:]) {
			return n - i
		}
	}
	return n
}
