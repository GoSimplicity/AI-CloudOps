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
	List(ctx context.Context, req *model.ListRolesRequest) ([]*model.Role, int64, error)
	Create(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error)
	Update(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error)
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.Role, error)
	CheckExists(ctx context.Context, name, code string, excludeID int) (bool, error)
	CheckHasUsers(ctx context.Context, roleID int) (bool, error)
	AssignApis(ctx context.Context, roleID int, apiIds []int) error
	RevokeApis(ctx context.Context, roleID int, apiIds []int) error
	GetApis(ctx context.Context, roleID int) ([]*model.Api, error)
	AssignRolesToUser(ctx context.Context, userID int, roleIds []int, grantedBy int) error
	RevokeRolesFromUser(ctx context.Context, userID int, roleIds []int) error
	GetUsers(ctx context.Context, roleID int) ([]*model.User, error)
	GetRoles(ctx context.Context, userID int) ([]*model.Role, error)
	CheckPermission(ctx context.Context, userID int, method, path string) (bool, error)
	GetPermissions(ctx context.Context, userID int) ([]*model.Api, error)
}

type roleDAO struct {
	db *gorm.DB
}

func NewRoleDAO(db *gorm.DB) RoleDAO {
	return &roleDAO{
		db: db,
	}
}

// List 获取角色列表
func (d *roleDAO) List(ctx context.Context, req *model.ListRolesRequest) ([]*model.Role, int64, error) {
	var roles []*model.Role
	var total int64

	query := d.db.WithContext(ctx).Model(&model.Role{})

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

// Create 创建角色
func (d *roleDAO) Create(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error) {
	tx := d.db.WithContext(ctx).Begin()
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
	return d.GetByID(ctx, role.ID)
}

// Update 更新角色
func (d *roleDAO) Update(ctx context.Context, role *model.Role, apiIds []int) (*model.Role, error) {
	tx := d.db.WithContext(ctx).Begin()
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
	return d.GetByID(ctx, role.ID)
}

// Delete 删除角色
func (d *roleDAO) Delete(ctx context.Context, id int) error {
	tx := d.db.WithContext(ctx).Begin()
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

// GetByID 根据ID获取角色
func (d *roleDAO) GetByID(ctx context.Context, id int) (*model.Role, error) {
	var role model.Role
	if err := d.db.WithContext(ctx).Preload("Apis").Preload("Users").
		First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// CheckExists 检查角色名称或编码是否已存在
func (d *roleDAO) CheckExists(ctx context.Context, name, code string, excludeID int) (bool, error) {
	var count int64
	query := d.db.WithContext(ctx).Model(&model.Role{}).
		Where("(name = ? OR code = ?)", name, code)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// CheckHasUsers 检查角色是否有关联用户
func (d *roleDAO) CheckHasUsers(ctx context.Context, roleID int) (bool, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&model.UserRole{}).
		Where("role_id = ?", roleID).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// AssignApis 为角色分配API权限
func (d *roleDAO) AssignApis(ctx context.Context, roleID int, apiIds []int) error {
	tx := d.db.WithContext(ctx).Begin()
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

// RevokeApis 撤销角色的API权限
func (d *roleDAO) RevokeApis(ctx context.Context, roleID int, apiIds []int) error {
	return d.db.WithContext(ctx).Where("role_id = ? AND api_id IN ?", roleID, apiIds).
		Delete(&model.RoleApi{}).Error
}

// GetApis 获取角色的API权限列表
func (d *roleDAO) GetApis(ctx context.Context, roleID int) ([]*model.Api, error) {
	var apis []*model.Api
	if err := d.db.WithContext(ctx).Model(&model.Api{}).
		Joins("JOIN cl_system_role_apis ON cl_system_apis.id = cl_system_role_apis.api_id").
		Where("cl_system_role_apis.role_id = ?", roleID).
		Find(&apis).Error; err != nil {
		return nil, err
	}

	return apis, nil
}

// AssignRoles 为用户分配角色
func (d *roleDAO) AssignRolesToUser(ctx context.Context, userID int, roleIds []int, grantedBy int) error {
	tx := d.db.WithContext(ctx).Begin()
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

// RevokeRoles 撤销用户角色
func (d *roleDAO) RevokeRolesFromUser(ctx context.Context, userID int, roleIds []int) error {
	query := d.db.WithContext(ctx).Where("user_id = ?", userID)
	if len(roleIds) > 0 {
		query = query.Where("role_id IN ?", roleIds)
	}
	return query.Delete(&model.UserRole{}).Error
}

// GetUsers 获取角色下的用户列表
func (d *roleDAO) GetUsers(ctx context.Context, roleID int) ([]*model.User, error) {
	var users []*model.User
	if err := d.db.WithContext(ctx).Model(&model.User{}).
		Joins("JOIN cl_system_user_roles ON cl_system_users.id = cl_system_user_roles.user_id").
		Where("cl_system_user_roles.role_id = ?", roleID).
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserRoles 获取用户的角色列表
func (d *roleDAO) GetRoles(ctx context.Context, userID int) ([]*model.Role, error) {
	var roles []*model.Role
	if err := d.db.WithContext(ctx).
		Preload("Apis").
		Model(&model.Role{}).
		Joins("JOIN cl_system_user_roles ON cl_system_roles.id = cl_system_user_roles.role_id").
		Where("cl_system_user_roles.user_id = ? AND cl_system_roles.status = 1", userID).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// CheckUserPermission 检查用户权限
func (d *roleDAO) CheckPermission(ctx context.Context, userID int, method, path string) (bool, error) {
	var count int64

	// 通过用户角色查询是否有对应的API权限
	query := `
		 SELECT COUNT(DISTINCT a.id) 
		 FROM cl_system_apis a
		 JOIN cl_system_role_apis ra ON a.id = ra.api_id
		 JOIN cl_system_user_roles ur ON ra.role_id = ur.role_id
		 JOIN cl_system_roles r ON ur.role_id = d.id
		 WHERE cl_system_user_roles.user_id = ? 
		 AND d.status = 1 
		 AND a.method = ? 
		 AND a.path = ?
	 `

	if err := d.db.WithContext(ctx).Raw(query, userID, method, path).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserPermissions 获取用户的所有权限
func (d *roleDAO) GetPermissions(ctx context.Context, userID int) ([]*model.Api, error) {
	var apis []*model.Api

	query := `
		 SELECT DISTINCT a.*
		 FROM cl_system_apis a
		 JOIN cl_system_role_apis ra ON a.id = ra.api_id
		 JOIN cl_system_user_roles ur ON ra.role_id = ur.role_id
		 JOIN cl_system_roles r ON ur.role_id = d.id
		 WHERE ur.user_id = ? AND d.status = 1
		 ORDER BY a.created_at DESC
	 `

	if err := d.db.WithContext(ctx).Raw(query, userID).Scan(&apis).Error; err != nil {
		return nil, err
	}

	return apis, nil
}
