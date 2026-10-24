package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type grantRoles struct {
	service.RoleService
	apis []*model.Api
}

func (g grantRoles) GetUserRoles(_ context.Context, _ int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: g.apis}}, Total: 1}, nil
}

func TestEmbeddedAIOpsCatalogResolvesEveryEntryToItself(t *testing.T) {
	routes, err := parseAIOpsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) == 0 {
		t.Fatal("empty AIOps catalog")
	}
	permission.SetProxyCatalog(permission.AIOpsProxyTemplate, routes)
	t.Cleanup(func() { permission.SetProxyCatalog(permission.AIOpsProxyTemplate, nil) })
	for _, route := range routes {
		concrete := route.Path
		for _, part := range strings.Split(route.Path, "/") {
			if strings.HasPrefix(part, ":") {
				concrete = strings.Replace(concrete, part, "sample-"+part[1:], 1)
			}
		}
		if got := permission.ResolveProxyTemplate(route.Method, permission.AIOpsProxyTemplate, concrete); got != route.Path {
			t.Errorf("%s %s resolved to %q", route.Method, concrete, got)
		}
	}
}

func TestRegisterAIOpsInstallsAuthorizationCatalog(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()
	t.Cleanup(func() { permission.SetProxyCatalog(permission.AIOpsProxyTemplate, nil) })

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
	engine.Use(middleware.NewAuthMiddleware(grantRoles{apis: []*model.Api{
		{Method: 1, Path: "/api/v1/assistant/session/:session_id"},
		{Method: 1, Path: "/api/v1/inspection/report/:report_id"},
	}}).CheckAuth())
	a := &App{cfg: &config.Config{AIOps: config.AIOpsConfig{BaseURL: upstream.URL, APIKey: "test-only"}}, server: engine}
	if err := a.registerAIOps(); err != nil {
		t.Fatal(err)
	}
	// 用真实 HTTP 服务承载网关：ReverseProxy 需要 CloseNotifier，ResponseRecorder 不提供
	gateway := httptest.NewServer(engine)
	defer gateway.Close()
	for _, tt := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/v1/assistant/session/s-1", 200},
		{"GET", "/api/v1/inspection/report/r-1", 200},
		{"GET", "/api/v1/inspection/report/r-1/markdown", 403},
		{"DELETE", "/api/v1/assistant/session/s-1", 403},
		{"GET", "/api/v1/assistant/session/s-1/extra", 403},
		{"POST", "/api/v1/autofix", 403},
	} {
		req, err := http.NewRequest(tt.method, gateway.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := gateway.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tt.want {
			t.Errorf("%s %s: status=%d want=%d", tt.method, tt.path, resp.StatusCode, tt.want)
		}
	}
}

func TestRegisterAIOpsWithoutBaseURLLeavesNoCatalog(t *testing.T) {
	permission.SetProxyCatalog(permission.AIOpsProxyTemplate, nil)
	a := &App{cfg: &config.Config{}, server: gin.New()}
	if err := a.registerAIOps(); err != nil {
		t.Fatal(err)
	}
	if got := permission.ProxyCatalog(permission.AIOpsProxyTemplate); len(got) != 0 {
		t.Fatalf("catalog installed without proxy: %d entries", len(got))
	}
}
