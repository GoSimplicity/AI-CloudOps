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

type ElbService interface {
	GetElbUnbindList(ctx context.Context) ([]*model.ResourceElb, error)
	GetElbList(ctx context.Context) ([]*model.ResourceElb, error)
	BindElb(ctx context.Context, elbID int, treeNodeID int) error
	UnBindElb(ctx context.Context, elbID int, treeNodeID int) error
}

type elbService struct {
	logger  *zap.Logger
	elbDao  dao.TreeElbDAO
	nodeDao dao.TreeNodeDAO
}

func NewElbService(logger *zap.Logger, elbDao dao.TreeElbDAO, nodeDao dao.TreeNodeDAO) ElbService {
	return &elbService{
		logger:  logger,
		elbDao:  elbDao,
		nodeDao: nodeDao,
	}
}

func (s *elbService) BindElb(ctx context.Context, elbID int, treeNodeID int) error {
	elb, err := s.elbDao.GetByIDNoPreload(ctx, elbID)
	if err != nil {
		s.logger.Error("BindElb 获取 ELB 失败", zap.Error(err))
		return err
	}

	node, err := s.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		s.logger.Error("BindElb 获取树节点失败", zap.Error(err))
		return err
	}

	return s.elbDao.AddBindNodes(ctx, elb, node)
}

func (s *elbService) GetElbList(ctx context.Context) ([]*model.ResourceElb, error) {
	list, err := s.elbDao.GetAll(ctx)
	if err != nil {
		s.logger.Error("GetElbList failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (s *elbService) GetElbUnbindList(ctx context.Context) ([]*model.ResourceElb, error) {
	elb, err := s.elbDao.GetAll(ctx)
	if err != nil {
		s.logger.Error("GetElbUnbindList failed", zap.Error(err))
		return nil, err
	}

	// 筛选出未绑定的 ELB 资源
	unbindElb := make([]*model.ResourceElb, 0, len(elb))
	for _, e := range elb {
		if len(e.BindNodes) == 0 {
			unbindElb = append(unbindElb, e)
		}
	}

	return unbindElb, nil
}

func (s *elbService) UnBindElb(ctx context.Context, elbID int, treeNodeID int) error {
	elb, err := s.elbDao.GetByIDNoPreload(ctx, elbID)
	if err != nil {
		s.logger.Error("UnBindElb 获取 ELB 失败", zap.Error(err))
		return err
	}

	node, err := s.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		s.logger.Error("UnBindElb 获取树节点失败", zap.Error(err))
		return err
	}

	return s.elbDao.RemoveBindNodes(ctx, elb, node)
}
