//go:build !windows

package builtin

import (
	"time"

	"mvdan.cc/sh/v3/interp"
)

func execHandlerOpt() interp.RunnerOption {
	return interp.ExecHandler(interp.DefaultExecHandler(2 * time.Second))
}
