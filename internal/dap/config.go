package dap

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// resolveCommand checks if command exists in PATH or relative to cwd.
func resolveCommand(command string, cwd string) string {
	if filepath.IsAbs(command) {
		if _, err := os.Stat(command); err == nil {
			return command
		}
		return ""
	}
	// Check relative to cwd
	local := filepath.Join(cwd, command)
	if _, err := os.Stat(local); err == nil {
		return local
	}
	// Check PATH
	p, err := exec.LookPath(command)
	if err != nil {
		return ""
	}
	return p
}

// hasRootMarkers checks if any of the given marker files exist in cwd.
func hasRootMarkers(cwd string, markers []string) bool {
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(cwd, m)); err == nil {
			return true
		}
	}
	return false
}

// resolveAdapter resolves an adapter name to a concrete DapResolvedAdapter.
func resolveAdapter(adapterName string, cwd string) *DapResolvedAdapter {
	config, ok := DefaultAdapters[adapterName]
	if !ok {
		return nil
	}
	resolved := resolveCommand(config.Command, cwd)
	if resolved == "" {
		return nil
	}
	connectMode := config.ConnectMode
	if connectMode == "" {
		connectMode = "stdio"
	}
	return &DapResolvedAdapter{
		Name:            adapterName,
		Command:         config.Command,
		Args:            append([]string{}, config.Args...),
		ResolvedCommand: resolved,
		Languages:       append([]string{}, config.Languages...),
		FileTypes:       append([]string{}, config.FileTypes...),
		RootMarkers:     append([]string{}, config.RootMarkers...),
		LaunchDefaults:  copyMap(config.LaunchDefaults),
		AttachDefaults:  copyMap(config.AttachDefaults),
		ConnectMode:     connectMode,
		Env:             copyMapStr(config.Env),
	}
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyMapStr(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// GetAvailableAdapters returns all adapters that are available on this system.
func GetAvailableAdapters(cwd string) []*DapResolvedAdapter {
	var adapters []*DapResolvedAdapter
	for name := range DefaultAdapters {
		if a := resolveAdapter(name, cwd); a != nil {
			adapters = append(adapters, a)
		}
	}
	sort.Slice(adapters, func(i, j int) bool {
		return adapters[i].Name < adapters[j].Name
	})
	return adapters
}

var extensionlessDebuggerOrder = []string{"gdb", "lldb-dap"}

// selectLaunchAdapter selects the best adapter for launching a program.
func selectLaunchAdapter(program string, cwd string, adapterName string) *DapResolvedAdapter {
	if adapterName != "" {
		return resolveAdapter(adapterName, cwd)
	}
	ext := strings.ToLower(filepath.Ext(program))
	available := GetAvailableAdapters(cwd)

	if ext == "" {
		// Extensionless binary: prefer native debuggers (gdb, lldb-dap)
		for _, pref := range extensionlessDebuggerOrder {
			for _, a := range available {
				if a.Name == pref {
					return a
				}
			}
		}
		// Fall back to root marker matching
		for _, a := range available {
			if hasRootMarkers(cwd, a.RootMarkers) {
				return a
			}
		}
		return nil
	}

	// Match by file extension
	for _, a := range available {
		for _, ft := range a.FileTypes {
			if ft == ext {
				return a
			}
		}
	}
	// Fall back to any available
	if len(available) > 0 {
		return available[0]
	}
	return nil
}

// selectAttachAdapter selects the best adapter for attaching to a process.
func selectAttachAdapter(cwd string, adapterName string) *DapResolvedAdapter {
	if adapterName != "" {
		return resolveAdapter(adapterName, cwd)
	}
	available := GetAvailableAdapters(cwd)
	// Prefer gdb/lldb-dap for attach
	for _, pref := range extensionlessDebuggerOrder {
		for _, a := range available {
			if a.Name == pref {
				return a
			}
		}
	}
	if len(available) > 0 {
		return available[0]
	}
	return nil
}
