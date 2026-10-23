//go:build smoke

package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	jwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	ws "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestAIOpsGatewayCredentialBoundary(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "server-only" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("gateway leaked client credentials or failed service authentication")
		}
		if r.Header.Get(aiopsPrincipalHeader) != "7" || r.Header.Get("Sec-WebSocket-Protocol") != "cloudops-v1" || r.URL.Query().Has("token") {
			t.Error("gateway accepted spoofed identity or forwarded a browser credential")
		}
		if r.URL.Path != "/api/v1/chat" || r.URL.Query().Get("q") != "yes" {
			t.Error("gateway changed path/query")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.Copy(w, r.Body)
	}))
	defer upstream.Close()
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
	a := &App{cfg: &config.Config{AIOps: config.AIOpsConfig{BaseURL: upstream.URL, APIKey: "server-only"}}, server: engine}
	if err := a.registerAIOps(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/chat?q=yes&token=browser-private", strings.NewReader("data: hello\n\n"))
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer browser-session")
	req.Header.Set("Cookie", "session=private")
	req.Header.Set("X-API-Key", "attacker-value")
	req.Header.Set(aiopsPrincipalHeader, "999")
	req.Header.Set("Sec-WebSocket-Protocol", "cloudops-v1, cloudops-auth.browser-private")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "data: hello\n\n" {
		t.Fatalf("unexpected proxy response: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAIOpsGatewayRejectsMissingPrincipal(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) }))
	defer upstream.Close()
	for _, uid := range []int{0, -1} {
		engine := gin.New()
		if uid < 0 {
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: uid}); c.Next() })
		}
		a := &App{cfg: &config.Config{AIOps: config.AIOpsConfig{BaseURL: upstream.URL, APIKey: "server-only"}}, server: engine}
		if err := a.registerAIOps(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("GET", "/api/v1/assistant/ready", nil)
		request.Header.Set(aiopsPrincipalHeader, "7")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || called {
			t.Fatal("unverified principal reached the AI service")
		}
	}
}

func TestAIOpsWebSocketIdentityBoundary(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(aiopsPrincipalHeader) != "7" || r.Header.Get("X-API-Key") != "server-only" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Sec-WebSocket-Protocol") != ws.Protocol || r.URL.Query().Has("token") {
			t.Error("websocket proxy crossed credential or identity boundary")
		}
		upgrader := websocket.Upgrader{Subprotocols: []string{ws.Protocol}, CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.TextMessage, []byte("proxy-ready"))
	}))
	defer upstream.Close()
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
	a := &App{cfg: &config.Config{AIOps: config.AIOpsConfig{BaseURL: upstream.URL, APIKey: "server-only"}}, server: engine}
	if err := a.registerAIOps(); err != nil {
		t.Fatal(err)
	}
	gateway := httptest.NewServer(engine)
	defer gateway.Close()
	dialer := websocket.Dialer{Subprotocols: []string{ws.Protocol, ws.CredentialProtocolPrefix + "browser-probe"}}
	headers := http.Header{aiopsPrincipalHeader: []string{"999"}, "Authorization": []string{"Bearer browser-probe"}, "Cookie": []string{"session=private"}}
	conn, response, err := dialer.Dial("ws"+strings.TrimPrefix(gateway.URL, "http")+"/api/v1/assistant/stream?token=browser-probe", headers)
	if err != nil {
		t.Fatal("upstream websocket boundary did not negotiate")
	}
	defer conn.Close()
	if response.StatusCode != 101 || conn.Subprotocol() != ws.Protocol {
		t.Fatal("public websocket protocol lost in proxy")
	}
	_, message, err := conn.ReadMessage()
	if err != nil || string(message) != "proxy-ready" {
		t.Fatal("upgraded proxy did not carry data")
	}
}
