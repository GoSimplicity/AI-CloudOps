//go:build smoke

package ssh

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh/sshtest"
	"go.uber.org/zap"
	gossh "golang.org/x/crypto/ssh"
)

func TestClientRealSSHProtocol(t *testing.T) {
	for _, mode := range []AuthMode{AuthModePassword, AuthModeKey} {
		t.Run(map[AuthMode]string{AuthModePassword: "password", AuthModeKey: "key"}[mode], func(t *testing.T) {
			server := sshtest.Start(t, false)
			t.Setenv("ENV", "production")
			t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
			client := NewClient(zap.NewNop())
			defer client.Close()
			if err := client.ConnectContext(context.Background(), &Config{Host: server.Host, Port: server.Port, Username: "fixture", Password: sshtest.Password, Key: server.Key, Mode: mode}); err != nil {
				t.Fatal(err)
			}
			output, err := client.RunContext(context.Background(), "success")
			if err != nil || output != "executed" {
				t.Fatalf("execution failed: %v", err)
			}
			output, err = client.RunContext(context.Background(), "large")
			if err != nil || len(output) > MaxCommandOutputBytes+len(outputTruncated) || !strings.HasSuffix(output, outputTruncated) {
				t.Fatalf("output bound failed: length=%d error=%v", len(output), err)
			}
			output, err = client.RunContext(context.Background(), "fail")
			if err == nil || output != "partial" {
				t.Fatal("failed command lost partial output")
			}
		})
	}
}

func TestRunCancellationAfterRemoteStarts(t *testing.T) {
	for _, stall := range []bool{false, true} {
		t.Run(map[bool]string{false: "execution", true: "session request"}[stall], func(t *testing.T) {
			server := sshtest.Start(t, stall)
			t.Setenv("ENV", "production")
			t.Setenv("SSH_KNOWN_HOSTS", server.KnownHosts)
			client := NewClient(zap.NewNop())
			defer client.Close()
			if err := client.Connect(&Config{Host: server.Host, Port: server.Port, Username: "fixture", Password: sshtest.Password, Mode: AuthModePassword}); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := client.RunContext(ctx, "block"); done <- err }()
			select {
			case <-server.Started:
			case <-time.After(3 * time.Second):
				t.Fatal("remote did not acknowledge started request")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("SSH operation ignored cancellation")
			}
		})
	}
}

func TestDialCancellationDuringHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := DialContext(ctx, "tcp", listener.Addr().String(), fixtureClientConfig())
		done <- err
	}()
	var conn net.Conn
	select {
	case conn = <-accepted:
		defer conn.Close()
	case <-time.After(time.Second):
		t.Fatal("connection not accepted")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handshake cancellation lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("handshake did not cancel")
	}
}

func fixtureClientConfig() *gossh.ClientConfig {
	return &gossh.ClientConfig{User: "fixture", HostKeyCallback: gossh.InsecureIgnoreHostKey(), Timeout: 10 * time.Second}
}

func TestClientRejectsUntrustedHostAndBadAuthentication(t *testing.T) {
	for _, kind := range []string{"unknown host", "bad password"} {
		t.Run(kind, func(t *testing.T) {
			server := sshtest.Start(t, false)
			trust := server.KnownHosts
			password := sshtest.Password
			if kind == "unknown host" {
				trust = filepath.Join(t.TempDir(), "empty_known_hosts")
				if err := os.WriteFile(trust, nil, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				password = "wrong-fixture-password"
			}
			t.Setenv("ENV", "production")
			t.Setenv("SSH_KNOWN_HOSTS", trust)
			client := NewClient(zap.NewNop())
			defer client.Close()
			if err := client.ConnectContext(context.Background(), &Config{Host: server.Host, Port: server.Port, Username: "fixture", Password: password, Mode: AuthModePassword}); err == nil {
				t.Fatal("SSH trust/authentication failure accepted")
			}
		})
	}
}
