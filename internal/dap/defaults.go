package dap

var DefaultAdapters = map[string]DapAdapterConfig{
	"dlv": {
		Command:     "dlv",
		Args:        []string{"dap"},
		Languages:   []string{"go"},
		FileTypes:   []string{".go"},
		RootMarkers: []string{"go.mod"},
		ConnectMode: "stdio",
	},
	"debugpy": {
		Command:     "python",
		Args:        []string{"-m", "debugpy.adapter"},
		Languages:   []string{"python"},
		FileTypes:   []string{".py"},
		RootMarkers: []string{"pyproject.toml", "setup.py", "requirements.txt"},
		ConnectMode: "stdio",
	},
	"gdb": {
		Command:     "gdb",
		Args:        []string{"--interpreter=dap"},
		Languages:   []string{"c", "cpp", "rust"},
		FileTypes:   []string{".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".rs"},
		RootMarkers: []string{"Cargo.toml", "CMakeLists.txt", "Makefile"},
		ConnectMode: "stdio",
	},
	"lldb-dap": {
		Command:     "lldb-dap",
		Args:        []string{},
		Languages:   []string{"c", "cpp", "rust", "swift"},
		FileTypes:   []string{".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".rs", ".swift"},
		RootMarkers: []string{"Cargo.toml", "CMakeLists.txt", "Makefile"},
		ConnectMode: "stdio",
	},
	"js-debug": {
		Command:     "js-debug",
		Args:        []string{},
		Languages:   []string{"javascript", "typescript"},
		FileTypes:   []string{".js", ".ts", ".mjs", ".cjs", ".jsx", ".tsx"},
		RootMarkers: []string{"package.json"},
		ConnectMode: "stdio",
	},
}
