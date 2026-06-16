package memory

import "strings"

func (s *KBStore) BuildMemoryBlock() string {
	var sb strings.Builder

	globalProfile, _ := s.ReadFile(ScopeGlobal, "wiki/profile.md")
	globalKnowledge, _ := s.ReadFile(ScopeGlobal, "wiki/knowledge.md")
	projectKnowledge, _ := s.ReadFile(ScopeProject, "wiki/knowledge.md")

	hasGlobal := globalProfile != "" || globalKnowledge != ""
	hasProject := projectKnowledge != ""

	if !hasGlobal && !hasProject {
		return ""
	}

	if hasGlobal {
		sb.WriteString("\n\n<global_memory>\n")
		if globalProfile != "" {
			sb.WriteString("<user_profile>\n")
			sb.WriteString(globalProfile)
			sb.WriteString("\n</user_profile>\n")
		}
		if globalKnowledge != "" {
			sb.WriteString("\n<core_knowledge>\n")
			sb.WriteString(globalKnowledge)
			sb.WriteString("\n</core_knowledge>\n")
		}
		sb.WriteString("</global_memory>")
	}

	if hasProject {
		sb.WriteString("\n\n<project_memory>\n")
		sb.WriteString("<project_knowledge>\n")
		sb.WriteString(projectKnowledge)
		sb.WriteString("\n</project_knowledge>\n")
		sb.WriteString("</project_memory>")
	}

	return sb.String()
}
