package builtin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"monika/internal/tool"
)

// Directories always skipped regardless of .gitignore.
// Hidden directories (starting with '.') are skipped separately in the walk.
var alwaysSkipDirs = map[string]bool{}

// Fallback directories skipped when no .gitignore exists in the project.
// Only non-hidden entries needed here; hidden ones are already filtered by name.
var fallbackSkipDirs = map[string]bool{
	"node_modules": true, "vendor": true,
	"__pycache__": true,
	"dist": true, "build": true, "target": true,
	"coverage": true,
	"bin": true, "obj": true,
	"tmp": true,
}

// Binary file extensions that should never be grepped.
var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
	".svg": true, ".webp": true, ".bmp": true, ".tiff": true,
	".mp3": true, ".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
	".webm": true, ".wav": true, ".ogg": true, ".flac": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
	".7z": true, ".rar": true, ".zst": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".wasm": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
	".eot": true, ".map": true,
	".class": true, ".pyc": true, ".pyo": true,
	".o": true, ".a": true, ".lib": true,
	".syso": true,
}

const maxWalkFiles = 5000
const maxResults = 200

type grepTool struct {
	projectDir string
	tsQuery    TSQueryFunc
}

func NewGrep(projectDir string, tsQuery TSQueryFunc) tool.Tool {
	return &grepTool{projectDir: projectDir, tsQuery: tsQuery}
}

func (g *grepTool) Name() string { return "grep" }
func (g *grepTool) Description() string {
	return "Search file contents using regular expressions. Returns file path, line number, and matching line content. Filter by file pattern using the include parameter. Capped at 200 results. Set ast_pattern to use tree-sitter AST queries (S-expressions) for structural code search instead of regex — e.g. '(function_declaration name: (identifier) @fn)'."
}

func (g *grepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regex pattern to search for in file contents",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to search in. Defaults to the project directory.",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "File pattern to include in the search (e.g. \"*.go\", \"*.{ts,tsx}\")",
			},
			"ast_pattern": map[string]any{
				"type":        "string",
				"description": "Tree-sitter query pattern (S-expression). When set, uses AST-aware matching instead of regex. E.g. '(function_declaration name: (identifier) @fn)'",
			},
		},
		"required": []string{"pattern"},
	}
}

type fileEntry struct {
	absPath string
	relPath string
}

func (g *grepTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Pattern    string `json:"pattern"`
		Path       string `json:"path"`
		Include    string `json:"include"`
		ASTPattern string `json:"ast_pattern"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	// AST grep mode
	if params.ASTPattern != "" {
		return g.executeASTGrep(ctx, params.ASTPattern, params.Path, params.Include)
	}

	re, err := regexp.Compile(params.Pattern)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("invalid regex: %s", err), IsError: true}, nil
	}

	absProject, err := filepath.Abs(tool.ProjectDirOrDefault(ctx, g.projectDir))
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if real, err := filepath.EvalSymlinks(absProject); err == nil {
		absProject = real
	}

	searchDir := absProject
	if params.Path != "" {
		if !filepath.IsAbs(params.Path) {
			return tool.ExecutionResult{Content: "path must be absolute", IsError: true}, nil
		}
		searchDir, err = filepath.Abs(params.Path)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if real, err := filepath.EvalSymlinks(searchDir); err == nil {
			searchDir = real
		}
	}

	rel, err := filepath.Rel(absProject, searchDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return tool.ExecutionResult{Content: "path is outside project directory", IsError: true}, nil
	}

	var includeRe *regexp.Regexp
	if params.Include != "" {
		pattern := globToRegex(params.Include)
		includeRe, err = regexp.Compile(pattern)
		if err != nil {
			return tool.ExecutionResult{Content: fmt.Sprintf("invalid include pattern: %s", err), IsError: true}, nil
		}
	}

	ignore := newIgnoreMatcher(absProject)

	// Phase 1: Collect candidate files using os.ReadDir (batch readdir, no per-entry lstat).
	var files []fileEntry
	var walkErrors []string
	fileCount := 0

	var walk func(dir string, relToSearch string)
	walk = func(dir string, relToSearch string) {
		select {
		case <-ctx.Done():
			return
		default:
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("%s: %s", dir, err))
			return
		}

		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return
			default:
			}

			name := entry.Name()
			if len(name) > 0 && name[0] == '.' {
				continue
			}

			absPath := filepath.Join(dir, name)
			relPath := filepath.Join(relToSearch, name)
			relToProject, _ := filepath.Rel(absProject, absPath)

			if entry.IsDir() {
				if ignore.hasRules {
					if ignore.Match(relToProject, true) {
						continue
					}
					ignore.loadNestedGitignore(absPath, toGitignorePath(relToProject))
				} else if fallbackSkipDirs[name] {
					continue
				}
				walk(absPath, relPath)
				continue
			}

			if fileCount >= maxWalkFiles {
				return
			}
			if ignore.hasRules && ignore.Match(relToProject, false) {
				continue
			}
			if binaryExts[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			if includeRe != nil && !includeRe.MatchString(name) {
				continue
			}
			fileCount++
			files = append(files, fileEntry{absPath: absPath, relPath: relPath})
		}
	}

	walk(searchDir, ".")

	if len(walkErrors) > 0 && len(files) == 0 {
		return tool.ExecutionResult{
			Content: "walk errors:\n" + strings.Join(walkErrors, "\n"),
			IsError: true,
		}, nil
	}

	if len(files) == 0 {
		return tool.ExecutionResult{Content: "No matches found"}, nil
	}

	// Phase 2: Concurrent grep with worker pool.
	type grepResult struct {
		relPath string
		lines   []string
		err     error
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > len(files) {
		numWorkers = len(files)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	fileCh := make(chan fileEntry, len(files))
	resultCh := make(chan grepResult, len(files))
	var totalMatchCount int64

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fe := range fileCh {
				if atomic.LoadInt64(&totalMatchCount) >= int64(maxResults) {
					return
				}
				lines, err := grepFile(fe.absPath, re, fe.relPath)
				resultCh <- grepResult{relPath: fe.relPath, lines: lines, err: err}
			}
		}()
	}

	// Feed files and close channel.
	go func() {
		for _, fe := range files {
			fileCh <- fe
		}
		close(fileCh)
	}()

	// Close resultCh when all workers done.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var results []string
	for gr := range resultCh {
		if gr.err != nil {
			walkErrors = append(walkErrors, fmt.Sprintf("%s: %s", gr.relPath, gr.err))
			continue
		}
		if len(gr.lines) == 0 {
			continue
		}
		for _, l := range gr.lines {
			if len(results) >= maxResults {
				break
			}
			results = append(results, l)
		}
		atomic.AddInt64(&totalMatchCount, int64(len(gr.lines)))
	}

	if len(results) == 0 {
		msg := "No matches found"
		if fileCount >= maxWalkFiles {
			msg += fmt.Sprintf(" (walked %d files before hitting limit)", fileCount)
		}
		if len(walkErrors) > 0 {
			msg += "\n\nwalk errors:\n" + strings.Join(walkErrors, "\n")
			return tool.ExecutionResult{Content: msg, IsError: true}, nil
		}
		return tool.ExecutionResult{Content: msg}, nil
	}

	// Sort by file path then line number for deterministic output.
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	// Group output by file for compact display.
	output := groupGrepResults(results)
	if fileCount >= maxWalkFiles {
		output += fmt.Sprintf("\n\nwalk truncated: reached %d file limit", maxWalkFiles)
	}
	if len(walkErrors) > 0 {
		output += "\n\nwalk errors:\n" + strings.Join(walkErrors, "\n")
	}
	return tool.ExecutionResult{Content: output}, nil
}

func grepFile(path string, re *regexp.Regexp, relPath string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if re.MatchString(scanner.Text()) {
			results = append(results, fmt.Sprintf("%s:%d: %s", relPath, lineNum, scanner.Text()))
		}
	}
	return results, scanner.Err()
}

func globToRegex(glob string) string {
	re := regexp.QuoteMeta(glob)
	re = strings.ReplaceAll(re, `\*`, ".*")
	re = strings.ReplaceAll(re, `\{`, "(?:")
	re = strings.ReplaceAll(re, `\}`, ")")
	re = strings.ReplaceAll(re, `\,`, "|")
	return "^" + re + "$"
}

// groupGrepResults groups grep output by file for compact display.
// Input format: "relPath:lineNum: content"
// Output format:
//
//	relPath
//	  lineNum│ content
//	  lineNum│ content
//	relPath2
//	  lineNum│ content
func groupGrepResults(results []string) string {
	type lineEntry struct {
		lineNum string
		content string
	}
	type fileGroup struct {
		path  string
		lines []lineEntry
	}

	var groups []fileGroup
	groupMap := make(map[string]*fileGroup)

	for _, r := range results {
		// Split "path:lineNum: content" — content may contain ":"
		firstColon := strings.Index(r, ":")
		if firstColon < 0 {
			continue
		}
		rest := r[firstColon+1:]
		secondColon := strings.Index(rest, ":")
		if secondColon < 0 {
			continue
		}
		path := r[:firstColon]
		lineNum := rest[:secondColon]
		content := rest[secondColon+1:] // includes leading space

		if g, ok := groupMap[path]; ok {
			g.lines = append(g.lines, lineEntry{lineNum, content})
		} else {
			groups = append(groups, fileGroup{path: path})
			groupMap[path] = &groups[len(groups)-1]
			groupMap[path].lines = append(groupMap[path].lines, lineEntry{lineNum, content})
		}
	}

	var sb strings.Builder
	for i, g := range groups {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(g.path)
		sb.WriteString("\n")
		for _, l := range g.lines {
			sb.WriteString("  ")
			// Right-align line number to 4 chars for consistency with file_read
			num := strings.TrimSpace(l.lineNum)
			if n, err := strconv.Atoi(num); err == nil {
				sb.WriteString(fmt.Sprintf("%4d│", n))
			} else {
				sb.WriteString(l.lineNum)
				sb.WriteString("│")
			}
			sb.WriteString(l.content)
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (g *grepTool) executeASTGrep(ctx context.Context, pattern, searchPath, include string) (tool.ExecutionResult, error) {
	if g.tsQuery == nil {
		return tool.ExecutionResult{Content: "AST grep requires tree-sitter, which is not available", IsError: true}, nil
	}

	absProject, err := filepath.Abs(tool.ProjectDirOrDefault(ctx, g.projectDir))
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if real, err := filepath.EvalSymlinks(absProject); err == nil {
		absProject = real
	}

	searchDir := absProject
	if searchPath != "" {
		if !filepath.IsAbs(searchPath) {
			return tool.ExecutionResult{Content: "path must be absolute", IsError: true}, nil
		}
		searchDir, err = filepath.Abs(searchPath)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if real, err := filepath.EvalSymlinks(searchDir); err == nil {
			searchDir = real
		}
	}

	rel, err := filepath.Rel(absProject, searchDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return tool.ExecutionResult{Content: "path is outside project directory", IsError: true}, nil
	}

	var includeRe *regexp.Regexp
	if include != "" {
		p := globToRegex(include)
		includeRe, err = regexp.Compile(p)
		if err != nil {
			return tool.ExecutionResult{Content: fmt.Sprintf("invalid include pattern: %s", err), IsError: true}, nil
		}
	}

	ignore := newIgnoreMatcher(absProject)

	// Collect files
	var files []fileEntry
	fileCount := 0
	var walk func(dir string, relToSearch string)
	walk = func(dir string, relToSearch string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := entry.Name()
			if len(name) > 0 && name[0] == '.' {
				continue
			}
			absPath := filepath.Join(dir, name)
			relPath := filepath.Join(relToSearch, name)
			relToProject, _ := filepath.Rel(absProject, absPath)
			if entry.IsDir() {
				if ignore.hasRules {
					if ignore.Match(relToProject, true) {
						continue
					}
					ignore.loadNestedGitignore(absPath, toGitignorePath(relToProject))
				} else if fallbackSkipDirs[name] {
					continue
				}
				walk(absPath, relPath)
				continue
			}
			if fileCount >= maxWalkFiles {
				return
			}
			if ignore.hasRules && ignore.Match(relToProject, false) {
				continue
			}
			if includeRe != nil && !includeRe.MatchString(name) {
				continue
			}
			lang := langFromExt(filepath.Ext(name))
			if lang == "" {
				continue // skip files we can't parse
			}
			fileCount++
			files = append(files, fileEntry{absPath: absPath, relPath: relPath})
		}
	}
	walk(searchDir, ".")

	if len(files) == 0 {
		return tool.ExecutionResult{Content: "No matching files found for AST grep"}, nil
	}

	// Query each file via tree-sitter
	var results []string
	for _, fe := range files {
		select {
		case <-ctx.Done():
			return tool.ExecutionResult{Content: "cancelled", IsError: true}, nil
		default:
		}
		if len(results) >= maxResults {
			break
		}
		data, err := os.ReadFile(fe.absPath)
		if err != nil {
			continue
		}
		lang := langFromExt(filepath.Ext(fe.absPath))
		raw, err := g.tsQuery(ctx, "query", map[string]any{
			"lang":    lang,
			"source":  string(data),
			"pattern": pattern,
		})
		if err != nil || raw == nil {
			continue
		}
		var matches []struct {
			Captures []struct {
				Name string `json:"name"`
				Node struct {
					Type         string `json:"type"`
					Text         string `json:"text"`
					StartRow     int    `json:"startRow"`
					StartColumn  int    `json:"startColumn"`
					EndRow       int    `json:"endRow"`
					EndColumn    int    `json:"endColumn"`
				} `json:"node"`
			} `json:"captures"`
		}
		if err := json.Unmarshal(raw, &matches); err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, m := range matches {
			if len(results) >= maxResults {
				break
			}
			for _, c := range m.Captures {
				startLine := c.Node.StartRow + 1
				endLine := c.Node.EndRow + 1
				if endLine > len(lines) {
					endLine = len(lines)
				}
				var matchText string
				if startLine == endLine {
					matchText = fmt.Sprintf("%s:%d: [%s:%s] %s", fe.relPath, startLine, c.Node.Type, c.Name, truncateLine(c.Node.Text, 120))
				} else {
					matchText = fmt.Sprintf("%s:%d-%d: [%s:%s] %s", fe.relPath, startLine, endLine, c.Node.Type, c.Name, truncateLine(c.Node.Text, 120))
				}
				results = append(results, matchText)
			}
		}
	}

	if len(results) == 0 {
		return tool.ExecutionResult{Content: "No AST matches found"}, nil
	}
	sort.Strings(results)
	return tool.ExecutionResult{Content: strings.Join(results, "\n")}, nil
}


func truncateLine(s string, max int) string {
	oneLine := strings.ReplaceAll(s, "\n", "\\n")
	if len(oneLine) <= max {
		return oneLine
	}
	return oneLine[:max-3] + "..."
}
