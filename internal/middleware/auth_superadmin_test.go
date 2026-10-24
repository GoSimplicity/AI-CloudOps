package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// fixedRoles 返回固定角色集合的 RoleService 假实现；err 非空时模拟数据库故障。
type fixedRoles struct {
	service.RoleService
	roles []*model.Role
	err   error
}

func (r fixedRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	if r.err != nil {
		return nil, r.err
	}
	return &model.ListResp[*model.Role]{Items: r.roles}, nil
}

func TestPrivilegeComesFromSuperAdminRoleNotUsername(t *testing.T) {
	superRole := &model.Role{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 1}
	for _, tt := range []struct {
		name   string
		claims jwt.UserClaims
		roles  []*model.Role
		status int
		super  bool
	}{
		{"admin username alone is ordinary", jwt.UserClaims{Uid: 1, Username: "admin"}, nil, 403, false},
		{"admin with super role", jwt.UserClaims{Uid: 1, Username: "admin"}, []*model.Role{superRole}, 204, true},
		{"renamed account with super role", jwt.UserClaims{Uid: 9, Username: "ops-lead"}, []*model.Role{superRole}, 204, true},
		{"custom role reusing super code", jwt.UserClaims{Uid: 9, Username: "ops"}, []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 0, Status: 1}}, 403, false},
		{"disabled super role", jwt.UserClaims{Uid: 1, Username: "admin"}, []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 0}}, 403, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", tt.claims); c.Next() })
			engine.Use(NewAuthMiddleware(fixedRoles{roles: tt.roles}).CheckAuth())
			var sawSuper bool
			engine.DELETE("/api/user/:id/delete", func(c *gin.Context) { sawSuper = permission.IsSuperAdmin(c); c.Status(204) })
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/user/3/delete", nil))
			if rec.Code != tt.status {
				t.Fatalf("status=%d want=%d", rec.Code, tt.status)
			}
			if tt.status == 204 && sawSuper != tt.super {
				t.Fatalf("super mark=%v want=%v", sawSuper, tt.super)
			}
		})
	}
}

func TestOrdinaryGrantMarksRequestAsNotSuperAdmin(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "reader"}); c.Next() })
	engine.Use(NewAuthMiddleware(fixedRoles{roles: []*model.Role{{Status: 1, Apis: []*model.Api{{Path: "/api/item/:id", Method: 1}}}}}).CheckAuth())
	sawSuper := true
	engine.GET("/api/item/:id", func(c *gin.Context) { sawSuper = permission.IsSuperAdmin(c); c.Status(204) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/item/1", nil))
	if rec.Code != 204 || sawSuper {
		t.Fatalf("status=%d super=%v", rec.Code, sawSuper)
	}
}

func TestRoleLookupFailureFailsClosed(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 1, Username: "admin"}); c.Next() })
	engine.Use(NewAuthMiddleware(fixedRoles{err: errors.New("db down")}).CheckAuth())
	called := false
	engine.GET("/api/item/:id", func(c *gin.Context) { called = true; c.Status(204) })
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/item/1", nil))
	if called {
		t.Fatalf("handler reached despite role lookup failure (status=%d)", rec.Code)
	}
}
