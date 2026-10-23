package middleware

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSafeRecoveryDoesNotLogCredentials(t *testing.T) {
	for _, mode := range []string{gin.DebugMode, gin.ReleaseMode} {
		t.Run(mode, func(t *testing.T) {
			previous := gin.Mode()
			gin.SetMode(mode)
			defer gin.SetMode(previous)
			var logs bytes.Buffer
			engine := gin.New()
			engine.Use(SafeRecovery(&logs))
			engine.POST("/panic", func(c *gin.Context) { panic(fmt.Errorf("panic-private-value")) })
			req := httptest.NewRequest("POST", "/panic?token=query-private-value", strings.NewReader(`{"password":"body-private-value"}`))
			req.Header.Set("Authorization", "Bearer header-private-value")
			req.Header.Set("Sec-WebSocket-Protocol", "cloudops-v1, cloudops-auth.protocol-private-value")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != 500 {
				t.Fatalf("status %d", rec.Code)
			}
			for _, value := range []string{"panic-private-value", "query-private-value", "body-private-value", "header-private-value", "protocol-private-value"} {
				if strings.Contains(logs.String(), value) || strings.Contains(rec.Body.String(), value) {
					t.Fatal("recovery disclosed private data")
				}
			}
			if !strings.Contains(logs.String(), `route="/panic"`) || !strings.Contains(logs.String(), "panic_type=") {
				t.Fatal("safe diagnostic metadata missing")
			}
		})
	}
}

func TestSafeRecoveryPreservesWrittenResponse(t *testing.T) {
	var logs bytes.Buffer
	engine := gin.New()
	engine.Use(SafeRecovery(&logs))
	engine.GET("/stream", func(c *gin.Context) { c.String(200, "stream-prefix"); panic("private-value") })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/stream", nil))
	if rec.Code != 200 || rec.Body.String() != "stream-prefix" {
		t.Fatal("recovery appended JSON to an active stream")
	}
	if strings.Contains(logs.String(), "private-value") {
		t.Fatal("stream panic leaked private data")
	}
}
