//go:build windows

package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func execHandlerOpt() interp.RunnerOption {
	hideNext := func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			hc := interp.HandlerCtx(ctx)
			path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
			if err != nil {
				fmt.Fprintln(hc.Stderr, err)
				return interp.ExitStatus(127)
			}

			envList := buildEnvList(hc.Env)
			cmd := exec.Cmd{
				Path:        path,
				Args:        args,
				Env:         envList,
				Dir:         hc.Dir,
				Stdin:       hc.Stdin,
				Stdout:      hc.Stdout,
				Stderr:      hc.Stderr,
				SysProcAttr: &syscall.SysProcAttr{HideWindow: true},
			}

			err = cmd.Start()
			if err == nil {
				stopf := context.AfterFunc(ctx, func() {
					_ = cmd.Process.Signal(os.Kill)
				})
				defer stopf()
				err = cmd.Wait()
			}

			switch e := err.(type) {
			case *exec.ExitError:
				return interp.ExitStatus(e.ExitCode())
			case *exec.Error:
				fmt.Fprintf(hc.Stderr, "%v\n", e)
				return interp.ExitStatus(127)
			default:
				return err
			}
		}
	}
	return interp.ExecHandlers(hideNext)
}

func buildEnvList(env expand.Environ) []string {
	var list []string
	env.Each(func(name string, vr expand.Variable) bool {
		if vr.Exported {
			list = append(list, name+"="+vr.String())
		}
		return true
	})
	return list
}
