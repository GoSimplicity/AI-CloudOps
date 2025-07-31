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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type InstanceFlowService interface {
	CreateInstanceFlow(ctx context.Context, req *model.CreateWorkorderInstanceFlowReq) error
	ListInstanceFlows(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) (model.ListResp[*model.WorkorderInstanceFlow], error)
	DetailInstanceFlow(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error)
}

type instanceFlowService struct {
	dao         dao.InstanceFlowDAO
	processDao  dao.ProcessDAO
	instanceDao dao.InstanceFlowDAO
	logger      *zap.Logger
}

func NewInstanceFlowService(dao dao.InstanceFlowDAO, processDao dao.ProcessDAO, instanceDao dao.InstanceFlowDAO, logger *zap.Logger) InstanceFlowService {
	return &instanceFlowService{
		dao:         dao,
		processDao:  processDao,
		instanceDao: instanceDao,
		logger:      logger,
	}
}

// ProcessInstanceFlow 处理工单流程
func (s *instanceFlowService) CreateInstanceFlow(ctx context.Context, req *model.CreateWorkorderInstanceFlowReq) error {
	if err := s.validateActionRequest(req); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 获取工单实例
	instance, err := s.instanceDao.GetInstanceFlows(ctx, req.InstanceID)
	if err != nil {
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	// 验证操作权限
	if err := s.validateOperationPermission(instance, req.OperatorID, req.Action); err != nil {
		return err
	}

	// 获取流程定义
	process, err := s.processDao.GetProcessByID(ctx, instance.ProcessID)
	if err != nil {
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	processDef, err := s.parseProcessDefinition(process.Definition.String())
	if err != nil {
		return fmt.Errorf("解析流程定义失败: %w", err)
	}

	// 创建流程记录
	flow := s.buildFlowRecord(req, instance, operatorID, operatorName)

	// 根据操作类型处理
	if err := s.handleFlowAction(ctx, instance, processDef, flow, req); err != nil {
		return err
	}

	// 保存流程记录
	if err := s.dao.CreateInstanceFlow(ctx, flow); err != nil {
		s.logger.Error("创建流程记录失败", zap.Error(err))
		return fmt.Errorf("创建流程记录失败: %w", err)
	}

	// 更新实例
	if err := s.instanceDao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新实例失败", zap.Error(err))
		return fmt.Errorf("更新实例失败: %w", err)
	}

	s.logger.Info("工单流程处理成功",
		zap.Int("instanceID", req.InstanceID),
		zap.String("action", req.Action),
		zap.Int("operatorID", operatorID))

	return nil
}

// GetInstanceFlows 获取工单流程记录
func (s *instanceFlowService) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error) {
	if instanceID <= 0 {
		return nil, ErrInvalidRequest
	}

	flows, err := s.dao.GetInstanceFlows(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取工单流程记录失败: %w", err)
	}

	respFlows := make([]model.InstanceFlowResp, 0, len(flows))
	for _, flow := range flows {
		respFlows = append(respFlows, *s.convertToFlowResp(&flow))
	}

	return respFlows, nil
}

// GetProcessDefinition 获取流程定义
func (s *instanceFlowService) GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error) {
	if processID <= 0 {
		return nil, ErrInvalidRequest
	}

	process, err := s.processDao.GetProcess(ctx, processID)
	if err != nil {
		if errors.Is(err, dao.ErrProcessNotFound) {
			return nil, ErrProcessNotFound
		}
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	processDef, err := s.parseProcessDefinition(process.Definition.String())
	if err != nil {
		return nil, fmt.Errorf("解析流程定义失败: %w", err)
	}

	return processDef, nil
}

// DetailInstanceFlow 获取工单流转记录详情
func (s *instanceFlowService) DetailInstanceFlow(ctx context.Context, id int) (*model.InstanceFlowResp, error) {
	if id <= 0 {
		return nil, ErrInvalidRequest
	}
	flow, err := s.dao.GetInstanceFlowByID(ctx, id)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceFlowNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, fmt.Errorf("获取工单流转记录失败: %w", err)
	}
	return s.convertToFlowResp(flow), nil
}

// 辅助方法

// parseProcessDefinition 解析流程定义
func (s *instanceFlowService) parseProcessDefinition(definition string) (*model.ProcessDefinition, error) {
	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(definition), &processDef); err != nil {
		return nil, ErrProcessDefinition
	}
	return &processDef, nil
}

// validateActionRequest 验证操作请求
func (s *instanceFlowService) validateActionRequest(req *model.InstanceActionReq) error {
	if req == nil {
		return ErrInvalidRequest
	}

	if req.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}

	validActions := []string{"approve", "reject", "cancel", "transfer", "revoke"}
	actionValid := false
	for _, action := range validActions {
		if req.Action == action {
			actionValid = true
			break
		}
	}

	if !actionValid {
		return ErrInvalidAction
	}

	if req.Action == "transfer" && (req.AssigneeID == nil || *req.AssigneeID <= 0) {
		return fmt.Errorf("转移操作需要指定有效的处理人")
	}

	return nil
}

// validateOperationPermission 验证操作权限
func (s *instanceFlowService) validateOperationPermission(instance *model.Instance, operatorID int, action string) error {
	// 检查工单状态
	if instance.Status == model.InstanceStatusCompleted ||
		instance.Status == model.InstanceStatusCancelled ||
		instance.Status == model.InstanceStatusRejected {
		return fmt.Errorf("工单已结束，不能进行此操作")
	}

	// 检查操作人权限
	switch action {
	case "approve", "reject":
		// 只有当前处理人可以审批或拒绝
		if instance.AssigneeID == nil || *instance.AssigneeID != operatorID {
			return fmt.Errorf("您不是当前工单的处理人，无权进行此操作")
		}
	case "cancel", "revoke":
		// 只有创建人可以取消或撤销
		if instance.CreatorID != operatorID {
			return fmt.Errorf("只有工单创建人可以进行此操作")
		}
	case "transfer":
		// 当前处理人或创建人可以转移
		if instance.AssigneeID != nil && *instance.AssigneeID != operatorID && instance.CreatorID != operatorID {
			return fmt.Errorf("您无权转移此工单")
		}
	}

	return nil
}

// buildFlowRecord 构建流程记录
func (s *instanceFlowService) buildFlowRecord(req *model.InstanceActionReq, instance *model.Instance, operatorID int, operatorName string) *model.InstanceFlow {
	// 处理表单数据
	var formData model.JSONMap
	if req.FormData != nil {
		formData = model.JSONMap(req.FormData)
	}

	return &model.InstanceFlow{
		InstanceID:   req.InstanceID,
		StepID:       instance.CurrentStep,
		Action:       req.Action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Comment:      strings.TrimSpace(req.Comment),
		FormData:     formData,
		FromStepID:   instance.CurrentStep,
	}
}

// handleFlowAction 处理流程操作
func (s *instanceFlowService) handleFlowAction(ctx context.Context, instance *model.Instance, processDef *model.ProcessDefinition, flow *model.InstanceFlow, req *model.InstanceActionReq) error {
	switch req.Action {
	case "approve":
		return s.handleApproveAction(ctx, instance, processDef, flow, req)
	case "reject":
		return s.handleRejectAction(ctx, instance, flow)
	case "cancel":
		return s.handleCancelAction(ctx, instance, flow)
	case "transfer":
		return s.handleTransferAction(ctx, instance, flow, req.AssigneeID)
	case "revoke":
		return s.handleRevokeAction(ctx, instance, flow)
	default:
		return ErrInvalidAction
	}
}

// convertToFlowResp 转换流程记录为响应格式
func (s *instanceFlowService) convertToFlowResp(flow *model.InstanceFlow) *model.InstanceFlowResp {
	if flow == nil {
		return nil
	}

	// 解析表单数据
	var formData map[string]interface{}
	if flow.FormData != nil {
		formData = map[string]interface{}(flow.FormData)
	}

	return &model.InstanceFlowResp{
		ID:           flow.ID,
		InstanceID:   flow.InstanceID,
		StepID:       flow.StepID,
		StepName:     flow.StepName,
		Action:       flow.Action,
		OperatorID:   flow.OperatorID,
		OperatorName: flow.OperatorName,
		Comment:      flow.Comment,
		FormData:     formData,
		Duration:     flow.Duration,
		FromStepID:   flow.FromStepID,
		ToStepID:     flow.ToStepID,
		CreatedAt:    flow.CreatedAt,
	}
}

// handleApproveAction 处理审批操作
func (s *instanceFlowService) handleApproveAction(ctx context.Context, instance *model.Instance, processDef *model.ProcessDefinition, flow *model.InstanceFlow, _ *model.InstanceActionReq) error {
	// 查找当前步骤
	var currentStep *model.ProcessStep
	for i, step := range processDef.Steps {
		if step.ID == instance.CurrentStep {
			currentStep = &processDef.Steps[i]
			break
		}
	}

	if currentStep == nil {
		return fmt.Errorf("当前步骤不存在")
	}

	flow.StepName = currentStep.Name

	// 查找下一步
	var nextStepID string
	for _, conn := range processDef.Connections {
		if conn.From == instance.CurrentStep {
			nextStepID = conn.To
			break
		}
	}

	if nextStepID == "" {
		// 没有下一步，工单完成
		instance.Status = model.InstanceStatusCompleted
		now := time.Now()
		instance.CompletedAt = &now
		flow.ToStepID = instance.CurrentStep
		return nil
	}

	// 查找下一步详情
	var nextStep *model.ProcessStep
	for i, step := range processDef.Steps {
		if step.ID == nextStepID {
			nextStep = &processDef.Steps[i]
			break
		}
	}

	if nextStep == nil {
		return fmt.Errorf("下一步骤定义不存在")
	}

	// 更新实例
	instance.CurrentStep = nextStep.ID
	flow.ToStepID = nextStep.ID

	if nextStep.Type == "end" {
		instance.Status = model.InstanceStatusCompleted
		now := time.Now()
		instance.CompletedAt = &now
		instance.AssigneeID = nil
		instance.AssigneeName = ""
	} else {
		// 分配下一步处理人
		if len(nextStep.Users) > 0 {
			instance.AssigneeID = &nextStep.Users[0]
			if user, err := s.userDao.GetUserByID(ctx, nextStep.Users[0]); err == nil {
				instance.AssigneeName = user.Username
			}
		} else {
			instance.AssigneeID = nil
			instance.AssigneeName = ""
		}
	}

	return nil
}

// handleRejectAction 处理拒绝操作
func (s *instanceFlowService) handleRejectAction(_ context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusRejected
	flow.ToStepID = instance.CurrentStep
	return nil
}

// handleCancelAction 处理取消操作
func (s *instanceFlowService) handleCancelAction(_ context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusCancelled
	flow.ToStepID = instance.CurrentStep
	return nil
}

// handleTransferAction 处理转移操作
func (s *instanceFlowService) handleTransferAction(ctx context.Context, instance *model.Instance, flow *model.InstanceFlow, assigneeID *int) error {
	if assigneeID == nil || *assigneeID <= 0 {
		return fmt.Errorf("转移操作需要指定有效的处理人")
	}

	// 验证目标用户是否存在
	user, err := s.userDao.GetUserByID(ctx, *assigneeID)
	if err != nil {
		return ErrUserNotFound
	}

	instance.AssigneeID = assigneeID
	instance.AssigneeName = user.Username
	flow.ToStepID = instance.CurrentStep

	return nil
}

// handleRevokeAction 处理撤销操作
func (s *instanceFlowService) handleRevokeAction(_ context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusDraft
	instance.AssigneeID = nil
	instance.AssigneeName = ""
	flow.ToStepID = instance.CurrentStep
	return nil
}
