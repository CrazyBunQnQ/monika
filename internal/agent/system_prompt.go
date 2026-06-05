package agent

import (
	"fmt"
	"strings"

	"monika/internal/prompt"
	"monika/pkg/engine"
)

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

var (
	PromptIdentity         = prompt.Get("").Identity
	PromptToolUsage        = prompt.Get("").ToolUsage
	PromptPlanning         = prompt.Get("").Planning
	PromptCodeQuality      = prompt.Get("").CodeQuality
	PromptResponseStyle    = prompt.Get("").ResponseStyle
	PromptSafetyBoundaries = prompt.Get("").SafetyBoundaries
	PromptRemember         = prompt.Get("").Remember
)

func PromptForModel(model string) prompt.PromptSet {
	return prompt.Get(model)
}

func BuildSkillsPrompt(skills []engine.SkillMeta) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## Skills — Specialized Workflows\n\n")
	b.WriteString("Skills provide detailed, step-by-step workflows for specific tasks. When you load a skill, you are committing to executing its ENTIRE workflow.\n\n")
	b.WriteString("### Skill Adherence Rules (MANDATORY)\n\n")
	b.WriteString("1. **Complete execution required**: When you load a skill via the skill tool, you MUST follow ALL of its steps in order. Partial execution is a failure.\n")
	b.WriteString("2. **No silent skipping**: Never skip a step without explicitly acknowledging it and explaining why. If a step does not apply, state that and why — do not just move on.\n")
	b.WriteString("3. **Follow the workflow to the end**: Skills often have a defined terminal state. You must reach that terminal state.\n")
	b.WriteString("4. **Step-by-step discipline**: Execute one step at a time. Complete each step fully before moving to the next. Do not batch or merge steps.\n")
	b.WriteString("5. **Self-check before responding**: After loading a skill, mentally track which step you are on. Before each response, verify: \"Am I still following the skill workflow? Have I completed all prior steps?\"\n\n")
	b.WriteString("Use the skill tool to load a skill when a task matches its description.\n\n")
	b.WriteString("### Skill Management\n\n")
	b.WriteString("When a user asks to install, add, or download a skill from a URL, use the **install_skill** tool. When a user asks to remove or uninstall a skill, use the **uninstall_skill** tool. After installing or uninstalling, report what was done to the user.\n\n")
	b.WriteString("<available_skills>\n")
	for _, s := range skills {
		if s.Enabled != nil && !*s.Enabled {
			continue
		}
		fmt.Fprintf(&b, "  <skill>\n    <name>%s</name>\n    <description>%s</description>\n  </skill>\n", xmlEscape(s.Name), xmlEscape(s.Description))
	}
	b.WriteString("</available_skills>")
	return b.String()
}

func BuildMCPPrompt(registry *engine.MCPRegistry) string {
	tools := registry.GetTools()
	if len(tools) == 0 {
		return ""
	}

	servers := registry.GetServers()
	byServer := make(map[string][]engine.MCPTool)
	for _, t := range tools {
		byServer[t.ServerID] = append(byServer[t.ServerID], t)
	}

	var b strings.Builder
	b.WriteString("\n\n## Available MCP Servers\n\n")
	b.WriteString("These MCP tools are available **right now** — every tool listed below is already connected and callable. ")
	b.WriteString("Each tool name is prefixed with its server ID (e.g., server 'foo' with tool 'bar' becomes 'foo_bar'). ")
	b.WriteString("**Before using bash for any external operation** (HTTP requests, web scraping, search, documentation lookup), ")
	b.WriteString("scan this list for a matching tool.\n\n")
	b.WriteString("### How to use MCP tools\n\n")
	b.WriteString("- Scan the server list below. Each entry shows what capabilities the server provides.\n")
	b.WriteString("- When a task involves external operations (web search, HTTP requests, documentation lookup, database access, browser automation), check the list for a matching tool.\n")
	b.WriteString("- Prefer MCP tools over bash (curl/wget) for these operations.\n")
	b.WriteString("- If nothing matches, or you're unsure what's available, call `list_mcp_servers` to get the full tool listing with descriptions.\n\n")

	for _, srv := range servers {
		srvTools := byServer[srv.ID]
		if len(srvTools) == 0 {
			continue
		}
		if srv.Name != "" {
			fmt.Fprintf(&b, "### %s\n", srv.Name)
		} else {
			fmt.Fprintf(&b, "### %s\n", srv.ID)
		}
		if srv.Instructions != "" {
			b.WriteString(srv.Instructions)
			b.WriteString("\n\n")
		}
		for _, t := range srvTools {
			annTags := buildAnnotationTags(t.Annotations)
			desc := t.Description
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Fprintf(&b, "- **%s**%s: %s\n", t.Name, annTags, desc)
		}
		b.WriteString("\n")
	}

	b.WriteString("### MCP Server Management\n\n")
	b.WriteString("Use **install_mcp_server** / **uninstall_mcp_server** / **list_mcp_servers** for MCP server management.\n")
	return b.String()
}

func buildAnnotationTags(a engine.MCPAnnotations) string {
	var tags []string
	if a.ReadOnly {
		tags = append(tags, "read-only")
	}
	if a.Destructive {
		tags = append(tags, "destructive")
	}
	if a.Idempotent {
		tags = append(tags, "idempotent")
	}
	if len(tags) == 0 {
		return ""
	}
	return " [" + strings.Join(tags, ", ") + "]"
}
