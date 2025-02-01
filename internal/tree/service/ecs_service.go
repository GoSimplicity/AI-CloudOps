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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

type EcsService interface {
	GetEcsUnbindList(ctx context.Context) ([]*model.ResourceEcs, error)
	GetEcsList(ctx context.Context) ([]*model.ResourceEcs, error)
	BindEcs(ctx context.Context, ecsID int, treeNodeID int) error
	UnBindEcs(ctx context.Context, ecsID int, treeNodeID int) error
	GetEcsById(ctx context.Context, id int) (*model.ResourceEcs, error)
}

type ecsService struct {
	logger  *zap.Logger
	ecsDao  dao.TreeEcsDAO
	nodeDao dao.TreeNodeDAO
}

func NewEcsService(logger *zap.Logger, ecsDao dao.TreeEcsDAO, nodeDao dao.TreeNodeDAO) EcsService {
	return &ecsService{
		logger:  logger,
		ecsDao:  ecsDao,
		nodeDao: nodeDao,
	}
}

func (s *ecsService) BindEcs(ctx context.Context, ecsID int, treeNodeID int) error {
	ecs, err := s.ecsDao.GetByID(ctx, ecsID)
	if err != nil {
		s.logger.Error("BindEcs 获取 ECS 失败", zap.Error(err))
		return err
	}

	node, err := s.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		s.logger.Error("BindEcs 获取树节点失败", zap.Error(err))
		return err
	}

	return s.ecsDao.AddBindNodes(ctx, ecs, node)
}

func (s *ecsService) GetEcsList(ctx context.Context) ([]*model.ResourceEcs, error) {
	list, err := s.ecsDao.GetAll(ctx)
	if err != nil {
		s.logger.Error("GetEcsList failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (s *ecsService) GetEcsUnbindList(ctx context.Context) ([]*model.ResourceEcs, error) {
	ecs, err := s.ecsDao.GetAll(ctx)
	if err != nil {
		s.logger.Error("GetEcsUnbindList failed", zap.Error(err))
		return nil, err
	}

	// 筛选出未绑定的 ECS 资源
	unbindEcs := make([]*model.ResourceEcs, 0, len(ecs))
	for _, e := range ecs {
		if len(e.BindNodes) == 0 {
			unbindEcs = append(unbindEcs, e)
		}
	}

	return unbindEcs, nil
}

func (s *ecsService) UnBindEcs(ctx context.Context, ecsID int, treeNodeID int) error {
	ecs, err := s.ecsDao.GetByID(ctx, ecsID)
	if err != nil {
		s.logger.Error("UnBindEcs 获取 ECS 失败", zap.Error(err))
		return err
	}

	node, err := s.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		s.logger.Error("UnBindEcs 获取树节点失败", zap.Error(err))
		return err
	}

	return s.ecsDao.RemoveBindNodes(ctx, ecs, node)
}

// GetEcsById 根据ID获取ECS资源
func (s *ecsService) GetEcsById(ctx context.Context, id int) (*model.ResourceEcs, error) {
	ecs, err := s.ecsDao.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("获取ECS资源失败", zap.Error(err))
		return nil, err
	}

	return ecs, nil
}
