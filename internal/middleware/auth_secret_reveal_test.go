package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func TestSecretRevealRequiresExplicitGrant(t *testing.T) {
	const detailRoute = "/api/k8s/secret/:cluster_id/:namespace/:name/detail"
	for _, tt := range []struct {
		name   string
		grants []*model.Api
		path   string
		want   int
	}{
		{"explicit reveal grant", []*model.Api{{Path: model.K8sSecretRevealRoute, Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 204},
		{"module wildcard does not cover reveal", []*model.Api{{Path: "/api/k8s/*", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"global wildcard does not cover reveal", []*model.Api{{Path: "/*", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"detail grant does not cover reveal", []*model.Api{{Path: detailRoute, Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"module wildcard still covers detail", []*model.Api{{Path: "/api/k8s/*", Method: 1}}, "/api/k8s/secret/1/ops/app/detail", 204},
		// 参数化路径（不含 *）在 matchWildcardPath 中按段匹配，但敏感路由只认与路由模板逐字相等的授权
		{"parameterized action path does not cover reveal", []*model.Api{{Path: "/api/k8s/secret/:cluster_id/:namespace/:name/:action", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"renamed params do not cover reveal", []*model.Api{{Path: "/api/k8s/secret/:a/:b/:c/reveal", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"fully parameterized path does not cover reveal", []*model.Api{{Path: "/:a/:b/:c/:d/:e/:f/:g", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"concrete request path does not cover reveal", []*model.Api{{Path: "/api/k8s/secret/1/ops/app/reveal", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"middle wildcard does not cover reveal", []*model.Api{{Path: "/api/k8s/*/reveal", Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"reveal route with wrong method does not cover reveal", []*model.Api{{Path: model.K8sSecretRevealRoute, Method: 2}}, "/api/k8s/secret/1/ops/app/reveal", 403},
		{"exact grant among wildcard grants", []*model.Api{{Path: "/*", Method: 1}, {Path: model.K8sSecretRevealRoute, Method: 1}}, "/api/k8s/secret/1/ops/app/reveal", 204},
		{"parameterized grant still covers detail", []*model.Api{{Path: "/api/k8s/secret/:a/:b/:c/detail", Method: 1}}, "/api/k8s/secret/1/ops/app/detail", 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "operator"}); c.Next() })
			engine.Use(NewAuthMiddleware(aiProxyRoles{apis: tt.grants}).CheckAuth())
			engine.GET(model.K8sSecretRevealRoute, func(c *gin.Context) { c.Status(204) })
			engine.GET(detailRoute, func(c *gin.Context) { c.Status(204) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("GET", tt.path, nil))
			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d", rec.Code, tt.want)
			}
		})
	}
}

func TestSecretRevealAdminBypass(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 1, Username: "admin"}); c.Next() })
	engine.Use(NewAuthMiddleware(aiProxyRoles{}).CheckAuth())
	engine.GET(model.K8sSecretRevealRoute, func(c *gin.Context) { c.Status(204) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/k8s/secret/1/ops/app/reveal", nil))
	if rec.Code != 204 {
		t.Fatalf("admin 应直接放行, status=%d", rec.Code)
	}
}
