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

package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type RoleDAO interface {
	// 角色管理
	ListRoles(ctx context.Context, req *model.ListRolesRequest) ([]*model.Role, int64, error)
	CreateRole(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error)
	DeleteRole(ctx context.Context, id int) error
	GetRoleByID(ctx context.Context, id int) (*model.Role, error)
	CheckRoleExists(ctx context.Context, name, code string, excludeID int) (bool, error)
	CheckRoleHasUsers(ctx context.Context, roleID int) (bool, error)

	// 角色权限管理
	AssignApisToRole(ctx context.Context, roleID int, apiIds []int) error
	RevokeApisFromRole(ctx context.Context, roleID int, apiIds []int) error
	GetRoleApis(ctx context.Context, roleID int) ([]*model.Api, error)

	// 用户角色管理
	AssignRolesToUser(ctx context.Context, userID int, roleIds []int, grantedBy int) error
	RevokeRolesFromUser(ctx context.Context, userID int, roleIds []int) error
	GetRoleUsers(ctx context.Context, roleID int) ([]*model.User, error)
	GetUserRoles(ctx context.Context, userID int) ([]*model.Role, error)

	// 权限检查
	CheckUserPermission(ctx context.Context, userID int, method, path string) (bool, error)
	GetUserPermissions(ctx context.Context, userID int) ([]*model.Api, error)
}

type roleDAO struct {
	db *gorm.DB
}

func NewRoleDAO(db *gorm.DB) RoleDAO {
	return &roleDAO{
		db: db,
	}
}

// ListRoles 获取角色列表
func (r *roleDAO) ListRoles(ctx context.Context, req *model.ListRolesRequest) ([]*model.Role, int64, error) {
	var roles []*model.Role
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Role{})

	// 状态筛选
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 名称模糊搜索
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Preload("Apis").Preload("Users").
		Offset(offset).Limit(req.Size).
		Order("created_at DESC").Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

// CreateRole 创建角色
func (r *roleDAO) CreateRole(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error) {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建角色
	if err := tx.Create(role).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 关联API权限
	if len(apiIds) > 0 {
		var roleApis []model.RoleApi
		for _, apiID := range apiIds {
			roleApis = append(roleApis, model.RoleApi{
				RoleID: role.ID,
				ApiID:  apiID,
			})
		}
		if err := tx.Create(&roleApis).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// 重新加载角色数据
	return r.GetRoleByID(ctx, role.ID)
}

// UpdateRole 更新角色
func (r *roleDAO) UpdateRole(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error) {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新角色基本信息
	if err := tx.Model(role).Updates(map[string]interface{}{
		"name":        role.Name,
		"code":        role.Code,
		"description": role.Description,
		"status":      role.Status,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 删除原有的API权限关联
	if err := tx.Where("role_id = ?", role.ID).Delete(&model.RoleApi{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 重新关联API权限
	if len(apiIds) > 0 {
		var roleApis []model.RoleApi
		for _, apiID := range apiIds {
			roleApis = append(roleApis, model.RoleApi{
				RoleID: role.ID,
				ApiID:  apiID,
			})
		}
		if err := tx.Create(&roleApis).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// 重新加载角色数据
	return r.GetRoleByID(ctx, role.ID)
}

// DeleteRole 删除角色
func (r *roleDAO) DeleteRole(ctx context.Context, id int) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除角色API权限关联
	if err := tx.Where("role_id = ?", id).Delete(&model.RoleApi{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除用户角色关联
	if err := tx.Where("role_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除角色
	if err := tx.Delete(&model.Role{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetRoleByID 根据ID获取角色
func (r *roleDAO) GetRoleByID(ctx context.Context, id int) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Preload("Apis").Preload("Users").
		First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// CheckRoleExists 检查角色名称或编码是否已存在
func (r *roleDAO) CheckRoleExists(ctx context.Context, name, code string, excludeID int) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.Role{}).
		Where("(name = ? OR code = ?)", name, code)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// CheckRoleHasUsers 检查角色是否有关联用户
func (r *roleDAO) CheckRoleHasUsers(ctx context.Context, roleID int) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.UserRole{}).
		Where("role_id = ?", roleID).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// AssignApisToRole 为角色分配API权限
func (r *roleDAO) AssignApisToRole(ctx context.Context, roleID int, apiIds []int) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除已存在的关联
	if err := tx.Where("role_id = ? AND api_id IN ?", roleID, apiIds).
		Delete(&model.RoleApi{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建新的关联
	var roleApis []model.RoleApi
	for _, apiID := range apiIds {
		roleApis = append(roleApis, model.RoleApi{
			RoleID: roleID,
			ApiID:  apiID,
		})
	}

	if err := tx.Create(&roleApis).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RevokeApisFromRole 撤销角色的API权限
func (r *roleDAO) RevokeApisFromRole(ctx context.Context, roleID int, apiIds []int) error {
	return r.db.WithContext(ctx).Where("role_id = ? AND api_id IN ?", roleID, apiIds).
		Delete(&model.RoleApi{}).Error
}

// GetRoleApis 获取角色的API权限列表
func (r *roleDAO) GetRoleApis(ctx context.Context, roleID int) ([]*model.Api, error) {
	var apis []*model.Api
	if err := r.db.WithContext(ctx).Table("apis").
		Joins("JOIN role_apis ON apis.id = role_apis.api_id").
		Where("role_apis.role_id = ?", roleID).
		Find(&apis).Error; err != nil {
		return nil, err
	}

	return apis, nil
}

// AssignRolesToUser 为用户分配角色
func (r *roleDAO) AssignRolesToUser(ctx context.Context, userID int, roleIds []int, grantedBy int) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除已存在的关联
	if err := tx.Where("user_id = ? AND role_id IN ?", userID, roleIds).
		Delete(&model.UserRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建新的关联
	var userRoles []model.UserRole
	for _, roleID := range roleIds {
		userRoles = append(userRoles, model.UserRole{
			UserID: userID,
			RoleID: roleID,
		})
	}

	if err := tx.Create(&userRoles).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RevokeRolesFromUser 撤销用户角色
func (r *roleDAO) RevokeRolesFromUser(ctx context.Context, userID int, roleIds []int) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND role_id IN ?", userID, roleIds).
		Delete(&model.UserRole{}).Error
}

// GetRoleUsers 获取角色下的用户列表
func (r *roleDAO) GetRoleUsers(ctx context.Context, roleID int) ([]*model.User, error) {
	var users []*model.User
	if err := r.db.WithContext(ctx).Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Where("user_roles.role_id = ?", roleID).
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserRoles 获取用户的角色列表
func (r *roleDAO) GetUserRoles(ctx context.Context, userID int) ([]*model.Role, error) {
	var roles []*model.Role
	if err := r.db.WithContext(ctx).Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.status = 1", userID).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// CheckUserPermission 检查用户权限
func (r *roleDAO) CheckUserPermission(ctx context.Context, userID int, method, path string) (bool, error) {
	var count int64

	// 通过用户角色查询是否有对应的API权限
	query := `
		 SELECT COUNT(DISTINCT a.id) 
		 FROM apis a
		 JOIN role_apis ra ON a.id = ra.api_id
		 JOIN user_roles ur ON ra.role_id = ur.role_id
		 JOIN roles r ON ur.role_id = r.id
		 WHERE ur.user_id = ? 
		 AND r.status = 1 
		 AND a.method = ? 
		 AND a.path = ?
	 `

	if err := r.db.WithContext(ctx).Raw(query, userID, method, path).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserPermissions 获取用户的所有权限
func (r *roleDAO) GetUserPermissions(ctx context.Context, userID int) ([]*model.Api, error) {
	var apis []*model.Api

	query := `
		 SELECT DISTINCT a.*
		 FROM apis a
		 JOIN role_apis ra ON a.id = ra.api_id
		 JOIN user_roles ur ON ra.role_id = ur.role_id
		 JOIN roles r ON ur.role_id = r.id
		 WHERE ur.user_id = ? AND r.status = 1
		 ORDER BY a.created_at DESC
	 `

	if err := r.db.WithContext(ctx).Raw(query, userID).Scan(&apis).Error; err != nil {
		return nil, err
	}

	return apis, nil
}
