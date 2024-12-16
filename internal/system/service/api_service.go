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
)

type ApiService interface {
	CreateApi(ctx context.Context, api *model.Api) error
	GetApiById(ctx context.Context, id int) (*model.Api, error)
	UpdateApi(ctx context.Context, api *model.Api) error
	DeleteApi(ctx context.Context, id int) error
	ListApis(ctx context.Context, page, pageSize int) ([]*model.Api, int, error)
}

type apiService struct {
	l   *zap.Logger
	dao dao.ApiDAO
}

func NewApiService(l *zap.Logger, dao dao.ApiDAO) ApiService {
	return &apiService{
		l:   l,
		dao: dao,
	}
}

// CreateApi 创建新的API
func (a *apiService) CreateApi(ctx context.Context, api *model.Api) error {
	if api == nil {
		a.l.Warn("API不能为空")
		return errors.New("api不能为空")
	}

	return a.dao.CreateApi(ctx, api)
}

// GetApiById 根据ID获取API
func (a *apiService) GetApiById(ctx context.Context, id int) (*model.Api, error) {
	if id <= 0 {
		a.l.Warn("API ID无效", zap.Int("ID", id))
		return nil, errors.New("api id无效")
	}

	return a.dao.GetApiById(ctx, id)
}

// UpdateApi 更新API信息
func (a *apiService) UpdateApi(ctx context.Context, api *model.Api) error {
	if api == nil {
		a.l.Warn("API不能为空")
		return errors.New("api不能为空")
	}

	return a.dao.UpdateApi(ctx, api)
}

// DeleteApi 删除指定ID的API
func (a *apiService) DeleteApi(ctx context.Context, id int) error {
	if id <= 0 {
		a.l.Warn("API ID无效", zap.Int("ID", id))
		return errors.New("api id无效")
	}

	return a.dao.DeleteApi(ctx, id)
}

// ListApis 分页获取API列表
func (a *apiService) ListApis(ctx context.Context, page, pageSize int) ([]*model.Api, int, error) {
	if page < 1 || pageSize < 1 {
		a.l.Warn("分页参数无效", zap.Int("页码", page), zap.Int("每页数量", pageSize))
		return nil, 0, errors.New("分页参数无效")
	}

	return a.dao.ListApis(ctx, page, pageSize)
}
