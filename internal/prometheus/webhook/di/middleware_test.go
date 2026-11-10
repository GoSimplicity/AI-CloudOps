package di

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestWebhookDoesNotEmitCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(InitMiddlewares(zap.NewNop())...)
	engine.Any("/probe", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})
	for _, origin := range []string{"https://evil.example", "http://localhost:3000", "null", ""} {
		t.Run(origin, func(t *testing.T) {
			for _, method := range []string{http.MethodOptions, http.MethodGet, http.MethodPost} {
				request := httptest.NewRequest(method, "/probe", nil)
				if origin != "" {
					request.Header.Set("Origin", origin)
					request.Header.Set("Access-Control-Request-Method", http.MethodPost)
					request.Header.Set("Access-Control-Request-Headers", "Authorization")
				}
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
					t.Fatalf("%s %s allowed origin %q", method, origin, got)
				}
				if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
					t.Fatalf("%s %s allowed credentials %q", method, origin, got)
				}
			}
		})
	}
}
