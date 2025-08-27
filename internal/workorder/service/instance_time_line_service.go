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
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type WorkorderInstanceTimeLineService interface {
	CreateInstanceTimeLine(ctx context.Context, req *model.CreateWorkorderInstanceTimelineReq, creatorID int, creatorName string) (*model.WorkorderInstanceTimeline, error)
	GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error)
	ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) (*model.ListResp[*model.WorkorderInstanceTimeline], error)
}

type instanceTimeLineService struct {
	logger *zap.Logger
	dao    dao.WorkorderInstanceTimelineDAO
}

func NewWorkorderInstanceTimeLineService(
	logger *zap.Logger,
	dao dao.WorkorderInstanceTimelineDAO,
) WorkorderInstanceTimeLineService {
	return &instanceTimeLineService{
		logger: logger,
		dao:    dao,
	}
}

// CreateInstanceTimeLine 创建时间线
func (i *instanceTimeLineService) CreateInstanceTimeLine(ctx context.Context, req *model.CreateWorkorderInstanceTimelineReq, creatorID int, creatorName string) (*model.WorkorderInstanceTimeline, error) {
	timeline := &model.WorkorderInstanceTimeline{
		InstanceID:   req.InstanceID,
		Action:       req.Action,
		Comment:      req.Comment,
		OperatorID:   creatorID,
		OperatorName: creatorName,
	}

	if err := i.dao.Create(ctx, timeline); err != nil {
		i.logger.Error("创建时间线记录失败", zap.Error(err))
		return nil, err
	}

	return timeline, nil
}

// GetInstanceTimeLine 获取时间线记录
func (i *instanceTimeLineService) GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error) {
	timeline, err := i.dao.GetByID(ctx, id)
	if err != nil {
		i.logger.Error("获取时间线记录失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}
	return timeline, nil
}

// ListInstanceTimeLine 获取时间线列表
func (i *instanceTimeLineService) ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) (*model.ListResp[*model.WorkorderInstanceTimeline], error) {
	timelines, total, err := i.dao.List(ctx, req)
	if err != nil {
		i.logger.Error("获取时间线记录列表失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.WorkorderInstanceTimeline]{Total: total, Items: timelines}, nil
}
