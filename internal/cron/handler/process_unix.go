//go:build unix

package handler

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureTaskProcess(cmd *exec.Cmd) {
	// 独立进程组使取消可覆盖继承输出管道的普通子进程
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
