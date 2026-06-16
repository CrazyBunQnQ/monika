package dbdiscovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Discoverer interface {
	Name() string
	Scan(projectDir string) ([]DiscoveredDB, error)
}

type DiscoveredDB struct {
	Name        string `json:"name"`
	Driver      string `json:"driver"`
	DSN         string `json:"dsn"`
	Source      string `json:"source"`
	RuntimeHint string `json:"runtime_hint,omitempty"`
}

type CacheFile struct {
	ScannedAt   time.Time      `json:"scanned_at"`
	Runtime     string         `json:"runtime"`
	Connections []DiscoveredDB `json:"connections"`
}

var (
	discMu      sync.RWMutex
	discoverers []Discoverer
)

func RegisterDiscoverer(d Discoverer) {
	discMu.Lock()
	defer discMu.Unlock()
	discoverers = append(discoverers, d)
}

func Scan(projectDir string) (*CacheFile, error) {
	rt := DetectRuntime(projectDir)

	var all []DiscoveredDB
	discMu.RLock()
	defer discMu.RUnlock()
	for _, d := range discoverers {
		results, err := d.Scan(projectDir)
		if err != nil {
			continue
		}
		all = append(all, results...)
	}

	all = deduplicate(all)
	for i := range all {
		if all[i].RuntimeHint == "" {
			all[i].RuntimeHint = rt
		}
	}

	cache := &CacheFile{
		ScannedAt:   time.Now(),
		Runtime:     rt,
		Connections: all,
	}

	cachePath := filepath.Join(projectDir, ".monika", "databases.json")
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[monika] dbdiscovery: failed to create cache dir: %v\n", err)
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[monika] dbdiscovery: failed to marshal cache: %v\n", err)
	} else if err := os.WriteFile(cachePath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "[monika] dbdiscovery: failed to write cache: %v\n", err)
	}

	return cache, nil
}

func LoadCache(projectDir string) (*CacheFile, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, ".monika", "databases.json"))
	if err != nil {
		return nil, err
	}
	var cache CacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func deduplicate(conns []DiscoveredDB) []DiscoveredDB {
	seen := map[string]bool{}
	var result []DiscoveredDB
	for _, c := range conns {
		key := c.Driver + ":" + c.DSN
		if !seen[key] {
			seen[key] = true
			result = append(result, c)
		}
	}
	return result
}
