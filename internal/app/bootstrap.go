package app

import (
	"context"
	"fmt"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// bootstrapAdmin creates an administrator only on an empty installation.
func (a *App) bootstrapAdmin(ctx context.Context) error {
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if password == "" {
		return nil
	}
	if len(password) < 12 {
		return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD 至少12字符")
	}
	var count int64
	if err := a.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return databaseLogger.WrapDatabaseError("初始化管理员: 查询用户失败", err)
	}
	if count != 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = a.db.WithContext(ctx).Create(&model.User{Username: "admin", Password: string(hash), RealName: "管理员", AccountType: 1, Enable: 1, HomePath: "/"}).Error
	return databaseLogger.WrapDatabaseError("初始化管理员: 创建用户失败", err)
}
