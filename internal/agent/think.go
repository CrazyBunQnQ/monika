package agent

import (
	"regexp"
	"strings"
)

var thinkTagRe = regexp.MustCompile(`(?s)<think\b[^>]*>.*?</think\s*>`)

// splitThinkTags separates <think>...</think> reasoning blocks from regular
// content in a complete (non-streaming) string. Returns (cleanText, thinkContent).
func splitThinkTags(s string) (text, thinking string) {
	var thinkParts []string
	text = thinkTagRe.ReplaceAllStringFunc(s, func(match string) string {
		openEnd := strings.Index(match, ">")
		if openEnd < 0 {
			return ""
		}
		rest := match[openEnd+1:]
		closeIdx := strings.Index(rest, "</think")
		if closeIdx >= 0 {
			thinkParts = append(thinkParts, strings.TrimSpace(rest[:closeIdx]))
		} else {
			thinkParts = append(thinkParts, strings.TrimSpace(rest))
		}
		return ""
	})
	if idx := strings.Index(text, "<think"); idx >= 0 {
		openEnd := strings.Index(text[idx:], ">")
		if openEnd < 0 {
			thinkParts = append(thinkParts, strings.TrimSpace(text[idx+len("<think"):]))
			text = text[:idx]
		} else {
			cs := idx + openEnd + 1
			closeIdx := strings.Index(text[cs:], "</think")
			if closeIdx >= 0 {
				ce := cs + closeIdx
				thinkParts = append(thinkParts, strings.TrimSpace(text[cs:ce]))
				if gt := strings.Index(text[ce:], ">"); gt >= 0 {
					text = text[:idx] + text[ce+gt+1:]
				} else {
					text = text[:idx]
				}
			} else {
				thinkParts = append(thinkParts, strings.TrimSpace(text[cs:]))
				text = text[:idx]
			}
		}
	}
	text = strings.TrimSpace(text)
	thinking = strings.Join(thinkParts, "\n")
	return
}

// stripThinkTags removes <think>...</think> reasoning blocks from a complete
// string.
func stripThinkTags(s string) string {
	text, _ := splitThinkTags(s)
	return text
}

// thinkStreamFilter separates <think>...</think> reasoning blocks from
// regular content in a streaming text channel.
type thinkStreamFilter struct {
	pending string
	inThink bool
}

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
			gtIdx := strings.Index(f.pending, ">")
			if gtIdx < 0 {
				break
			}
			f.pending = f.pending[gtIdx+1:]
			f.inThink = true
		}

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

func (f *thinkStreamFilter) Flush() (text, thinking string) {
	remaining := f.pending
	f.pending = ""
	if f.inThink {
		return "", remaining
	}
	return remaining, ""
}

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
