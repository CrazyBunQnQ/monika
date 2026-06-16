package api

import (
	"encoding/json"
	"fmt"

	"monika/internal/memory"
)

type KBFileInfo struct {
	Path       string   `json:"path"`
	Scope      string   `json:"scope"`
	Category   string   `json:"category"`
	Title      string   `json:"title"`
	Tags       []string `json:"tags"`
	Confidence string   `json:"confidence"`
	Status     string   `json:"status"`
	CharCount  int      `json:"char_count"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type KBStats struct {
	Total      int    `json:"total"`
	Active     int    `json:"active"`
	Archived   int    `json:"archived"`
	LastUpdate string `json:"last_update"`
}

func (a *App) KBListFiles(scope string) ([]KBFileInfo, error) {
	if a.kbStore == nil {
		return nil, fmt.Errorf("kb not initialized")
	}
	files, err := a.kbStore.ListFiles(scope, "")
	if err != nil {
		return nil, err
	}
	var result []KBFileInfo
	for _, f := range files {
		result = append(result, kbFileToInfo(f))
	}
	return result, nil
}

func (a *App) KBReadFile(scope, path string) (string, error) {
	if a.kbStore == nil {
		return "", fmt.Errorf("kb not initialized")
	}
	return a.kbStore.ReadFile(scope, path)
}

func (a *App) KBWriteFile(args json.RawMessage) error {
	if a.kbStore == nil {
		return fmt.Errorf("kb not initialized")
	}
	var p struct {
		Scope      string   `json:"scope"`
		Category   string   `json:"category"`
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	return a.kbStore.WriteFile(p.Scope, p.Category, p.Title, p.Content, p.Tags, p.Confidence)
}

func (a *App) KBDeleteFile(scope, path string) error {
	if a.kbStore == nil {
		return fmt.Errorf("kb not initialized")
	}
	return a.kbStore.SoftDelete(scope, path)
}

func (a *App) KBSearch(query, scope string) ([]KBFileInfo, error) {
	if a.kbStore == nil {
		return nil, fmt.Errorf("kb not initialized")
	}
	files, err := a.kbStore.Search(query, scope, 10)
	if err != nil {
		return nil, err
	}
	var result []KBFileInfo
	for _, f := range files {
		result = append(result, kbFileToInfo(f))
	}
	return result, nil
}

func (a *App) KBStatistics(scope string) (*KBStats, error) {
	if a.kbStore == nil {
		return nil, fmt.Errorf("kb not initialized")
	}
	total, active, archived, lastUpdate, err := a.kbStore.GetStatistics(scope)
	if err != nil {
		return nil, err
	}
	return &KBStats{
		Total:      total,
		Active:     active,
		Archived:   archived,
		LastUpdate: lastUpdate,
	}, nil
}

func (a *App) KBUploadDocument(args json.RawMessage) error {
	if a.kbStore == nil {
		return fmt.Errorf("kb not initialized")
	}
	var p struct {
		Scope    string `json:"scope"`
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	return a.kbStore.WriteFile(p.Scope, memory.CategoryRawDoc, p.Filename, p.Content, nil, "medium")
}

func kbFileToInfo(f memory.KBFile) KBFileInfo {
	return KBFileInfo{
		Path:       f.Path,
		Scope:      f.Scope,
		Category:   f.Category,
		Title:      f.Title,
		Tags:       f.Tags,
		Confidence: f.Confidence,
		Status:     f.Status,
		CharCount:  f.CharCount,
		CreatedAt:  f.CreatedAt.Format("2006-01-02"),
		UpdatedAt:  f.UpdatedAt.Format("2006-01-02"),
	}
}
