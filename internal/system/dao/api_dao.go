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
		return gorm.ErrRecordNotFound
	}

	api.CreateTime = time.Now().Unix()
	api.UpdateTime = time.Now().Unix()

	return a.db.WithContext(ctx).Create(api).Error
}

// GetApiById 根据ID获取API记录
func (a *apiDAO) GetApiById(ctx context.Context, id int) (*model.Api, error) {
	var api model.Api

	if err := a.db.WithContext(ctx).Where("id = ? AND is_deleted = 0", id).First(&api).Error; err != nil {
		return nil, err
	}

	return &api, nil
}

// UpdateApi 更新API记录
func (a *apiDAO) UpdateApi(ctx context.Context, api *model.Api) error {
	if api == nil {
		return gorm.ErrRecordNotFound
	}

	// 获取旧的API记录
	oldApi, err := a.GetApiById(ctx, api.ID)
	if err != nil {
		return err
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
		Where("id = ? AND is_deleted = 0", api.ID).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
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
		policies, err := a.enforcer.GetFilteredPolicy(1, oldApi.Path, methodMap[api.Method])
		if err != nil {
			tx.Rollback()
			a.l.Error("获取旧的API策略失败", zap.Error(err))
			return err
		}

		if len(policies) > 0 {
			// 遍历每个策略进行更新
			for _, policy := range policies {
				// 检查旧的策略是否存在
				hasPolicy, err := a.enforcer.HasPolicy(policy[0], oldApi.Path, methodMap[oldApi.Method])
				if err != nil {
					tx.Rollback()
					a.l.Error("检查旧的策略是否存在失败", zap.Error(err))
					return err
				}

				// 如果旧策略存在，需要更新策略
				if hasPolicy {
					// 删除旧的策略
					removed, err := a.enforcer.RemoveFilteredPolicy(1, oldApi.Path, methodMap[oldApi.Method])
					if err != nil {
						tx.Rollback()
						a.l.Error("删除旧的API策略失败", zap.Error(err))
						return err
					}

					if !removed {
						tx.Rollback()
						a.l.Warn("没有找到要删除的API策略", zap.String("path", oldApi.Path))
						return fmt.Errorf("没有找到要删除的API策略: %s", oldApi.Path)
					}

					// 添加新的策略
					_, err = a.enforcer.AddPolicy(policy[0], api.Path, methodMap[api.Method])
					if err != nil {
						tx.Rollback()
						a.l.Error("添加新的API策略失败", zap.Error(err))
						return err
					}

					// 保存策略变更
					if err = a.enforcer.SavePolicy(); err != nil {
						tx.Rollback()
						a.l.Error("保存API策略失败", zap.Error(err))
						return err
					}
				}
			}
		}
	}

	return tx.Commit().Error
}

// DeleteApi 软删除API记录
func (a *apiDAO) DeleteApi(ctx context.Context, id int) error {
	updates := map[string]interface{}{
		"is_deleted":  1,
		"update_time": time.Now().Unix(),
	}

	return a.db.WithContext(ctx).Model(&model.Api{}).Where("id = ? AND is_deleted = 0", id).Updates(updates).Error
}

// ListApis 分页获取API列表
func (a *apiDAO) ListApis(ctx context.Context, page, pageSize int) ([]*model.Api, int, error) {
	var apis []*model.Api
	var total int64

	// 构建基础查询
	db := a.db.WithContext(ctx).Model(&model.Api{}).Where("is_deleted = 0")

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("id ASC").Find(&apis).Error; err != nil {
		return nil, 0, err
	}

	return apis, int(total), nil
}
