package di

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestOriginAllowlistUsesLoadedConfigurationSnapshot(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{AllowedOrigins: " https://console.example.com , ,https://second.example.com "}}
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://environment.example.com")
	corsMiddleware := InitMiddlewares(cfg, nil, zap.NewNop(), nil, nil, nil)[0]
	upgrader := InitWebSocketUpgrader(cfg)
	// 已构造的HTTP和WebSocket实例不应随外部配置对象或进程环境变化。
	cfg.Server.AllowedOrigins = "https://mutated.example.com"
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://changed.example.com")

	for _, tc := range []struct {
		name   string
		origin string
		allow  bool
	}{
		{"yaml-first", "https://console.example.com", true},
		{"yaml-second", "https://second.example.com", true},
		{"environment-bypass", "https://environment.example.com", false},
		{"environment-mutated", "https://changed.example.com", false},
		{"config-mutated", "https://mutated.example.com", false},
		{"subdomain-confusion", "https://console.example.com.evil.test", false},
		{"different-port", "https://console.example.com:8443", false},
		{"opaque-origin", "null", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "https://api.example.com/probe", nil)
			request.Header.Set("Origin", tc.origin)
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			corsMiddleware(ctx)
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); (got == tc.origin) != tc.allow {
				t.Fatalf("CORS allow=%v, expected %v", got != "", tc.allow)
			}
			if got := upgrader.CheckOrigin(request); got != tc.allow {
				t.Fatalf("WebSocket allow=%v, expected %v", got, tc.allow)
			}
		})
	}
}

func TestOriginAllowlistWithoutConfigurationFailsClosed(t *testing.T) {
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://external.example.com")
	for _, tc := range []struct {
		name string
		cfg  *config.Config
	}{
		{"nil", nil},
		{"empty", &config.Config{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			corsMiddleware := InitMiddlewares(tc.cfg, nil, zap.NewNop(), nil, nil, nil)[0]
			upgrader := InitWebSocketUpgrader(tc.cfg)
			for _, origin := range []string{"https://external.example.com", "https://api.example.com", ""} {
				request := httptest.NewRequest(http.MethodOptions, "https://api.example.com/probe", nil)
				request.Header.Set("Origin", origin)
				request.Header.Set("Access-Control-Request-Method", http.MethodGet)
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				ctx.Request = request
				corsMiddleware(ctx)
				if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Fatal("unconfigured CORS must not emit an allow-origin header")
				}
				wantWebSocket := origin == "" || origin == "https://api.example.com"
				if upgrader.CheckOrigin(request) != wantWebSocket {
					t.Fatal("WebSocket must preserve same-origin and no-origin clients without allowing cross-origin")
				}
			}
		})
	}
}
