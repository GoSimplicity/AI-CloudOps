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
