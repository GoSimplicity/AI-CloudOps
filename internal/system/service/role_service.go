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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RoleService interface {
	// 角色管理
	ListRoles(ctx context.Context, req *model.ListRolesRequest) (*model.ListResp[*model.Role], error)
	CreateRole(ctx context.Context, req *model.CreateRoleRequest) (*model.Role, error)
	UpdateRole(ctx context.Context, req *model.UpdateRoleRequest) (*model.Role, error)
	DeleteRole(ctx context.Context, id int) error
	GetRoleByID(ctx context.Context, id int) (*model.Role, error)

	// 角色权限管理
	AssignApisToRole(ctx context.Context, roleID int, apiIds []int) error
	RevokeApisFromRole(ctx context.Context, roleID int, apiIds []int) error
	GetRoleApis(ctx context.Context, roleID int) (*model.ListResp[*model.Api], error)

	// 用户角色管理
	AssignRolesToUser(ctx context.Context, userID int, roleIds []int, grantedBy int) error
	RevokeRolesFromUser(ctx context.Context, userID int, roleIds []int) error
	GetRoleUsers(ctx context.Context, roleID int) (*model.ListResp[*model.User], error)
	GetUserRoles(ctx context.Context, userID int) (*model.ListResp[*model.Role], error)

	// 权限检查
	CheckUserPermission(ctx context.Context, userID int, method, path string) (bool, error)
	GetUserPermissions(ctx context.Context, userID int) (*model.ListResp[*model.Api], error)
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
func (s *roleService) ListRoles(ctx context.Context, req *model.ListRolesRequest) (*model.ListResp[*model.Role], error) {
	roles, total, err := s.roleDao.List(ctx, req)
	if err != nil {
		s.l.Error("获取角色列表失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.Role]{
		Items: roles,
		Total: total,
	}, nil
}

// CreateRole 创建角色
func (s *roleService) CreateRole(ctx context.Context, req *model.CreateRoleRequest) (*model.Role, error) {
	// 检查角色名称和编码是否已存在
	exists, err := s.roleDao.CheckExists(ctx, req.Name, req.Code, 0)
	if err != nil {
		s.l.Error("检查角色是否存在失败", zap.Error(err))
		return nil, err
	}
	if exists {
		s.l.Error("角色名称或编码已存在", zap.String("name", req.Name), zap.String("code", req.Code))
		return nil, errors.New("角色名称或编码已存在")
	}

	role := &model.Role{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      int8(req.Status),
		IsSystem:    0, // 创建的角色默认不是系统角色
	}

	createdRole, err := s.roleDao.Create(ctx, role, req.ApiIds)
	if err != nil {
		s.l.Error("创建角色失败", zap.Error(err))
		return nil, err
	}

	return createdRole, nil
}

// UpdateRole 更新角色
func (s *roleService) UpdateRole(ctx context.Context, req *model.UpdateRoleRequest) (*model.Role, error) {
	// 检查角色是否存在
	existingRole, err := s.roleDao.GetByID(ctx, req.ID)
	if err != nil {
		s.l.Error("获取角色失败", zap.Error(err))
		return nil, err
	}

	// 系统角色不允许修改某些字段
	if existingRole.IsSystem == 1 {
		s.l.Error("系统角色不允许修改", zap.Int("role_id", req.ID))
		return nil, errors.New("系统角色不允许修改")
	}

	// 检查角色名称和编码是否已被其他角色使用
	exists, err := s.roleDao.CheckExists(ctx, req.Name, req.Code, req.ID)
	if err != nil {
		s.l.Error("检查角色是否存在失败", zap.Error(err))
		return nil, err
	}
	if exists {
		s.l.Error("角色名称或编码已被其他角色使用", zap.String("name", req.Name), zap.String("code", req.Code))
		return nil, errors.New("角色名称或编码已被其他角色使用")
	}

	role := &model.Role{
		Model:       model.Model{ID: req.ID},
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      int8(req.Status),
	}

	updatedRole, err := s.roleDao.Update(ctx, role, req.ApiIds)
	if err != nil {
		s.l.Error("更新角色失败", zap.Error(err))
		return nil, err
	}

	return updatedRole, nil
}

// DeleteRole 删除角色
func (s *roleService) DeleteRole(ctx context.Context, id int) error {
	// 检查角色是否存在
	role, err := s.roleDao.GetByID(ctx, id)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.l.Error("获取角色失败", zap.Error(err))
		return err
	}

	if role == nil {
		return nil
	}

	// 系统角色不允许删除
	if role.IsSystem == 1 {
		s.l.Error("系统角色不允许删除", zap.Int("role_id", id))
		return errors.New("系统角色不允许删除")
	}

	// 检查是否有用户关联该角色
	hasUsers, err := s.roleDao.CheckHasUsers(ctx, id)
	if err != nil {
		s.l.Error("检查角色是否有关联用户失败", zap.Error(err))
		return err
	}
	if hasUsers {
		s.l.Error("角色下还有关联用户，不允许删除", zap.Int("role_id", id))
		return errors.New("角色下还有关联用户，不允许删除")
	}

	err = s.roleDao.Delete(ctx, id)
	if err != nil {
		s.l.Error("删除角色失败", zap.Error(err))
		return err
	}

	return nil
}

// GetRoleByID 根据ID获取角色
func (s *roleService) GetRoleByID(ctx context.Context, id int) (*model.Role, error) {
	role, err := s.roleDao.GetByID(ctx, id)
	if err != nil {
		s.l.Error("获取角色详情失败", zap.Error(err))
		return nil, err
	}

	return role, nil
}

// AssignApisToRole 为角色分配API权限
func (s *roleService) AssignApisToRole(ctx context.Context, roleID int, apiIds []int) error {
	s.l.Info("为角色分配API权限", zap.Int("role_id", roleID), zap.Ints("api_ids", apiIds))

	err := s.roleDao.AssignApis(ctx, roleID, apiIds)
	if err != nil {
		s.l.Error("为角色分配API权限失败", zap.Error(err))
		return err
	}

	return nil
}

// RevokeApisFromRole 撤销角色的API权限
func (s *roleService) RevokeApisFromRole(ctx context.Context, roleID int, apiIds []int) error {
	err := s.roleDao.RevokeApis(ctx, roleID, apiIds)
	if err != nil {
		s.l.Error("撤销角色API权限失败", zap.Error(err))
		return err
	}

	return nil
}

// GetRoleApis 获取角色的API权限列表
func (s *roleService) GetRoleApis(ctx context.Context, roleID int) (*model.ListResp[*model.Api], error) {
	apis, err := s.roleDao.GetApis(ctx, roleID)
	if err != nil {
		s.l.Error("获取角色API权限失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.Api]{
		Items: apis,
	}, nil
}

// AssignRolesToUser 为用户分配角色
func (s *roleService) AssignRolesToUser(ctx context.Context, userID int, roleIds []int, grantedBy int) error {
	err := s.roleDao.AssignRolesToUser(ctx, userID, roleIds, grantedBy)
	if err != nil {
		s.l.Error("为用户分配角色失败", zap.Error(err))
		return err
	}

	return nil
}

// RevokeRolesFromUser 撤销用户角色
func (s *roleService) RevokeRolesFromUser(ctx context.Context, userID int, roleIds []int) error {
	err := s.roleDao.RevokeRolesFromUser(ctx, userID, roleIds)
	if err != nil {
		s.l.Error("撤销用户角色失败", zap.Error(err))
		return err
	}

	return nil
}

// GetRoleUsers 获取角色下的用户列表
func (s *roleService) GetRoleUsers(ctx context.Context, roleID int) (*model.ListResp[*model.User], error) {
	users, err := s.roleDao.GetUsers(ctx, roleID)
	if err != nil {
		s.l.Error("获取角色用户列表失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.User]{
		Items: users,
	}, nil
}

// GetUserRoles 获取用户的角色列表
func (s *roleService) GetUserRoles(ctx context.Context, userID int) (*model.ListResp[*model.Role], error) {
	roles, err := s.roleDao.GetRoles(ctx, userID)
	if err != nil {
		s.l.Error("获取用户角色列表失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.Role]{
		Items: roles,
	}, nil
}

// CheckUserPermission 检查用户权限
func (s *roleService) CheckUserPermission(ctx context.Context, userID int, method, path string) (bool, error) {
	hasPermission, err := s.roleDao.CheckPermission(ctx, userID, method, path)
	if err != nil {
		s.l.Error("检查用户权限失败", zap.Error(err))
		return false, err
	}

	return hasPermission, nil
}

// GetUserPermissions 获取用户的所有权限
func (s *roleService) GetUserPermissions(ctx context.Context, userID int) (*model.ListResp[*model.Api], error) {
	permissions, err := s.roleDao.GetPermissions(ctx, userID)
	if err != nil {
		s.l.Error("获取用户权限列表失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.Api]{
		Items: permissions,
	}, nil
}
