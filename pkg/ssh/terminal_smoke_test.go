//go:build smoke

package ssh

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh/sshtest"
	wsprotocol "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestWebTerminalContextOwnsCancellation(t *testing.T) {
	for _, mode := range []string{"session creation cancellation", "session creation peer disconnect", "live cancellation", "peer disconnect"} {
		t.Run(mode, func(t *testing.T) {
			var server *sshtest.Server
			if strings.HasPrefix(mode, "session creation") {
				server = sshtest.Start(t, true)
			} else {
				server = sshtest.StartTerminal(t)
			}
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
			web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{Subprotocols: []string{wsprotocol.Protocol}}
				connection, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					done <- err
					return
				}
				done <- client.WebTerminalContext(ctx, 7, connection)
			}))
			defer web.Close()
			connection, _, err := (&websocket.Dialer{Subprotocols: []string{wsprotocol.Protocol}}).Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			select {
			case <-server.Started:
			case <-time.After(time.Second):
				t.Fatal("SSH terminal did not acknowledge initialization")
			}
			if !strings.HasPrefix(mode, "session creation") {
				if err := connection.WriteMessage(websocket.TextMessage, []byte("echo-fixture")); err != nil {
					t.Fatal(err)
				}
				_ = connection.SetReadDeadline(time.Now().Add(time.Second))
				_, output, err := connection.ReadMessage()
				if err != nil || string(output) != "echo-fixture\n" {
					t.Fatal("real SSH terminal did not echo input")
				}
			}
			if mode == "peer disconnect" || mode == "session creation peer disconnect" {
				_ = connection.Close()
			} else {
				cancel()
			}
			select {
			case err := <-done:
				if mode == "session creation peer disconnect" {
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("setup deadline lost: %v", err)
					}
				} else if mode != "peer disconnect" && !errors.Is(err, context.Canceled) {
					t.Fatalf("terminal cancellation lost: %v", err)
				}
			case <-time.After(12 * time.Second):
				t.Fatal("SSH/WS terminal was not reclaimed")
			}
			if err := client.Close(); err != nil {
				t.Fatal("cancelled transport reported a cleanup failure")
			}
		})
	}
}

func TestTerminalWriterSerializesStdoutAndStderr(t *testing.T) {
	finished := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			finished <- err
			return
		}
		defer connection.Close()
		writer := &TerminalWriter{conn: connection, logger: zap.NewNop()}
		results := make(chan error, 2)
		for _, stream := range []string{"stdout", "stderr"} {
			go func(value string) {
				for i := 0; i < 64; i++ {
					if _, err := writer.Write([]byte(value)); err != nil {
						results <- err
						return
					}
				}
				results <- nil
			}(stream)
		}
		first, second := <-results, <-results
		if first != nil {
			finished <- first
		} else {
			finished <- second
		}
	}))
	defer server.Close()
	connection, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_ = connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	counts := map[string]int{}
	for i := 0; i < 128; i++ {
		_, message, err := connection.ReadMessage()
		if err != nil {
			t.Fatal("concurrent terminal streams lost a frame")
		}
		counts[string(message)]++
	}
	if counts["stdout"] != 64 || counts["stderr"] != 64 {
		t.Fatal("terminal streams crossed or lost frames")
	}
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}
