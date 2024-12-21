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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type RoleService interface {
	ListRoles(ctx context.Context, page, pageSize int) ([]*model.Role, int, error)
	CreateRole(ctx context.Context, role *model.Role, apiIds []int) error
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id int) error
	GetRole(ctx context.Context, id int) (*model.Role, error)
	GetUserRole(ctx context.Context, userId int) (*model.Role, error)
}

type roleService struct {
	roleDao dao.RoleDAO
	l       *zap.Logger
}

func NewRoleService(roleDao dao.RoleDAO, l *zap.Logger) RoleService {
	return &roleService{
		roleDao: roleDao,
		l:       l,
	}
}

// ListRoles 获取角色列表
func (r *roleService) ListRoles(ctx context.Context, page, pageSize int) ([]*model.Role, int, error) {
	return r.roleDao.ListRoles(ctx, page, pageSize)
}

// CreateRole 创建新角色
func (r *roleService) CreateRole(ctx context.Context, role *model.Role, apiIds []int) error {
	return r.roleDao.CreateRole(ctx, role, apiIds)
}

// UpdateRole 更新角色信息
func (r *roleService) UpdateRole(ctx context.Context, role *model.Role) error {
	return r.roleDao.UpdateRole(ctx, role)
}

// DeleteRole 删除角色
func (r *roleService) DeleteRole(ctx context.Context, id int) error {
	return r.roleDao.DeleteRole(ctx, id)
}

// GetRole 获取角色详情
func (r *roleService) GetRole(ctx context.Context, id int) (*model.Role, error) {
	return r.roleDao.GetRole(ctx, id)
}

// GetUserRole 获取用户角色
func (r *roleService) GetUserRole(ctx context.Context, userId int) (*model.Role, error) {
	return r.roleDao.GetUserRole(ctx, userId)
}
