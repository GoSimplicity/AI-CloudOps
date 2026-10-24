package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func TestSignupRequiresAuthenticatedAdministrator(t *testing.T) {
	superRole := &model.Role{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 1}
	signupGrant := &model.Role{Status: 1, Apis: []*model.Api{{Path: "/api/user/signup", Method: 2}, {Path: "/*", Method: 2}}}
	for _, tt := range []struct {
		name   string
		claims *ijwt.UserClaims
		roles  []*model.Role
		status int
	}{
		{"anonymous", nil, nil, 401},
		{"ordinary account", &ijwt.UserClaims{Uid: 2, Username: "member", AccountType: 1}, nil, 403},
		{"service account", &ijwt.UserClaims{Uid: 3, Username: "service", AccountType: 2}, nil, 403},
		// 角色里显式授予 signup 也不够：建号只属于超管
		{"granted but not super admin", &ijwt.UserClaims{Uid: 4, Username: "operator"}, []*model.Role{signupGrant}, 403},
		// 用户名叫 admin 但没有超管角色：不再有任何特权
		{"username admin without role", &ijwt.UserClaims{Uid: 1, Username: "admin"}, nil, 403},
		{"super admin role", &ijwt.UserClaims{Uid: 5, Username: "alice"}, []*model.Role{superRole}, 204},
		{"administrator", &ijwt.UserClaims{Uid: 1, Username: "admin"}, []*model.Role{superRole}, 204},
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
			engine.Use(NewAuthMiddleware(fixedRoles{roles: tt.roles}).CheckAuth())
			engine.POST("/api/user/signup", func(c *gin.Context) { c.Status(204) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("POST", "/api/user/signup", nil))
			if rec.Code != tt.status {
				t.Fatalf("got %d want %d", rec.Code, tt.status)
			}
		})
	}
}
