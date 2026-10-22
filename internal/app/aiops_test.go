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
	"github.com/gin-gonic/gin"
)

func TestAIOpsGatewayCredentialBoundary(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "server-only" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("gateway leaked client credentials or failed service authentication")
		}
		if r.URL.Path != "/api/v1/chat" || r.URL.Query().Get("q") != "yes" {
			t.Error("gateway changed path/query")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.Copy(w, r.Body)
	}))
	defer upstream.Close()
	engine := gin.New()
	a := &App{cfg: &config.Config{AIOps: config.AIOpsConfig{BaseURL: upstream.URL, APIKey: "server-only"}}, server: engine}
	if err := a.registerAIOps(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/chat?q=yes", strings.NewReader("data: hello\n\n"))
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer browser-session")
	req.Header.Set("Cookie", "session=private")
	req.Header.Set("X-API-Key", "attacker-value")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "data: hello\n\n" {
		t.Fatalf("unexpected proxy response: %d %s", rec.Code, rec.Body.String())
	}
}
