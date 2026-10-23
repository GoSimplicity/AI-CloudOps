package middleware

import (
	"net/http/httptest"
	"testing"

	k8sapi "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	treeapi "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// productionRouter 用真实的 handler RegisterRouters 注册路由（与线上同一份路由表），
// 在鉴权之后用一个终止中间件代替业务 handler：放行返回 204 并记录命中的模板，
// 拒绝则由鉴权中间件返回 403。
func productionRouter(t *testing.T, grants []*model.Api) (*gin.Engine, *string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	matched := new(string)
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 42, Username: "operator"}); c.Next() })
	engine.Use(NewAuthMiddleware(fixedRoles{roles: []*model.Role{{Status: 1, Apis: grants}}}).CheckAuth())
	engine.Use(func(c *gin.Context) { *matched = c.FullPath(); c.AbortWithStatus(204) })
	k8sapi.NewK8sEventHandler(nil).RegisterRouters(engine)
	treeapi.NewCloudAccountHandler(nil).RegisterRouters(engine)
	treeapi.NewTreeCloudHandler(nil, nil, nil, nil).RegisterRouters(engine)
	return engine, matched
}

func TestParamGrantsDoNotLeakToSiblingStaticRoutes(t *testing.T) {
	for _, tt := range []struct {
		name        string
		grant       model.Api
		method      string
		allowed     string // 授予本身覆盖的请求
		allowedTmpl string
		leaked      string // 改动前会被授予外溢放行的请求
		leakedTmpl  string
	}{
		{
			name:        "k8s event detail vs node events (event.go:49 vs :54)",
			grant:       model.Api{Method: 1, Path: "/api/k8s/clusters/:cluster_id/events/:namespace/:name"},
			method:      "GET",
			allowed:     "/api/k8s/clusters/1/events/default/pod-oom.17a",
			allowedTmpl: "/api/k8s/clusters/:cluster_id/events/:namespace/:name",
			leaked:      "/api/k8s/clusters/1/events/nodes/worker-1",
			leakedTmpl:  "/api/k8s/clusters/:cluster_id/events/nodes/:node_name",
		},
		{
			name:        "cloud account status vs batch status (cloud_account_handler.go:54 vs :57)",
			grant:       model.Api{Method: 3, Path: "/api/tree/cloud/account/:id/status"},
			method:      "PUT",
			allowed:     "/api/tree/cloud/account/7/status",
			allowedTmpl: "/api/tree/cloud/account/:id/status",
			leaked:      "/api/tree/cloud/account/batch/status",
			leakedTmpl:  "/api/tree/cloud/account/batch/status",
		},
		{
			name:        "cloud resource status vs batch status (tree_cloud_handler.go:69 vs :74)",
			grant:       model.Api{Method: 3, Path: "/api/tree/cloud/:id/status"},
			method:      "PUT",
			allowed:     "/api/tree/cloud/9/status",
			allowedTmpl: "/api/tree/cloud/:id/status",
			leaked:      "/api/tree/cloud/batch/status",
			leakedTmpl:  "/api/tree/cloud/batch/status",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			grant := tt.grant
			engine, matched := productionRouter(t, []*model.Api{&grant})

			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.allowed, nil))
			if rec.Code != 204 || *matched != tt.allowedTmpl {
				t.Fatalf("granted request: status=%d template=%q", rec.Code, *matched)
			}

			// 确认泄露请求确实命中的是兄弟静态路由，而不是 404
			*matched = ""
			rec = httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.leaked, nil))
			if rec.Code != 403 {
				t.Fatalf("sibling route %s leaked: status=%d", tt.leakedTmpl, rec.Code)
			}
			if !routeRegistered(engine, tt.method, tt.leakedTmpl) {
				t.Fatalf("test setup: %s %s is not registered", tt.method, tt.leakedTmpl)
			}

			// 显式授予兄弟路由后才放行，说明拒绝来自授权而非路由
			sibling := model.Api{Method: grant.Method, Path: tt.leakedTmpl}
			engine, matched = productionRouter(t, []*model.Api{&grant, &sibling})
			rec = httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.leaked, nil))
			if rec.Code != 204 || *matched != tt.leakedTmpl {
				t.Fatalf("explicit sibling grant: status=%d template=%q", rec.Code, *matched)
			}
		})
	}
}

func routeRegistered(engine *gin.Engine, method, path string) bool {
	for _, route := range engine.Routes() {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func TestWildcardGrantsMatchTemplatesOnly(t *testing.T) {
	for _, tt := range []struct {
		name   string
		grant  model.Api
		method string
		path   string
		want   int
	}{
		{"global wildcard", model.Api{Method: 1, Path: "/*"}, "GET", "/api/k8s/clusters/1/events/nodes/worker-1", 204},
		{"global wildcard keeps method", model.Api{Method: 1, Path: "/*"}, "PUT", "/api/tree/cloud/batch/status", 403},
		{"module prefix", model.Api{Method: 3, Path: "/api/tree/cloud/account/*"}, "PUT", "/api/tree/cloud/account/batch/status", 204},
		{"module prefix does not reach parent module", model.Api{Method: 3, Path: "/api/tree/cloud/account/*"}, "PUT", "/api/tree/cloud/batch/status", 403},
		// 前缀只作用于模板：真实路径里的值不能“拼”出被授予的前缀
		{"prefix on real id is not honoured", model.Api{Method: 3, Path: "/api/tree/cloud/9/*"}, "PUT", "/api/tree/cloud/9/status", 403},
		{"prefix written with template param", model.Api{Method: 3, Path: "/api/tree/cloud/:id/*"}, "PUT", "/api/tree/cloud/9/status", 204},
		{"prefix with param does not cover static sibling", model.Api{Method: 3, Path: "/api/tree/cloud/:id/*"}, "PUT", "/api/tree/cloud/batch/status", 403},
		{"exact real path is not a template", model.Api{Method: 3, Path: "/api/tree/cloud/9/status"}, "PUT", "/api/tree/cloud/9/status", 403},
		{"leading wildcard dropped", model.Api{Method: 3, Path: "*/status"}, "PUT", "/api/tree/cloud/9/status", 403},
		{"middle wildcard dropped", model.Api{Method: 3, Path: "/api/tree/*/status"}, "PUT", "/api/tree/cloud/:id/status", 403},
	} {
		t.Run(tt.name, func(t *testing.T) {
			grant := tt.grant
			engine, _ := productionRouter(t, []*model.Api{&grant})
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
		})
	}
}

func TestUnmatchedRouteIsForbiddenForOrdinaryUsersAndNotFoundForSuperAdmin(t *testing.T) {
	for _, tt := range []struct {
		name  string
		roles []*model.Role
		want  int
	}{
		{"global wildcard grant", []*model.Role{{Status: 1, Apis: []*model.Api{{Method: 1, Path: "/*"}}}}, 403},
		{"no roles", nil, 403},
		{"super admin", []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 1}}, 404},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 1}); c.Next() })
			engine.Use(NewAuthMiddleware(fixedRoles{roles: tt.roles}).CheckAuth())
			engine.GET("/api/item/:id", func(c *gin.Context) { c.Status(204) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/no-such-route", nil))
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
		})
	}
}

func TestProxyGrantsResolveToMostSpecificCatalogTemplate(t *testing.T) {
	useProxyCatalog(t, []permission.Route{
		{Method: "GET", Path: "/api/v1/inspection/report/:report_id"},
		{Method: "GET", Path: "/api/v1/inspection/report/latest"},
		{Method: "GET", Path: "/api/v1/inspection/report/:report_id/markdown"},
		{Method: "POST", Path: "/api/v1/autofix"},
		{Method: "POST", Path: "/api/v1/autofix/"},
	})
	for _, tt := range []struct {
		name   string
		grants []*model.Api
		method string
		path   string
		want   int
	}{
		{"param grant", []*model.Api{{Method: 1, Path: "/api/v1/inspection/report/:report_id"}}, "GET", "/api/v1/inspection/report/r-1", 204},
		{"param grant does not reach static sibling", []*model.Api{{Method: 1, Path: "/api/v1/inspection/report/:report_id"}}, "GET", "/api/v1/inspection/report/latest", 403},
		{"param grant does not reach deeper template", []*model.Api{{Method: 1, Path: "/api/v1/inspection/report/:report_id"}}, "GET", "/api/v1/inspection/report/r-1/markdown", 403},
		{"static grant", []*model.Api{{Method: 1, Path: "/api/v1/inspection/report/latest"}}, "GET", "/api/v1/inspection/report/latest", 204},
		{"prefix grant", []*model.Api{{Method: 1, Path: "/api/v1/inspection/*"}}, "GET", "/api/v1/inspection/report/r-1/markdown", 204},
		{"prefix grant ignores uncatalogued endpoints", []*model.Api{{Method: 1, Path: "/api/v1/inspection/*"}}, "GET", "/api/v1/inspection/internal/debug", 403},
		{"module grant covers uncatalogued endpoints", []*model.Api{{Method: 1, Path: "/api/v1/*path"}}, "GET", "/api/v1/inspection/internal/debug", 204},
		{"trailing slash alias is distinct", []*model.Api{{Method: 2, Path: "/api/v1/autofix"}}, "POST", "/api/v1/autofix/", 403},
		{"trailing slash alias granted", []*model.Api{{Method: 2, Path: "/api/v1/autofix/"}}, "POST", "/api/v1/autofix/", 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
			engine.Use(NewAuthMiddleware(fixedRoles{roles: []*model.Role{{Status: 1, Apis: tt.grants}}}).CheckAuth())
			engine.Any(permission.AIOpsProxyTemplate, func(c *gin.Context) { c.Status(204) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
		})
	}
}

func TestDisabledRoleGrantsNothing(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
	engine.Use(NewAuthMiddleware(fixedRoles{roles: []*model.Role{{Status: 0, Apis: []*model.Api{{Method: 1, Path: "/*"}}}}}).CheckAuth())
	engine.GET("/api/item/:id", func(c *gin.Context) { c.Status(204) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/item/1", nil))
	if rec.Code != 403 {
		t.Fatalf("status=%d", rec.Code)
	}
}
