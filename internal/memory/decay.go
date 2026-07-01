package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DecayPolicy controls when memories are archived or deleted based on age and confidence.
type DecayPolicy struct {
	ArchiveAfterDays     int                // default 90: archive if not updated in 90 days
	DeleteAfterDays      int                // default 180: soft-delete if not updated in 180 days
	ConfidenceMultiplier map[string]float64 // low=2.0 (faster decay), medium=1.0, high=0.5 (slower)
}

func DefaultDecayPolicy() DecayPolicy {
	return DecayPolicy{
		ArchiveAfterDays: 90,
		DeleteAfterDays:  180,
		ConfidenceMultiplier: map[string]float64{
			"high":   0.5,
			"medium": 1.0,
			"low":    2.0,
		},
	}
}

// RunDecay archives and deletes stale memories based on the policy.
// Returns counts of archived and deleted items.
func (s *KBStore) RunDecay(scope string, policy DecayPolicy) (archived, deleted int, err error) {
	files, err := s.ListFiles(scope, "")
	if err != nil {
		return 0, 0, err
	}
	now := time.Now()

	for _, f := range files {
		if f.Status != "active" {
			continue
		}
		multiplier := policy.ConfidenceMultiplier[f.Confidence]
		if multiplier == 0 {
			multiplier = 1.0
		}

		ageDays := now.Sub(f.UpdatedAt).Hours() / 24
		effectiveAge := ageDays * multiplier

		if effectiveAge >= float64(policy.DeleteAfterDays) {
			if f.AccessCount >= 10 {
				if err := s.SetFileStatus(scope, f.Path, "archived"); err != nil {
					continue
				}
				s.LogEntry(scope, "高频保护", fmt.Sprintf("%s 已归档但保留 (age=%.0fd, conf=%s, access=%d)", f.Path, ageDays, f.Confidence, f.AccessCount))
				archived++
			} else {
				if err := s.SoftDelete(scope, f.Path); err != nil {
					continue
				}
				s.LogEntry(scope, "自动遗忘", fmt.Sprintf("%s 已过期删除 (age=%.0fd, conf=%s)", f.Path, ageDays, f.Confidence))
				deleted++
			}
		} else if effectiveAge >= float64(policy.ArchiveAfterDays) {
			if err := s.SetFileStatus(scope, f.Path, "archived"); err != nil {
				continue
			}
			s.LogEntry(scope, "自动归档", fmt.Sprintf("%s 已归档 (age=%.0fd, conf=%s)", f.Path, ageDays, f.Confidence))
			archived++
		}
	}
	return archived, deleted, nil
}

// LastDecayTime reads the last decay execution timestamp from .index/last_decay.txt.
// Returns zero time if file doesn't exist.
func (s *KBStore) LastDecayTime(scope string) time.Time {
	root := s.rootFor(scope)
	data, err := os.ReadFile(filepath.Join(root, ".index", "last_decay.txt"))
	if err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	return t
}

// SetLastDecayTime writes the current time to .index/last_decay.txt.
func (s *KBStore) SetLastDecayTime(scope string) error {
	root := s.rootFor(scope)
	dir := filepath.Join(root, ".index")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "last_decay.txt"), []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
}
