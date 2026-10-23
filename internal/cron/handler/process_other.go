//go:build !unix

package handler

import "os/exec"

// 非Unix保留CommandContext的直接进程取消，通用WaitDelay限制管道等待
func configureTaskProcess(_ *exec.Cmd) {}
