package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const configDir = ".monika"

// ResolveServers returns the merged list of server configs:
// defaults filtered by root markers present in workdir,
// then overridden by .monika/lsp.json if present.
func ResolveServers(workdir string) map[string]ServerConfig {
	result := make(map[string]ServerConfig)
	for name, cfg := range DefaultServers {
		if cfg.Disabled {
			continue
		}
		if !hasRootMarker(workdir, cfg.RootMarkers) {
			continue
		}
		avail := binaryAvailable(cfg.Command, workdir)
		if !avail {
			continue
		}
		result[name] = cfg
	}

	userCfg := loadUserConfig(workdir)
	for name, cfg := range userCfg {
		if cfg.Disabled {
			delete(result, name)
			continue
		}
		result[name] = cfg
	}

	return result
}

func hasRootMarker(workdir string, markers []string) bool {
	if len(markers) == 0 {
		return true
	}
	for _, m := range markers {
		matches, _ := filepath.Glob(filepath.Join(workdir, m))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

func binaryAvailable(command string, workdir string) bool {
	if filepath.IsAbs(command) {
		_, err := os.Stat(command)
		return err == nil
	}

	searchPaths := []string{}

	if nmBin := filepath.Join(workdir, "node_modules", ".bin"); dirExists(nmBin) {
		searchPaths = append(searchPaths, nmBin)
	}

	for _, venv := range []string{".venv", "venv"} {
		binDir := filepath.Join(workdir, venv, "bin")
		if runtime.GOOS == "windows" {
			binDir = filepath.Join(workdir, venv, "Scripts")
		}
		if dirExists(binDir) {
			searchPaths = append(searchPaths, binDir)
		}
	}

	for _, p := range searchPaths {
		candidate := filepath.Join(p, resolveExe(command))
		if fileExists(candidate) {
			return true
		}
	}

	_, err := exec.LookPath(resolveExe(command))
	return err == nil
}

func resolveExe(name string) string {
	if runtime.GOOS == "windows" {
		for _, ext := range []string{".exe", ".cmd", ".bat"} {
			if _, err := os.Stat(name + ext); err == nil {
				return name + ext
			}
		}
	}
	return name
}

func loadUserConfig(workdir string) map[string]ServerConfig {
	configPath := filepath.Join(workdir, configDir, "lsp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var userCfg map[string]ServerConfig
	if err := json.Unmarshal(data, &userCfg); err != nil {
		return nil
	}

	for name, cfg := range userCfg {
		if len(cfg.FileTypes) > 0 && !strings.HasPrefix(cfg.Command, "/") && !strings.HasPrefix(cfg.Command, ".") {
			cfg.Command = resolveExe(cfg.Command)
		}
		userCfg[name] = cfg
	}

	return userCfg
}

// FileTypeToServer maps a file extension to server names that handle it,
// given a set of resolved server configs.
func FileTypeToServer(ext string, servers map[string]ServerConfig) []string {
	var names []string
	for name, cfg := range servers {
		for _, ft := range cfg.FileTypes {
			if strings.EqualFold(ft, ext) {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ResolveCommand returns the full path to a server command,
// searching node_modules/.bin, venv, then PATH.
func ResolveCommand(command string, workdir string) string {
	if filepath.IsAbs(command) {
		return command
	}

	searchPaths := []string{}
	if nmBin := filepath.Join(workdir, "node_modules", ".bin"); dirExists(nmBin) {
		searchPaths = append(searchPaths, nmBin)
	}
	for _, venv := range []string{".venv", "venv"} {
		binDir := filepath.Join(workdir, venv, "bin")
		if runtime.GOOS == "windows" {
			binDir = filepath.Join(workdir, venv, "Scripts")
		}
		if dirExists(binDir) {
			searchPaths = append(searchPaths, binDir)
		}
	}

	exeName := resolveExe(command)
	for _, p := range searchPaths {
		candidate := filepath.Join(p, exeName)
		if fileExists(candidate) {
			return candidate
		}
	}

	if found, err := exec.LookPath(exeName); err == nil {
		return found
	}

	return command
}

// FormatDiagnostics formats diagnostics for a file into human-readable text.
func FormatDiagnostics(uri string, diags []Diagnostic) string {
	if len(diags) == 0 {
		return fmt.Sprintf("No diagnostics for %s", uri)
	}

	severityName := map[DiagnosticSeverity]string{
		SeverityError:       "Error",
		SeverityWarning:     "Warning",
		SeverityInformation: "Info",
		SeverityHint:        "Hint",
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Diagnostics for %s:\n", uri))
	for _, d := range diags {
		sev := "Error"
		if name, ok := severityName[d.Severity]; ok {
			sev = name
		}
		loc := fmt.Sprintf("%d:%d-%d:%d", d.Range.Start.Line+1, d.Range.Start.Character+1, d.Range.End.Line+1, d.Range.End.Character+1)
		sb.WriteString(fmt.Sprintf("  [%s] %s %s", sev, loc, d.Message))
		if d.Source != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", d.Source))
		}
		if d.Code != nil {
			sb.WriteString(fmt.Sprintf(" code=%v", d.Code))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// FormatLocations formats a list of locations into human-readable text.
func FormatLocations(locs []Location) string {
	if len(locs) == 0 {
		return "No results found."
	}
	var sb strings.Builder
	for _, loc := range locs {
		path := uriToPath(loc.URI)
		r := loc.Range
		sb.WriteString(fmt.Sprintf("%s:%d:%d\n", path, r.Start.Line+1, r.Start.Character+1))
	}
	return sb.String()
}

// FormatSymbols formats document symbols into a tree.
func FormatSymbols(syms []DocumentSymbol, indent string) string {
	if len(syms) == 0 {
		return "No symbols found."
	}
	var sb strings.Builder
	for _, s := range syms {
		kindName := symbolKindName(s.Kind)
		sb.WriteString(fmt.Sprintf("%s[%s] %s (%d:%d)\n", indent, kindName, s.Name, s.SelectionRange.Start.Line+1, s.SelectionRange.Start.Character+1))
		if len(s.Children) > 0 {
			sb.WriteString(FormatSymbols(s.Children, indent+"  "))
		}
	}
	return sb.String()
}

func symbolKindName(k SymbolKind) string {
	names := map[SymbolKind]string{
		SKFile: "File", SKModule: "Module", SKNamespace: "Namespace", SKPackage: "Package",
		SKClass: "Class", SKMethod: "Method", SKProperty: "Property", SKField: "Field",
		SKConstructor: "Constructor", SKEnum: "Enum", SKInterface: "Interface", SKFunction: "Function",
		SKVariable: "Variable", SKConstant: "Constant", SKString: "String", SKNumber: "Number",
		SKBoolean: "Boolean", SKArray: "Array", SKObject: "Object", SKKey: "Key",
		SKNull: "Null", SKEnumMember: "EnumMember", SKStruct: "Struct", SKEvent: "Event",
		SKOperator: "Operator", SKTypeParameter: "TypeParameter",
	}
	if n, ok := names[k]; ok {
		return n
	}
	return fmt.Sprintf("Kind(%d)", k)
}
