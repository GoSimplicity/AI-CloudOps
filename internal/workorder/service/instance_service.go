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
	"encoding/json"
	"fmt"
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

// CreateInstance 创建工单实例
func (i *instanceService) CreateInstance(ctx context.Context, req model.InstanceReq) error {
	instance, err := utils.ConvertInstanceReq(&req)
	if err != nil {
		i.l.Error("转换实例请求失败", zap.Error(err))
		return err
	}
	return i.dao.CreateInstance(ctx, instance)
}

// DeleteInstance 删除工单实例
func (i *instanceService) DeleteInstance(ctx context.Context, req model.DeleteInstanceReq) error {
	return i.dao.DeleteInstance(ctx, req.ID)
}

// DetailInstance 获取工单实例详情
func (i *instanceService) DetailInstance(ctx context.Context, id int) (model.Instance, error) {
	return i.dao.GetInstance(ctx, id)
}

// ListInstance 获取工单实例列表
func (i *instanceService) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) {
	return i.dao.ListInstance(ctx, req)
}

// UpdateInstance 更新工单实例
func (i *instanceService) UpdateInstance(ctx context.Context, req model.InstanceReq) error {
	instance, err := utils.ConvertInstanceReq(&req)
	if err != nil {
		i.l.Error("转换实例请求失败", zap.Error(err))
		return err
	}
	return i.dao.UpdateInstance(ctx, instance)
}

// ApproveInstance 审批工单实例
func (i *instanceService) ApproveInstance(ctx context.Context, req model.InstanceFlowReq) error {
	// 1. 创建流程记录
	flowReq, err := utils.ConvertInstanceFlowReq(&req)
	if err != nil {
		i.l.Error("转换实例流程请求失败", zap.Error(err))
		return err
	}
	
	if err := i.dao.CreateInstanceFlow(ctx, flowReq); err != nil {
		i.l.Error("创建实例流程记录失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return err
	}
	
	// 2. 更新实例截止日期
	days := req.FormData.ApproveDays
	instance, err := i.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		i.l.Error("获取实例失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return err
	}

	// 解析FormData
	var formData map[string]interface{}
	if err := json.Unmarshal([]byte(instance.FormData), &formData); err != nil {
		i.l.Error("解析FormData失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("解析FormData失败: %w", err)
	}

	// 获取日期范围
	dateRange, ok := formData["DateRange"].([]interface{})
	if !ok || len(dateRange) == 0 {
		i.l.Error("获取日期范围失败", zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("获取日期范围失败")
	}

	// 解析开始日期
	startDateStr, ok := dateRange[0].(string)
	if !ok {
		i.l.Error("日期格式错误", zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("日期格式错误")
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		i.l.Error("时间转换失败", zap.Error(err), zap.String("startDate", startDateStr))
		return fmt.Errorf("时间转换失败: %w", err)
	}

	// 更新截止日期并保存
	instance.DueDate = startDate.AddDate(0, 0, int(days))
	if err := i.dao.UpdateInstance(ctx, &instance); err != nil {
		i.l.Error("更新实例失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("更新实例失败: %w", err)
	}
	
	return nil
}

// ActionInstance 处理工单实例操作
func (i *instanceService) ActionInstance(ctx context.Context, req model.InstanceFlowReq) error {
	flowReq, err := utils.ConvertInstanceFlowReq(&req)
	if err != nil {
		i.l.Error("转换实例流程请求失败", zap.Error(err))
		return fmt.Errorf("转换实例流程请求失败: %w", err)
	}
	
	if err := i.dao.CreateInstanceFlow(ctx, flowReq); err != nil {
		i.l.Error("创建实例流程记录失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("创建实例流程记录失败: %w", err)
	}
	
	return nil
}

// CommentInstance 添加工单实例评论
func (i *instanceService) CommentInstance(ctx context.Context, req model.InstanceCommentReq) error {
	commentReq, err := utils.ConvertInstanceCommentReq(&req)
	if err != nil {
		i.l.Error("转换实例评论请求失败", zap.Error(err))
		return fmt.Errorf("转换实例评论请求失败: %w", err)
	}
	
	if err := i.dao.CreateInstanceComment(ctx, commentReq); err != nil {
		i.l.Error("创建实例评论失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("创建实例评论失败: %w", err)
	}
	
	return nil
}
