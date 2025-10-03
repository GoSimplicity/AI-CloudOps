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
	CreateApi(ctx context.Context, req *model.CreateApiRequest) error
	GetApiById(ctx context.Context, id int) (*model.Api, error)
	UpdateApi(ctx context.Context, req *model.UpdateApiRequest) error
	DeleteApi(ctx context.Context, id int) error
	ListApis(ctx context.Context, req *model.ListApisRequest) (model.ListResp[*model.Api], error)
	GetApiStatistics(ctx context.Context) (*model.ApiStatistics, error)
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
func (s *apiService) CreateApi(ctx context.Context, req *model.CreateApiRequest) error {
	if req == nil {
		s.l.Warn("API不能为空")
		return errors.New("api不能为空")
	}

	return s.dao.CreateApi(ctx, s.buildCreateApi(req))
}

// GetApiById 根据ID获取API
func (s *apiService) GetApiById(ctx context.Context, id int) (*model.Api, error) {
	if id <= 0 {
		s.l.Warn("API ID无效", zap.Int("ID", id))
		return nil, errors.New("api id无效")
	}

	return s.dao.GetApiById(ctx, id)
}

// UpdateApi 更新API信息
func (s *apiService) UpdateApi(ctx context.Context, req *model.UpdateApiRequest) error {
	if req == nil {
		s.l.Warn("API不能为空")
		return errors.New("api不能为空")
	}

	return s.dao.UpdateApi(ctx, s.buildUpdateApi(req))
}

// DeleteApi 删除指定ID的API
func (s *apiService) DeleteApi(ctx context.Context, id int) error {
	if id <= 0 {
		s.l.Warn("API ID无效", zap.Int("ID", id))
		return errors.New("api id无效")
	}

	return s.dao.DeleteApi(ctx, id)
}

// ListApis 分页获取API列表
func (s *apiService) ListApis(ctx context.Context, req *model.ListApisRequest) (model.ListResp[*model.Api], error) {
	if req.Page < 1 || req.Size < 1 {
		s.l.Warn("分页参数无效", zap.Int("页码", req.Page), zap.Int("每页数量", req.Size))
		return model.ListResp[*model.Api]{}, errors.New("分页参数无效")
	}

	apis, total, err := s.dao.ListApis(ctx, req.Page, req.Size, req.Search, req.IsPublic, req.Method)
	if err != nil {
		s.l.Error("获取API列表失败", zap.Error(err))
		return model.ListResp[*model.Api]{}, err
	}

	return model.ListResp[*model.Api]{
		Items: apis,
		Total: total,
	}, nil
}

func (s *apiService) buildCreateApi(req *model.CreateApiRequest) *model.Api {
	return &model.Api{
		Name:        req.Name,
		Path:        req.Path,
		Method:      int8(req.Method),
		Description: req.Description,
		Version:     req.Version,
		Category:    int8(req.Category),
		IsPublic:    int8(req.IsPublic),
	}
}

func (s *apiService) buildUpdateApi(req *model.UpdateApiRequest) *model.Api {
	return &model.Api{
		Model: model.Model{
			ID: req.ID,
		},
		Name:        req.Name,
		Path:        req.Path,
		Method:      int8(req.Method),
		Description: req.Description,
		Version:     req.Version,
		Category:    int8(req.Category),
		IsPublic:    int8(req.IsPublic),
	}
}

func (s *apiService) GetApiStatistics(ctx context.Context) (*model.ApiStatistics, error) {
	statistics, err := s.dao.GetApiStatistics(ctx)
	if err != nil {
		s.l.Error("获取API统计失败", zap.Error(err))
		return nil, err
	}

	return statistics, nil
}
