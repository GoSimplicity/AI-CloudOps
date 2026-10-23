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
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestLegacyTerminalPeerDisconnectReclaimsSSH(t *testing.T) {
	upstream := sshtest.StartTerminal(t)
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", upstream.KnownHosts)
	client := NewSSH(zap.NewNop())
	defer client.Close()
	if err := client.ConnectContext(context.Background(), upstream.Host, upstream.Port, "fixture", sshtest.Password, "", 1, 7); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			done <- err
			return
		}
		done <- client.Web2SSHContext(ctx, conn)
	}))
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-upstream.Started:
	case <-time.After(time.Second):
		t.Fatal("shell handshake missing")
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("fixture")); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil || string(data) != "fixture\n" {
		t.Fatal("legacy wire changed")
	}
	_ = conn.Close()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("peer disconnect left legacy SSH alive")
	}
}
