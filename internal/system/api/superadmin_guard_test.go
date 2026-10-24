package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	superRoleID  = 1
	memberRoleID = 2
	fakeRoleID   = 3 // 普通角色借用 super_admin 编码
)

type guardRoles struct {
	service.RoleService
	targetRoles []*model.Role
	assigned    *[]int
	revoked     *[]int
}

func (guardRoles) GetRoleByID(_ context.Context, id int) (*model.Role, error) {
	switch id {
	case superRoleID:
		return &model.Role{Model: model.Model{ID: id}, Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 0}, nil
	case memberRoleID:
		return &model.Role{Model: model.Model{ID: id}, Code: "member", Status: 1}, nil
	case fakeRoleID:
		return &model.Role{Model: model.Model{ID: id}, Code: permission.SuperAdminRoleCode, IsSystem: 0, Status: 1}, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r guardRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: r.targetRoles}, nil
}
func (r guardRoles) AssignRolesToUser(_ context.Context, _ int, roleIds []int, _ int) error {
	*r.assigned = append(*r.assigned, roleIds...)
	return nil
}
func (r guardRoles) RevokeRolesFromUser(_ context.Context, _ int, roleIds []int) error {
	*r.revoked = append(*r.revoked, -1)
	return nil
}

func TestOnlySuperAdminCanGrantOrRevokeSuperAdminRole(t *testing.T) {
	superTarget := []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 1}}
	for _, tt := range []struct {
		name        string
		path        string
		roleIds     []int
		targetRoles []*model.Role
		callerSuper bool
		allowed     bool
	}{
		{"assign member role", "/api/role/assign_users", []int{memberRoleID}, nil, false, true},
		{"assign custom role reusing code", "/api/role/assign_users", []int{fakeRoleID}, nil, false, true},
		// 即使超管角色被禁用，授予它也必须由超管完成，否则被重新启用时会立刻提权
		{"assign super role by ordinary", "/api/role/assign_users", []int{memberRoleID, superRoleID}, nil, false, false},
		{"assign super role by super", "/api/role/assign_users", []int{superRoleID}, nil, true, true},
		{"assign unknown role", "/api/role/assign_users", []int{99}, nil, false, true},
		{"revoke super role by ordinary", "/api/role/revoke_users", []int{superRoleID}, nil, false, false},
		{"revoke all roles of super admin by ordinary", "/api/role/revoke_users", []int{}, superTarget, false, false},
		{"revoke all roles of member by ordinary", "/api/role/revoke_users", []int{}, nil, false, true},
		{"revoke super role by super", "/api/role/revoke_users", []int{superRoleID}, nil, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var assigned, revoked []int
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("user", jwt.UserClaims{Uid: 7, Username: "operator"})
				permission.MarkSuperAdmin(c, tt.callerSuper)
				c.Next()
			})
			NewRoleHandler(guardRoles{targetRoles: tt.targetRoles, assigned: &assigned, revoked: &revoked}).RegisterRouters(router)
			body, _ := json.Marshal(map[string]any{"user_id": 8, "role_ids": tt.roleIds})
			req := httptest.NewRequest("POST", tt.path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			called := len(assigned)+len(revoked) > 0
			if called != tt.allowed {
				t.Fatalf("service called=%v want %v (status=%d body=%s)", called, tt.allowed, rec.Code, rec.Body.String())
			}
			if !tt.allowed && rec.Code != 403 {
				t.Fatalf("status=%d want 403", rec.Code)
			}
		})
	}
}

type profileRecorder struct {
	service.UserService
	called       bool
	allowAccount bool
}

func (p *profileRecorder) UpdateProfile(_ context.Context, _ *model.UpdateProfileReq, allow bool) error {
	p.called, p.allowAccount = true, allow
	return nil
}

func TestUpdateProfilePrivilegeComesFromSuperAdminRole(t *testing.T) {
	for _, tt := range []struct {
		name         string
		username     string
		super        bool
		target       int
		allowed      bool
		allowAccount bool
	}{
		{"self edit", "member", false, 7, true, false},
		{"other user by ordinary", "member", false, 8, false, false},
		{"other user by username admin without role", "admin", false, 8, false, false},
		{"self edit by username admin without role", "admin", false, 7, true, false},
		{"other user by super admin", "ops-lead", true, 8, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := &profileRecorder{}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("user", jwt.UserClaims{Uid: 7, Username: tt.username})
				permission.MarkSuperAdmin(c, tt.super)
				c.Next()
			})
			NewUserHandler(svc, nil).RegisterRoutes(router)
			body, _ := json.Marshal(map[string]any{"real_name": "n", "mobile": "1", "account_type": 1, "home_path": "/", "enable": 2})
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/user/profile/update/%d", tt.target), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if svc.called != tt.allowed {
				t.Fatalf("called=%v want %v (status=%d)", svc.called, tt.allowed, rec.Code)
			}
			if !tt.allowed && rec.Code != 403 {
				t.Fatalf("status=%d want 403", rec.Code)
			}
			if svc.allowAccount != tt.allowAccount {
				t.Fatalf("allowAccountChanges=%v want %v", svc.allowAccount, tt.allowAccount)
			}
		})
	}
}
