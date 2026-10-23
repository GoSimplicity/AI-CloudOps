//go:build smoke

package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh/sshtest"
	wsprotocol "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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

func TestLegacyReaderPreservesLongFrameAcrossSmallReads(t *testing.T) {
	want := strings.Repeat("长input-", 4000) + "\n"
	done := make(chan error, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		reader := NewTerminalReader(conn, zap.NewNop(), nil)
		var result strings.Builder
		var small [17]byte
		for result.Len() < len(want) {
			n, err := reader.Read(small[:])
			if err != nil {
				done <- err
				return
			}
			result.Write(small[:n])
		}
		if result.String() != want {
			done <- errors.New("legacy input was truncated or newline changed")
			return
		}
		done <- nil
	}))
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(strings.TrimSuffix(want, "\n"))); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("long input stalled")
	}
}

func TestLegacyReaderOversizedFrameCancelsWithoutReturningInput(t *testing.T) {
	done := make(chan error, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reader := NewTerminalReader(conn, zap.NewNop(), cancel)
		var buf [8]byte
		n, err := reader.Read(buf[:])
		if err == nil || n != 0 || ctx.Err() == nil {
			done <- errors.New("oversized frame crossed input boundary")
			return
		}
		done <- nil
	}))
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteMessage(websocket.TextMessage, []byte(strings.Repeat("x", MaxTerminalFrameBytes+1)))
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("oversized input stalled")
	}
}

func TestPTYInvalidControlsCloseWithoutDeliveringStdin(t *testing.T) {
	for _, message := range []string{`not-json`, `{"op":"stdin","row_size":1,"col_size":1}`, `{"op":"resize","row_size":0,"col_size":80}`, `{"op":"resize","row_size":25,"col_size":1001}`, `{"op":"resize","row_size":25,"col_size":80,"unknown":1}`, `{"op":"resize","row_size":25,"col_size":80} {}`, strings.Repeat(" ", MaxTerminalControlBytes+1)} {
		done := make(chan error, 1)
		web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
			if err != nil {
				done <- err
				return
			}
			defer conn.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resized := false
			reader := NewTerminalReaderWithOptions(conn, zap.NewNop(), cancel, TerminalOptions{Mode: TerminalModePTY}, func(int, int) error { resized = true; return nil })
			var buf [8]byte
			n, err := reader.Read(buf[:])
			if err == nil || n != 0 || ctx.Err() == nil || resized {
				done <- errors.New("invalid control affected SSH input or window")
				return
			}
			done <- nil
		}))
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
		if err != nil {
			web.Close()
			t.Fatal(err)
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(message))
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Fatal("invalid control left peer connected")
		}
		conn.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("invalid control blocked reader")
		}
		web.Close()
	}
}

func TestPTYMalformedInputDoesNotEnterLifecycleLogs(t *testing.T) {
	upstream := sshtest.StartTerminal(t)
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", upstream.KnownHosts)
	core, logs := observer.New(zap.DebugLevel)
	client := NewClient(zap.New(core))
	defer client.Close()
	if err := client.ConnectContext(context.Background(), &Config{Host: upstream.Host, Port: upstream.Port, Username: "fixture", Password: sshtest.Password, Mode: AuthModePassword}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			done <- err
			return
		}
		done <- client.WebTerminalWithOptionsContext(context.Background(), 7, conn, TerminalOptions{Mode: TerminalModePTY})
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	select {
	case <-upstream.Started:
	case <-time.After(time.Second):
		t.Fatal("shell handshake missing")
	}
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"op":"private-input-canary","data":"private-input-canary"}`))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("invalid control left bridge alive")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), "private-input-canary") {
			t.Fatal("input reached SSH lifecycle log")
		}
	}
}

func TestCompressedPTYMessagesRespectDecodedInputLimit(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		size := MaxTerminalFrameBytes
		if oversized {
			size += 4096
		}
		payload := strings.Repeat("x", size)
		done := make(chan error, 1)
		web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := (&websocket.Upgrader{EnableCompression: true}).Upgrade(w, r, nil)
			if err != nil {
				done <- err
				return
			}
			defer conn.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reader := NewTerminalReaderWithOptions(conn, zap.NewNop(), cancel, TerminalOptions{Mode: TerminalModePTY}, nil)
			if oversized {
				var buf [17]byte
				n, err := reader.Read(buf[:])
				if n != 0 || err == nil || ctx.Err() == nil {
					done <- errors.New("compressed oversize delivered partial input")
					return
				}
			} else {
				decoded := make([]byte, size)
				n, err := io.ReadFull(reader, decoded)
				if err != nil || n != size || string(decoded) != payload {
					done <- errors.New("valid compressed input changed")
					return
				}
			}
			done <- nil
		}))
		conn, _, err := (&websocket.Dialer{EnableCompression: true}).Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
		if err != nil {
			web.Close()
			t.Fatal(err)
		}
		_ = conn.WriteMessage(websocket.BinaryMessage, []byte(payload))
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("compressed input stalled")
		}
		conn.Close()
		web.Close()
	}
}
