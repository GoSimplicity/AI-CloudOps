//go:build smoke && unix

package handler

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type processReadyWriter struct {
	ready chan int
	text  string
}

func (w *processReadyWriter) Write(p []byte) (int, error) {
	w.text += string(p)
	if strings.Contains(w.text, "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(w.text))
		if err == nil {
			select {
			case w.ready <- pid:
			default:
			}
		}
	}
	return len(p), nil
}

func TestCancellationTerminatesStartedProcessGroup(t *testing.T) {
	for _, mode := range []string{"cancel", "timeout", "parent exits with open child pipe"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if mode == "timeout" {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			}
			defer cancel()
			script := `sleep 60 & child=$!; printf '%s\n' "$child"; wait "$child"`
			if mode == "parent exits with open child pipe" {
				script = `sleep 60 & child=$!; printf '%s\n' "$child"; exit 0`
			}
			cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
			ready := make(chan int, 1)
			cmd.Stdout = &processReadyWriter{ready: ready}
			done := make(chan error, 1)
			go func() { done <- runTaskCommand(cmd) }()
			var child int
			select {
			case child = <-ready:
			case err := <-done:
				t.Fatalf("process exited before readiness: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("process did not signal readiness")
			}
			// 若测试失败仍清理隔离进程组，不遗留60秒子进程
			t.Cleanup(func() {
				if cmd.Process != nil {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				}
				_ = syscall.Kill(child, syscall.SIGKILL)
			})
			started := time.Now()
			if mode == "cancel" {
				cancel()
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("cancelled command succeeded")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("cancelled command did not return within bounded deadline")
			}
			if time.Since(started) > 3*time.Second {
				t.Fatal("cancellation exceeded deadline")
			}
			deadline := time.NewTimer(2 * time.Second)
			defer deadline.Stop()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				if err := syscall.Kill(child, 0); errors.Is(err, syscall.ESRCH) {
					break
				}
				select {
				case <-ticker.C:
				case <-deadline.C:
					t.Fatal("child process still exists after cancellation")
				}
			}
		})
	}
}
