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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req model.InstanceReq) error
	UpdateInstance(ctx context.Context, req model.InstanceReq) error
	DeleteInstance(ctx context.Context, req model.DeleteInstanceReq) error
	ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error)
	DetailInstance(ctx context.Context, id int) (model.Instance, error)
	ApproveInstance(ctx context.Context, req model.InstanceFlowReq) error
	ActionInstance(ctx context.Context, req model.InstanceFlowReq) error
	CommentInstance(ctx context.Context, req model.InstanceCommentReq) error
}

type instanceService struct {
	dao dao.InstanceDAO
	l   *zap.Logger
}

func NewInstanceService(dao dao.InstanceDAO, l *zap.Logger) InstanceService {
	return &instanceService{
		dao: dao,
		l:   l,
	}
}

// CreateInstance implements InstanceService.
func (i *instanceService) CreateInstance(ctx context.Context, req model.InstanceReq) error {
	instance, err := utils.ConvertInstanceReq(&req)
	if err != nil {
		return err
	}
	return i.dao.CreateInstance(ctx, instance)
}

// DeleteInstance implements InstanceService.
func (i *instanceService) DeleteInstance(ctx context.Context, req model.DeleteInstanceReq) error {
	return i.dao.DeleteInstance(ctx, req.ID)
}

// DetailInstance implements InstanceService.
func (i *instanceService) DetailInstance(ctx context.Context, id int) (model.Instance, error) {
	return i.dao.GetInstance(ctx, id)
}

// ListInstance implements InstanceService.
func (i *instanceService) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) {
	return i.dao.ListInstance(ctx, req)
}

// UpdateInstance implements InstanceService.
func (i *instanceService) UpdateInstance(ctx context.Context, req model.InstanceReq) error {
	instance, err := utils.ConvertInstanceReq(&req)
	if err != nil {
		return err
	}
	return i.dao.UpdateInstance(ctx, instance)
}

func (i *instanceService) ApproveInstance(ctx context.Context, req model.InstanceFlowReq) error {
	// step1:给数据库中的InstanceFlow表添加一条记录
	flowReq, err := utils.ConvertInstanceFlowReq(&req)
	if err != nil {
		return err
	}
	err = i.dao.CreateInstanceFlow(ctx, flowReq)
	if err != nil {
		return err
	}
	// step2:更新Instance表中的DueDate字段
	days := req.FormData.ApproveDays
	// 查找Instance
	instance, err := i.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}
	// 转换InstanceReq=>Instance
	convertInstanceReq, err := utils.ConvertInstance(&instance)
	if err != nil {
		return err
	}
	// 更改Instance的截至时间
	startDate, err := time.Parse("2006-01-02", convertInstanceReq.FormData.DateRange[0])
	if err != nil {
		i.l.Error("时间转换失败", zap.Error(err))
		return err
	}
	convertInstanceReq.DueDate = startDate.AddDate(0, 0, int(days))
	// 转换回去
	newinstance, err := utils.ConvertInstanceReq(convertInstanceReq)
	if err != nil {
		return err
	}
	err = i.dao.UpdateInstance(ctx, newinstance)
	if err != nil {
		return err
	}
	return nil
}

func (i *instanceService) ActionInstance(ctx context.Context, req model.InstanceFlowReq) error {
	// step1:给数据库中的InstanceFlow表添加一条记录
	flowReq, err := utils.ConvertInstanceFlowReq(&req)
	if err != nil {
		return err
	}
	err = i.dao.CreateInstanceFlow(ctx, flowReq)
	if err != nil {
		return err
	}
	return nil
}
func (i *instanceService) CommentInstance(ctx context.Context, req model.InstanceCommentReq) error {
	// step1:给数据库中的InstanceComment表添加一条记录
	commentReq, err := utils.ConvertInstanceCommentReq(&req)
	if err != nil {
		return err
	}
	err = i.dao.CreateInstanceComment(ctx, commentReq)
	if err != nil {
		return err
	}
	return nil
}
