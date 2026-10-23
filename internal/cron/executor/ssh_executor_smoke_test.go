//go:build smoke

package executor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh/sshtest"
	"go.uber.org/zap"
)

func TestSSHStructuredValuesStayLiteralInRealShell(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "path '; $(printf injected) ")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	value := "value '; $(printf injected); \"quoted\"\nsecond line"
	job := testSSHJob()
	job.SSHWorkDir = directory
	job.SSHEnvironment = model.KeyValueList{{Key: "VALUE", Value: value}}
	job.SSHCommand = `printf '%s' "$VALUE"; printf '|'; pwd # trailing comment`
	command, err := NewSSHExecutor(nil, nil).buildSSHCommand(job)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/bin/sh", "-c", command).CombinedOutput()
	if err != nil || string(output) != value+"|"+directory+"\n" {
		t.Fatal("directory/environment were evaluated or shell semantics changed")
	}
	// 工作目录失败时阻止整个命令块，而不只阻止第一个分号前的命令
	job.SSHWorkDir = filepath.Join(t.TempDir(), "missing")
	job.SSHCommand = "true; printf should-not-run"
	command, err = NewSSHExecutor(nil, nil).buildSSHCommand(job)
	if err != nil {
		t.Fatal(err)
	}
	output, err = exec.CommandContext(ctx, "/bin/sh", "-c", command).CombinedOutput()
	if err == nil || strings.Contains(string(output), "should-not-run") {
		t.Fatal("failed prefix did not gate entire shell command")
	}
}

func TestCronSSHUsesDecryptedCredentialsOverRealProtocol(t *testing.T) {
	configureEncryption(t)
	server := sshtest.Start(t, false)
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
	encrypted, err := treeUtils.EncryptPassword(sshtest.Password)
	if err != nil {
		t.Fatal(err)
	}
	resource := &model.TreeLocalResource{Status: model.RUNNING, IpAddr: server.Host, Port: server.Port, Username: "fixture", AuthMode: model.AuthModePassword, Password: encrypted}
	executor := NewSSHExecutor(zap.NewNop(), &resourceStub{resource: resource})
	output, err := executor.ExecuteSSHJob(context.Background(), testSSHJob())
	if err != nil || output != "executed" {
		t.Fatalf("real SSH authentication/execution failed: %v", err)
	}
	if resource.Password != encrypted {
		t.Fatal("DAO resource credentials were mutated")
	}
}

func TestCronSSHConnectCancellationOverRealProtocol(t *testing.T) {
	server := sshtest.Start(t, true)
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
	resource := &model.TreeLocalResource{Status: model.RUNNING, IpAddr: server.Host, Port: server.Port, Username: "fixture", AuthMode: model.AuthModeKey, Key: server.Key}
	executor := NewSSHExecutor(zap.NewNop(), &resourceStub{resource: resource})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := executor.ExecuteSSHJob(ctx, testSSHJob()); done <- err }()
	select {
	case <-server.Started:
	case <-time.After(time.Second):
		t.Fatal("remote connection did not reach session request")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("connection cancellation lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Cron SSH connection ignored context")
	}
}

func TestCronSSHCancelAndTimeoutAfterRemoteExecutionStarted(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "task deadline"}[timeout], func(t *testing.T) {
			server := sshtest.Start(t, false)
			t.Setenv("ENV", "production")
			t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
			resource := &model.TreeLocalResource{Status: model.RUNNING, IpAddr: server.Host, Port: server.Port, Username: "fixture", AuthMode: model.AuthModeKey, Key: server.Key}
			executor := NewSSHExecutor(zap.NewNop(), &resourceStub{resource: resource})
			job := testSSHJob()
			job.SSHCommand = "block"
			if timeout {
				job.Timeout = 1
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := executor.ExecuteSSHJob(ctx, job); done <- err }()
			select {
			case <-server.Started:
			case <-time.After(time.Second):
				t.Fatal("command did not reach execution")
			}
			expected := context.Canceled
			if timeout {
				expected = context.DeadlineExceeded
			} else {
				cancel()
			}
			select {
			case err := <-done:
				if !errors.Is(err, expected) {
					t.Fatalf("execution cancellation lost: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("Cron SSH execution exceeded its cancellation bound")
			}
		})
	}
}
