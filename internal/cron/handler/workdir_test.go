package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestShellTaskWritesDefaultWorkDir(t *testing.T) {
	marker := "shell-write-" + strings.ReplaceAll(t.Name(), "/", "-")
	path := filepath.Join(defaultLocalWorkDir, marker)
	t.Cleanup(func() { _ = os.Remove(path) })

	output, err := NewScriptExecutor(zap.NewNop()).Execute(context.Background(), &ScriptTask{
		Type:    "shell",
		Content: "pwd; touch " + marker,
		Timeout: 10,
	})
	if err != nil {
		t.Fatalf("shell 任务写入默认工作目录失败: %v, 输出: %s", err, output)
	}
	if !strings.Contains(output, defaultLocalWorkDir) {
		t.Fatalf("工作目录不是默认目录，输出: %s", output)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("默认工作目录中没有写入文件: %v", err)
	}
	if info.IsDir() {
		t.Fatal("写入目标应是文件")
	}
}

func TestCommandWithoutWorkDirUsesDefault(t *testing.T) {
	marker := "command-write-" + strings.ReplaceAll(t.Name(), "/", "-")
	path := filepath.Join(defaultLocalWorkDir, marker)
	t.Cleanup(func() { _ = os.Remove(path) })

	output, err := NewCommandExecutor(zap.NewNop()).Execute(context.Background(), &CommandTask{
		Command: "touch",
		Args:    model.StringList{marker},
		Timeout: 10,
	})
	if err != nil {
		t.Fatalf("空 work_dir 的命令写入默认目录失败: %v, 输出: %s", err, output)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("默认工作目录中没有命令写入的文件: %v", err)
	}
}

func TestCommandExplicitWorkDirIsNotReplaced(t *testing.T) {
	dir := t.TempDir()
	output, err := NewCommandExecutor(zap.NewNop()).Execute(context.Background(), &CommandTask{
		Command: "pwd",
		WorkDir: dir,
		Timeout: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output) != dir {
		t.Fatalf("显式 work_dir 未被使用: %q", output)
	}
}
