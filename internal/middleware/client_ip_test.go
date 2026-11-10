package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTrustedRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := UseTrustedRealIP(engine, nil); err != nil {
		t.Fatal(err)
	}
	engine.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	public := httptest.NewRequest(http.MethodGet, "/ip", nil)
	public.RemoteAddr = "203.0.113.10:1234"
	public.Header.Set("X-Real-IP", "198.51.100.8")
	public.Header.Set("X-Forwarded-For", "198.51.100.9")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, public)
	if rec.Body.String() != "203.0.113.10" {
		t.Fatalf("public peer trusted forged header: %s", rec.Body.String())
	}

	proxy := httptest.NewRequest(http.MethodGet, "/ip", nil)
	proxy.RemoteAddr = "10.1.2.3:4321"
	proxy.Header.Set("X-Real-IP", "203.0.113.9")
	proxy.Header.Set("X-Forwarded-For", "198.51.100.77")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, proxy)
	if rec.Body.String() != "203.0.113.9" {
		t.Fatalf("trusted proxy ignored X-Real-IP: %s", rec.Body.String())
	}
}
