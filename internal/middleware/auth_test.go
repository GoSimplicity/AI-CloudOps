package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type readOnlyRoles struct{ service.RoleService }

func (readOnlyRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: []*model.Api{{Path: "/api/item/:id", Method: 1}}}}}, nil
}
func TestRegisteredReadPermissionDoesNotAllowWrites(t *testing.T) {
	for _, tt := range []struct {
		method string
		status int
	}{{"GET", 204}, {"POST", 403}} {
		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "reader"}); c.Next() })
		engine.Use(NewAuthMiddleware(readOnlyRoles{}).CheckAuth())
		handler := func(c *gin.Context) { c.Status(204) }
		engine.GET("/api/item/:id", handler)
		engine.POST("/api/item/:id", handler)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(tt.method, "/api/item/42", nil))
		if rec.Code != tt.status {
			t.Fatalf("%s returned %d", tt.method, rec.Code)
		}
	}
}

type aiProxyRoles struct {
	service.RoleService
	apis []*model.Api
}

func (r aiProxyRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: r.apis}}}, nil
}
func TestAssistantProxyPermissionsDoNotGrantInfrastructureActions(t *testing.T) {
	// 代理入口只有通配模板，鉴权依赖启动时登记的 AIOps 接口目录（真实目录见 internal/app 的测试）
	useProxyCatalog(t, []permission.Route{
		{Method: "POST", Path: "/api/v1/assistant/query"},
		{Method: "GET", Path: "/api/v1/assistant/session/:session_id"},
		{Method: "POST", Path: "/api/v1/autofix/workflow/confirm"},
		{Method: "POST", Path: "/api/v1/autofix/notify"},
	})
	grants := []*model.Api{{Path: "/api/v1/assistant/query", Method: 2}, {Path: "/api/v1/assistant/session/:session_id", Method: 1}}
	for _, test := range []struct {
		method, path string
		want         int
	}{
		{"POST", "/api/v1/assistant/query", 204},
		{"GET", "/api/v1/assistant/session/owned-sid", 204},
		{"GET", "/api/v1/assistant/session/owned-sid/extra", 403},
		{"GET", "/api/v1/assistant/session/", 403},
		{"GET", "/api/v1/assistant/session/..", 403},
		{"GET", "/api/v1/assistant/sessionx/owned-sid", 403},
		{"GET", "/api/v1/assistant/query", 403},
		{"POST", "/api/v1/autofix/workflow/confirm", 403},
		{"POST", "/api/v1/autofix/notify", 403},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "assistant-reader"}); c.Next() })
			engine.Use(NewAuthMiddleware(aiProxyRoles{apis: grants}).CheckAuth())
			engine.Any("/api/v1/*path", func(c *gin.Context) { c.Status(204) })
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != test.want {
				t.Fatalf("status=%d want=%d", recorder.Code, test.want)
			}
		})
	}
}

func useProxyCatalog(t *testing.T, routes []permission.Route) {
	t.Helper()
	previous := permission.ProxyCatalog(permission.AIOpsProxyTemplate)
	permission.SetProxyCatalog(permission.AIOpsProxyTemplate, routes)
	t.Cleanup(func() { permission.SetProxyCatalog(permission.AIOpsProxyTemplate, previous) })
}

func TestLegacyModuleGrantRemainsExplicitlyBroad(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "operator"}); c.Next() })
	engine.Use(NewAuthMiddleware(aiProxyRoles{apis: []*model.Api{{Path: "/api/v1/*path", Method: 2}}}).CheckAuth())
	engine.Any("/api/v1/*path", func(c *gin.Context) { c.Status(204) })
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("POST", "/api/v1/autofix/workflow/confirm", nil))
	if recorder.Code != 204 {
		t.Fatalf("既有显式整模块授权不应被静默改写: status=%d", recorder.Code)
	}
}
