package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const handlerFixtureValue = "plain-handler-fixture-777"

type handlerSecretManager struct {
	manager.SecretManager
}

func (handlerSecretManager) GetSecret(_ context.Context, _ int, namespace, name string) (*corev1.Secret, error) {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"alpha": []byte(handlerFixtureValue)},
	}, nil
}

// recordingAudit 同时记录 service 的同步审计与中间件的异步审计
type recordingAudit struct {
	systemService.AuditService
	mu    sync.Mutex
	sync  []*model.CreateAuditLogRequest
	async []*model.CreateAuditLogRequest
}

func (a *recordingAudit) CreateAuditLog(_ context.Context, req *model.CreateAuditLogRequest) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sync = append(a.sync, req)
	return nil
}

func (a *recordingAudit) CreateAuditLogAsync(_ context.Context, req *model.CreateAuditLogRequest) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.async = append(a.async, req)
}

func TestRevealRouteMatchesPermissionCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewK8sSecretHandler(nil).RegisterRouters(engine)
	found := false
	for _, r := range engine.Routes() {
		if r.Method == "GET" && r.Path == model.K8sSecretRevealRoute {
			found = true
		}
	}
	if !found {
		t.Fatalf("未注册 reveal 路由 %s", model.K8sSecretRevealRoute)
	}
}

func TestRevealEndpointReturnsPlaintextAndAuditsWithoutValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audit := &recordingAudit{}
	svc := service.NewSecretService(handlerSecretManager{}, audit, zap.NewNop())

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("user", jwt.UserClaims{Uid: 9, Username: "bob"})
		c.Next()
	})
	// 与线上一致挂载审计中间件，确认它不会把响应体（明文）写入审计
	engine.Use(middleware.NewAuditLogMiddleware(audit, zap.NewNop()).AuditLog())
	NewK8sSecretHandler(svc).RegisterRouters(engine)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/k8s/secret/3/ops/app/reveal", nil))
	if rec.Code != 200 {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("reveal 响应应禁止缓存")
	}
	var resp struct {
		Code int                   `json:"code"`
		Data model.K8sSecretReveal `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if string(resp.Data.Data["alpha"]) != handlerFixtureValue || resp.Data.Namespace != "ops" || resp.Data.Name != "app" || resp.Data.ClusterID != 3 {
		t.Fatalf("reveal 响应不符: %s", rec.Body.String())
	}

	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.sync) != 1 {
		t.Fatalf("reveal 应写 1 条审计，实际 %d", len(audit.sync))
	}
	all := append(append([]*model.CreateAuditLogRequest{}, audit.sync...), audit.async...)
	raw, _ := json.Marshal(all)
	if strings.Contains(string(raw), handlerFixtureValue) || strings.Contains(string(raw), base64.StdEncoding.EncodeToString([]byte(handlerFixtureValue))) {
		t.Fatal("审计记录中出现了明文")
	}
	for _, l := range all {
		if len(l.ResponseBody) != 0 {
			t.Fatal("审计不应记录响应体")
		}
	}
	got := audit.sync[0]
	if got.UserID != 9 || got.TargetID != "3/ops/app" || got.Endpoint != "/api/k8s/secret/3/ops/app/reveal" {
		t.Fatalf("审计元数据不符: %+v", got)
	}
	var body model.SecretRevealAuditRecord
	if err := json.Unmarshal(got.RequestBody, &body); err != nil || body.Username != "bob" || strings.Join(body.Keys, ",") != "alpha" {
		t.Fatalf("审计内容不符: %s", got.RequestBody)
	}
}

type failingAudit struct {
	systemService.AuditService
}

func (failingAudit) CreateAuditLog(context.Context, *model.CreateAuditLogRequest) error {
	return errors.New("audit db unavailable")
}

// 审计写入失败：HTTP 500、不返回明文、仍带 no-store
func TestRevealEndpointAuditFailureReturns500WithoutPlaintext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewSecretService(handlerSecretManager{}, failingAudit{}, zap.NewNop())
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 9, Username: "bob"}); c.Next() })
	NewK8sSecretHandler(svc).RegisterRouters(engine)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/k8s/secret/3/ops/app/reveal", nil))
	if rec.Code != 500 {
		t.Fatalf("审计失败应返回 500, got %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("失败响应也应禁止缓存")
	}
	body := rec.Body.String()
	if strings.Contains(body, handlerFixtureValue) || strings.Contains(body, base64.StdEncoding.EncodeToString([]byte(handlerFixtureValue))) {
		t.Fatal("审计失败时响应中出现了明文")
	}
}
