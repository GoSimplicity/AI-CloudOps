/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */
package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/mock"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// SEC-27：SEC-27 之前 mock/user_mock.go 会用一个内置的弱口令创建 admin。删除代码不会修改已有账号，
// 因此启动时检测 admin 是否仍在使用该口令。
//
// 为了比对 bcrypt 哈希，运行时必须拿到候选口令本身；为避免明文出现在源码、日志和扫描结果中，
// 源码只保存候选口令与 legacyMockAdminMaskKey 按字节循环异或后的结果，在内存中还原、比对后立即清零。
// 这只是防止误抄和扫描命中的遮蔽，不是加密：任何人都能还原它，而该口令本身已存在于公开的 Git 历史中。
var legacyMockAdminMasked = []byte{0x20, 0x2d, 0x40, 0x2a, 0x02}

// legacyMockAdminMaskKey 是遮蔽用的固定字节序列，不是密钥。
const legacyMockAdminMaskKey = "AI-CloudOps/SEC-27/legacy-mock-admin"

// ErrLegacyAdminPassword 表示 admin 仍在使用旧的内置弱口令；提示中不包含口令。
var ErrLegacyAdminPassword = errors.New("[SECURITY] admin 账号仍在使用 SEC-27 之前 mock 内置的弱口令，生产环境拒绝启动；" +
	"请先为 admin 设置新的强口令（至少12字符）后再启动，处置方式见 docs/security-baseline.md（SEC-27）")

// legacyMockAdminCandidate 在内存中还原旧内置口令，调用方用完后应清零。
func legacyMockAdminCandidate() []byte {
	out := make([]byte, len(legacyMockAdminMasked))
	for i, b := range legacyMockAdminMasked {
		out[i] = b ^ legacyMockAdminMaskKey[i%len(legacyMockAdminMaskKey)]
	}
	return out
}

// legacyAdminPasswordInUse 检查用户名为 admin 的账号是否仍在使用旧内置口令。
// 用户名当前没有唯一约束，因此会检查所有同名行；不存在 admin 时返回 false。
func (a *App) legacyAdminPasswordInUse(ctx context.Context) (bool, error) {
	var admins []model.User
	if err := a.db.WithContext(ctx).Select("id", "password").Where("username = ?", mock.AdminUsername).Find(&admins).Error; err != nil {
		return false, databaseLogger.WrapDatabaseError("检查管理员弱口令: 查询用户失败", err)
	}
	if len(admins) == 0 {
		return false, nil
	}
	candidate := legacyMockAdminCandidate()
	defer clear(candidate)
	for _, admin := range admins {
		if bcrypt.CompareHashAndPassword([]byte(admin.Password), candidate) == nil {
			return true, nil
		}
	}
	return false, nil
}

// checkLegacyAdminPassword 执行旧内置弱口令策略：
//   - 生产环境：admin 仍在使用旧口令或检查本身失败时拒绝启动；
//   - 非生产环境：只打印显眼的 [SECURITY] Warn 并继续启动；
//   - admin 不存在时跳过。
func (a *App) checkLegacyAdminPassword(ctx context.Context) error {
	inUse, err := a.legacyAdminPasswordInUse(ctx)
	production := config.IsProduction()
	if err != nil {
		if production {
			return fmt.Errorf("[SECURITY] 无法确认 admin 是否仍在使用旧内置弱口令，生产环境拒绝启动: %w", err)
		}
		a.logger.Warn("[SECURITY] 无法确认 admin 是否仍在使用旧内置弱口令（非生产环境继续启动），请检查数据库连接与用户表", zap.Error(err))
		return nil
	}
	if !inUse {
		return nil
	}
	if production {
		return ErrLegacyAdminPassword
	}
	a.logger.Warn("[SECURITY] admin 账号仍在使用 SEC-27 之前 mock 内置的弱口令（非生产环境仅告警）；" +
		"请立即在用户管理中修改 admin 口令（至少12字符），否则该库迁移到生产环境后将拒绝启动")
	return nil
}
