//go:build !windows

package api

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

func openInExplorer(absPath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}
	return cmd.Start()
}

func listDrives() []FileNode {
	if runtime.GOOS != "darwin" {
		return nil
	}

	var nodes []FileNode
	nodes = append(nodes, FileNode{
		Name:  "Macintosh HD",
		Path:  "/",
		IsDir: true,
	})

	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return nodes
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		volPath := filepath.Join("/Volumes", name)
		if info, err := os.Lstat(volPath); err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		nodes = append(nodes, FileNode{
			Name:  name,
			Path:  filepath.ToSlash(volPath),
			IsDir: true,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Path == "/" {
			return true
		}
		if nodes[j].Path == "/" {
			return false
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes
}
