//go:build smoke

package di

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ws "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gorilla/websocket"
)

func TestWebSocketNegotiatesPublicProtocolOnly(t *testing.T) {
	upgrader := InitWebSocketUpgrader()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.TextMessage, []byte("ready"))
	}))
	defer server.Close()
	dialer := websocket.Dialer{Subprotocols: []string{ws.Protocol, ws.CredentialProtocolPrefix + "private-probe"}}
	conn, response, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), http.Header{"Origin": []string{server.URL}})
	if err != nil {
		t.Fatal("same-origin handshake with explicit port failed")
	}
	defer conn.Close()
	if response.StatusCode != 101 || conn.Subprotocol() != ws.Protocol {
		t.Fatal("fixed public protocol not negotiated")
	}
	if strings.Contains(response.Header.Get("Sec-WebSocket-Protocol"), "private-probe") {
		t.Fatal("credential was selected as response protocol")
	}
	_, message, err := conn.ReadMessage()
	if err != nil || string(message) != "ready" {
		t.Fatal("upgraded connection could not carry data")
	}
}
