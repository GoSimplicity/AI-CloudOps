package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// 测试用 token（非真实凭据）
const (
	handlerOwnToken    = "fake-handler-sa-token-01"
	handlerIssuedToken = "fake-handler-sa-token-02"
)

type handlerSAManager struct {
	manager.ServiceAccountManager
	tokenTarget string
	tokenReq    *authv1.TokenRequest
}

func (handlerSAManager) GetServiceAccount(_ context.Context, _ int, namespace, name string) (*corev1.ServiceAccount, error) {
	return &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: types.UID("uid-" + name)}}, nil
}

func (handlerSAManager) GetServiceAccountSecrets(_ context.Context, _ int, namespace, name string) ([]corev1.Secret, error) {
	return []corev1.Secret{{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-token", Namespace: namespace, Annotations: map[string]string{
			corev1.ServiceAccountNameKey: name, corev1.ServiceAccountUIDKey: "uid-" + name,
		}},
		Type: corev1.SecretTypeServiceAccountToken,
		Data: map[string][]byte{corev1.ServiceAccountTokenKey: []byte(handlerOwnToken)},
	}}, nil
}

func (m *handlerSAManager) CreateServiceAccountToken(_ context.Context, _ int, _, name string, tr *authv1.TokenRequest) (*authv1.TokenRequest, error) {
	m.tokenTarget = name
	m.tokenReq = tr.DeepCopy()
	out := tr.DeepCopy()
	out.Status.Token = handlerIssuedToken
	return out, nil
}

func TestServiceAccountTokenRoutesMatchPermissionCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewK8sServiceAccountHandler(nil).RegisterRouters(engine)
	found := map[string]bool{}
	for _, r := range engine.Routes() {
		if r.Path == model.K8sServiceAccountTokenRoute {
			found[r.Method] = true
		}
	}
	if !found["GET"] || !found["POST"] {
		t.Fatalf("token 路由与权限码路径不一致: %v", found)
	}
}

func newSATokenEngine(audit service.ServiceAccountService) *gin.Engine {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 9, Username: "bob"}); c.Next() })
	NewK8sServiceAccountHandler(audit).RegisterRouters(engine)
	return engine
}

func TestServiceAccountTokenEndpointsNoStoreAndAuditWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		method, body, token, op string
	}{
		{"GET", "", handlerOwnToken, model.ServiceAccountTokenRevealAuditOperation},
		// 请求体中的 service_account_name 不能覆盖 URL 中的 SA
		{"POST", `{"expiration_seconds":600,"service_account_name":"admin"}`, handlerIssuedToken, model.ServiceAccountTokenCreateAuditOperation},
	} {
		t.Run(tt.method, func(t *testing.T) {
			audit := &recordingAudit{}
			mgr := &handlerSAManager{}
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 9, Username: "bob"}); c.Next() })
			engine.Use(middleware.NewAuditLogMiddleware(audit, zap.NewNop()).AuditLog())
			NewK8sServiceAccountHandler(service.NewServiceAccountService(mgr, audit, zap.NewNop())).RegisterRouters(engine)

			var req = httptest.NewRequest(tt.method, "/api/k8s/serviceaccount/3/ops/app/token", strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("Pragma") != "no-cache" {
				t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
			var resp struct {
				Code int                           `json:"code"`
				Data model.ServiceAccountTokenInfo `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Data.Token != tt.token {
				t.Fatalf("响应不符: %s", rec.Body.String())
			}
			if tt.method == "POST" && mgr.tokenTarget != "app" {
				t.Fatalf("TokenRequest 目标应为 URL 中的 SA, got %q", mgr.tokenTarget)
			}

			audit.mu.Lock()
			defer audit.mu.Unlock()
			if len(audit.sync) != 1 || audit.sync[0].OperationType != tt.op || audit.sync[0].TargetID != "3/ops/app" ||
				audit.sync[0].Endpoint != "/api/k8s/serviceaccount/3/ops/app/token" || audit.sync[0].UserID != 9 {
				t.Fatalf("同步审计不符: %+v", audit.sync)
			}
			raw, _ := json.Marshal(append(append([]*model.CreateAuditLogRequest{}, audit.sync...), audit.async...))
			if strings.Contains(string(raw), tt.token) {
				t.Fatal("审计记录中出现了 token")
			}
			for _, l := range audit.async {
				if len(l.ResponseBody) != 0 {
					t.Fatal("审计中间件不应记录响应体")
				}
			}
		})
	}
}

func TestServiceAccountTokenEndpointsAuditFailureReturns500WithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			engine := newSATokenEngine(service.NewServiceAccountService(&handlerSAManager{}, failingAudit{}, zap.NewNop()))
			req := httptest.NewRequest(method, "/api/k8s/serviceaccount/3/ops/app/token", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != 500 {
				t.Fatalf("审计失败应返回 500, got %d", rec.Code)
			}
			if rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("失败响应也应禁止缓存")
			}
			body := rec.Body.String()
			if strings.Contains(body, handlerOwnToken) || strings.Contains(body, handlerIssuedToken) {
				t.Fatal("审计失败时响应中出现了 token")
			}
		})
	}
}

func TestServiceAccountTokenCreateExpirationValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, body string
		wantHTTP   int
		wantExp    int64 // 传给 K8s 的有效期；0 表示不应调用 K8s
	}{
		{"不传", `{}`, 200, 3600},
		{"传0", `{"expiration_seconds":0}`, 200, 3600},
		{"599", `{"expiration_seconds":599}`, 400, 0},
		{"600", `{"expiration_seconds":600}`, 200, 600},
		{"3600", `{"expiration_seconds":3600}`, 200, 3600},
		{"3601", `{"expiration_seconds":3601}`, 400, 0},
		{"负数", `{"expiration_seconds":-1}`, 400, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			audit := &recordingAudit{}
			mgr := &handlerSAManager{}
			engine := newSATokenEngine(service.NewServiceAccountService(mgr, audit, zap.NewNop()))
			req := httptest.NewRequest("POST", "/api/k8s/serviceaccount/3/ops/app/token", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)
			if rec.Code != tt.wantHTTP || rec.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
			}
			var resp struct {
				Code int                           `json:"code"`
				Data model.ServiceAccountTokenInfo `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatal(err)
			}
			audit.mu.Lock()
			defer audit.mu.Unlock()
			if len(audit.sync) != 1 || audit.sync[0].StatusCode != tt.wantHTTP {
				t.Fatalf("应写一条状态码为 %d 的审计: %+v", tt.wantHTTP, audit.sync)
			}
			if tt.wantExp == 0 {
				if resp.Code != 1 || resp.Data.Token != "" || strings.Contains(rec.Body.String(), handlerIssuedToken) {
					t.Fatalf("越界应返回参数错误且不含 token: %s", rec.Body.String())
				}
				if mgr.tokenTarget != "" || mgr.tokenReq != nil {
					t.Fatal("越界时不应调用 K8s TokenRequest")
				}
				return
			}
			if resp.Code != 0 || resp.Data.Token != handlerIssuedToken || mgr.tokenReq == nil ||
				mgr.tokenReq.Spec.ExpirationSeconds == nil || *mgr.tokenReq.Spec.ExpirationSeconds != tt.wantExp {
				t.Fatalf("应以 %d 秒签发: %s %+v", tt.wantExp, rec.Body.String(), mgr.tokenReq)
			}
		})
	}
}
