package permission

import (
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
)

func TestSuperAdminIdentityRequiresSystemRoleAndStatus(t *testing.T) {
	for _, tt := range []struct {
		name  string
		role  *model.Role
		super bool
	}{
		{"nil", nil, false},
		{"system enabled", &model.Role{Code: SuperAdminRoleCode, IsSystem: 1, Status: 1}, true},
		{"system disabled", &model.Role{Code: SuperAdminRoleCode, IsSystem: 1, Status: 0}, false},
		{"custom role reusing code", &model.Role{Code: SuperAdminRoleCode, IsSystem: 0, Status: 1}, false},
		{"other system role", &model.Role{Code: "auditor", IsSystem: 1, Status: 1}, false},
		{"name only", &model.Role{Name: SuperAdminRoleName, Code: "x", IsSystem: 1, Status: 1}, false},
	} {
		if got := IsSuperAdminRole(tt.role); got != tt.super {
			t.Fatalf("%s: got %v want %v", tt.name, got, tt.super)
		}
		if got := HasSuperAdminRole([]*model.Role{{Code: "member", Status: 1}, tt.role}); got != tt.super {
			t.Fatalf("%s (set): got %v want %v", tt.name, got, tt.super)
		}
	}
}

func TestSuperAdminContextFailsClosed(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if IsSuperAdmin(c) || IsSuperAdmin(nil) {
		t.Fatal("missing mark must not be super admin")
	}
	c.Set(superAdminContextKey, "true")
	if IsSuperAdmin(c) {
		t.Fatal("non-bool mark must not be super admin")
	}
	MarkSuperAdmin(c, true)
	if !IsSuperAdmin(c) {
		t.Fatal("mark not read back")
	}
	MarkSuperAdmin(c, false)
	if IsSuperAdmin(c) {
		t.Fatal("cleared mark still super admin")
	}
}
