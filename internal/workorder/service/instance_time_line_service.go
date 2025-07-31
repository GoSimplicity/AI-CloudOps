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
	UpdateInstanceTimeLine(ctx context.Context, req *model.UpdateWorkorderInstanceTimelineReq, operatorID int) error
	DeleteInstanceTimeLine(ctx context.Context, id int, operatorID int) error
	GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error)
	ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) (model.ListResp[*model.WorkorderInstanceTimeline], error)
}

type instanceTimeLineService struct {
	dao    dao.InstanceTimeLineDAO
	logger *zap.Logger
}

func NewWorkorderInstanceTimeLineService(
	logger *zap.Logger,
) WorkorderInstanceTimeLineService {
	return &instanceTimeLineService{
		logger: logger,
	}
}

// CreateInstanceTimeLine implements WorkorderInstanceTimeLineService.
func (i *instanceTimeLineService) CreateInstanceTimeLine(ctx context.Context, req *model.CreateWorkorderInstanceTimelineReq, creatorID int, creatorName string) (*model.WorkorderInstanceTimeline, error) {
	panic("unimplemented")
}

// DeleteInstanceTimeLine implements WorkorderInstanceTimeLineService.
func (i *instanceTimeLineService) DeleteInstanceTimeLine(ctx context.Context, id int, operatorID int) error {
	panic("unimplemented")
}

// GetInstanceTimeLine implements WorkorderInstanceTimeLineService.
func (i *instanceTimeLineService) GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error) {
	panic("unimplemented")
}

// ListInstanceTimeLine implements WorkorderInstanceTimeLineService.
func (i *instanceTimeLineService) ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) (model.ListResp[*model.WorkorderInstanceTimeline], error) {
	panic("unimplemented")
}

// UpdateInstanceTimeLine implements WorkorderInstanceTimeLineService.
func (i *instanceTimeLineService) UpdateInstanceTimeLine(ctx context.Context, req *model.UpdateWorkorderInstanceTimelineReq, operatorID int) error {
	panic("unimplemented")
}
