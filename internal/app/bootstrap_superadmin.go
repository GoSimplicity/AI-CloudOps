package app

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// bootstrapSuperAdmin 幂等地确保内置超管角色存在，并把内置 admin 账号绑定到该角色。
// 运行期鉴权只认角色，不认用户名，因此这一步是 admin 不被锁在门外的前提。
func (a *App) bootstrapSuperAdmin(ctx context.Context) error {
	return ensureSuperAdmin(ctx, a.db, a.logger)
}

func ensureSuperAdmin(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var roles []model.Role
		if err := tx.Where("code = ?", permission.SuperAdminRoleCode).Order("id").Find(&roles).Error; err != nil {
			return databaseLogger.WrapDatabaseError("初始化超级管理员角色: 查询角色失败", err)
		}
		var role *model.Role
		for i := range roles {
			if permission.IsSuperAdminIdentity(&roles[i]) {
				if role == nil {
					role = &roles[i]
				}
				continue
			}
			// 普通角色借用了超管编码：不提升它，只提示人工清理
			logger.Warn("存在使用超管编码的非系统角色，该角色不具备超管权限",
				zap.Int("role_id", roles[i].ID), zap.String("role_name", roles[i].Name))
		}
		if role == nil {
			role = &model.Role{
				Name:        permission.SuperAdminRoleName,
				Code:        permission.SuperAdminRoleCode,
				Description: permission.SuperAdminRoleDescription,
				Status:      1,
				IsSystem:    1,
			}
			if err := tx.Create(role).Error; err != nil {
				return databaseLogger.WrapDatabaseError("初始化超级管理员角色: 创建角色失败", err)
			}
			logger.Info("已创建内置超级管理员角色", zap.Int("role_id", role.ID))
		} else if role.Status != 1 {
			logger.Warn("内置超级管理员角色已被禁用，启动引导不会自动启用；持有该角色的账号当前没有超管权限",
				zap.Int("role_id", role.ID))
		}

		var users []model.User
		if err := tx.Select("id", "username").Where("username = ?", permission.BootstrapAdminUsername).Find(&users).Error; err != nil {
			return databaseLogger.WrapDatabaseError("初始化超级管理员角色: 查询管理员失败", err)
		}
		for _, user := range users {
			// MySQL 默认排序规则大小写不敏感，这里再按字节精确比较，避免绑定 "Admin" 等账号
			if user.Username != permission.BootstrapAdminUsername {
				continue
			}
			var bound int64
			if err := tx.Model(&model.UserRole{}).Where("user_id = ? AND role_id = ?", user.ID, role.ID).Count(&bound).Error; err != nil {
				return databaseLogger.WrapDatabaseError("初始化超级管理员角色: 查询绑定失败", err)
			}
			if bound > 0 {
				continue
			}
			if err := tx.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
				return databaseLogger.WrapDatabaseError("初始化超级管理员角色: 绑定管理员失败", err)
			}
			logger.Info("已将内置管理员绑定到超级管理员角色", zap.Int("user_id", user.ID), zap.Int("role_id", role.ID))
		}
		return nil
	})
}
