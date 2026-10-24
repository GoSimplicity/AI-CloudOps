package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openRBACTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rbac.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.User{}, &model.Api{}, &model.Role{}, &model.RoleApi{}, &model.UserRole{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func superRoles(t *testing.T, db *gorm.DB) []model.Role {
	t.Helper()
	var roles []model.Role
	if err := db.Where("code = ? AND is_system = 1", permission.SuperAdminRoleCode).Find(&roles).Error; err != nil {
		t.Fatal(err)
	}
	return roles
}

func isSuper(t *testing.T, db *gorm.DB, userID int) bool {
	t.Helper()
	roles, err := dao.NewRoleDAO(db).GetRoles(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	return permission.HasSuperAdminRole(roles)
}

func TestEnsureSuperAdminIsIdempotentAndBindsOnlyAdmin(t *testing.T) {
	db := openRBACTestDB(t)
	admin := model.User{Username: "admin", Enable: 1}
	lookalike := model.User{Username: "Admin", Enable: 1}
	member := model.User{Username: "member", Enable: 1}
	for _, u := range []*model.User{&admin, &lookalike, &member} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	// 历史上手工建的同编码普通角色，不能被提升
	custom := model.Role{Name: "自建超管", Code: permission.SuperAdminRoleCode, Status: 1}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: member.ID, RoleID: custom.ID}).Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if err := ensureSuperAdmin(context.Background(), db, zap.NewNop()); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}

	roles := superRoles(t, db)
	if len(roles) != 1 || roles[0].Status != 1 || roles[0].Name != permission.SuperAdminRoleName {
		t.Fatalf("unexpected system roles: %+v", roles)
	}
	var custom2 model.Role
	if err := db.First(&custom2, custom.ID).Error; err != nil || custom2.IsSystem != 0 {
		t.Fatalf("custom role promoted: %+v err=%v", custom2, err)
	}
	var bindings []model.UserRole
	if err := db.Where("role_id = ?", roles[0].ID).Find(&bindings).Error; err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 1 || bindings[0].UserID != admin.ID {
		t.Fatalf("bindings=%+v want only admin(%d)", bindings, admin.ID)
	}
	if !isSuper(t, db, admin.ID) || isSuper(t, db, lookalike.ID) || isSuper(t, db, member.ID) {
		t.Fatal("super admin identity resolved incorrectly")
	}
}

func TestEnsureSuperAdminRebindsAndDoesNotReenable(t *testing.T) {
	db := openRBACTestDB(t)
	if err := ensureSuperAdmin(context.Background(), db, nil); err != nil {
		t.Fatal(err)
	}
	if n := len(superRoles(t, db)); n != 1 {
		t.Fatalf("empty install created %d roles", n)
	}
	// admin 在角色之后才创建（例如 Mock），下次启动应补绑
	admin := model.User{Username: "admin", Enable: 1}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureSuperAdmin(context.Background(), db, nil); err != nil {
		t.Fatal(err)
	}
	if !isSuper(t, db, admin.ID) {
		t.Fatal("admin not bound on second boot")
	}
	// 人为撤销后重启会重新绑定
	if err := db.Where("user_id = ?", admin.ID).Delete(&model.UserRole{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureSuperAdmin(context.Background(), db, nil); err != nil {
		t.Fatal(err)
	}
	if !isSuper(t, db, admin.ID) {
		t.Fatal("admin not re-bound after revoke")
	}
	// 被直接改库禁用的超管角色不会被自动启用
	if err := db.Model(&model.Role{}).Where("code = ?", permission.SuperAdminRoleCode).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureSuperAdmin(context.Background(), db, nil); err != nil {
		t.Fatal(err)
	}
	if roles := superRoles(t, db); len(roles) != 1 || roles[0].Status != 0 {
		t.Fatalf("disabled role changed: %+v", roles)
	}
	if isSuper(t, db, admin.ID) {
		t.Fatal("disabled super role still grants super admin")
	}
}

// 端到端：真实 RoleDAO/RoleService + 真实鉴权中间件，admin 经引导后仍能使用
// 仅超管可用的接口，而同名但未绑定角色的账号不能。
func TestBootstrappedAdminKeepsAccessThroughRealMiddleware(t *testing.T) {
	db := openRBACTestDB(t)
	admin := model.User{Username: "admin", Enable: 1}
	member := model.User{Username: "member", Enable: 1}
	for _, u := range []*model.User{&admin, &member} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := ensureSuperAdmin(context.Background(), db, nil); err != nil {
		t.Fatal(err)
	}
	roles := service.NewRoleService(dao.NewRoleDAO(db), zap.NewNop())
	for _, tt := range []struct {
		claims jwt.UserClaims
		want   int
	}{
		{jwt.UserClaims{Uid: admin.ID, Username: "admin"}, 204},
		{jwt.UserClaims{Uid: member.ID, Username: "member"}, 403},
		// 伪造用户名没有用：特权跟着 uid 的角色走
		{jwt.UserClaims{Uid: member.ID, Username: "admin"}, 403},
	} {
		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set("user", tt.claims); c.Next() })
		engine.Use(middleware.NewAuthMiddleware(roles).CheckAuth())
		engine.POST("/api/user/signup", func(c *gin.Context) { c.Status(204) })
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest("POST", "/api/user/signup", nil))
		if rec.Code != tt.want {
			t.Fatalf("%+v: status=%d want=%d", tt.claims, rec.Code, tt.want)
		}
	}
}
