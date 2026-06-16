package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *KBStore) LogEntry(scope, action, detail string) error {
	now := time.Now().UTC()
	dateHeader := now.Format("2006-01-02")
	timeHeader := now.Format("15:04")

	entry := fmt.Sprintf("\n### %s — %s\n- %s\n", timeHeader, action, detail)

	content, err := s.ReadFile(scope, "wiki/log.md")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if content == "" {
		content = "# Operation Log\n\n> 类型：episodic\n"
	}

	dateSection := "## " + dateHeader
	if idx := strings.Index(content, dateSection); idx >= 0 {
		rest := content[idx+len(dateSection):]
		nextSection := strings.Index(rest, "\n## ")
		if nextSection >= 0 {
			content = content[:idx+len(dateSection)] + entry + rest[:nextSection] + rest[nextSection:]
		} else {
			content = content[:idx+len(dateSection)] + entry + rest
		}
	} else {
		content += "\n" + dateSection + entry
	}

	root := s.rootFor(scope)
	return os.WriteFile(filepath.Join(root, "wiki", "log.md"), []byte(content), 0644)
}
