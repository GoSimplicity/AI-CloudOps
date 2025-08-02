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
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

// 错误定义
var (
	ErrInstanceNotFound = errors.New("工单实例不存在")
)

type InstanceFlowService interface {
	ListInstanceFlows(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) (*model.ListResp[*model.WorkorderInstanceFlow], error)
	DetailInstanceFlow(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error)
}

type instanceFlowService struct {
	dao    dao.WorkorderInstanceFlowDAO
	logger *zap.Logger
}

func NewInstanceFlowService(dao dao.WorkorderInstanceFlowDAO, logger *zap.Logger) InstanceFlowService {
	return &instanceFlowService{
		dao:    dao,
		logger: logger,
	}
}

// ListInstanceFlows 分页获取工单流程记录列表
func (s *instanceFlowService) ListInstanceFlows(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) (*model.ListResp[*model.WorkorderInstanceFlow], error) {
	if req == nil {
		return nil, dao.ErrInstanceInvalidID
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	flows, total, err := s.dao.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取工单流程记录列表失败: %w", err)
	}

	// 转换为指针切片
	flowPtrs := make([]*model.WorkorderInstanceFlow, len(flows))
	for i := range flows {
		flowPtrs[i] = &flows[i]
	}

	return &model.ListResp[*model.WorkorderInstanceFlow]{
		Items: flowPtrs,
		Total: total,
	}, nil
}

// DetailInstanceFlow 获取工单流转记录详情
func (s *instanceFlowService) DetailInstanceFlow(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error) {
	if id <= 0 {
		return nil, dao.ErrInstanceInvalidID
	}

	flow, err := s.dao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceFlowNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, fmt.Errorf("获取工单流转记录失败: %w", err)
	}

	return flow, nil
}
