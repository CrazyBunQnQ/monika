package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"monika/internal/tool"
)

type fileList struct {
	projectDir string
}

func NewFileList(projectDir string) tool.Tool {
	return &fileList{projectDir: projectDir}
}

func (f *fileList) Name() string        { return "file_list" }
func (f *fileList) Description() string {
	return "List files and directories in a given path. Use this to discover project structure before targeting specific files with grep or file_read. Set tree=true for a recursive tree view with depth control (default depth 3)."
}

func (f *fileList) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"dirPath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the directory to list",
			},
			"tree": map[string]any{
				"type":        "boolean",
				"description": "If true, output a recursive tree view instead of a flat list",
			},
			"depth": map[string]any{
				"type":        "integer",
				"description": "Maximum depth for tree view (default 3). Only used when tree=true.",
			},
		},
		"required": []string{"dirPath"},
	}
}

func (f *fileList) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		DirPath string `json:"dirPath"`
		Tree    bool   `json:"tree"`
		Depth   int    `json:"depth"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	absPath, err := f.resolvePath(ctx, params.DirPath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Tree {
		maxDepth := params.Depth
		if maxDepth < 1 {
			maxDepth = 3
		}
		return f.readTree(absPath, maxDepth)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	var lines []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		lines = append(lines, name)
	}
	return tool.ExecutionResult{Content: strings.Join(lines, "\n")}, nil
}

func (f *fileList) resolvePath(ctx context.Context, p string) (string, error) {
	return resolveToolPath(p, tool.ProjectDirOrDefault(ctx, f.projectDir))
}

func (f *fileList) readTree(root string, maxDepth int) (tool.ExecutionResult, error) {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s/\n", filepath.Base(root))
	f.buildTree(&buf, root, "", maxDepth, 0)
	return tool.ExecutionResult{Content: strings.TrimRight(buf.String(), "\n")}, nil
}

func (f *fileList) buildTree(buf *strings.Builder, dir, prefix string, maxDepth, depth int) {
	if depth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Skip common noise directories
	skipDirs := map[string]bool{".git": true, "node_modules": true, ".svn": true, "__pycache__": true, ".hg": true}
	var filtered []os.DirEntry
	for _, e := range entries {
		if e.IsDir() && skipDirs[e.Name()] {
			continue
		}
		filtered = append(filtered, e)
	}

	for i, e := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		fmt.Fprintf(buf, "%s%s%s\n", prefix, connector, name)

		if e.IsDir() {
			extension := "│   "
			if isLast {
				extension = "    "
			}
			f.buildTree(buf, filepath.Join(dir, e.Name()), prefix+extension, maxDepth, depth+1)
		}
	}
}
