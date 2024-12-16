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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

var (
	ErrResourceNotFound = errors.New("资源未找到")
)

type EcsResourceService interface {
	CreateEcsResource(ctx context.Context, obj *model.ResourceEcs) error
	UpdateEcsResource(ctx context.Context, obj *model.ResourceEcs) error
	DeleteEcsResource(ctx context.Context, id int) error
	GetAllResourcesByType(ctx context.Context, nid int, resourceType string, page int, size int) ([]*model.ResourceTree, error)
}

type ecsResourceService struct {
	logger         *zap.Logger
	ecsResourceDao dao.TreeEcsResourceDAO
	ecsDao         dao.TreeEcsDAO
}

func NewEcsResourceService(logger *zap.Logger, ecsResourceDao dao.TreeEcsResourceDAO, ecsDao dao.TreeEcsDAO) EcsResourceService {
	return &ecsResourceService{
		logger:         logger,
		ecsResourceDao: ecsResourceDao,
		ecsDao:         ecsDao,
	}
}

func (s *ecsResourceService) CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	// 生成资源的唯一哈希值
	resource.Hash = generateResourceHash(resource)
	if err := s.ecsResourceDao.Create(ctx, resource); err != nil {
		s.logger.Error("CreateEcsResource 创建 ECS 资源失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *ecsResourceService) DeleteEcsResource(ctx context.Context, id int) error {
	// 获取资源
	resource, err := s.ecsDao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			s.logger.Warn("DeleteEcsResource: 资源未找到", zap.Int("id", id))
			return fmt.Errorf("资源未找到: %w", err)
		}
		s.logger.Error("DeleteEcsResource 获取 ECS 资源失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取 ECS 资源失败: %w", err)
	}

	// 检查资源是否被绑定
	if len(resource.BindNodes) > 0 {
		s.logger.Warn("DeleteEcsResource: 资源被绑定，无法删除", zap.Int("id", id), zap.Any("bindNodes", resource.BindNodes))
		return ErrResourceBound
	}

	// 删除资源
	if err := s.ecsResourceDao.Delete(ctx, id); err != nil {
		s.logger.Error("DeleteEcsResource 删除 ECS 资源失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("删除 ECS 资源失败: %w", err)
	}

	s.logger.Info("DeleteEcsResource 删除 ECS 资源成功", zap.Int("id", id))
	return nil
}

func (s *ecsResourceService) GetAllResourcesByType(ctx context.Context, nid int, resourceType string, page, size int) ([]*model.ResourceTree, error) {
	// 获取子节点 ID Map
	nodeIdsMap := s.getChildrenTreeNodeIds(ctx, nid)

	// 查询对应类型的资源
	resources := make([]*model.ResourceTree, 0)

	switch resourceType {
	case "ecs":
		ecs, err := s.ecsDao.GetAll(ctx)
		if err != nil {
			s.logger.Error("GetEcsList failed", zap.Error(err))
			return nil, err
		}
		for _, e := range ecs {
			for _, n := range e.BindNodes {
				if _, ok := nodeIdsMap[n.ID]; ok {
					resources = append(resources, &e.ResourceTree)
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", resourceType)
	}

	// 分页处理
	offset := (page - 1) * size
	if offset >= len(resources) {
		return nil, nil
	}

	end := offset + size
	if end > len(resources) {
		end = len(resources)
	}

	return resources[offset:end], nil
}

func (s *ecsResourceService) UpdateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	if err := s.ecsResourceDao.Update(ctx, resource); err != nil {
		s.logger.Error("UpdateEcsResource 更新 ECS 资源失败", zap.Error(err))
		return err
	}

	return nil
}

// 辅助方法
func (s *ecsResourceService) getChildrenTreeNodeIds(ctx context.Context, nid int) map[int]struct{} {
	// TODO: 实现获取子节点ID的逻辑
	return make(map[int]struct{})
}
