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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PermissionDAO interface {
	AssignRole(ctx context.Context, roleId int, apiIds []int) error
	AssignRoleToUser(ctx context.Context, userId int, roleIds []int, apiIds []int) error
	AssignRoleToUsers(ctx context.Context, userIds []int, roleIds []int) error

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
func (p *permissionDAO) AssignRole(ctx context.Context, roleId int, apiIds []int) error {
	const batchSize = 1000

	if roleId <= 0 {
		return errors.New("无效的角色ID")
	}

	var role model.Role
	if err := p.db.WithContext(ctx).Where("id = ? AND deleted_at = ?", roleId, 0).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return fmt.Errorf("获取角色失败: %v", err)
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先移除角色现有的权限
		if _, err := p.enforcer.RemoveFilteredPolicy(0, role.Name); err != nil {
			return fmt.Errorf("移除角色现有权限失败: %v", err)
		}

		// 添加API权限
		if err := p.assignAPIPermissions(ctx, role.Name, apiIds, batchSize); err != nil {
			return err
		}

		if err := p.enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("加载策略失败: %v", err)
		}

		return nil
	})
}

// AssignRoleToUser 为用户分配角色和权限
func (p *permissionDAO) AssignRoleToUser(ctx context.Context, userId int, roleIds []int, apiIds []int) error {
	const batchSize = 1000

	if userId <= 0 {
		return errors.New("无效的用户ID")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取用户信息
		var user model.User
		if err := tx.First(&user, userId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("用户不存在")
			}
			return fmt.Errorf("获取用户信息失败: %v", err)
		}

		// 获取角色信息
		var roles []*model.Role
		if len(roleIds) > 0 {
			if err := tx.Where("id IN ? AND deleted_at = ?", roleIds, 0).Find(&roles).Error; err != nil {
				return fmt.Errorf("获取角色信息失败: %v", err)
			}
			if len(roles) != len(roleIds) {
				return errors.New("部分角色不存在或已被删除")
			}
		}

		// 先移除用户现有的角色关联和权限
		userStr := fmt.Sprintf("%d", userId)
		if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
			return fmt.Errorf("移除用户现有角色关联失败: %v", err)
		}
		if _, err := p.enforcer.RemoveFilteredPolicy(0, userStr); err != nil {
			return fmt.Errorf("移除用户现有权限失败: %v", err)
		}

		// 更新用户的角色关联
		if err := tx.Model(&user).Association("Roles").Replace(roles); err != nil {
			return fmt.Errorf("更新用户角色关联失败: %v", err)
		}

		// 更新用户的API关联
		if len(apiIds) > 0 {
			var apis []*model.Api
			if err := tx.Where("id IN ? AND deleted_at = ?", apiIds, 0).Find(&apis).Error; err != nil {
				return fmt.Errorf("获取API信息失败: %v", err)
			}
			if len(apis) != len(apiIds) {
				return errors.New("部分API不存在或已被删除")
			}
			if err := tx.Model(&user).Association("Apis").Replace(apis); err != nil {
				return fmt.Errorf("更新用户API关联失败: %v", err)
			}
		}

		// 添加角色关联策略
		if len(roles) > 0 {
			rolePolicies := make([][]string, 0, len(roles))
			for _, role := range roles {
				rolePolicies = append(rolePolicies, []string{userStr, role.Name})
			}
			if _, err := p.enforcer.AddGroupingPolicies(rolePolicies); err != nil {
				return fmt.Errorf("添加用户角色关联失败: %v", err)
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

	// 不允许删除userId为1的权限
	if userId == 1 {
		return errors.New("不允许删除超级管理员权限")
	}

	userStr := fmt.Sprintf("%d", userId)

	// 移除用户的角色关联
	if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
		return fmt.Errorf("移除用户角色关联失败: %v", err)
	}

	// 移除用户的所有权限策略
	if _, err := p.enforcer.RemoveFilteredPolicy(0, userStr); err != nil {
		return fmt.Errorf("移除用户权限策略失败: %v", err)
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
		return errors.New("无效的角色ID")
	}

	// 查询角色名称
	var role model.Role
	if err := p.db.WithContext(ctx).Where("id = ? AND deleted_at = ?", roleId, 0).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("角色不存在")
		}
		return fmt.Errorf("查询角色失败: %v", err)
	}

	// 直接移除角色的所有权限策略
	if _, err := p.enforcer.RemoveFilteredPolicy(0, role.Name); err != nil {
		return fmt.Errorf("移除角色权限失败: %v", err)
	}

	// 重新加载策略
	if err := p.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("加载策略失败: %v", err)
	}

	return nil
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
		method, ok := methodMap[int(api.Method)]
		if !ok {
			return fmt.Errorf("无效的HTTP方法: %d", api.Method)
		}

		policies = append(policies, []string{roleName, api.Path, method})
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

// AssignRoleToUsers 批量为用户分配角色
func (p *permissionDAO) AssignRoleToUsers(ctx context.Context, userIds []int, roleIds []int) error {
	const batchSize = 1000

	if len(userIds) == 0 {
		return errors.New("用户ID列表不能为空")
	}

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取角色信息
		var roles []*model.Role
		if len(roleIds) > 0 {
			if err := tx.Where("id IN ? AND deleted_at = ?", roleIds, 0).Find(&roles).Error; err != nil {
				return fmt.Errorf("获取角色信息失败: %v", err)
			}
			if len(roles) != len(roleIds) {
				return errors.New("部分角色不存在或已被删除")
			}
		}

		// 为每个用户添加角色
		for _, userId := range userIds {
			userStr := fmt.Sprintf("%d", userId)

			// 获取用户信息
			var user model.User
			if err := tx.First(&user, userId).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("用户不存在: %d", userId)
				}
				return fmt.Errorf("获取用户信息失败: %v", err)
			}

			// 先移除用户现有的角色关联
			if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
				return fmt.Errorf("移除用户现有角色关联失败: %v", err)
			}

			// 更新用户的角色关联
			if err := tx.Model(&user).Association("Roles").Replace(roles); err != nil {
				return fmt.Errorf("更新用户角色关联失败: %v", err)
			}

			// 添加角色关联策略
			if len(roles) > 0 {
				rolePolicies := make([][]string, 0, len(roles))
				for _, role := range roles {
					rolePolicies = append(rolePolicies, []string{userStr, role.Name})
				}
				if _, err := p.enforcer.AddGroupingPolicies(rolePolicies); err != nil {
					return fmt.Errorf("添加用户角色关联失败: %v", err)
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
		for _, userId := range userIds {
			// 不允许删除userId为1的权限
			if userId == 1 {
				continue
			}

			userStr := fmt.Sprintf("%d", userId)

			// 获取用户信息
			var user model.User
			if err := tx.First(&user, userId).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("用户不存在: %d", userId)
				}
				return fmt.Errorf("获取用户信息失败: %v", err)
			}

			// 获取用户当前的API列表
			var apis []*model.Api
			if err := tx.Model(&user).Association("Apis").Find(&apis); err != nil {
				return fmt.Errorf("获取用户API关联失败: %v", err)
			}

			// 移除每个API的权限策略
			for _, api := range apis {
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
				method := methodMap[int(api.Method)]

				if _, err := p.enforcer.RemovePolicy(userStr, api.Path, method); err != nil {
					return fmt.Errorf("移除用户API权限失败: %v", err)
				}
			}

			// 移除用户的角色关联
			if _, err := p.enforcer.RemoveFilteredGroupingPolicy(0, userStr); err != nil {
				return fmt.Errorf("移除用户角色关联失败: %v", err)
			}

			// 清空用户的关联
			if err := tx.Model(&user).Association("Roles").Clear(); err != nil {
				return fmt.Errorf("清空用户角色关联失败: %v", err)
			}

			if err := tx.Model(&user).Association("Apis").Clear(); err != nil {
				return fmt.Errorf("清空用户API关联失败: %v", err)
			}
		}

		// 加载最新的策略
		if err := p.enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("加载策略失败: %v", err)
		}

		return nil
	})
}
