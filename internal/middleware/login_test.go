package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
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
