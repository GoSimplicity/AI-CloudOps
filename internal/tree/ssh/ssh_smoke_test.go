//go:build smoke

package ssh

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pkgssh "github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh/sshtest"
	"go.uber.org/zap"
)

func TestTreeSSHProtocolExecutionAndCancellation(t *testing.T) {
	server := sshtest.Start(t, false)
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
	client := NewSSH(zap.NewNop())
	defer client.Close()
	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", sshtest.Password, "", 1, 7); err != nil {
		t.Fatal(err)
	}
	output, err := client.RunContext(context.Background(), "large")
	if err != nil || len(output) > pkgssh.MaxCommandOutputBytes+len("[输出已截断]") || !strings.HasSuffix(output, "[输出已截断]") {
		t.Fatal("tree SSH output was not bounded")
	}
	// 清除前一次exec握手，再等待当前block实际开始
	<-server.Started
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := client.RunContext(ctx, "block"); done <- err }()
	select {
	case <-server.Started:
	case <-time.After(time.Second):
		t.Fatal("command did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("command did not cancel")
	}
}

func TestTreeSSHConnectCancelsSessionCreation(t *testing.T) {
	server := sshtest.Start(t, true)
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
	client := NewSSH(zap.NewNop())
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- client.ConnectContext(ctx, server.Host, server.Port, "fixture", sshtest.Password, "", 1, 7)
	}()
	select {
	case <-server.Started:
	case <-time.After(time.Second):
		t.Fatal("session request did not reach server")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("session cancellation lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("session creation ignored cancellation")
	}
}
