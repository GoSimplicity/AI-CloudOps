package middleware

import (
	"net/http/httptest"
	"testing"

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func TestSignupRequiresAuthenticatedAdministrator(t *testing.T) {
	for _, tt := range []struct {
		name   string
		claims *ijwt.UserClaims
		status int
	}{
		{"anonymous", nil, 401},
		{"ordinary account", &ijwt.UserClaims{Username: "member", AccountType: 1}, 403},
		{"service account", &ijwt.UserClaims{Username: "service", AccountType: 2}, 403},
		{"administrator", &ijwt.UserClaims{Username: "admin"}, 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			if tt.claims == nil {
				jwtHandler, err := ijwt.NewJWTHandler(nil, ijwt.Config{Issuer: "test-issuer"})
				if err != nil {
					t.Fatal(err)
				}
				engine.Use(NewJWTMiddleware(jwtHandler, nil).CheckLogin())
			} else {
				engine.Use(func(c *gin.Context) { c.Set("user", *tt.claims); c.Next() })
			}
			engine.Use(NewAuthMiddleware(nil).CheckAuth())
			engine.POST("/api/user/signup", func(c *gin.Context) { c.Status(204) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("POST", "/api/user/signup", nil))
			if rec.Code != tt.status {
				t.Fatalf("got %d want %d", rec.Code, tt.status)
			}
		})
	}
}
