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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RoleDAO interface {
	CreateRole(ctx context.Context, role *model.Role, menuIds []int, apiIds []int) error
	GetRoleById(ctx context.Context, id int) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) error
	DeleteRole(ctx context.Context, id int) error
	ListRoles(ctx context.Context, page, pageSize int) ([]*model.Role, int, error)
	GetRole(ctx context.Context, roleId int) (*model.Role, error)
	GetUserRole(ctx context.Context, userId int) (*model.Role, error)
}

type roleDAO struct {
	db            *gorm.DB
	l             *zap.Logger
	enforcer      *casbin.Enforcer
	permissionDao PermissionDAO
}

func NewRoleDAO(db *gorm.DB, l *zap.Logger, enforcer *casbin.Enforcer, permissionDao PermissionDAO) RoleDAO {
	return &roleDAO{
		db:            db,
		l:             l,
		enforcer:      enforcer,
		permissionDao: permissionDao,
	}
}

// CreateRole 创建角色
func (r *roleDAO) CreateRole(ctx context.Context, role *model.Role, menuIds []int, apiIds []int) error {
	if role == nil {
		return errors.New("角色对象不能为空")
	}

	if role.Name == "" {
		return errors.New("角色名称不能为空")
	}

	var roleId int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64

		// 检查角色名是否已存在
		if err := tx.Model(&model.Role{}).Where("name = ? AND is_deleted = ?", role.Name, 0).Count(&count).Error; err != nil {
			return fmt.Errorf("检查角色名称失败: %v", err)
		}
		if count > 0 {
			return errors.New("角色名称已存在")
		}

		// 设置创建时间和更新时间
		now := time.Now().Unix()
		role.CreateTime = now
		role.UpdateTime = now
		role.IsDeleted = 0

		// 创建角色并返回ID
		result := tx.Create(role)
		if result.Error != nil {
			return fmt.Errorf("创建角色失败: %v", result.Error)
		}

		roleId = result.Statement.Model.(*model.Role).ID

		return nil
	})

	if err != nil {
		return err
	}

	// 分配权限
	if len(menuIds) > 0 || len(apiIds) > 0 {
		if err := r.permissionDao.AssignRole(ctx, roleId, menuIds, apiIds); err != nil {
			return fmt.Errorf("分配权限失败: %v", err)
		}
	}

	return nil
}

// GetRoleById 根据ID获取角色
func (r *roleDAO) GetRoleById(ctx context.Context, id int) (*model.Role, error) {
	if id <= 0 {
		return nil, errors.New("无效的角色ID")
	}

	var role model.Role
	if err := r.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, 0).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询角色失败: %v", err)
	}

	return &role, nil
}

// UpdateRole 更新角色信息
func (r *roleDAO) UpdateRole(ctx context.Context, role *model.Role) error {
	if role == nil {
		return errors.New("角色对象不能为空")
	}
	if role.ID <= 0 {
		return errors.New("无效的角色ID")
	}
	if role.Name == "" {
		return errors.New("角色名称不能为空")
	}

	// 获取原角色信息
	var oldRole model.Role
	if err := r.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", role.ID, 0).First(&oldRole).Error; err != nil {
		return fmt.Errorf("获取原角色信息失败: %v", err)
	}

	// 检查角色名是否已被其他角色使用
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Role{}).
		Where("name = ? AND id != ? AND is_deleted = ?", role.Name, role.ID, 0).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查角色名称失败: %v", err)
	}
	if count > 0 {
		return errors.New("角色名称已被使用")
	}

	updates := map[string]interface{}{
		"name":        role.Name,
		"description": role.Description,
		"role_type":   role.RoleType,
		"is_default":  role.IsDefault,
		"update_time": time.Now().Unix(),
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Role{}).
			Where("id = ? AND is_deleted = ?", role.ID, 0).
			Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("更新角色失败: %v", result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.New("角色不存在或已被删除")
		}

		// 如果角色名发生变化，需要更新casbin中的权限
		if oldRole.Name != role.Name {
			// 获取原角色的所有权限
			policies, err := r.enforcer.GetFilteredPolicy(0, oldRole.Name)
			if err != nil {
				return fmt.Errorf("获取原角色权限失败: %v", err)
			}

			// 删除原角色的所有权限
			if _, err := r.enforcer.DeleteRole(oldRole.Name); err != nil {
				return fmt.Errorf("删除原角色权限失败: %v", err)
			}

			// 为新角色名添加相同的权限
			for _, policy := range policies {
				policy[0] = role.Name // 更新角色名
				if _, err := r.enforcer.AddPolicy(policy); err != nil {
					return fmt.Errorf("添加新角色权限失败: %v", err)
				}
			}

			// 确保所有策略都已添加后再保存
			if err := r.enforcer.SavePolicy(); err != nil {
				return fmt.Errorf("保存权限策略失败: %v", err)
			}
		}

		return nil
	})

	return err
}

// DeleteRole 删除角色
func (r *roleDAO) DeleteRole(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("无效的角色ID")
	}

	// 检查是否为默认角色
	var role model.Role
	if err := r.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, 0).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return fmt.Errorf("查询角色失败: %v", err)
	}

	if role.IsDefault == 1 {
		return errors.New("默认角色不能删除")
	}

	updates := map[string]interface{}{
		"is_deleted":  1,
		"update_time": time.Now().Unix(),
	}

	result := r.db.WithContext(ctx).Model(&model.Role{}).Where("id = ? AND is_deleted = ?", id, 0).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("删除角色失败: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("角色不存在或已被删除")
	}

	// 删除角色关联的权限
	if _, err := r.enforcer.DeleteRole(role.Name); err != nil {
		return fmt.Errorf("删除角色权限失败: %v", err)
	}

	return nil
}

// ListRoles 获取角色列表
func (r *roleDAO) ListRoles(ctx context.Context, page, pageSize int) ([]*model.Role, int, error) {
	if page <= 0 || pageSize <= 0 {
		return nil, 0, errors.New("无效的分页参数")
	}

	var roles []*model.Role
	var total int64

	db := r.db.WithContext(ctx).Model(&model.Role{}).Where("is_deleted = ?", 0)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取角色总数失败: %v", err)
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("id ASC").Find(&roles).Error; err != nil {
		return nil, 0, fmt.Errorf("获取角色列表失败: %v", err)
	}

	return roles, int(total), nil
}

// GetRole 获取角色详细信息(包含权限)
func (r *roleDAO) GetRole(ctx context.Context, roleId int) (*model.Role, error) {
	if roleId <= 0 {
		return nil, errors.New("无效的角色ID")
	}

	var role model.Role
	if err := r.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", roleId, 0).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询角色失败: %v", err)
	}

	// 获取角色的所有权限策略
	policies, err := r.enforcer.GetFilteredPolicy(0, role.Name)
	if err != nil {
		return nil, fmt.Errorf("获取角色权限策略失败: %v", err)
	}

	// 解析权限策略获取菜单和API的ID
	menuIds := make([]int, 0)
	apiIds := make([]int, 0)
	for _, policy := range policies {
		if len(policy) < 2 {
			continue
		}
		if strings.HasPrefix(policy[1], "menu:") {
			if id, err := strconv.Atoi(strings.TrimPrefix(policy[1], "menu:")); err == nil {
				menuIds = append(menuIds, id)
			}
		} else if strings.HasPrefix(policy[1], "api:") {
			parts := strings.Split(policy[1], ":")
			if len(parts) >= 2 {
				if id, err := strconv.Atoi(parts[1]); err == nil {
					apiIds = append(apiIds, id)
				}
			}
		}
	}

	// 查询菜单和API详细信息
	if len(menuIds) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ? AND is_deleted = ?", menuIds, 0).Find(&role.Menus).Error; err != nil {
			return nil, fmt.Errorf("查询菜单失败: %v", err)
		}
	}
	if len(apiIds) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ? AND is_deleted = ?", apiIds, 0).Find(&role.Apis).Error; err != nil {
			return nil, fmt.Errorf("查询API失败: %v", err)
		}
	}

	return &role, nil
}

// GetUserRole 获取用户的角色信息
func (r *roleDAO) GetUserRole(ctx context.Context, userId int) (*model.Role, error) {
	if userId <= 0 {
		return nil, errors.New("无效的用户ID")
	}

	// 先从数据库中获取用户的角色
	var user model.User
	if err := r.db.WithContext(ctx).Preload("Roles", "is_deleted = ?", 0).Where("id = ? AND deleted_at = ?", userId, 0).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %v", err)
	}

	// 如果用户没有角色,返回nil
	if user.Roles == nil || len(user.Roles) == 0 {
		return nil, nil
	}

	// 获取第一个角色
	role := user.Roles[0]

	// 获取用户的所有权限(包括直接权限和角色权限)
	allPolicies := make([][]string, 0)

	// 获取用户直接权限
	userStr := fmt.Sprintf("%d", userId)
	userPolicies, err := r.enforcer.GetFilteredPolicy(0, userStr)
	if err == nil && len(userPolicies) > 0 {
		allPolicies = append(allPolicies, userPolicies...)
	}

	// 获取角色权限
	if role.ID > 0 {
		rolePolicies, err := r.enforcer.GetFilteredPolicy(0, role.Name)
		if err == nil && len(rolePolicies) > 0 {
			allPolicies = append(allPolicies, rolePolicies...)
		}
	}

	// 解析权限策略获取菜单和API的ID
	menuIdsMap := make(map[int]struct{})
	apiIdsMap := make(map[int]struct{})

	for _, policy := range allPolicies {
		if len(policy) < 2 {
			continue
		}
		if strings.HasPrefix(policy[1], "menu:") {
			if id, err := strconv.Atoi(strings.TrimPrefix(policy[1], "menu:")); err == nil {
				menuIdsMap[id] = struct{}{}
			}
		} else if strings.HasPrefix(policy[1], "api:") {
			parts := strings.Split(policy[1], ":")
			if len(parts) >= 2 {
				if id, err := strconv.Atoi(parts[1]); err == nil {
					apiIdsMap[id] = struct{}{}
				}
			}
		}
	}

	// 转换为切片
	menuIds := make([]int, 0, len(menuIdsMap))
	for id := range menuIdsMap {
		menuIds = append(menuIds, id)
	}

	apiIds := make([]int, 0, len(apiIdsMap))
	for id := range apiIdsMap {
		apiIds = append(apiIds, id)
	}

	// 查询菜单和API详细信息
	var menus []*model.Menu
	var apis []*model.Api
	
	if len(menuIds) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ? AND is_deleted = ?", menuIds, 0).Find(&menus).Error; err != nil {
			return nil, fmt.Errorf("查询菜单失败: %v", err)
		}
		role.Menus = menus
	}

	if len(apiIds) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ? AND is_deleted = ?", apiIds, 0).Find(&apis).Error; err != nil {
			return nil, fmt.Errorf("查询API失败: %v", err)
		}
		role.Apis = apis
	}

	return role, nil
}
