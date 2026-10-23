//go:build smoke

package ssh_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	legacy "github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	terminal "github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh/sshtest"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestPTYProtocolPreservesKeystrokesResizeAndBinaryOutput(t *testing.T) {
	for _, implementation := range []string{"main", "legacy"} {
		t.Run(implementation, func(t *testing.T) {
			upstream := sshtest.StartPTY(t)
			t.Setenv("ENV", "production")
			t.Setenv("SSH_KNOWN_HOSTS", upstream.KnownHosts)
			var run func(context.Context, *websocket.Conn, terminal.TerminalOptions) error
			if implementation == "main" {
				client := terminal.NewClient(zap.NewNop())
				defer client.Close()
				if err := client.ConnectContext(context.Background(), &terminal.Config{Host: upstream.Host, Port: upstream.Port, Username: "fixture", Password: sshtest.Password, Mode: terminal.AuthModePassword}); err != nil {
					t.Fatal(err)
				}
				run = func(ctx context.Context, conn *websocket.Conn, options terminal.TerminalOptions) error {
					return client.WebTerminalWithOptionsContext(ctx, 7, conn, options)
				}
			} else {
				client := legacy.NewSSH(zap.NewNop())
				defer client.Close()
				if err := client.ConnectContext(context.Background(), upstream.Host, upstream.Port, "fixture", sshtest.Password, "", 1, 7); err != nil {
					t.Fatal(err)
				}
				run = client.Web2SSHWithOptionsContext
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					done <- err
					return
				}
				done <- run(ctx, conn, terminal.TerminalOptions{Mode: terminal.TerminalModePTY, Rows: 37, Cols: 96})
			}))
			defer server.Close()
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			select {
			case size := <-upstream.PTY:
				if size.Rows != 37 || size.Cols != 96 || size.Echo != 1 {
					t.Fatal("PTY initial size or remote echo mode lost")
				}
			case <-time.After(time.Second):
				t.Fatal("PTY handshake missing")
			}
			select {
			case <-upstream.Started:
			case <-time.After(time.Second):
				t.Fatal("shell handshake missing")
			}
			counts := map[byte]int{}
			total := 0
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			for total < 4096*3+8192 {
				kind, data, err := conn.ReadMessage()
				if err != nil {
					t.Fatal("concurrent SSH output lost")
				}
				if kind != websocket.BinaryMessage {
					t.Fatal("PTY output is not binary")
				}
				for _, value := range data {
					counts[value]++
				}
				total += len(data)
			}
			if counts[0] != 4096 || counts[255] != 4096 || counts['A'] != 4096 || counts['B'] != 8192 {
				t.Fatal("stdout/stderr bytes crossed or changed")
			}
			prefix := []byte{3, 27, '[', 'A', 'x', 0, 255}
			rawJSON := []byte(`{"op":"resize","row_size":1,"col_size":2}`)
			long := bytes.Repeat([]byte("partial-"), 8192) // exact 64KiB, without newline
			expected := append(append(append([]byte{}, prefix...), rawJSON...), long...)
			for _, frame := range [][]byte{prefix, rawJSON, long} {
				if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
					t.Fatal(err)
				}
			}
			var actual []byte
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()
			for len(actual) < len(expected) {
				select {
				case chunk := <-upstream.Input:
					actual = append(actual, chunk...)
				case <-timer.C:
					t.Fatal("partial input truncated or stalled")
				}
			}
			if !bytes.Equal(actual, expected) {
				t.Fatal("keystrokes were buffered, modified or newline appended")
			}
			select {
			case <-upstream.Resize:
				t.Fatal("binary JSON stdin was guessed as control")
			default:
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"op":"resize","row_size":42,"col_size":120}`)); err != nil {
				t.Fatal(err)
			}
			select {
			case size := <-upstream.Resize:
				if size.Rows != 42 || size.Cols != 120 {
					t.Fatal("resize dimensions swapped")
				}
			case <-time.After(time.Second):
				t.Fatal("SSH window-change missing")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatal("parent cancellation lost")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("terminal blocked after cancellation")
			}
		})
	}
}

func TestPTYPeerCloseAndDeadlineWakeBlockedInput(t *testing.T) {
	for _, implementation := range []string{"main", "legacy"} {
		for _, ending := range []string{"peer close", "deadline"} {
			t.Run(implementation+"/"+ending, func(t *testing.T) {
				upstream := sshtest.StartPTY(t)
				t.Setenv("ENV", "production")
				t.Setenv("SSH_KNOWN_HOSTS", upstream.KnownHosts)
				var run func(context.Context, *websocket.Conn, terminal.TerminalOptions) error
				if implementation == "main" {
					client := terminal.NewClient(zap.NewNop())
					defer client.Close()
					if err := client.ConnectContext(context.Background(), &terminal.Config{Host: upstream.Host, Port: upstream.Port, Username: "fixture", Password: sshtest.Password, Mode: terminal.AuthModePassword}); err != nil {
						t.Fatal(err)
					}
					run = func(ctx context.Context, conn *websocket.Conn, options terminal.TerminalOptions) error {
						return client.WebTerminalWithOptionsContext(ctx, 7, conn, options)
					}
				} else {
					client := legacy.NewSSH(zap.NewNop())
					defer client.Close()
					if err := client.ConnectContext(context.Background(), upstream.Host, upstream.Port, "fixture", sshtest.Password, "", 1, 7); err != nil {
						t.Fatal(err)
					}
					run = client.Web2SSHWithOptionsContext
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				done := make(chan error, 1)
				web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
					if err != nil {
						done <- err
						return
					}
					done <- run(ctx, conn, terminal.TerminalOptions{Mode: terminal.TerminalModePTY})
				}))
				defer web.Close()
				conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				select {
				case <-upstream.Started:
				case <-ctx.Done():
					t.Fatal("shell did not acknowledge readiness")
				}
				if ending == "peer close" {
					conn.Close()
				} else {
					<-ctx.Done()
				}
				select {
				case err := <-done:
					if ending == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
						t.Fatal("deadline identity lost")
					}
				case <-time.After(2 * time.Second):
					t.Fatal("blocked SSH/WS was not reclaimed")
				}
			})
		}
	}
}
