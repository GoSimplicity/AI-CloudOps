package casbin

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
	"context"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
)

// CasbinDAO 定义了与 Casbin 相关的权限管理操作
type CasbinDAO interface {
	// CheckPermission 检查角色是否具有某资源的访问权限
	CheckPermission(ctx context.Context, roleName, path, method string) (bool, error)
	// AddPolicies 批量添加权限策略
	AddPolicies(ctx context.Context, rules [][]string) (bool, error)
	// AddPolicy 添加单条权限策略
	AddPolicy(ctx context.Context, sub, obj, act string) (bool, error)
}

type casbinDAO struct {
	casbin *casbin.Enforcer
	l      *zap.Logger
}

func NewCasbinDAO(casbin *casbin.Enforcer, l *zap.Logger) CasbinDAO {
	return &casbinDAO{
		casbin: casbin,
		l:      l,
	}
}

// CheckPermission 检查指定的角色 roleName 是否有权访问某个资源 path 并执行特定操作 method
func (c *casbinDAO) CheckPermission(_ context.Context, roleName, path, method string) (bool, error) {
	// 使用 Casbin 的 Enforce 方法进行权限检查
	enforce, err := c.casbin.Enforce(roleName, path, method)
	if err != nil {
		c.l.Error("casbin check permission failed", zap.Error(err))
		return false, err
	}

	return enforce, nil
}

// AddPolicies 批量添加多条权限规则
func (c *casbinDAO) AddPolicies(_ context.Context, rules [][]string) (bool, error) {
	// 使用 Casbin 的 AddPolicies 方法批量添加权限规则
	added, err := c.casbin.AddPolicies(rules)
	if err != nil {
		c.l.Error("casbin add policies failed", zap.Error(err))
		return false, err
	}

	return added, nil
}

// AddPolicy 添加单条权限规则
func (c *casbinDAO) AddPolicy(_ context.Context, sub, obj, act string) (bool, error) {
	// 使用 Casbin 的 AddPolicy 方法添加单条权限规则
	added, err := c.casbin.AddPolicy(sub, obj, act)
	if err != nil {
		c.l.Error("casbin add policy failed", zap.Error(err))
		return false, err
	}

	return added, nil
}
