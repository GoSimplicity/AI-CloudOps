package di

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

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/gorm-adapter/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitCasbin 初始化 Casbin 并使用 Gorm 作为策略存储
func InitCasbin(db *gorm.DB, logger *zap.Logger) *casbin.Enforcer {
	// 使用 Gorm 数据库适配器作为 Casbin 的存储后端
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		logger.Error("failed to initialize Casbin adapter", zap.Error(err))
		return nil
	}

	// Casbin 模型定义：定义了请求、策略、角色和匹配规则
	modelText := `
		[request_definition]
		r = sub, obj, act

		[policy_definition]
		p = sub, obj, act

		[role_definition]
		g = _, _

		[policy_effect]
		e = some(where (p.eft == allow))

		[matchers]
		m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
	`

	// 从字符串中加载模型
	casbinModel, err := model.NewModelFromString(modelText)
	if err != nil {
		logger.Error("failed to initialize Casbin model", zap.Error(err))
		return nil
	}

	// 创建 Casbin Enforcer，结合模型和适配器（存储后端）
	enforcer, err := casbin.NewEnforcer(casbinModel, adapter)
	if err != nil {
		logger.Error("failed to initialize Casbin enforcer", zap.Error(err))
		return nil
	}

	// 加载策略到 Enforcer 中
	if err := enforcer.LoadPolicy(); err != nil {
		logger.Error("failed to load Casbin policies", zap.Error(err))
		return nil
	}

	logger.Info("Casbin initialized successfully")

	return enforcer
}
