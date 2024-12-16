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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model" // 添加model包导入
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PermissionDAO interface {
	AssignRole(ctx context.Context, roleId int, menuIds []int, apiIds []int) error
	AssignRoleToUser(ctx context.Context, userId int, roleIds []int, menuIds []int, apiIds []int) error
	AssignRoleToUsers(ctx context.Context, userIds []int, roleIds []int, menuIds []int, apiIds []int) error

	RemoveUserPermissions(ctx context.Context, userId int) error
	RemoveRolePermissions(ctx context.Context, roleId int) error
	RemoveUsersPermissions(ctx context.Context, userIds []int) error
}

type permissionDAO struct {
	db       *gorm.DB
	l        *zap.Logger
	enforcer *casbin.Enforcer
	apiDao   ApiDAO
}

func NewPermissionDAO(db *gorm.DB, l *zap.Logger, enforcer *casbin.Enforcer, apiDao ApiDAO) PermissionDAO {
	return &permissionDAO{
		db:       db,
		l:        l,
		enforcer: enforcer,
		apiDao:   apiDao,
	}
}

// AssignRole 为角色分配权限
func (p *permissionDAO) AssignRole(ctx context.Context, roleId int, menuIds []int, apiIds []int) error {
	const batchSize = 1000

	if roleId <= 0 {
		return errors.New("无效的角色ID")
	}

	var role model.Role
	if err := p.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", roleId, 0).First(&role).Error; err != nil {
		return fmt.Errorf("获取角色失败: %v", err)
	}

	if role.ID <= 0 {
		return errors.New("角色不存在")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先移除角色现有的权限
		if _, err := p.enforcer.RemoveFilteredPolicy(0, role.Name); err != nil {
			return fmt.Errorf("移除角色现有权限失败: %v", err)
		}

		// 添加菜单权限
		if err := p.assignMenuPermissions(role.Name, menuIds, batchSize); err != nil {
			return err
		}

		// 添加API权限
		if err := p.assignAPIPermissions(ctx, role.Name, apiIds, batchSize); err != nil {
			return err
		}

		// 加载最新的策略
		if err := p.enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("加载策略失败: %v", err)
		}

		return nil
	})
}

// AssignRoleToUser 为用户分配角色和权限
func (p *permissionDAO) AssignRoleToUser(ctx context.Context, userId int, roleIds []int, menuIds []int, apiIds []int) error {
	const batchSize = 1000

	if userId <= 0 {
		return errors.New("无效的用户ID")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取角色信息
		var roles []*model.Role
		if err := tx.Where("id IN ? AND is_deleted = ?", roleIds, 0).Find(&roles).Error; err != nil {
			return fmt.Errorf("获取角色信息失败: %v", err)
		}

		if len(roles) == 0 && len(roleIds) > 0 {
			return errors.New("未找到有效的角色")
		}

		// 先移除用户现有的角色关联和权限
		userStr := fmt.Sprintf("%d", userId)
		if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
			return fmt.Errorf("移除用户现有角色关联失败: %v", err)
		}
		if _, err := p.enforcer.RemoveFilteredPolicy(0, userStr); err != nil {
			return fmt.Errorf("移除用户现有权限失败: %v", err)
		}

		// 构建角色关联策略
		rolePolicies := make([][]string, 0, len(roles))
		for _, role := range roles {
			rolePolicies = append(rolePolicies, []string{userStr, role.Name})
		}

		// 更新用户的角色列表
		roleIdsStr := "[]"
		if len(roleIds) > 0 {
			roleIdsBytes, err := json.Marshal(roleIds)
			if err != nil {
				return fmt.Errorf("序列化角色ID列表失败: %v", err)
			}
			roleIdsStr = string(roleIdsBytes)
		}

		// 更新用户表中的角色字段
		if err := tx.Model(&model.User{}).Where("id = ?", userId).UpdateColumn("roles", roleIdsStr).Error; err != nil {
			return fmt.Errorf("更新用户角色列表失败: %v", err)
		}

		// 添加用户角色关联
		if len(rolePolicies) > 0 {
			if _, err := p.enforcer.AddGroupingPolicies(rolePolicies); err != nil {
				return fmt.Errorf("添加用户角色关联失败: %v", err)
			}
		}

		// 添加菜单权限
		if len(menuIds) > 0 {
			if err := p.assignMenuPermissions(userStr, menuIds, batchSize); err != nil {
				return err
			}
		}

		// 添加API权限
		if len(apiIds) > 0 {
			if err := p.assignAPIPermissions(ctx, userStr, apiIds, batchSize); err != nil {
				return err
			}
		}

		// 加载最新的策略
		if err := p.enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("加载策略失败: %v", err)
		}

		return nil
	})
}

// RemoveUserPermissions 移除用户权限
func (p *permissionDAO) RemoveUserPermissions(ctx context.Context, userId int) error {
	if userId <= 0 {
		return errors.New("无效的用户ID")
	}

	userStr := fmt.Sprintf("%d", userId)

	// 移除该用户的所有权限策略
	if _, err := p.enforcer.RemoveFilteredPolicy(0, userStr); err != nil {
		return fmt.Errorf("移除用户权限策略失败: %v", err)
	}

	// 移除该用户的所有角色关联
	if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
		return fmt.Errorf("移除用户角色关联失败: %v", err)
	}

	// 清空用户的角色字段
	if err := p.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userId).UpdateColumn("roles", "[]").Error; err != nil {
		return fmt.Errorf("清空用户角色列表失败: %v", err)
	}

	// 重新加载策略
	if err := p.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("加载策略失败: %v", err)
	}

	return nil
}

// RemoveRolePermissions 批量移除角色对应api权限
func (p *permissionDAO) RemoveRolePermissions(ctx context.Context, roleId int) error {
	if roleId <= 0 {
		return nil
	}

	// 查询角色名称
	var role model.Role
	if err := p.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", roleId, 0).First(&role).Error; err != nil {
		return fmt.Errorf("查询角色失败: %v", err)
	}

	// 移除该角色的所有API权限策略
	if _, err := p.enforcer.RemoveFilteredPolicy(0, role.Name); err != nil {
		return fmt.Errorf("移除角色API权限策略失败: %v", err)
	}

	// 重新加载策略
	if err := p.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("加载策略失败: %v", err)
	}

	return nil
}

// assignMenuPermissions 分配菜单权限
func (p *permissionDAO) assignMenuPermissions(roleName string, menuIds []int, batchSize int) error {
	if roleName == "" {
		return errors.New("角色名称不能为空")
	}

	// 如果菜单ID列表为空,直接返回
	if len(menuIds) == 0 {
		return nil
	}

	// 构建casbin策略规则
	policies := make([][]string, 0, len(menuIds))
	for _, menuId := range menuIds {
		if menuId <= 0 {
			return fmt.Errorf("无效的菜单ID: %d", menuId)
		}
		policies = append(policies, []string{roleName, fmt.Sprintf("menu:%d", menuId), "read"})
	}

	// 批量添加策略
	return p.batchAddPolicies(policies, batchSize)
}

// assignAPIPermissions 分配API权限
func (p *permissionDAO) assignAPIPermissions(ctx context.Context, roleName string, apiIds []int, batchSize int) error {
	if roleName == "" {
		return errors.New("角色名称不能为空")
	}

	// 如果API ID列表为空,直接返回
	if len(apiIds) == 0 {
		return nil
	}

	// HTTP方法映射表
	methodMap := map[int]string{
		1: "GET",
		2: "POST",
		3: "PUT",
		4: "DELETE",
		5: "PATCH",
		6: "OPTIONS",
		7: "HEAD",
	}

	// 构建casbin策略规则
	policies := make([][]string, 0, len(apiIds))
	for _, apiId := range apiIds {
		if apiId <= 0 {
			return fmt.Errorf("无效的API ID: %d", apiId)
		}

		// 获取API信息
		api, err := p.apiDao.GetApiById(ctx, apiId)
		if err != nil {
			return fmt.Errorf("获取API信息失败: %v", err)
		}

		if api == nil {
			return fmt.Errorf("API不存在: %d", apiId)
		}

		// 获取HTTP方法
		method, ok := methodMap[api.Method]
		if !ok {
			return fmt.Errorf("无效的HTTP方法: %d", api.Method)
		}

		policies = append(policies, []string{roleName, fmt.Sprintf("api:%d", api.ID), method})
	}

	// 批量添加策略
	return p.batchAddPolicies(policies, batchSize)
}

// batchAddPolicies 批量添加策略
func (p *permissionDAO) batchAddPolicies(policies [][]string, batchSize int) error {
	if len(policies) == 0 {
		return nil
	}

	if batchSize <= 0 {
		return errors.New("无效的批次大小")
	}

	// 按批次处理策略规则
	for i := 0; i < len(policies); i += batchSize {
		end := i + batchSize
		if end > len(policies) {
			end = len(policies)
		}

		// 添加一批策略规则
		if _, err := p.enforcer.AddPolicies(policies[i:end]); err != nil {
			return fmt.Errorf("添加权限策略失败: %v", err)
		}
	}

	return nil
}

// AssignRoleToUsers 批量为用户分配角色和权限
func (p *permissionDAO) AssignRoleToUsers(ctx context.Context, userIds []int, roleIds []int, menuIds []int, apiIds []int) error {
	const batchSize = 1000

	if len(userIds) == 0 {
		return errors.New("用户ID列表不能为空")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取角色信息
		var roles []*model.Role
		if err := tx.Where("id IN ? AND is_deleted = ?", roleIds, 0).Find(&roles).Error; err != nil {
			return fmt.Errorf("获取角色信息失败: %v", err)
		}

		if len(roles) == 0 && len(roleIds) > 0 {
			return errors.New("未找到有效的角色")
		}

		// 序列化角色ID列表
		roleIdsStr := "[]"
		if len(roleIds) > 0 {
			roleIdsBytes, err := json.Marshal(roleIds)
			if err != nil {
				return fmt.Errorf("序列化角色ID列表失败: %v", err)
			}
			roleIdsStr = string(roleIdsBytes)
		}

		// 为每个用户添加角色和权限
		for _, userId := range userIds {
			userStr := fmt.Sprintf("%d", userId)

			// 先移除用户现有的角色关联和权限
			if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
				return fmt.Errorf("移除用户现有角色关联失败: %v", err)
			}
			if _, err := p.enforcer.RemoveFilteredPolicy(0, userStr); err != nil {
				return fmt.Errorf("移除用户现有权限失败: %v", err)
			}

			// 更新用户表中的角色字段
			if err := tx.Model(&model.User{}).Where("id = ?", userId).UpdateColumn("roles", roleIdsStr).Error; err != nil {
				return fmt.Errorf("更新用户角色列表失败: %v", err)
			}

			// 构建角色关联策略
			rolePolicies := make([][]string, 0, len(roles))
			for _, role := range roles {
				rolePolicies = append(rolePolicies, []string{userStr, role.Name})
			}

			// 添加用户角色关联
			if len(rolePolicies) > 0 {
				if _, err := p.enforcer.AddGroupingPolicies(rolePolicies); err != nil {
					return fmt.Errorf("添加用户角色关联失败: %v", err)
				}
			}

			// 添加菜单权限
			if len(menuIds) > 0 {
				if err := p.assignMenuPermissions(userStr, menuIds, batchSize); err != nil {
					return err
				}
			}

			// 添加API权限
			if len(apiIds) > 0 {
				if err := p.assignAPIPermissions(ctx, userStr, apiIds, batchSize); err != nil {
					return err
				}
			}
		}

		// 加载最新的策略
		if err := p.enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("加载策略失败: %v", err)
		}

		return nil
	})
}

// RemoveUsersPermissions 批量移除用户权限
func (p *permissionDAO) RemoveUsersPermissions(ctx context.Context, userIds []int) error {
	if len(userIds) == 0 {
		return errors.New("用户ID列表不能为空")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 移除所有用户的权限
		for _, userId := range userIds {
			userStr := fmt.Sprintf("%d", userId)

			// 移除用户的所有角色关联
			if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
				return fmt.Errorf("移除用户角色关联失败: %v", err)
			}

			// 移除用户的所有权限策略
			if _, err := p.enforcer.RemoveFilteredPolicy(0, userStr); err != nil {
				return fmt.Errorf("移除用户权限策略失败: %v", err)
			}

			// 清空用户的角色字段
			if err := tx.Model(&model.User{}).Where("id = ?", userId).UpdateColumn("roles", "[]").Error; err != nil {
				return fmt.Errorf("清空用户角色列表失败: %v", err)
			}
		}

		// 加载最新的策略
		if err := p.enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("加载策略失败: %v", err)
		}

		return nil
	})
}
