package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type liveUserReader struct {
	service.UserService
	user *model.User
	err  error
}

func (s liveUserReader) GetProfile(context.Context, int) (*model.User, error) { return s.user, s.err }

type existingSession struct{ ijwt.Handler }

func (existingSession) ExtractToken(*gin.Context) string { return "existing-token" }
func (existingSession) ParseUserClaims(string) (*ijwt.UserClaims, error) {
	return &ijwt.UserClaims{Uid: 7, Username: "admin", AccountType: 2, UserAgent: "browser", Ssid: "session"}, nil
}
func (existingSession) CheckSession(*gin.Context, string) error { return nil }

func TestExistingTokenUsesCurrentAccountState(t *testing.T) {
	for _, tt := range []struct {
		name   string
		user   *model.User
		err    error
		status int
	}{
		{"active", &model.User{Username: "admin", Enable: 1, AccountType: 1}, nil, 204},
		{"frozen", &model.User{Username: "admin", Enable: 2}, nil, 401},
		{"deleted", nil, errors.New("not found"), 401},
		{"renamed", &model.User{Username: "former-admin", Enable: 1}, nil, 401},
		{"database unavailable", nil, errors.New("unavailable"), 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(NewJWTMiddleware(existingSession{}, liveUserReader{user: tt.user, err: tt.err}).CheckLogin())
			engine.GET("/protected", func(c *gin.Context) {
				claims := c.MustGet("user").(ijwt.UserClaims)
				if claims.AccountType != 1 {
					t.Error("stale account type retained")
				}
				c.Status(204)
			})
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("GET", "/protected", nil))
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d", rec.Code, tt.status)
			}
		})
	}
}

type transportSession struct {
	existingSession
	token string
}

func (s *transportSession) ExtractToken(c *gin.Context) string {
	return strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
}
func (s *transportSession) ParseUserClaims(token string) (*ijwt.UserClaims, error) {
	s.token = token
	if token != "test-token" {
		return nil, errors.New("invalid")
	}
	return s.existingSession.ParseUserClaims(token)
}

func TestCredentialTransport(t *testing.T) {
	for _, tt := range []struct {
		name, path, authorization, protocol string
		upgrade                             bool
		status                              int
	}{
		{"normal execute bearer", "/api/cron/job/execute/1", "Bearer test-token", "", false, 204},
		{"websocket protocol", "/api/k8s/pod/1/ns/pod/containers/web/exec", "", "cloudops-v1, cloudops-auth.test-token", true, 204},
		{"legacy websocket query", "/api/k8s/pod/1/ns/pod/containers/web/exec?token=test-token", "", "", true, 204},
		{"fake ordinary HTTP upgrade", "/api/cron/job/execute/1?token=test-token", "", "", true, 401},
		{"assistant websocket protocol", "/api/v1/assistant/stream", "", "cloudops-v1, cloudops-auth.test-token", true, 204},
		{"non websocket AI path", "/api/v1/assistant/query?token=test-token", "", "cloudops-v1, cloudops-auth.test-token", true, 401},
		{"HTTP query forbidden", "/api/cron/job/execute/1?token=test-token", "", "", false, 401},
		{"duplicate protocol rejected", "/api/k8s/pod/1/ns/pod/containers/web/exec?token=test-token", "", "cloudops-auth.first, cloudops-auth.second", true, 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			session := &transportSession{}
			engine := gin.New()
			engine.Use(NewJWTMiddleware(session, liveUserReader{user: &model.User{Username: "admin", Enable: 1, AccountType: 1}}).CheckLogin())
			engine.Any("/api/cron/job/execute/:id", func(c *gin.Context) { c.Status(204) })
			engine.GET("/api/k8s/pod/:cluster_id/:namespace/:name/containers/:container/exec", func(c *gin.Context) { c.Status(204) })
			engine.Any("/api/v1/*path", func(c *gin.Context) { c.Status(204) })
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("Authorization", tt.authorization)
			req.Header.Set("Sec-WebSocket-Protocol", tt.protocol)
			if tt.upgrade {
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Sec-WebSocket-Version", "13")
				req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			}
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status %d expected %d", rec.Code, tt.status)
			}
		})
	}
}

func TestFakeUpgradePOSTCannotUseBrowserCredential(t *testing.T) {
	session := &transportSession{}
	engine := gin.New()
	engine.Use(NewJWTMiddleware(session, liveUserReader{user: &model.User{Username: "admin", Enable: 1, AccountType: 1}}).CheckLogin())
	called := false
	engine.POST("/api/cron/job/execute/:id", func(c *gin.Context) { called = true; c.Status(204) })
	req := httptest.NewRequest("POST", "/api/cron/job/execute/1?token=test-token", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Protocol", "cloudops-v1, cloudops-auth.test-token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != 401 || called {
		t.Fatal("ordinary POST accepted a forged WebSocket credential")
	}
}
