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

// issuerCheckingRecorder 使用真实 JWT 解析，仅记录是否签发了新访问令牌
type issuerCheckingRecorder struct {
	ijwt.Handler
	issued bool
}

func (*issuerCheckingRecorder) CheckSession(*gin.Context, string) error { return nil }
func (s *issuerCheckingRecorder) SetJWTToken(*gin.Context, int, string, string, int8) (string, error) {
	s.issued = true
	return "new-token", nil
}

func TestRefreshRejectsTokenFromOtherIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key1, key2 := strings.Repeat("a", 32), strings.Repeat("b", 32)
	oldIssuer := ijwt.NewJWTHandler(nil, ijwt.Config{Key1: key1, Key2: key2, Issuer: "cloudops-old"})
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/login", nil)
	_, refreshToken, err := oldIssuer.SetLoginToken(ctx, 7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		issuer     string
		wantIssued bool
	}{{"cloudops-new", false}, {"cloudops-old", true}} {
		t.Run(tt.issuer, func(t *testing.T) {
			tokens := &issuerCheckingRecorder{Handler: ijwt.NewJWTHandler(nil, ijwt.Config{Key1: key1, Key2: key2, Issuer: tt.issuer})}
			handler := NewUserHandler(refreshUserReader{user: &model.User{Username: "admin", Enable: 1}}, tokens)
			engine := gin.New()
			engine.POST("/refresh", handler.RefreshToken)
			req := httptest.NewRequest("POST", "/refresh", strings.NewReader(`{"refreshToken":"`+refreshToken+`"}`))
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(httptest.NewRecorder(), req)
			if tokens.issued != tt.wantIssued {
				t.Fatalf("issuer=%s 时 issued=%v，期望 %v", tt.issuer, tokens.issued, tt.wantIssued)
			}
		})
	}
}
