package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// ShellEngine executes shell commands using the mvdan/sh interpreter.
// It provides cross-platform shell semantics (POSIX + bash extensions)
// without depending on an external system shell.
type ShellEngine struct {
	parser *syntax.Parser
}

// NewShellEngine creates a new ShellEngine.
func NewShellEngine() *ShellEngine {
	return &ShellEngine{
		parser: syntax.NewParser(),
	}
}

// RunResult holds the output and exit status of a command.
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run executes a shell command and waits for it to complete.
func (e *ShellEngine) Run(ctx context.Context, command, workdir string, env []string) RunResult {
	var stdout, stderr bytes.Buffer
	runner, err := e.newRunner(workdir, env, nil, &stdout, &stderr)
	if err != nil {
		return RunResult{Stderr: err.Error(), ExitCode: 1}
	}

	prog, err := e.parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return RunResult{Stderr: fmt.Sprintf("parse error: %v", err), ExitCode: 127}
	}

	// Reset parser state for next use (avoid accumulated state issues).
	e.parser = syntax.NewParser()

	err = runner.Run(ctx, prog)
	exitCode := 0
	if err != nil {
		if status, ok := interp.IsExitStatus(err); ok {
			exitCode = int(status)
		} else {
			exitCode = 1
		}
	}

	return RunResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// StartBackground starts a command in the background, streaming output lines to the caller.
// Returns a cancel function to kill the process and a channel that receives exit code when done.
func (e *ShellEngine) StartBackground(ctx context.Context, command, workdir string, env []string, onLine func(line string)) (context.CancelFunc, <-chan int, error) {
	ctx, cancel := context.WithCancel(ctx)

	pr, pw := io.Pipe()

	runner, err := e.newRunner(workdir, env, nil, pw, pw)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("create runner: %w", err)
	}

	prog, err := e.parser.Parse(strings.NewReader(command), "")
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("parse error: %v", err)
	}
	e.parser = syntax.NewParser()

	exitCh := make(chan int, 1)

	// Read lines from the combined stdout/stderr pipe.
	go func() {
		readLines(pr, onLine)
	}()

	// Run the command.
	go func() {
		defer pw.Close()
		err := runner.Run(ctx, prog)
		code := 0
		if status, ok := interp.IsExitStatus(err); ok {
			code = int(status)
		} else {
			code = 1
		}
		exitCh <- code
	}()

	return cancel, exitCh, nil
}

func (e *ShellEngine) newRunner(workdir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (*interp.Runner, error) {
	env = enrichEnv(env)
	opts := []interp.RunnerOption{
		interp.Env(expand.ListEnviron(env...)),
		interp.Dir(workdir),
		interp.StdIO(stdin, stdout, stderr),
		execHandlerOpt(),
		interp.OpenHandler(openHandler),
	}

	r, err := interp.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("init shell interpreter: %w", err)
	}
	return r, nil
}

func readLines(r io.Reader, onLine func(string)) {
	buf := make([]byte, 4096)
	var lineBuf []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			lineBuf = append(lineBuf, buf[:n]...)
			for {
				i := bytes.IndexByte(lineBuf, '\n')
				if i < 0 {
					break
				}
				line := string(lineBuf[:i])
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				onLine(line)
				lineBuf = lineBuf[i+1:]
			}
		}
		if err != nil {
			// Flush remaining data.
			if len(lineBuf) > 0 {
				line := string(lineBuf)
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				onLine(line)
			}
			return
		}
	}
}

// openHandler is a custom file opener that handles Windows paths.
func openHandler(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return interp.DefaultOpenHandler()(ctx, path, flag, perm)
}

// enrichEnv adds common tool directories to PATH on Windows (e.g. Git for Windows' usr/bin)
// so that standard Unix utilities (ls, cat, grep, etc.) are available in the shell.
func enrichEnv(env []string) []string {
	if runtime.GOOS != "windows" {
		return env
	}

	gitUsrBin := gitUsrBinPath()
	if gitUsrBin == "" {
		return env
	}

	for i, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if !ok || !strings.EqualFold(k, "PATH") {
			continue
		}
		for _, p := range filepath.SplitList(v) {
			if strings.EqualFold(p, gitUsrBin) {
				return env
			}
		}
		env[i] = k + "=" + gitUsrBin + string(os.PathListSeparator) + v
		return env
	}

	return env
}

// gitUsrBinPath returns the path to Git for Windows' usr/bin directory, or "" if not found.
func gitUsrBinPath() string {
	dirs := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "usr", "bin"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "usr", "bin"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git", "usr", "bin"),
	}
	for _, d := range dirs {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			return d
		}
	}

	if gitExe, err := exec.LookPath("git.exe"); err == nil {
		gitRoot := filepath.Dir(filepath.Dir(gitExe))
		candidate := filepath.Join(gitRoot, "usr", "bin")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
	}

	return ""
}

// GlobalShellEngine is a shared instance for use across the application.
var GlobalShellEngine = NewShellEngine()

// ParseError checks if the given error is a shell parse error.
func ParseError(err error) bool {
	_, ok := err.(*syntax.ParseError)
	return ok
}
