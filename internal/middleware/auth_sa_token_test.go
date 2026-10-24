package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// SEC-29：GET（token Secret）与 POST（TokenRequest）/token 只接受按路由模板 + 方法的显式授权
func TestServiceAccountTokenRequiresExplicitGrant(t *testing.T) {
	const (
		detailRoute = "/api/k8s/serviceaccount/:cluster_id/:namespace/:name/detail"
		tokenPath   = "/api/k8s/serviceaccount/1/ops/app/token"
		detailPath  = "/api/k8s/serviceaccount/1/ops/app/detail"
	)
	for _, tt := range []struct {
		name   string
		grants []*model.Api
		method string
		path   string
		want   int
	}{
		{"explicit GET grant", []*model.Api{{Path: model.K8sServiceAccountTokenRoute, Method: 1}}, "GET", tokenPath, 204},
		{"explicit POST grant", []*model.Api{{Path: model.K8sServiceAccountTokenRoute, Method: 2}}, "POST", tokenPath, 204},
		{"GET grant does not cover POST TokenRequest", []*model.Api{{Path: model.K8sServiceAccountTokenRoute, Method: 1}}, "POST", tokenPath, 403},
		{"POST grant does not cover GET", []*model.Api{{Path: model.K8sServiceAccountTokenRoute, Method: 2}}, "GET", tokenPath, 403},
		{"module wildcard does not cover GET token", []*model.Api{{Path: "/api/k8s/*", Method: 1}}, "GET", tokenPath, 403},
		{"module wildcard does not cover POST token", []*model.Api{{Path: "/api/k8s/*", Method: 2}}, "POST", tokenPath, 403},
		{"global wildcard does not cover GET token", []*model.Api{{Path: "/*", Method: 1}}, "GET", tokenPath, 403},
		{"global wildcard does not cover POST token", []*model.Api{{Path: "/*", Method: 2}}, "POST", tokenPath, 403},
		{"parameterized action path does not cover token", []*model.Api{{Path: "/api/k8s/serviceaccount/:cluster_id/:namespace/:name/:action", Method: 1}}, "GET", tokenPath, 403},
		{"renamed params do not cover token", []*model.Api{{Path: "/api/k8s/serviceaccount/:a/:b/:c/token", Method: 2}}, "POST", tokenPath, 403},
		{"fully parameterized path does not cover token", []*model.Api{{Path: "/:a/:b/:c/:d/:e/:f/:g", Method: 1}}, "GET", tokenPath, 403},
		{"concrete request path does not cover token", []*model.Api{{Path: tokenPath, Method: 1}}, "GET", tokenPath, 403},
		{"middle wildcard does not cover token", []*model.Api{{Path: "/api/k8s/*/token", Method: 1}}, "GET", tokenPath, 403},
		{"detail grant does not cover token", []*model.Api{{Path: detailRoute, Method: 1}}, "GET", tokenPath, 403},
		{"exact grant among wildcard grants", []*model.Api{{Path: "/api/k8s/*", Method: 1}, {Path: model.K8sServiceAccountTokenRoute, Method: 1}}, "GET", tokenPath, 204},
		{"module wildcard still covers detail", []*model.Api{{Path: "/api/k8s/*", Method: 1}}, "GET", detailPath, 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "operator"}); c.Next() })
			engine.Use(NewAuthMiddleware(aiProxyRoles{apis: tt.grants}).CheckAuth())
			ok := func(c *gin.Context) { c.Status(204) }
			engine.GET(model.K8sServiceAccountTokenRoute, ok)
			engine.POST(model.K8sServiceAccountTokenRoute, ok)
			engine.GET(detailRoute, ok)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
		})
	}
}

func TestServiceAccountTokenAdminBypass(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 1, Username: "admin"}); c.Next() })
		engine.Use(NewAuthMiddleware(fixedRoles{roles: []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 1}}}).CheckAuth())
		engine.Handle(method, model.K8sServiceAccountTokenRoute, func(c *gin.Context) { c.Status(204) })
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(method, "/api/k8s/serviceaccount/1/ops/app/token", nil))
		if rec.Code != 204 {
			t.Fatalf("%s admin 应直接放行, status=%d", method, rec.Code)
		}
	}
}
