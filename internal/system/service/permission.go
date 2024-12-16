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

package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type PermissionService interface {
	AssignRole(ctx context.Context, roleId int, menuIds []int, apiIds []int) error
	AssignRoleToUser(ctx context.Context, userId int, roleIds []int, menuIds []int, apiIds []int) error
	AssignRoleToUsers(ctx context.Context, userIds []int, roleIds []int) error
}

type permissionService struct {
	l   *zap.Logger
	dao dao.PermissionDAO
}

func NewPermissionService(l *zap.Logger, dao dao.PermissionDAO) PermissionService {
	return &permissionService{
		l:   l,
		dao: dao,
	}
}

// TODO: 下述接口遗留问题
// 1.每次角色分配权限，都会先移除旧权限，再重新分配新权限，这样会导致角色被赋予新权限后，用户没有旧权限，导致无法访问
// 目前解决办法: 在前端处理，打开前端的角色分配权限页面，先获取当前角色的权限，再进行分配，这样不会出现上述问题
// 2.每次移除旧的权限都是对数据库的一次写操作，但是每次分配新权限，都会对数据库进行一次读操作，这样会导致性能问题，需要优化
// 3.由于casbin_rule表id主键为自增的，所以每次写入都会导致id自增，导致性能问题，需要优化

// AssignRole 为角色分配权限
func (p *permissionService) AssignRole(ctx context.Context, roleId int, menuIds []int, apiIds []int) error {
	// 参数校验
	if roleId <= 0 {
		p.l.Warn("角色ID无效", zap.Int("roleId", roleId))
		return nil
	}

	// 先移除旧权限
	if err := p.dao.RemoveRolePermissions(ctx, roleId); err != nil {
		p.l.Error("移除角色API权限失败", zap.Error(err))
		return err
	}

	// 分配新权限
	return p.dao.AssignRole(ctx, roleId, menuIds, apiIds)
}

// AssignRoleToUser 为用户分配角色和权限
func (p *permissionService) AssignRoleToUser(ctx context.Context, userId int, roleIds []int, menuIds []int, apiIds []int) error {
	// 参数校验
	if userId <= 0 {
		p.l.Warn("用户ID无效", zap.Int("userId", userId))
		return nil
	}

	// 先移除旧角色和权限
	if err := p.dao.RemoveUserPermissions(ctx, userId); err != nil {
		p.l.Error("移除用户角色失败", zap.Error(err))
		return err
	}

	// 分配新角色和权限
	return p.dao.AssignRoleToUser(ctx, userId, roleIds, menuIds, apiIds)
}

// AssignRoleToUsers 为多个用户批量分配角色和权限
func (p *permissionService) AssignRoleToUsers(ctx context.Context, userIds []int, roleIds []int) error {
	// 参数校验
	if len(userIds) == 0 {
		p.l.Warn("用户ID列表不能为空")
		return nil
	}

	// 先移除旧角色和权限
	if err := p.dao.RemoveUsersPermissions(ctx, userIds); err != nil {
		p.l.Error("移除用户角色失败", zap.Error(err))
		return err
	}

	// 批量分配新角色和权限
	return p.dao.AssignRoleToUsers(ctx, userIds, roleIds)
}
