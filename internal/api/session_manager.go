package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"monika/internal/tool"
	"monika/pkg/engine"
)

// Session status constants.
const (
	StatusIdle       = "idle"
	StatusGenerating = "generating"
	StatusPending    = "pending"
	StatusArchived   = "archived"
)

type Session struct {
	ID              string               `json:"id"`
	Title           string               `json:"title"`
	CustomTitle     bool                 `json:"custom_title,omitempty"`
	ProjectDir      string               `json:"project_dir"`
	Messages        []engine.ChatMessage `json:"messages"`
	Model           string               `json:"model"`
	Provider        string               `json:"provider"`
	Status          string               `json:"status"`
	Pinned          bool                 `json:"pinned"`
	TokenCount      int64                `json:"token_count,omitempty"`
	TokenMax        int64                `json:"token_max,omitempty"`
	CompactionCount int                  `json:"compaction_count,omitempty"`
	CompactionFrom  int                  `json:"compaction_from,omitempty"`
	ParentID        string               `json:"parent_id,omitempty"`
	Tasks           []tool.Task          `json:"tasks,omitempty"`
	LastViewedAt    *time.Time           `json:"last_viewed_at,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	WorktreePath    string               `json:"worktree_path,omitempty"`
	Queue           []QueuedMessage      `json:"queue,omitempty"`
	QueuePaused     bool                 `json:"queue_paused,omitempty"`
}

type SessionManager struct {
	mu          sync.Mutex
	home        string
	projectDir  string
	sessionsDir string
}

func projectSlug(projectDir string) string {
	s := strings.ToLower(projectDir)
	s = strings.ReplaceAll(s, `\`, "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.Trim(s, "-")
	return s
}

func NewSessionManager(home, projectDir string) *SessionManager {
	sessionsDir := filepath.Join(home, ".monika", "projects", projectSlug(projectDir), "sessions")
	return &SessionManager{
		home:        home,
		projectDir:  projectDir,
		sessionsDir: sessionsDir,
	}
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (sm *SessionManager) New(model, provider string) (*Session, error) {
	now := time.Now()
	return &Session{
		ID:         generateID(),
		ProjectDir: sm.projectDir,
		Model:      model,
		Provider:   provider,
		Status:     StatusIdle,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (sm *SessionManager) Load(id string) (*Session, error) {
	p := filepath.Join(sm.sessionsDir, id+".json")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Status == "" {
		s.Status = StatusIdle
	}
	// Repair title truncated mid-rune from old byte-based slicing
	if s.Title != "" && len(s.Messages) > 0 && !s.CustomTitle {
		sm.SetTitle(&s)
	}
	return &s, nil
}

func (sm *SessionManager) Save(s *Session) error {
	s.UpdatedAt = time.Now()
	p := filepath.Join(sm.sessionsDir, s.ID+".json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (sm *SessionManager) SetStatus(s *Session, status string) {
	s.Status = status
}

func (sm *SessionManager) AppendShellMessages(sessionID string, msgs []engine.ChatMessage) error {
	sm.Lock()
	defer sm.Unlock()

	s, err := sm.Load(sessionID)
	if err != nil {
		return err
	}
	s.Messages = append(s.Messages, msgs...)
	return sm.Save(s)
}

func (sm *SessionManager) Delete(id string) error {
	p := filepath.Join(sm.sessionsDir, id+".json")
	return os.Remove(p)
}

func (sm *SessionManager) List() ([]SessionInfo, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionInfo{}, nil
		}
		return nil, err
	}
	var infos []SessionInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		s, err := sm.Load(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		// Migrate legacy statuses to archived
		if s.Status == "completed" || s.Status == "success" || s.Status == "failure" || s.Status == "stopped" {
			s.Status = StatusArchived
			sm.Save(s)
		}
		// Lazy archival: pending + viewed > 1h ago → archiveded
		if s.Status == StatusPending && s.LastViewedAt != nil {
			if time.Since(*s.LastViewedAt) > time.Hour {
				s.Status = StatusArchived
				sm.Save(s)
			}
		}
		info := SessionInfo{
			ID:           s.ID,
			Title:        s.Title,
			Status:       s.Status,
			Pinned:       s.Pinned,
			UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
			TokenCount:   s.TokenCount,
			TokenMax:     s.TokenMax,
			WorktreePath: s.WorktreePath,
		}
		// Resolve branch name from WorktreePath if set
		if s.WorktreePath != "" {
			base := filepath.Base(s.WorktreePath)
			info.WorktreeBranch = base
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (sm *SessionManager) SetTitle(s *Session) {
	if s.CustomTitle {
		return
	}
	for _, m := range s.Messages {
		if m.Role == "user" && m.Content != "" {
			content := m.Content
			// Strip injected prefix blocks (<env>, <recalled-memory>, etc.)
			for strings.HasPrefix(content, "<") {
				idx := strings.Index(content, ">\n\n")
				if idx < 0 {
					break
				}
				content = content[idx+len(">\n\n"):]
			}
			runes := []rune(content)
			if len(runes) > 40 {
				s.Title = string(runes[:40])
			} else {
				s.Title = content
			}
			return
		}
	}
}

func (sm *SessionManager) Lock() {
	sm.mu.Lock()
}

func (sm *SessionManager) Unlock() {
	sm.mu.Unlock()
}

func (sm *SessionManager) EnqueueQueueItem(s *Session, item QueuedMessage) {
	s.Queue = append(s.Queue, item)
}

func (sm *SessionManager) FindQueueItem(s *Session, itemID string) int {
	for i, item := range s.Queue {
		if item.ID == itemID {
			return i
		}
	}
	return -1
}

func (sm *SessionManager) UpdateQueueItem(s *Session, itemID string, fn func(*QueuedMessage)) {
	for i := range s.Queue {
		if s.Queue[i].ID == itemID {
			fn(&s.Queue[i])
			return
		}
	}
}

func (sm *SessionManager) RemoveQueueItem(s *Session, itemID string) {
	idx := sm.FindQueueItem(s, itemID)
	if idx >= 0 {
		s.Queue = append(s.Queue[:idx], s.Queue[idx+1:]...)
	}
}

func (sm *SessionManager) ReorderQueue(s *Session, itemIDs []string) {
	itemMap := make(map[string]QueuedMessage)
	for _, item := range s.Queue {
		itemMap[item.ID] = item
	}
	seen := make(map[string]bool)
	var reordered []QueuedMessage
	for _, id := range itemIDs {
		if item, ok := itemMap[id]; ok {
			reordered = append(reordered, item)
			seen[id] = true
		}
	}
	for _, item := range s.Queue {
		if !seen[item.ID] {
			reordered = append(reordered, item)
		}
	}
	s.Queue = reordered
}

func (sm *SessionManager) NextQueuedItem(s *Session) *QueuedMessage {
	for i := range s.Queue {
		if s.Queue[i].Status == "queued" {
			return &s.Queue[i]
		}
	}
	return nil
}
