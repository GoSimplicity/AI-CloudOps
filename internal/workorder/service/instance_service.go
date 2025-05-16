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
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req model.CreateInstanceReq, creatorID int, creatorName string) error
	UpdateInstance(ctx context.Context, req model.UpdateInstanceReq) error
	DeleteInstance(ctx context.Context, id int) error
	ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error)
	DetailInstance(ctx context.Context, id int) (model.Instance, error)
	ProcessInstanceFlow(ctx context.Context, req model.InstanceFlowReq, operatorID int, operatorName string) error
	CommentInstance(ctx context.Context, req model.InstanceCommentReq, creatorID int, creatorName string) error
	GetWorkflowDefinition(ctx context.Context, workflowID int) (model.WorkflowDefinition, error)
	GetInstanceStatistics(ctx context.Context) (interface{}, error)
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
func (i *instanceService) CreateInstance(ctx context.Context, req model.CreateInstanceReq, creatorID int, creatorName string) error {
	// 将请求转换为实例对象
	formDataBytes, err := json.Marshal(req.FormData)
	if err != nil {
		i.l.Error("序列化表单数据失败", zap.Error(err))
		return fmt.Errorf("序列化表单数据失败: %w", err)
	}

	instance := model.Instance{
		Title:       req.Title,
		WorkflowID:  req.WorkflowID,
		Description: req.Description,
		FormData:    string(formDataBytes),
		Status:      model.InstanceStatusDraft,
		Priority:    req.Priority,
		CategoryID:  req.CategoryID,
		CreatorID:   creatorID,
		CreatorName: creatorName,
	}

	// 获取工作流定义
	workflow, err := i.dao.GetWorkflow(ctx, req.WorkflowID)
	if err != nil {
		i.l.Error("获取工作流定义失败", zap.Error(err), zap.Int("workflowID", req.WorkflowID))
		return fmt.Errorf("获取工作流定义失败: %w", err)
	}

	// 设置初始状态为工作流的第一步
	var workflowDef model.WorkflowDefinition
	if err := json.Unmarshal([]byte(workflow.Definition), &workflowDef); err != nil {
		i.l.Error("解析工作流定义失败", zap.Error(err))
		return fmt.Errorf("解析工作流定义失败: %w", err)
	}

	if len(workflowDef.Steps) > 0 {
		instance.CurrentStep = workflowDef.Steps[0].Step
		instance.CurrentRole = workflowDef.Steps[0].Role
		instance.Status = model.InstanceStatusProcessing

		// 设置初始处理人（如果有）
		// TODO: 根据角色查找对应的处理人
	}

	return i.dao.CreateInstance(ctx, instance)
}

// DeleteInstance 删除工单实例
func (i *instanceService) DeleteInstance(ctx context.Context, id int) error {
	// 先检查工单状态，只有草稿状态可以删除
	instance, err := i.dao.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以删除")
	}

	return i.dao.DeleteInstance(ctx, id)
}

// DetailInstance 获取工单实例详情
func (i *instanceService) DetailInstance(ctx context.Context, id int) (model.Instance, error) {
	instance, err := i.dao.GetInstance(ctx, id)
	if err != nil {
		return model.Instance{}, err
	}

	// 获取实例相关的流程记录
	flows, err := i.dao.GetInstanceFlows(ctx, id)
	if err != nil {
		i.l.Warn("获取实例流程记录失败", zap.Error(err), zap.Int("instanceID", id))
	} else {
		instance.Flows = flows
	}

	// 获取实例相关的评论
	comments, err := i.dao.GetInstanceComments(ctx, id)
	if err != nil {
		i.l.Warn("获取实例评论失败", zap.Error(err), zap.Int("instanceID", id))
	} else {
		instance.Comments = comments
	}

	return instance, nil
}

// ListInstance 获取工单实例列表
func (i *instanceService) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) {
	instances, _, err := i.dao.ListInstance(ctx, req)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// UpdateInstance 更新工单实例
func (i *instanceService) UpdateInstance(ctx context.Context, req model.UpdateInstanceReq) error {
	// 获取当前实例
	instance, err := i.dao.GetInstance(ctx, req.ID)
	if err != nil {
		return err
	}

	// 只有草稿状态可以更新
	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以更新")
	}

	// 更新表单数据
	formDataBytes, err := json.Marshal(req.FormData)
	if err != nil {
		i.l.Error("序列化表单数据失败", zap.Error(err))
		return fmt.Errorf("序列化表单数据失败: %w", err)
	}

	instance.Title = req.Title
	instance.FormData = string(formDataBytes)
	instance.Priority = req.Priority
	instance.CategoryID = req.CategoryID

	return i.dao.UpdateInstance(ctx, &instance)
}

// ProcessInstanceFlow 处理工单流程
func (i *instanceService) ProcessInstanceFlow(ctx context.Context, req model.InstanceFlowReq, operatorID int, operatorName string) error {
	// 1. 获取当前实例信息
	instance, err := i.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		i.l.Error("获取实例失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return err
	}

	// 检查工单状态
	if instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("当前工单状态不允许此操作")
	}

	// 检查操作人是否为当前处理人
	if instance.AssigneeID != 0 && instance.AssigneeID != operatorID {
		return fmt.Errorf("您不是当前工单的处理人")
	}

	// 2. 创建流程记录
	var formDataStr string
	// 检查FormData是否有内容
	if req.FormData.Reason != "" || req.FormData.Type != "" || req.FormData.ApproveDays != 0 || len(req.FormData.DateRange) > 0 {
		formDataBytes, err := json.Marshal(req.FormData)
		if err != nil {
			i.l.Error("序列化表单数据失败", zap.Error(err))
			return fmt.Errorf("序列化表单数据失败: %w", err)
		}
		formDataStr = string(formDataBytes)
	}

	flow := model.InstanceFlow{
		InstanceID:   req.InstanceID,
		Step:         instance.CurrentStep,
		Action:       req.Action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Comment:      req.Comment,
		FormData:     formDataStr,
	}

	if err := i.dao.CreateInstanceFlow(ctx, flow); err != nil {
		i.l.Error("创建实例流程记录失败", zap.Error(err))
		return err
	}

	// 3. 根据操作类型更新实例状态
	switch req.Action {
	case "approve":
		// 获取工作流定义
		workflow, err := i.dao.GetWorkflow(ctx, instance.WorkflowID)
		if err != nil {
			i.l.Error("获取工作流定义失败", zap.Error(err))
			return err
		}

		var workflowDef model.WorkflowDefinition
		if err := json.Unmarshal([]byte(workflow.Definition), &workflowDef); err != nil {
			i.l.Error("解析工作流定义失败", zap.Error(err))
			return err
		}

		// 查找当前步骤索引
		currentStepIndex := -1
		for idx, step := range workflowDef.Steps {
			if step.Step == instance.CurrentStep {
				currentStepIndex = idx
				break
			}
		}

		// 更新到下一步骤
		if currentStepIndex >= 0 && currentStepIndex < len(workflowDef.Steps)-1 {
			nextStep := workflowDef.Steps[currentStepIndex+1]
			instance.CurrentStep = nextStep.Step
			instance.CurrentRole = nextStep.Role

			// TODO: 根据角色查找下一步处理人
		} else {
			// 最后一步，完成工单
			instance.Status = model.InstanceStatusCompleted
			now := time.Now()
			instance.CompletedAt = &now
		}

	case "reject":
		instance.Status = model.InstanceStatusRejected
	case "cancel":
		instance.Status = model.InstanceStatusCancelled
	case "transfer":
		// TODO: 实现转交逻辑，需要更新AssigneeID和AssigneeName
	}

	// 4. 保存实例更新
	if err := i.dao.UpdateInstance(ctx, &instance); err != nil {
		i.l.Error("更新实例失败", zap.Error(err))
		return err
	}

	return nil
}

// CommentInstance 添加工单实例评论
func (i *instanceService) CommentInstance(ctx context.Context, req model.InstanceCommentReq, creatorID int, creatorName string) error {
	comment := model.InstanceComment{
		InstanceID:  req.InstanceID,
		Content:     req.Content,
		CreatorID:   creatorID,
		CreatorName: creatorName,
		ParentID:    req.ParentID,
	}

	if err := i.dao.CreateInstanceComment(ctx, comment); err != nil {
		i.l.Error("创建实例评论失败", zap.Error(err))
		return err
	}

	return nil
}

// GetWorkflowDefinition 获取工作流定义
func (i *instanceService) GetWorkflowDefinition(ctx context.Context, workflowID int) (model.WorkflowDefinition, error) {
	workflow, err := i.dao.GetWorkflow(ctx, workflowID)
	if err != nil {
		i.l.Error("获取工作流定义失败", zap.Error(err), zap.Int("workflowID", workflowID))
		return model.WorkflowDefinition{}, fmt.Errorf("获取工作流定义失败: %w", err)
	}

	var workflowDef model.WorkflowDefinition
	if err := json.Unmarshal([]byte(workflow.Definition), &workflowDef); err != nil {
		i.l.Error("解析工作流定义失败", zap.Error(err))
		return model.WorkflowDefinition{}, fmt.Errorf("解析工作流定义失败: %w", err)
	}

	return workflowDef, nil
}

// GetInstanceStatistics 获取工单统计信息
func (i *instanceService) GetInstanceStatistics(ctx context.Context) (interface{}, error) {
	// 获取各状态工单数量
	stats, err := i.dao.GetInstanceStatistics(ctx)
	if err != nil {
		return nil, err
	}

	// 获取最近工单趋势
	trend, err := i.dao.GetInstanceTrend(ctx)
	if err != nil {
		i.l.Warn("获取工单趋势失败", zap.Error(err))
		trend = []interface{}{}
	}

	return map[string]interface{}{
		"status_count": stats,
		"trend":        trend,
	}, nil
}
