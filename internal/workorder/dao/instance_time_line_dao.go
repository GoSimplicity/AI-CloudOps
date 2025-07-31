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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InstanceTimeLineDAO interface {
	CreateInstanceTimeLine(ctx context.Context, instance *model.WorkorderInstanceTimeline) error
	UpdateInstanceTimeLine(ctx context.Context, instance *model.WorkorderInstanceTimeline) error
	DeleteInstanceTimeLine(ctx context.Context, id int) error
	GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error)
	ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) ([]*model.WorkorderInstanceTimeline, int64, error)
}

type instanceTimeLineDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceTimeLineDAO(db *gorm.DB, logger *zap.Logger) InstanceTimeLineDAO {
	return &instanceTimeLineDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstanceTimeLine implements InstanceTimeLineDAO.
func (i *instanceTimeLineDAO) CreateInstanceTimeLine(ctx context.Context, instance *model.WorkorderInstanceTimeline) error {
	panic("unimplemented")
}

// DeleteInstanceTimeLine implements InstanceTimeLineDAO.
func (i *instanceTimeLineDAO) DeleteInstanceTimeLine(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetInstanceTimeLine implements InstanceTimeLineDAO.
func (i *instanceTimeLineDAO) GetInstanceTimeLine(ctx context.Context, id int) (*model.WorkorderInstanceTimeline, error) {
	panic("unimplemented")
}

// ListInstanceTimeLine implements InstanceTimeLineDAO.
func (i *instanceTimeLineDAO) ListInstanceTimeLine(ctx context.Context, req *model.ListWorkorderInstanceTimelineReq) ([]*model.WorkorderInstanceTimeline, int64, error) {
	panic("unimplemented")
}

// UpdateInstanceTimeLine implements InstanceTimeLineDAO.
func (i *instanceTimeLineDAO) UpdateInstanceTimeLine(ctx context.Context, instance *model.WorkorderInstanceTimeline) error {
	panic("unimplemented")
}
