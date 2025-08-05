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
	CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) error
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
	dao         dao.WorkorderInstanceDAO
	flowDao     dao.WorkorderInstanceFlowDAO
	timelineDao dao.WorkorderInstanceTimelineDAO
	commentDao  dao.WorkorderInstanceCommentDAO
	processDao  dao.WorkorderProcessDAO
	logger      *zap.Logger
}

func NewInstanceService(
	dao dao.WorkorderInstanceDAO,
	flowDao dao.WorkorderInstanceFlowDAO,
	timelineDao dao.WorkorderInstanceTimelineDAO,
	commentDao dao.WorkorderInstanceCommentDAO,
	processDao dao.WorkorderProcessDAO,
	logger *zap.Logger,
) InstanceService {
	return &instanceService{
		dao:         dao,
		flowDao:     flowDao,
		timelineDao: timelineDao,
		commentDao:  commentDao,
		processDao:  processDao,
		logger:      logger,
	}
}

// CreateInstance 创建工单实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) error {
	if req.Status < model.InstanceStatusDraft || req.Status > model.InstanceStatusCancelled {
		return fmt.Errorf("工单状态无效")
	}
	if req.Priority < model.PriorityLow || req.Priority > model.PriorityHigh {
		return fmt.Errorf("优先级无效")
	}

	// 确保实例名称唯一
	if _, err := s.dao.GetInstanceByTitle(ctx, req.Title); err != nil {
		if err != dao.ErrInstanceNotFound {
			s.logger.Error("获取工单实例失败", zap.Error(err), zap.String("title", req.Title))
			return err
		}
	} else {
		return fmt.Errorf("工单实例名称已存在")
	}

	// 验证流程是否存在
	if _, err := s.processDao.GetProcessByID(ctx, req.ProcessID); err != nil {
		if err != dao.ErrProcessNotFound {
			s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
			return err
		}
	}

	// 生成工单编号
	serialNumber, err := s.dao.GenerateSerialNumber(ctx)
	if err != nil {
		s.logger.Error("生成工单编号失败", zap.Error(err))
		return err
	}

	instance := &model.WorkorderInstance{
		Title:        req.Title,
		SerialNumber: serialNumber,
		ProcessID:    req.ProcessID,
		FormData:     req.FormData,
		Status:       req.Status,
		Priority:     req.Priority,
		OperatorID:   req.OperatorID,
		OperatorName: req.OperatorName,
		AssigneeID:   req.AssigneeID,
		Description:  req.Description,
		Tags:         req.Tags,
		DueDate:      req.DueDate,
	}

	if err := s.dao.CreateInstance(ctx, instance); err != nil {
		s.logger.Error("创建工单实例失败", zap.Error(err))
		return fmt.Errorf("创建工单实例失败: %w", err)
	}

	// 创建初始流转记录
	s.createFlowRecord(ctx, instance.ID, model.FlowActionSubmit, req.OperatorID, req.OperatorName, 0, req.Status, "", 1)

	// 创建时间线记录
	s.createTimelineRecord(ctx, instance.ID, model.TimelineActionCreate, req.OperatorID, req.OperatorName, "工单创建")

	return nil
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

	// 确保实例名称唯一 (排除当前实例)
	if instance, err := s.dao.GetInstanceByTitle(ctx, req.Title); err != nil {
		if err != dao.ErrInstanceNotFound {
			s.logger.Error("获取工单实例失败", zap.Error(err), zap.String("title", req.Title))
			return err
		}
	} else if instance.ID != req.ID {
		return fmt.Errorf("工单实例名称已存在")
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
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusPending

	// 更新状态
	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		return err
	}

	// 创建流转记录
	s.createFlowRecord(ctx, id, model.FlowActionSubmit, operatorID, operatorName, fromStatus, toStatus, "", 2)

	// 创建时间线记录
	s.createTimelineRecord(ctx, id, model.TimelineActionSubmit, operatorID, operatorName, "工单提交")

	return nil
}

// AssignInstance 指派工单
func (s *instanceService) AssignInstance(ctx context.Context, id int, assigneeID int, operatorID int, operatorName string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusProcessing

	// 更新指派人和状态
	if err := s.dao.UpdateInstanceAssignee(ctx, id, &assigneeID); err != nil {
		return err
	}
	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		return err
	}

	// 创建流转记录
	s.createFlowRecord(ctx, id, model.FlowActionAssign, operatorID, operatorName, fromStatus, toStatus, "", 2)

	// 创建时间线记录
	s.createTimelineRecord(ctx, id, model.TimelineActionAssign, operatorID, operatorName, fmt.Sprintf("工单指派给用户ID: %d", assigneeID))

	return nil
}

// ApproveInstance 审批通过工单
func (s *instanceService) ApproveInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusCompleted
	now := time.Now()

	// 更新状态和完成时间
	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	instance.CompletedAt = &now
	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单完成时间失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	// 创建流转记录
	s.createFlowRecord(ctx, id, model.FlowActionApprove, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	// 创建时间线记录
	s.createTimelineRecord(ctx, id, model.TimelineActionApprove, operatorID, operatorName, fmt.Sprintf("工单审批通过: %s", comment))

	// 如果有审批意见，则添加系统评论
	if comment != "" {
		commentEntity := &model.WorkorderInstanceComment{
			InstanceID:   id,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			Content:      fmt.Sprintf("审批通过：%s", comment),
			Type:         model.CommentTypeSystem,
			Status:       model.CommentStatusNormal,
			IsSystem:     1,
		}

		if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
			s.logger.Error("创建审批评论失败", zap.Error(err), zap.Int("instanceID", id))
		}
	}

	return nil
}

// RejectInstance 拒绝工单
func (s *instanceService) RejectInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusRejected

	// 更新工单状态为已拒绝
	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	// 创建流转记录
	s.createFlowRecord(ctx, id, model.FlowActionReject, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	// 创建时间线记录
	s.createTimelineRecord(ctx, id, model.TimelineActionReject, operatorID, operatorName, fmt.Sprintf("工单审批拒绝: %s", comment))

	// 添加拒绝原因的系统评论
	if comment != "" {
		commentEntity := &model.WorkorderInstanceComment{
			InstanceID:   id,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			Content:      fmt.Sprintf("审批拒绝：%s", comment),
			Type:         model.CommentTypeSystem,
			Status:       model.CommentStatusNormal,
			IsSystem:     1,
		}

		if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
			s.logger.Error("创建拒绝评论失败", zap.Error(err), zap.Int("instanceID", id))
		}
	}

	return nil
}

// 私有方法：创建流转记录
func (s *instanceService) createFlowRecord(ctx context.Context, instanceID int, action string, operatorID int, operatorName string, fromStatus, toStatus int8, comment string, isSystem int8) {
	flow := &model.WorkorderInstanceFlow{
		InstanceID:     instanceID,
		Action:         action,
		OperatorID:     operatorID,
		OperatorName:   operatorName,
		FromStatus:     fromStatus,
		ToStatus:       toStatus,
		Comment:        comment,
		IsSystemAction: isSystem,
	}

	if err := s.flowDao.Create(ctx, flow); err != nil {
		s.logger.Error("创建流转记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
	}
}

// 私有方法：创建时间线记录
func (s *instanceService) createTimelineRecord(ctx context.Context, instanceID int, action string, operatorID int, operatorName string, comment string) {
	timeline := &model.WorkorderInstanceTimeline{
		InstanceID:   instanceID,
		Action:       action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		ActionDetail: "", // 简单操作不需要详细信息
		Comment:      comment,
		RelatedID:    nil, // 无关联记录
	}

	if err := s.timelineDao.Create(ctx, timeline); err != nil {
		s.logger.Error("创建时间线记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
	}
}
