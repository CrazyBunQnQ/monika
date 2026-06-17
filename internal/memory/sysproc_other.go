//go:build !windows

package memory

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
