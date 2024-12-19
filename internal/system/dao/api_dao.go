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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
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
	db *gorm.DB
	l  *zap.Logger
}

func NewApiDAO(db *gorm.DB, l *zap.Logger) ApiDAO {
	return &apiDAO{
		db: db,
		l:  l,
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

	return a.db.WithContext(ctx).
		Model(&model.Api{}).
		Where("id = ? AND is_deleted = 0", api.ID).
		Updates(updates).Error
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
