package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type loginStub struct {
	service.UserService
	user *model.User
	err  error
	ip   string
}

func (s *loginStub) Login(_ context.Context, _ *model.UserLoginReq, ip string) (*model.User, error) {
	s.ip = ip
	return s.user, s.err
}

type loginJWT struct {
	ijwt.Handler
	called bool
}

func (j *loginJWT) SetLoginToken(*gin.Context, int, string, int8) (string, string, error) {
	j.called = true
	return "access-token", "refresh-token", nil
}

func TestLoginErrorContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
		wantMsg    string
		retryAfter string
		wantToken  bool
	}{
		{name: "missing user", err: constants.ErrorUserNotExist, wantStatus: http.StatusOK, wantCode: 1, wantMsg: "用户名或密码错误"},
		{name: "wrong password", err: constants.ErrorPasswordIncorrect, wantStatus: http.StatusOK, wantCode: 1, wantMsg: "用户名或密码错误"},
		{name: "unified", err: constants.ErrorInvalidCredentials, wantStatus: http.StatusOK, wantCode: 1, wantMsg: "用户名或密码错误"},
		{name: "locked", err: &service.LoginLockedError{RetryAfter: 90 * time.Second}, wantStatus: http.StatusTooManyRequests, wantCode: 1, wantMsg: "登录失败次数过多，请 2 分钟后再试", retryAfter: "90"},
		{name: "redis down", err: service.ErrLoginUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: 1, wantMsg: "登录服务暂不可用，请稍后重试"},
		{name: "success", wantStatus: http.StatusOK, wantCode: 0, wantToken: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stub := &loginStub{user: &model.User{Model: model.Model{ID: 7}, Username: "alice", RealName: "Alice"}, err: tt.err}
			tokens := &loginJWT{}
			handler := NewUserHandler(stub, tokens)
			engine := newLoginEngine(t, handler)
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"username":"alice","password":"s3cret-login-value"}`))
			req.RemoteAddr = "10.1.2.3:1234"
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "198.51.100.50")
			req.Header.Set("X-Real-IP", "203.0.113.9")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var body struct {
				Code    int            `json:"code"`
				Message string         `json:"message"`
				Data    map[string]any `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tt.wantCode || (tt.wantMsg != "" && body.Message != tt.wantMsg) {
				t.Fatalf("body=%s", rec.Body.String())
			}
			if tt.wantCode == 1 && !strings.Contains(rec.Body.String(), `"data":{}`) {
				t.Fatalf("error data=%s", rec.Body.String())
			}
			if rec.Header().Get("Retry-After") != tt.retryAfter {
				t.Fatalf("retry-after=%q", rec.Header().Get("Retry-After"))
			}
			if tokens.called != tt.wantToken {
				t.Fatalf("token issued=%v", tokens.called)
			}
			if stub.ip != "203.0.113.9" {
				t.Fatalf("ip=%s", stub.ip)
			}
			if strings.Contains(rec.Body.String(), "s3cret-login-value") || strings.Contains(rec.Body.String(), "用户不存在") {
				t.Fatalf("response leaked details: %s", rec.Body.String())
			}
		})
	}
}

func TestLoginClientIPTrustsOnlyProxyPeer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name   string
		peer   string
		realIP string
		want   string
	}{
		{name: "public peer ignores forged header", peer: "203.0.113.10:1234", realIP: "198.51.100.8", want: "203.0.113.10"},
		{name: "private proxy uses X-Real-IP", peer: "10.0.0.8:4321", realIP: "203.0.113.9", want: "203.0.113.9"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stub := &loginStub{err: constants.ErrorInvalidCredentials}
			engine := newLoginEngine(t, NewUserHandler(stub, &loginJWT{}))
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{"username":"alice","password":"pw"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Forwarded-For", "198.51.100.77")
			req.Header.Set("X-Real-IP", tt.realIP)
			req.RemoteAddr = tt.peer
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if stub.ip != tt.want {
				t.Fatalf("ip=%s", stub.ip)
			}
		})
	}
}

func newLoginEngine(t *testing.T, handler *UserHandler) *gin.Engine {
	t.Helper()
	engine := gin.New()
	if err := middleware.UseTrustedRealIP(engine, nil); err != nil {
		t.Fatal(err)
	}
	handler.RegisterRoutes(engine)
	return engine
}

func TestLoginRejectsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &loginStub{err: errors.New("should not run")}
	engine := newLoginEngine(t, NewUserHandler(stub, &loginJWT{}))
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || stub.ip != "" {
		t.Fatalf("status=%d ip=%s body=%s", rec.Code, stub.ip, rec.Body.String())
	}
}
