package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"monika/internal/lsp"
)

func migrateLSPJSON(projectDir string) error {
	lspPath := filepath.Join(projectDir, ".monika", "lsp.json")
	if _, err := os.Stat(lspPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	data, err := os.ReadFile(lspPath)
	if err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	var userServers map[string]lsp.ServerConfig
	if err := json.Unmarshal(data, &userServers); err != nil {
		// Corrupt lsp.json — rename so it doesn't fail on every Load()
		os.Rename(lspPath, lspPath+".migrated")
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	configPath := filepath.Join(projectDir, ".monika", "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.MkdirAll(filepath.Dir(configPath), 0o755)
			configData = []byte("{}")
		} else {
			return fmt.Errorf("migrate lsp.json: %w", err)
		}
	}

	var cfgMap map[string]json.RawMessage
	if err := json.Unmarshal(configData, &cfgMap); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	// Merge with any existing lsp config in config.json
	existingServers := make(map[string]lsp.ServerConfig)
	if raw, ok := cfgMap["lsp"]; ok {
		var existingLSP struct {
			Servers map[string]lsp.ServerConfig `json:"servers"`
		}
		if err := json.Unmarshal(raw, &existingLSP); err == nil && existingLSP.Servers != nil {
			existingServers = existingLSP.Servers
		}
	}
	for name, srv := range userServers {
		existingServers[name] = srv
	}

	serversData, err := json.Marshal(existingServers)
	if err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}
	cfgMap["lsp"] = json.RawMessage(fmt.Sprintf(`{"servers":%s}`, serversData))

	out, err := json.MarshalIndent(cfgMap, "", "  ")

	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0o600); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	if err := os.Rename(lspPath, lspPath+".migrated"); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	return nil
}
