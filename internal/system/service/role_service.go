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
	"errors"
	"fmt"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
)

type RoleService interface {
	ListRoles(ctx context.Context, page, pageSize int) (*model.GenerateRoleResp, error)
	CreateRole(ctx context.Context, req *model.CreateRoleRequest) error
	UpdateRole(ctx context.Context, req *model.UpdateRoleRequest) error
	DeleteRole(ctx context.Context, req *model.DeleteRoleRequest) error
	GetUserRoles(ctx context.Context, userId int, page, pageSize int) (*model.GenerateRoleResp, error)
	DeleteUserAllRoles(ctx context.Context, userId int) error
}

type roleService struct {
	roleDao  dao.RoleDAO
	enforcer *casbin.Enforcer
	l        *zap.Logger
}

func NewRoleService(roleDao dao.RoleDAO, enforcer *casbin.Enforcer, l *zap.Logger) RoleService {
	return &roleService{
		roleDao:  roleDao,
		enforcer: enforcer,
		l:        l,
	}
}

// ListRoles 获取角色列表
func (r *roleService) ListRoles(ctx context.Context, page, pageSize int) (*model.GenerateRoleResp, error) {
	resp, err := r.roleDao.ListRoles(ctx, page, pageSize)
	if err != nil {
		r.l.Error("获取角色列表失败", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

// CreateRole 创建新角色
func (r *roleService) CreateRole(ctx context.Context, req *model.CreateRoleRequest) error {
	exist, err := r.enforcer.HasPolicy(req.Name, req.Domain, req.Path, req.Method)
	if err != nil {
		r.l.Error("检查角色是否存在失败", zap.Error(err))
		return err
	}
	if exist {
		r.l.Error("角色已存在", zap.String("name", req.Name), zap.String("domain", req.Domain), zap.String("path", req.Path), zap.String("method", req.Method))
		return errors.New("角色已存在")
	}

	// 创建角色
	_, err = r.enforcer.AddPolicy(req.Name, req.Domain, req.Path, req.Method)
	if err != nil {
		r.l.Error("创建角色失败", zap.Error(err))
		return err
	}

	// 保存策略
	if err := r.enforcer.SavePolicy(); err != nil {
		r.l.Error("保存策略失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateRole 更新角色信息
func (r *roleService) UpdateRole(ctx context.Context, req *model.UpdateRoleRequest) error {
	oldRole := req.OldRole
	newRole := req.NewRole

	// 检查旧角色是否存在
	exist, err := r.enforcer.HasPolicy(oldRole.Name, oldRole.Domain, oldRole.Path, oldRole.Method)
	if err != nil {
		r.l.Error("检查旧角色是否存在失败", zap.Error(err), zap.Any("oldRole", oldRole))
		return fmt.Errorf("检查旧角色是否存在失败: %w", err)
	}

	if !exist {
		r.l.Error("旧角色不存在",
			zap.String("name", oldRole.Name),
			zap.String("domain", oldRole.Domain),
			zap.String("path", oldRole.Path),
			zap.String("method", oldRole.Method))
		return errors.New("旧角色不存在")
	}

	// 检查新角色是否已存在（避免冲突）
	if oldRole.Name != newRole.Name || oldRole.Domain != newRole.Domain ||
		oldRole.Path != newRole.Path || oldRole.Method != newRole.Method {
		exist, err = r.enforcer.HasPolicy(newRole.Name, newRole.Domain, newRole.Path, newRole.Method)
		if err != nil {
			r.l.Error("检查新角色是否存在失败", zap.Error(err), zap.Any("newRole", newRole))
			return fmt.Errorf("检查新角色是否存在失败: %w", err)
		}
		if exist {
			r.l.Error("新角色已存在", zap.Any("newRole", newRole))
			return errors.New("新角色已存在")
		}
	}
	// 更新角色策略
	_, err = r.enforcer.UpdatePolicy(
		[]string{oldRole.Name, oldRole.Domain, oldRole.Path, oldRole.Method},
		[]string{newRole.Name, newRole.Domain, newRole.Path, newRole.Method})
	if err != nil {
		r.l.Error("更新角色失败", zap.Error(err), zap.Any("oldRole", oldRole), zap.Any("newRole", newRole))
		return fmt.Errorf("更新角色失败: %w", err)
	}

	// 保存策略
	if err := r.enforcer.SavePolicy(); err != nil {
		r.l.Error("保存策略失败", zap.Error(err))
		return fmt.Errorf("保存策略失败: %w", err)
	}

	return nil
}

// DeleteRole 删除角色
func (r *roleService) DeleteRole(ctx context.Context, req *model.DeleteRoleRequest) error {
	// 检查角色是否存在
	exist, err := r.enforcer.HasPolicy(req.Name, req.Domain, req.Path, req.Method)
	if err != nil {
		r.l.Error("检查角色是否存在失败", zap.Error(err))
		return err
	}

	if !exist {
		r.l.Error("角色不存在", zap.Any("role", req))
		return errors.New("角色不存在")
	}

	// 删除角色策略
	_, err = r.enforcer.RemovePolicy(req.Name, req.Domain, req.Path, req.Method)
	if err != nil {
		r.l.Error("删除角色策略失败", zap.Error(err))
		return err
	}

	// 保存策略
	if err := r.enforcer.SavePolicy(); err != nil {
		r.l.Error("保存策略失败", zap.Error(err))
		return fmt.Errorf("保存策略失败: %w", err)
	}

	return nil
}

// GetUserRoles 获取用户角色
func (r *roleService) GetUserRoles(ctx context.Context, userId int, page, pageSize int) (*model.GenerateRoleResp, error) {
	roles, err := r.roleDao.GetRolesByUserId(ctx, userId, page, pageSize)
	if err != nil {
		r.l.Error("获取用户角色失败", zap.Error(err))
		return nil, err
	}

	return roles, nil
}

// DeleteUserAllRoles 删除用户所有角色
func (r *roleService) DeleteUserAllRoles(ctx context.Context, userId int) error {
	// 删除用户所有角色
	_, err := r.enforcer.RemoveFilteredPolicy(0, strconv.Itoa(userId))
	if err != nil {
		r.l.Error("删除用户所有角色失败", zap.Error(err))
		return err
	}

	// 保存策略
	if err := r.enforcer.SavePolicy(); err != nil {
		r.l.Error("保存策略失败", zap.Error(err))
		return fmt.Errorf("保存策略失败: %w", err)
	}

	return nil
}
