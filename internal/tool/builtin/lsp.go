package builtin

import (
	"monika/internal/lsp"
	"monika/internal/tool"
)

func NewLSPTool(projectDir string, lspServers map[string]lsp.ServerConfig, formatters map[string]lsp.FormatterConfig) (tool.Tool, error) {
	return lsp.NewLSPTool(projectDir, lspServers, formatters)
}
