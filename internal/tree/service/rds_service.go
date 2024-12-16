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

type RdsService interface {
	GetRdsUnbindList(ctx context.Context) ([]*model.ResourceRds, error)
	GetRdsList(ctx context.Context) ([]*model.ResourceRds, error)
	BindRds(ctx context.Context, rdsID int, treeNodeID int) error
	UnBindRds(ctx context.Context, rdsID int, treeNodeID int) error
}

type rdsService struct {
	logger  *zap.Logger
	rdsDao  dao.TreeRdsDAO
	nodeDao dao.TreeNodeDAO
}

func NewRdsService(logger *zap.Logger, rdsDao dao.TreeRdsDAO, nodeDao dao.TreeNodeDAO) RdsService {
	return &rdsService{
		logger:  logger,
		rdsDao:  rdsDao,
		nodeDao: nodeDao,
	}
}

func (s *rdsService) BindRds(ctx context.Context, rdsID int, treeNodeID int) error {
	rds, err := s.rdsDao.GetByIDNoPreload(ctx, rdsID)
	if err != nil {
		s.logger.Error("BindRds 获取 RDS 失败", zap.Error(err))
		return err
	}

	node, err := s.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		s.logger.Error("BindRds 获取树节点失败", zap.Error(err))
		return err
	}

	return s.rdsDao.AddBindNodes(ctx, rds, node)
}

func (s *rdsService) GetRdsList(ctx context.Context) ([]*model.ResourceRds, error) {
	list, err := s.rdsDao.GetAll(ctx)
	if err != nil {
		s.logger.Error("GetRdsList failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (s *rdsService) GetRdsUnbindList(ctx context.Context) ([]*model.ResourceRds, error) {
	rds, err := s.rdsDao.GetAll(ctx)
	if err != nil {
		s.logger.Error("GetRdsUnbindList failed", zap.Error(err))
		return nil, err
	}

	// 筛选出未绑定的 RDS 资源
	unbindRds := make([]*model.ResourceRds, 0, len(rds))
	for _, e := range rds {
		if len(e.BindNodes) == 0 {
			unbindRds = append(unbindRds, e)
		}
	}

	return unbindRds, nil
}

func (s *rdsService) UnBindRds(ctx context.Context, rdsID int, treeNodeID int) error {
	rds, err := s.rdsDao.GetByIDNoPreload(ctx, rdsID)
	if err != nil {
		s.logger.Error("UnBindRds 获取 RDS 失败", zap.Error(err))
		return err
	}

	node, err := s.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		s.logger.Error("UnBindRds 获取树节点失败", zap.Error(err))
		return err
	}

	return s.rdsDao.RemoveBindNodes(ctx, rds, node)
}
