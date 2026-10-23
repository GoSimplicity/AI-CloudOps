package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthRoutesBypassAuthentication(t *testing.T) {
	for _, path := range []string{"/health", "/ready"} {
		t.Run(path, func(t *testing.T) {
			engine := gin.New()
			// nil 依赖确保健康路由不会读取会话、账户或权限数据库
			engine.Use(NewJWTMiddleware(nil, nil).CheckLogin(), NewAuthMiddleware(nil).CheckAuth())
			engine.GET(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
			if response.Code != http.StatusNoContent {
				t.Fatalf("probe blocked: %d", response.Code)
			}
		})
	}
}
