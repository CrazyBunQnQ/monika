//go:build !windows

package dbbridge

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
