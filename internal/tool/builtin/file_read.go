package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"monika/internal/tool"
)

type fileRead struct {
	projectDir string
	homeDir    string
	tsQuery    TSQueryFunc
	symFunc    LSPDiagFunc
}

func NewFileRead(projectDir, homeDir string, tsQuery TSQueryFunc) tool.Tool {
	return &fileRead{projectDir: projectDir, homeDir: homeDir, tsQuery: tsQuery}
}

func (f *fileRead) SetSymFunc(fn LSPDiagFunc) { f.symFunc = fn }

func (f *fileRead) Name() string { return "file_read" }
func (f *fileRead) Description() string {
	return "Read a section of a file from the local filesystem, or read a skill file by name. Use grep first to find the relevant file and line range, then read only the section you need using offset and limit. Output lines are prefixed with line number and content hash in '42│a1b2c3│ text' format. Copy the 'a1b2c3:42' portion as the anchor parameter for file_edit. If the file has more lines beyond the requested range, a footer hints the next offset. Optionally specify 'ranges' (e.g. '5-16,40-80') to read multiple non-contiguous sections. Set summary=true to prepend an AST-based structural summary (requires tree-sitter). When skillName is set (instead of filePath), the tool automatically resolves the path to the skill's SKILL.md from the project or global skills directory."
}

func (f *fileRead) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to read. Mutually exclusive with skillName.",
			},
			"skillName": map[string]any{
				"type":        "string",
				"description": "Name of a skill to read (resolves to SKILL.md automatically). Mutually exclusive with filePath.",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "The line number to start reading from (1-indexed). Defaults to 1.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of lines to read. Defaults to 200.",
			},
			"ranges": map[string]any{
				"type":        "string",
				"description": "Comma-separated line ranges to read, e.g. '5-16,40-80'. Overrides offset/limit.",
			},
			"summary": map[string]any{
				"type":        "boolean",
				"description": "If true, prepend a structured AST summary of the file (requires tree-sitter).",
			},
		},
	}
}

func (f *fileRead) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		FilePath  string `json:"filePath"`
		SkillName string `json:"skillName"`
		Offset    int    `json:"offset"`
		Limit     int    `json:"limit"`
		Ranges    string `json:"ranges"`
		Summary   bool   `json:"summary"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Offset < 1 {
		params.Offset = 1
	}
	if params.Limit < 1 {
		params.Limit = 200
	}

	// Validate mutual exclusivity of filePath and skillName.
	if params.FilePath != "" && params.SkillName != "" {
		return tool.ExecutionResult{
			Content: "filePath and skillName are mutually exclusive; provide only one",
			IsError: true,
		}, nil
	}
	if params.FilePath == "" && params.SkillName == "" {
		return tool.ExecutionResult{
			Content: "either filePath or skillName is required",
			IsError: true,
		}, nil
	}

	// Determine the file path: resolve skillName if set, otherwise use filePath.
	var safePath string
	if params.SkillName != "" {
		var err error
		safePath, err = f.resolveSkillPath(params.SkillName)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
	} else {
		var err error
		safePath, err = f.resolvePath(ctx, params.FilePath)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
	}

	if params.Ranges != "" {
		ranges, err := parseRanges(params.Ranges)
		if err != nil {
			return tool.ExecutionResult{Content: fmt.Sprintf("invalid ranges: %s", err), IsError: true}, nil
		}
		result, err := readFileRanges(safePath, ranges)
		if err == nil && params.Summary {
			result.Content = f.appendSummary(ctx, safePath, result.Content)
			if f.symFunc != nil {
				result.Content += f.symFunc(ctx, safePath)
			}
		}
		return result, err
	}

	result, err := readFileLines(safePath, params.Offset, params.Limit)
	if err == nil && params.Summary {
		result.Content = f.appendSummary(ctx, safePath, result.Content)
		if f.symFunc != nil {
			result.Content += f.symFunc(ctx, safePath)
		}
	}
	return result, err
}

func (f *fileRead) resolvePath(ctx context.Context, p string) (string, error) {
	return resolveToolPath(p, tool.ProjectDirOrDefault(ctx, f.projectDir))
}

var skillNameRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func isValidSkillName(name string) bool {
	return len(name) > 0 && len(name) <= 64 && skillNameRE.MatchString(name)
}

// resolveSkillPath resolves a skill name to the SKILL.md file path.
// It checks the project skills directory first, then the global skills directory.
func (f *fileRead) resolveSkillPath(skillName string) (string, error) {
	if !isValidSkillName(skillName) {
		return "", fmt.Errorf("invalid skill name %q", skillName)
	}
	candidates := []struct {
		root string
		kind string
	}{
		{f.projectDir, "project"},
		{f.homeDir, "global"},
	}
	for _, c := range candidates {
		if c.root == "" {
			continue
		}
		p := filepath.Join(c.root, ".monika", "skills", skillName, "SKILL.md")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("skill %q not found in project or global skills", skillName)
}

func (f *fileRead) appendSummary(ctx context.Context, path, content string) string {
	if f.tsQuery == nil {
		return content
	}
	lang := langFromExt(filepath.Ext(path))
	if lang == "" {
		return content
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return content
	}
	source := string(data)
	if len(strings.Split(source, "\n")) < 100 {
		return content
	}
	raw, err := f.tsQuery(ctx, "summarize", map[string]any{"lang": lang, "source": source})
	if err != nil || raw == nil {
		return content
	}
	var summary struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		StartRow int    `json:"startRow"`
		EndRow   int    `json:"endRow"`
		Children []any  `json:"children"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		return content
	}
	formatted := formatSummaryNode(raw)
	return "═══ STRUCTURED SUMMARY ═══\n" + formatted + "\n══════════════════════════\n\n" + content
}

func langFromExt(ext string) string {
	m := map[string]string{
		".go": "go", ".js": "javascript", ".jsx": "javascript", ".mjs": "javascript",
		".ts": "typescript", ".tsx": "typescript",
		".py": "python", ".pyw": "python",
		".rs": "rust", ".java": "java",
		".c": "c", ".h": "c",
		".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".hpp": "cpp",
		".cs": "c_sharp", ".rb": "ruby", ".php": "php",
		".swift": "swift", ".kt": "kotlin", ".kts": "kotlin",
		".scala": "scala",
		".json":  "json", ".yaml": "yaml", ".yml": "yaml", ".toml": "toml",
		".html": "html", ".htm": "html", ".css": "css",
		".sh": "bash", ".bash": "bash", ".zsh": "bash",
		".cjs": "javascript",
	}
	return m[strings.ToLower(ext)]
}

func formatSummaryNode(raw json.RawMessage) string {
	var node struct {
		Type     string          `json:"type"`
		Text     string          `json:"text"`
		StartRow int             `json:"startRow"`
		EndRow   int             `json:"endRow"`
		Folded   bool            `json:"folded"`
		Children json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(raw, &node); err != nil {
		return ""
	}
	text := strings.ReplaceAll(node.Text, "\n", "\\n")
	if len(text) > 60 {
		text = text[:57] + "..."
	}
	line := fmt.Sprintf("%s [%d-%d] %s", node.Type, node.StartRow+1, node.EndRow+1, text)

	if node.Folded {
		return line + "\n  .."
	}

	var children []json.RawMessage
	if node.Children != nil {
		json.Unmarshal(node.Children, &children)
	}
	var parts []string
	for _, child := range children {
		parts = append(parts, formatSummaryNode(child))
	}
	childStr := strings.Join(parts, "\n")
	if childStr == "" {
		return line
	}
	return line + "\n" + indentLines(childStr)
}

func indentLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

func readFileLines(path string, offset, limit int) (tool.ExecutionResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	// Two-pass: first count lines, then emit with context padding.
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	totalLines := len(allLines)
	start := offset - 1 // 0-indexed
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > totalLines {
		end = totalLines
	}

	// Context padding: +1 line before, +3 lines after
	padStart := start - 1
	if padStart < 0 {
		padStart = 0
	}
	padEnd := end + 3
	if padEnd > totalLines {
		padEnd = totalLines
	}

	var buf strings.Builder
	for i := padStart; i < padEnd; i++ {
		fmt.Fprintf(&buf, "%4d│%s│ %s\n", i+1, lineHash(allLines[i]), allLines[i])
	}

	// Footer hint
	if end < totalLines {
		remaining := totalLines - end
		fmt.Fprintf(&buf, "\n[%d more lines below — use offset=%d to continue]", remaining, end+1)
	}

	return tool.ExecutionResult{Content: strings.TrimRight(buf.String(), "\n")}, nil
}

type lineRange struct {
	start, end int // 1-indexed, inclusive
}

func parseRanges(s string) ([]lineRange, error) {
	parts := strings.Split(s, ",")
	var ranges []lineRange
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		dashIdx := strings.Index(p, "-")
		if dashIdx < 0 {
			// Single line number
			n, err := strconv.Atoi(p)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("invalid range %q", p)
			}
			ranges = append(ranges, lineRange{start: n, end: n})
		} else {
			startStr := p[:dashIdx]
			endStr := p[dashIdx+1:]
			start, err1 := strconv.Atoi(startStr)
			end, err2 := strconv.Atoi(endStr)
			if err1 != nil || err2 != nil || start < 1 || end < start {
				return nil, fmt.Errorf("invalid range %q", p)
			}
			ranges = append(ranges, lineRange{start: start, end: end})
		}
	}
	if len(ranges) == 0 {
		return nil, fmt.Errorf("empty ranges")
	}
	// Sort by start line
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	return ranges, nil
}

func readFileRanges(path string, ranges []lineRange) (tool.ExecutionResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	// Build a set of lines to include
	include := make(map[int]bool)
	maxLine := 0
	for _, r := range ranges {
		for i := r.start; i <= r.end; i++ {
			include[i] = true
		}
		if r.end > maxLine {
			maxLine = r.end
		}
	}

	var collected []string
	lineNum := 0
	prevIncluded := false

	for scanner.Scan() {
		lineNum++
		if lineNum > maxLine {
			break
		}
		isIncluded := include[lineNum]
		if isIncluded {
			if !prevIncluded && len(collected) > 0 {
				collected = append(collected, "    ⋮")
			}
			collected = append(collected, fmt.Sprintf("%4d│%s│ %s", lineNum, lineHash(scanner.Text()), scanner.Text()))
		}
		prevIncluded = isIncluded
	}

	if err := scanner.Err(); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	return tool.ExecutionResult{Content: strings.Join(collected, "\n")}, nil
}

// lineHash returns a 6-character hex FNV-1a hash of the trimmed line content,
// used as a fingerprint for anchor verification in file_edit.
func lineHash(line string) string {
	h := fnv.New32a()
	h.Write([]byte(strings.TrimSpace(line)))
	return fmt.Sprintf("%06x", h.Sum32())
}
