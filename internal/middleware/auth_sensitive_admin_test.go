package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func TestAdminUsernameDoesNotBypassSensitiveGrants(t *testing.T) {
	for _, endpoint := range []struct {
		method, route, path string
	}{
		{"GET", model.K8sSecretRevealRoute, "/api/k8s/secret/1/ops/app/reveal"},
		{"GET", model.K8sServiceAccountTokenRoute, "/api/k8s/serviceaccount/1/ops/app/token"},
		{"POST", model.K8sServiceAccountTokenRoute, "/api/k8s/serviceaccount/1/ops/app/token"},
	} {
		for _, roles := range []struct {
			name  string
			items []*model.Role
		}{
			{name: "no roles"},
			{name: "wildcard grants", items: []*model.Role{{Status: 1, Apis: []*model.Api{{Method: 1, Path: "/*"}, {Method: 2, Path: "/*"}}}}},
		} {
			t.Run(endpoint.method+" "+endpoint.route+"/"+roles.name, func(t *testing.T) {
				engine := gin.New()
				engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 1, Username: "admin"}); c.Next() })
				engine.Use(NewAuthMiddleware(fixedRoles{roles: roles.items}).CheckAuth())
				engine.Handle(endpoint.method, endpoint.route, func(c *gin.Context) { c.Status(204) })
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, httptest.NewRequest(endpoint.method, endpoint.path, nil))
				if rec.Code != 403 {
					t.Fatalf("admin without a superadmin role reached sensitive endpoint: status=%d", rec.Code)
				}
			})
		}
	}
}
