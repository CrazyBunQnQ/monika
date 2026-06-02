package lsp

// ServerConfig defines a language server configuration.
type ServerConfig struct {
	Command     string         `json:"command"`
	Args        []string       `json:"args,omitempty"`
	FileTypes   []string       `json:"fileTypes"`
	RootMarkers []string       `json:"rootMarkers"`
	InitOptions map[string]any `json:"initOptions,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
	Disabled    bool           `json:"disabled,omitempty"`
}

// DefaultServers is the built-in language server registry, ported from
// oh-my-pi's defaults.json. Covers 40+ languages.
var DefaultServers = map[string]ServerConfig{
	"gopls": {
		Command:     "gopls",
		Args:        []string{"serve"},
		FileTypes:   []string{".go", ".mod", ".sum"},
		RootMarkers: []string{"go.mod", "go.work", "go.sum"},
		Settings: map[string]any{
			"gopls": map[string]any{"staticcheck": true},
		},
	},
	"typescript-language-server": {
		Command:   "typescript-language-server",
		Args:      []string{"--stdio"},
		FileTypes: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"},
		RootMarkers: []string{"package.json", "tsconfig.json", "jsconfig.json"},
		InitOptions: map[string]any{
			"hostInfo": "monika",
			"preferences": map[string]any{
				"includeInlayParameterNameHints":       "all",
				"includeInlayVariableTypeHints":        true,
				"includeInlayFunctionParameterTypeHints": true,
			},
		},
	},
	"rust-analyzer": {
		Command:     "rust-analyzer",
		FileTypes:   []string{".rs"},
		RootMarkers: []string{"Cargo.toml", "rust-analyzer.toml"},
		Settings: map[string]any{
			"rust-analyzer": map[string]any{"checkOnSave": false},
		},
	},
	"clangd": {
		Command:   "clangd",
		Args:      []string{"--background-index", "--clang-tidy", "--header-insertion=iwyu"},
		FileTypes: []string{".c", ".cpp", ".cc", ".cxx", ".h", ".hpp", ".hxx", ".m", ".mm"},
		RootMarkers: []string{"compile_commands.json", "CMakeLists.txt", ".clangd", ".clang-format", "Makefile"},
	},
	"pyright": {
		Command:     "pyright-langserver",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".py", ".pyi"},
		RootMarkers: []string{"pyproject.toml", "pyrightconfig.json", "setup.py", "setup.cfg", "requirements.txt", "Pipfile"},
		Settings: map[string]any{
			"python": map[string]any{
				"analysis": map[string]any{
					"autoSearchPaths":        true,
					"diagnosticMode":         "openFilesOnly",
					"useLibraryCodeForTypes": true,
				},
			},
		},
	},
	"basedpyright": {
		Command:     "basedpyright-langserver",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".py", ".pyi"},
		RootMarkers: []string{"pyproject.toml", "pyrightconfig.json", "setup.py", "requirements.txt"},
		Settings: map[string]any{
			"basedpyright": map[string]any{
				"analysis": map[string]any{
					"autoSearchPaths":        true,
					"diagnosticMode":         "openFilesOnly",
					"useLibraryCodeForTypes": true,
				},
			},
		},
	},
	"pylsp": {
		Command:     "pylsp",
		FileTypes:   []string{".py"},
		RootMarkers: []string{"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt", "Pipfile"},
	},
	"ruff": {
		Command:     "ruff",
		Args:        []string{"server"},
		FileTypes:   []string{".py", ".pyi"},
		RootMarkers: []string{"pyproject.toml", "ruff.toml", ".ruff.toml"},
	},
	"jdtls": {
		Command:     "jdtls",
		FileTypes:   []string{".java"},
		RootMarkers: []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", ".project"},
	},
	"kotlin-lsp": {
		Command:     "kotlin-lsp",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".kt", ".kts"},
		RootMarkers: []string{"build.gradle", "build.gradle.kts", "pom.xml", "settings.gradle", "settings.gradle.kts"},
	},
	"omnisharp": {
		Command:   "omnisharp",
		Args:      []string{"-z", "--hostPID", "$PID", "--encoding", "utf-8", "--languageserver"},
		FileTypes: []string{".cs", ".csx"},
		RootMarkers: []string{"*.sln", "*.csproj", "omnisharp.json", ".git"},
	},
	"sourcekit-lsp": {
		Command:     "sourcekit-lsp",
		FileTypes:   []string{".swift"},
		RootMarkers: []string{"Package.swift", "*.xcodeproj", "*.xcworkspace", "project.yml", ".swiftpm"},
	},
	"ruby-lsp": {
		Command:     "ruby-lsp",
		FileTypes:   []string{".rb", ".rake", ".gemspec", ".erb"},
		RootMarkers: []string{"Gemfile", ".ruby-version", ".ruby-gemset"},
		InitOptions: map[string]any{"formatter": "auto"},
	},
	"solargraph": {
		Command:     "solargraph",
		Args:        []string{"stdio"},
		FileTypes:   []string{".rb", ".rake", ".gemspec"},
		RootMarkers: []string{"Gemfile", ".solargraph.yml", "Rakefile"},
		InitOptions: map[string]any{"formatting": true},
		Settings: map[string]any{
			"solargraph": map[string]any{
				"diagnostics": true, "completion": true, "hover": true,
				"formatting": true, "references": true, "rename": true, "symbols": true,
			},
		},
	},
	"metals": {
		Command:     "metals",
		FileTypes:   []string{".scala", ".sbt", ".sc"},
		RootMarkers: []string{"build.sbt", "build.sc", "build.gradle", "pom.xml"},
		InitOptions: map[string]any{"statusBarProvider": "show-message", "isHttpEnabled": true},
	},
	"hls": {
		Command:     "haskell-language-server-wrapper",
		Args:        []string{"--lsp"},
		FileTypes:   []string{".hs", ".lhs"},
		RootMarkers: []string{"stack.yaml", "cabal.project", "hie.yaml", "package.yaml", "*.cabal"},
		Settings: map[string]any{
			"haskell": map[string]any{"formattingProvider": "ormolu", "checkProject": true},
		},
	},
	"ocamllsp": {
		Command:     "ocamllsp",
		FileTypes:   []string{".ml", ".mli", ".mll", ".mly"},
		RootMarkers: []string{"dune-project", "dune-workspace", "*.opam", ".ocamlformat"},
	},
	"elixirls": {
		Command:     "elixir-ls",
		FileTypes:   []string{".ex", ".exs", ".heex", ".eex"},
		RootMarkers: []string{"mix.exs", "mix.lock"},
		Settings: map[string]any{
			"elixirLS": map[string]any{"dialyzerEnabled": true, "fetchDeps": false},
		},
	},
	"erlangls": {
		Command:     "erlang_ls",
		FileTypes:   []string{".erl", ".hrl"},
		RootMarkers: []string{"rebar.config", "erlang.mk", "rebar.lock"},
	},
	"gleam": {
		Command:     "gleam",
		Args:        []string{"lsp"},
		FileTypes:   []string{".gleam"},
		RootMarkers: []string{"gleam.toml"},
	},
	"zls": {
		Command:     "zls",
		FileTypes:   []string{".zig"},
		RootMarkers: []string{"build.zig", "build.zig.zon", "zls.json"},
	},
	"denols": {
		Command:     "deno",
		Args:        []string{"lsp"},
		FileTypes:   []string{".ts", ".tsx", ".js", ".jsx"},
		RootMarkers: []string{"deno.json", "deno.jsonc", "deno.lock"},
		InitOptions: map[string]any{"enable": true, "lint": true, "unstable": true},
	},
	"vscode-html-language-server": {
		Command:     "vscode-html-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".html", ".htm"},
		RootMarkers: []string{"package.json", ".git"},
		InitOptions: map[string]any{"provideFormatter": true},
	},
	"vscode-css-language-server": {
		Command:     "vscode-css-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".css", ".scss", ".sass", ".less"},
		RootMarkers: []string{"package.json", ".git"},
		InitOptions: map[string]any{"provideFormatter": true},
	},
	"vscode-json-language-server": {
		Command:     "vscode-json-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".json", ".jsonc"},
		RootMarkers: []string{"package.json", ".git"},
		InitOptions: map[string]any{"provideFormatter": true},
	},
	"tailwindcss": {
		Command:   "tailwindcss-language-server",
		Args:      []string{"--stdio"},
		FileTypes: []string{".html", ".css", ".scss", ".js", ".jsx", ".ts", ".tsx", ".vue", ".svelte"},
		RootMarkers: []string{"tailwind.config.js", "tailwind.config.ts", "tailwind.config.mjs", "tailwind.config.cjs"},
	},
	"svelte": {
		Command:     "svelteserver",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".svelte"},
		RootMarkers: []string{"svelte.config.js", "svelte.config.mjs", "package.json"},
	},
	"vue-language-server": {
		Command:     "vue-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".vue"},
		RootMarkers: []string{"vue.config.js", "nuxt.config.js", "nuxt.config.ts", "package.json"},
	},
	"astro": {
		Command:     "astro-ls",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".astro"},
		RootMarkers: []string{"astro.config.mjs", "astro.config.js", "astro.config.ts"},
	},
	"lua-language-server": {
		Command:     "lua-language-server",
		FileTypes:   []string{".lua"},
		RootMarkers: []string{".luarc.json", ".luarc.jsonc", ".luacheckrc", ".stylua.toml", "stylua.toml"},
		Settings: map[string]any{
			"Lua": map[string]any{
				"runtime":     map[string]any{"version": "LuaJIT"},
				"workspace":   map[string]any{"checkThirdParty": false},
				"telemetry":   map[string]any{"enable": false},
			},
		},
	},
	"intelephense": {
		Command:     "intelephense",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".php", ".phtml"},
		RootMarkers: []string{"composer.json", "composer.lock", ".git"},
	},
	"phpactor": {
		Command:     "phpactor",
		Args:        []string{"language-server"},
		FileTypes:   []string{".php"},
		RootMarkers: []string{"composer.json", ".phpactor.json", ".phpactor.yml"},
	},
	"dartls": {
		Command:     "dart",
		Args:        []string{"language-server", "--protocol=lsp"},
		FileTypes:   []string{".dart"},
		RootMarkers: []string{"pubspec.yaml", "pubspec.lock"},
		InitOptions: map[string]any{"closingLabels": true, "flutterOutline": true, "outline": true},
	},
	"bashls": {
		Command:     "bash-language-server",
		Args:        []string{"start"},
		FileTypes:   []string{".sh", ".bash", ".zsh"},
		RootMarkers: []string{".git"},
		Settings: map[string]any{
			"bashIde": map[string]any{"globPattern": "*@(.sh|.inc|.bash|.command)"},
		},
	},
	"yamlls": {
		Command:     "yaml-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".yaml", ".yml"},
		RootMarkers: []string{".git"},
		Settings: map[string]any{
			"yaml":   map[string]any{"validate": true, "hover": true, "completion": true},
			"redhat": map[string]any{"telemetry": map[string]any{"enabled": false}},
		},
	},
	"terraformls": {
		Command:     "terraform-ls",
		Args:        []string{"serve"},
		FileTypes:   []string{".tf", ".tfvars"},
		RootMarkers: []string{".terraform", "terraform.tfstate", "*.tf"},
	},
	"dockerls": {
		Command:     "docker-langserver",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".dockerfile", "Dockerfile"},
		RootMarkers: []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml", ".dockerignore"},
	},
	"nixd": {
		Command:     "nixd",
		FileTypes:   []string{".nix"},
		RootMarkers: []string{"flake.nix", "default.nix", "shell.nix"},
	},
	"ols": {
		Command:     "ols",
		FileTypes:   []string{".odin"},
		RootMarkers: []string{"ols.json", ".git"},
	},
	"marksman": {
		Command:     "marksman",
		Args:        []string{"server"},
		FileTypes:   []string{".md", ".markdown"},
		RootMarkers: []string{".marksman.toml", ".git"},
	},
	"texlab": {
		Command:     "texlab",
		FileTypes:   []string{".tex", ".bib", ".sty", ".cls"},
		RootMarkers: []string{".latexmkrc", "latexmkrc", ".texlabroot", "texlabroot", "Tectonic.toml"},
		Settings: map[string]any{
			"texlab": map[string]any{
				"chktex": map[string]any{"onOpenAndSave": true},
			},
		},
	},
	"graphql": {
		Command:     "graphql-lsp",
		Args:        []string{"server", "-m", "stream"},
		FileTypes:   []string{".graphql", ".gql"},
		RootMarkers: []string{".graphqlrc", ".graphqlrc.json", ".graphqlrc.yml", ".graphqlrc.yaml", "graphql.config.js"},
	},
	"prismals": {
		Command:     "prisma-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".prisma"},
		RootMarkers: []string{"schema.prisma", "prisma/schema.prisma"},
	},
	"vimls": {
		Command:     "vim-language-server",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".vim", ".vimrc"},
		RootMarkers: []string{".git"},
		InitOptions: map[string]any{
			"isNeovim": true,
			"diagnostic": map[string]any{"enable": true},
		},
	},
	"emmet-language-server": {
		Command:   "emmet-language-server",
		Args:      []string{"--stdio"},
		FileTypes: []string{".html", ".css", ".scss", ".less", ".jsx", ".tsx", ".vue", ".svelte"},
		RootMarkers: []string{".git"},
	},
	"tlaplus": {
		Command:     "tlapm_lsp",
		Args:        []string{"--stdio"},
		FileTypes:   []string{".tla", ".tlaplus"},
		RootMarkers: []string{"*.tla"},
	},
	"biome": {
		Command:     "biome",
		Args:        []string{"lsp-proxy"},
		FileTypes:   []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".json", ".jsonc"},
		RootMarkers: []string{"biome.json", "biome.jsonc"},
	},
	"eslint": {
		Command:   "vscode-eslint-language-server",
		Args:      []string{"--stdio"},
		FileTypes: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".vue", ".svelte"},
		RootMarkers: []string{".eslintrc", ".eslintrc.js", ".eslintrc.json", ".eslintrc.yml", "eslint.config.js", "eslint.config.mjs"},
		Settings: map[string]any{"validate": "on", "run": "onType"},
	},
}
