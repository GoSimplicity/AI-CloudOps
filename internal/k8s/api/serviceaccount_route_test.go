package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemservice "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
)

type routeGrantRoles struct {
	systemservice.RoleService
	apis []*model.Api
}

func (r routeGrantRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: r.apis}}}, nil
}

// routeProbeSAManager 记录 token 相关的 manager 调用：handler 被执行时必然会调用其中之一
type routeProbeSAManager struct {
	*handlerSAManager
	reveals, issues int
}

func (m *routeProbeSAManager) GetServiceAccount(ctx context.Context, clusterID int, namespace, name string) (*corev1.ServiceAccount, error) {
	m.reveals++
	return m.handlerSAManager.GetServiceAccount(ctx, clusterID, namespace, name)
}

func (m *routeProbeSAManager) CreateServiceAccountToken(ctx context.Context, clusterID int, namespace, name string, tr *authv1.TokenRequest) (*authv1.TokenRequest, error) {
	m.issues++
	return m.handlerSAManager.CreateServiceAccountToken(ctx, clusterID, namespace, name, tr)
}

// newSARouteAuthEngine 与生产一致：gin.New()（RedirectTrailingSlash=true、RedirectFixedPath=false）+ 认证中间件 + 真实路由注册
// （不测 RedirectFixedPath=true：gin v1.10.0 的 findCaseInsensitivePath 在本路由树上会 panic，生产也未开启）
func newSARouteAuthEngine(mgr *routeProbeSAManager, apis []*model.Api) *gin.Engine {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 9, Username: "bob"}); c.Next() })
	engine.Use(middleware.NewAuthMiddleware(routeGrantRoles{apis: apis}).CheckAuth())
	NewK8sServiceAccountHandler(service.NewServiceAccountService(mgr, &recordingAudit{}, zap.NewNop())).RegisterRouters(engine)
	return engine
}

// serveFollowingRedirects 发请求并跟随最多 3 次重定向（模拟客户端自动跟随），返回最终状态码
func serveFollowingRedirects(engine *gin.Engine, method, path string) int {
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(method, path, strings.NewReader(`{"expiration_seconds":600}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code < 300 || rec.Code >= 400 {
			return rec.Code
		}
		loc := rec.Header().Get("Location")
		if loc == "" {
			return rec.Code
		}
		path = loc
	}
	return -1
}

// SEC-29 P3-4：只有通配/参数化授权（没有显式授予 token 权限码）时，末尾斜杠、大小写变体、重复斜杠等路径
// 都不能执行 k8s:serviceaccount:token（GET）或 k8s:serviceaccount:token:create（POST）的 handler；
// 即使跟随 gin 的末尾斜杠重定向，落到规范路径后仍被显式授权检查拦截。
func TestServiceAccountTokenRoutesTrailingSlashAndCaseCannotBypassPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wildcardOnly := []*model.Api{
		{Path: "/api/k8s/*", Method: 1}, {Path: "/api/k8s/*", Method: 2},
		{Path: "/*", Method: 1}, {Path: "/*", Method: 2},
		{Path: "/api/k8s/serviceaccount/:cluster_id/:namespace/:name/:action", Method: 1},
		{Path: "/api/k8s/serviceaccount/:cluster_id/:namespace/:name/:action", Method: 2},
	}
	variants := []string{
		"/api/k8s/serviceaccount/3/ops/app/token",
		"/api/k8s/serviceaccount/3/ops/app/token/",
		"/api/k8s/serviceaccount/3/ops/app/token//",
		"/api/k8s/serviceaccount/3/ops/app//token",
		"/api/k8s/serviceaccount/3/ops/app/Token",
		"/api/k8s/serviceaccount/3/ops/app/TOKEN",
		"/api/k8s/serviceaccount/3/ops/app/TOKEN/",
		"/api/k8s/ServiceAccount/3/ops/app/token",
		"/API/K8S/SERVICEACCOUNT/3/ops/app/token",
		"/api/k8s/serviceaccount/3/ops/app/token/.",
		"/api/k8s/serviceaccount/3/ops/app/./token",
	}
	for _, method := range []string{"GET", "POST"} {
		for _, path := range variants {
			mgr := &routeProbeSAManager{handlerSAManager: &handlerSAManager{}}
			code := serveFollowingRedirects(newSARouteAuthEngine(mgr, wildcardOnly), method, path)
			if mgr.reveals != 0 || mgr.issues != 0 || (code >= 200 && code < 300) {
				t.Fatalf("%s %s 绕过了 token 显式授权: status=%d reveals=%d issues=%d", method, path, code, mgr.reveals, mgr.issues)
			}
		}
	}

	// 只显式授予 GET（k8s:serviceaccount:token）时，POST 的任何路径变体都不能签发 token
	getOnly := append([]*model.Api{{Path: model.K8sServiceAccountTokenRoute, Method: 1}}, wildcardOnly...)
	for _, path := range variants {
		mgr := &routeProbeSAManager{handlerSAManager: &handlerSAManager{}}
		code := serveFollowingRedirects(newSARouteAuthEngine(mgr, getOnly), "POST", path)
		if mgr.issues != 0 || (code >= 200 && code < 300) {
			t.Fatalf("POST %s 未授予 token:create 却签发了 token: status=%d", path, code)
		}
	}

	// 对照：显式授权后规范路径可以正常访问（证明上面的拒绝来自授权检查而非路由缺失）
	for _, tt := range []struct {
		method string
		code   int8
	}{{"GET", 1}, {"POST", 2}} {
		mgr := &routeProbeSAManager{handlerSAManager: &handlerSAManager{}}
		engine := newSARouteAuthEngine(mgr, []*model.Api{{Path: model.K8sServiceAccountTokenRoute, Method: tt.code}})
		if code := serveFollowingRedirects(engine, tt.method, "/api/k8s/serviceaccount/3/ops/app/token"); code != 200 || mgr.reveals+mgr.issues != 1 {
			t.Fatalf("%s 显式授权后应可访问: status=%d", tt.method, code)
		}
	}
}
