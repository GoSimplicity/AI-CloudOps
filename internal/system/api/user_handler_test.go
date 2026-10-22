package api

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

type refreshUserReader struct {
	service.UserService
	user *model.User
	err  error
}

func (s refreshUserReader) GetProfile(context.Context, int) (*model.User, error) {
	return s.user, s.err
}

type refreshRecorder struct {
	ijwt.Handler
	issued      bool
	accountType int8
}

func (*refreshRecorder) ParseRefreshClaims(string) (*ijwt.RefreshClaims, error) {
	return &ijwt.RefreshClaims{Uid: 7, Username: "admin", AccountType: 2, Ssid: "session"}, nil
}
func (*refreshRecorder) CheckSession(*gin.Context, string) error { return nil }
func (s *refreshRecorder) SetJWTToken(_ *gin.Context, _ int, _ string, _ string, accountType int8) (string, error) {
	s.issued = true
	s.accountType = accountType
	return "new-token", nil
}

func TestRefreshRechecksAccount(t *testing.T) {
	for _, tt := range []struct {
		name       string
		user       *model.User
		err        error
		wantIssued bool
	}{
		{"active", &model.User{Username: "admin", Enable: 1, AccountType: 1}, nil, true},
		{"frozen", &model.User{Username: "admin", Enable: 2}, nil, false},
		{"deleted", nil, errors.New("not found"), false},
		{"renamed", &model.User{Username: "changed", Enable: 1}, nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tokens := &refreshRecorder{}
			handler := NewUserHandler(refreshUserReader{user: tt.user, err: tt.err}, tokens)
			engine := gin.New()
			engine.POST("/refresh", handler.RefreshToken)
			req := httptest.NewRequest("POST", "/refresh", strings.NewReader(`{"refreshToken":"old-token"}`))
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(httptest.NewRecorder(), req)
			if tokens.issued != tt.wantIssued {
				t.Fatalf("token issued=%v", tokens.issued)
			}
			if tokens.issued && tokens.accountType != 1 {
				t.Fatal("token retained obsolete account type")
			}
		})
	}
}
