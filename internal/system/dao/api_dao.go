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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ApiDAO interface {
	CreateApi(ctx context.Context, api *model.Api) error
	GetApiById(ctx context.Context, id int) (*model.Api, error)
	UpdateApi(ctx context.Context, api *model.Api) error
	DeleteApi(ctx context.Context, id int) error
	ListApis(ctx context.Context, page, pageSize int) ([]*model.Api, int, error)
}

type apiDAO struct {
	db       *gorm.DB
	enforcer *casbin.Enforcer
	l        *zap.Logger
}

func NewApiDAO(db *gorm.DB, enforcer *casbin.Enforcer, l *zap.Logger) ApiDAO {
	return &apiDAO{
		db:       db,
		enforcer: enforcer,
		l:        l,
	}
}

// CreateApi 创建新的API记录
func (a *apiDAO) CreateApi(ctx context.Context, api *model.Api) error {
	if api == nil {
		return errors.New("API对象不能为空")
	}

	if api.Path == "" {
		return errors.New("API路径不能为空")
	}

	if api.Method <= 0 {
		return errors.New("无效的HTTP方法")
	}

	api.CreatedAt = time.Now().Unix()
	api.UpdatedAt = time.Now().Unix()
	api.DeletedAt = 0

	return a.db.WithContext(ctx).Create(api).Error
}

// GetApiById 根据ID获取API记录
func (a *apiDAO) GetApiById(ctx context.Context, id int) (*model.Api, error) {
	if id <= 0 {
		return nil, errors.New("无效的API ID")
	}

	var api model.Api
	if err := a.db.WithContext(ctx).Where("id = ? AND deleted_at = ?", id, 0).First(&api).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询API失败: %v", err)
	}

	return &api, nil
}

// UpdateApi 更新API记录
func (a *apiDAO) UpdateApi(ctx context.Context, api *model.Api) error {
	if api == nil {
		return errors.New("API对象不能为空")
	}

	if api.ID <= 0 {
		return errors.New("无效的API ID")
	}

	if api.Path == "" {
		return errors.New("API路径不能为空")
	}

	if api.Method <= 0 {
		return errors.New("无效的HTTP方法")
	}

	// 获取旧的API记录
	oldApi, err := a.GetApiById(ctx, api.ID)
	if err != nil {
		return err
	}

	if oldApi == nil {
		return errors.New("API不存在")
	}

	updates := map[string]interface{}{
		"name":        api.Name,
		"path":        api.Path,
		"method":      api.Method,
		"description": api.Description,
		"version":     api.Version,
		"category":    api.Category,
		"is_public":   api.IsPublic,
		"update_time": time.Now().Unix(),
	}

	// 开启事务
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 更新API记录
	if err := tx.Model(&model.Api{}).
		Where("id = ? AND deleted_at = ?", api.ID, 0).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新API失败: %v", err)
	}

	// 如果API路径或方法发生变化，则需要更新casbin策略
	if oldApi.Path != api.Path || oldApi.Method != api.Method {
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

		// 获取所有包含旧路径的策略
		policies, err := a.enforcer.GetFilteredPolicy(1, fmt.Sprintf("%d", oldApi.ID))
		if err != nil {
			tx.Rollback()
			a.l.Error("获取旧的API策略失败", zap.Error(err))
			return fmt.Errorf("获取旧的API策略失败: %v", err)
		}

		if len(policies) > 0 {
			// 遍历每个策略进行更新
			for _, policy := range policies {
				// 删除旧的策略
				removed, err := a.enforcer.RemovePolicy(policy)
				if err != nil {
					tx.Rollback()
					a.l.Error("删除旧的API策略失败", zap.Error(err))
					return fmt.Errorf("删除旧的API策略失败: %v", err)
				}

				if !removed {
					tx.Rollback()
					a.l.Warn("没有找到要删除的API策略", zap.String("path", oldApi.Path))
					return fmt.Errorf("没有找到要删除的API策略: %s", oldApi.Path)
				}

				// 添加新的策略
				newPolicy := []string{policy[0], fmt.Sprintf("%d", api.ID), methodMap[int(api.Method)]}
				_, err = a.enforcer.AddPolicy(newPolicy)
				if err != nil {
					tx.Rollback()
					a.l.Error("添加新的API策略失败", zap.Error(err))
					return fmt.Errorf("添加新的API策略失败: %v", err)
				}
			}

			// 保存策略变更
			if err = a.enforcer.SavePolicy(); err != nil {
				tx.Rollback()
				a.l.Error("保存API策略失败", zap.Error(err))
				return fmt.Errorf("保存API策略失败: %v", err)
			}
		}
	}

	return tx.Commit().Error
}

// DeleteApi 软删除API记录
func (a *apiDAO) DeleteApi(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("无效的API ID")
	}

	updates := map[string]interface{}{
		"deleted_at":  1,
		"update_time": time.Now().Unix(),
	}

	result := a.db.WithContext(ctx).Model(&model.Api{}).Where("id = ? AND deleted_at = ?", id, 0).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("删除API失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("API不存在或已被删除")
	}

	// 删除相关的casbin策略
	policies, err := a.enforcer.GetFilteredPolicy(1, fmt.Sprintf("%d", id))
	if err != nil {
		return fmt.Errorf("获取API策略失败: %v", err)
	}

	if len(policies) > 0 {
		_, err = a.enforcer.RemoveFilteredPolicy(1, fmt.Sprintf("%d", id))
		if err != nil {
			return fmt.Errorf("删除API策略失败: %v", err)
		}

		if err = a.enforcer.SavePolicy(); err != nil {
			return fmt.Errorf("保存策略变更失败: %v", err)
		}
	}

	return nil
}

// ListApis 分页获取API列表
func (a *apiDAO) ListApis(ctx context.Context, page, pageSize int) ([]*model.Api, int, error) {
	if page <= 0 || pageSize <= 0 {
		return nil, 0, errors.New("无效的分页参数")
	}

	var apis []*model.Api
	var total int64

	// 构建基础查询
	db := a.db.WithContext(ctx).Model(&model.Api{}).Where("deleted_at = ?", 0)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取API总数失败: %v", err)
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("id ASC").Find(&apis).Error; err != nil {
		return nil, 0, fmt.Errorf("获取API列表失败: %v", err)
	}

	return apis, int(total), nil
}
