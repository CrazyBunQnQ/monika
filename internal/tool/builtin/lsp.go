package builtin

import (
	"monika/internal/lsp"
	"monika/internal/tool"
)

func NewLSPTool(projectDir string) (tool.Tool, error) {
	return lsp.NewLSPTool(projectDir)
}
