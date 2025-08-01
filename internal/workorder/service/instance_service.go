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
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

var (
	ErrInvalidRequest = fmt.Errorf("请求参数无效")
	ErrInvalidStatus  = fmt.Errorf("工单状态无效")
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) (*model.WorkorderInstance, error)
	UpdateInstance(ctx context.Context, req *model.UpdateWorkorderInstanceReq) error
	DeleteInstance(ctx context.Context, id int) error
	GetInstance(ctx context.Context, id int) (*model.WorkorderInstance, error)
	ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) (*model.ListResp[*model.WorkorderInstance], error)
	SubmitInstance(ctx context.Context, id int, operatorID int, operatorName string) error
	AssignInstance(ctx context.Context, id int, assigneeID int, operatorID int, operatorName string) error
	ApproveInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	RejectInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
}

type instanceService struct {
	dao        dao.WorkorderInstanceDAO
	processDao dao.ProcessDAO
	logger     *zap.Logger
}

func NewInstanceService(
	dao dao.WorkorderInstanceDAO,
	processDao dao.ProcessDAO,
	logger *zap.Logger,
) InstanceService {
	return &instanceService{
		dao:        dao,
		processDao: processDao,
		logger:     logger,
	}
}

// CreateInstance 创建工单实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) (*model.WorkorderInstance, error) {
	if req.Status < model.InstanceStatusDraft || req.Status > model.InstanceStatusCancelled {
		return nil, ErrInvalidStatus
	}
	if req.Priority < model.PriorityLow || req.Priority > model.PriorityHigh {
		return nil, fmt.Errorf("优先级无效")
	}

	// 验证流程是否存在
	_, err := s.processDao.GetProcessByID(ctx, req.ProcessID)
	if err != nil && err != dao.ErrProcessNotFound {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return nil, err
	}

	// 生成工单编号
	serialNumber, err := s.dao.GenerateSerialNumber(ctx)
	if err != nil {
		s.logger.Error("生成工单编号失败", zap.Error(err))
		return nil, err
	}

	instance := &model.WorkorderInstance{
		Title:          req.Title,
		SerialNumber:   serialNumber,
		ProcessID:      req.ProcessID,
		FormData:       req.FormData,
		Status:         req.Status,
		Priority:       req.Priority,
		OperatorID:     req.OperatorID,
		OperatorName:   req.OperatorName,
		AssigneeID:     req.AssigneeID,
		Description:    req.Description,
		Tags:           req.Tags,
		DueDate:        req.DueDate,
	}

	if err := s.dao.CreateInstance(ctx, instance); err != nil {
		s.logger.Error("创建工单实例失败", zap.Error(err))
		return nil, fmt.Errorf("创建工单实例失败: %w", err)
	}

	return instance, nil
}

// UpdateInstance 更新工单实例
func (s *instanceService) UpdateInstance(ctx context.Context, req *model.UpdateWorkorderInstanceReq) error {
	ins, err := s.dao.GetInstanceByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", req.ID))
		return err
	}

	// 只有草稿和待处理状态可以更新
	if ins.Status != model.InstanceStatusDraft && ins.Status != model.InstanceStatusPending {
		return fmt.Errorf("当前状态的工单不允许修改")
	}

	instance := &model.WorkorderInstance{
		Model:       model.Model{ID: req.ID},
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Tags:        req.Tags,
		DueDate:     req.DueDate,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		FormData:    req.FormData,
		CompletedAt: req.CompletedAt,
	}

	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单实例失败", zap.Error(err), zap.Int("instanceID", req.ID))
		return err
	}

	return nil
}

// DeleteInstance 删除工单实例
func (s *instanceService) DeleteInstance(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidRequest
	}

	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	if instance.Status != model.InstanceStatusDraft {
		return ErrInvalidStatus
	}

	if err := s.dao.DeleteInstance(ctx, id); err != nil {
		s.logger.Error("删除工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	return nil
}

// GetInstance 获取工单实例详情
func (s *instanceService) GetInstance(ctx context.Context, id int) (*model.WorkorderInstance, error) {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return nil, err
	}

	return instance, nil
}

// ListInstance 获取工单实例列表
func (s *instanceService) ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) (*model.ListResp[*model.WorkorderInstance], error) {
	result, total, err := s.dao.ListInstance(ctx, req)
	if err != nil {
		s.logger.Error("获取工单实例列表失败", zap.Error(err))
		return nil, err
	}

	return &model.ListResp[*model.WorkorderInstance]{
		Items: result,
		Total: total,
	}, nil
}

// SubmitInstance 提交工单
func (s *instanceService) SubmitInstance(ctx context.Context, id int, operatorID int, operatorName string) error {
	return s.dao.UpdateInstanceStatus(ctx, id, model.InstanceStatusPending)
}

// AssignInstance 指派工单
func (s *instanceService) AssignInstance(ctx context.Context, id int, assigneeID int, operatorID int, operatorName string) error {
	if err := s.dao.UpdateInstanceAssignee(ctx, id, &assigneeID); err != nil {
		return err
	}
	return s.dao.UpdateInstanceStatus(ctx, id, model.InstanceStatusProcessing)
}

// ApproveInstance 审批通过工单
func (s *instanceService) ApproveInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	now := time.Now()
	if err := s.dao.UpdateInstanceStatus(ctx, id, model.InstanceStatusCompleted); err != nil {
		return err
	}
	
	// 更新完成时间
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}
	instance.CompletedAt = &now
	return s.dao.UpdateInstance(ctx, instance)
}

// RejectInstance 拒绝工单
func (s *instanceService) RejectInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	return s.dao.UpdateInstanceStatus(ctx, id, model.InstanceStatusRejected)
}
