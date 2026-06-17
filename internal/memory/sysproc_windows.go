//go:build windows

package memory

import (
	"os/exec"
	"syscall"
)

// hideWindow 在 Windows 上隐藏子进程的控制台窗口，
// 避免应用启动时弹出 cmd 黑框一闪而过。
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
