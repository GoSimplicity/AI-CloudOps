package app

import (
	"context"
	"fmt"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/mock"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BootstrapAdminPasswordEnv 是首次启动创建管理员所用口令的环境变量名。
	BootstrapAdminPasswordEnv = "BOOTSTRAP_ADMIN_PASSWORD"
	// MinBootstrapAdminPasswordLen 是 BOOTSTRAP_ADMIN_PASSWORD 的最小长度。
	MinBootstrapAdminPasswordLen = 12
)

// takeSecretEnv 读取口令类环境变量后立即从进程环境中删除，避免被 cron 等子进程继承。
func (a *App) takeSecretEnv(key string) string {
	value := os.Getenv(key)
	if err := os.Unsetenv(key); err != nil && a.logger != nil {
		a.logger.Warn("[SECURITY] 清除口令环境变量失败，子进程可能继承该变量", zap.String("env", key), zap.Error(err))
	}
	return value
}

// scrubAdminPasswordEnv 兜底清除管理员口令环境变量（例如未开启 mock 时 MOCK_ADMIN_PASSWORD 不会被读取）。
func (a *App) scrubAdminPasswordEnv() {
	for _, key := range []string{BootstrapAdminPasswordEnv, mock.MockAdminPasswordEnv} {
		_ = a.takeSecretEnv(key)
	}
}

// bootstrapAdmin creates an administrator only on an empty installation.
// 口令读取后立即从进程环境中删除。口令过短时：生产环境拒绝启动；非生产环境打印 [SECURITY] Warn 并跳过创建。
func (a *App) bootstrapAdmin(ctx context.Context) error {
	password := a.takeSecretEnv(BootstrapAdminPasswordEnv)
	if password == "" {
		return nil
	}
	if len(password) < MinBootstrapAdminPasswordLen {
		if config.IsProduction() {
			return fmt.Errorf("%s 至少%d字符", BootstrapAdminPasswordEnv, MinBootstrapAdminPasswordLen)
		}
		a.logger.Warn(fmt.Sprintf("[SECURITY] %s 少于%d字符，非生产环境跳过创建初始管理员；请改为至少%d字符的强口令后重启（生产环境会拒绝启动）",
			BootstrapAdminPasswordEnv, MinBootstrapAdminPasswordLen, MinBootstrapAdminPasswordLen))
		return nil
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
	err = a.db.WithContext(ctx).Create(&model.User{Username: "admin", Password: string(hash), RealName: "管理员", AccountType: constants.AccountTypeUser, Enable: 1, HomePath: "/"}).Error
	return databaseLogger.WrapDatabaseError("初始化管理员: 创建用户失败", err)
}
