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
	"strconv"
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
	CreateInstanceFromTemplate(ctx context.Context, templateID int, req *model.CreateWorkorderInstanceFromTemplateReq) error
	UpdateInstance(ctx context.Context, req *model.UpdateWorkorderInstanceReq) error
	DeleteInstance(ctx context.Context, id int) error
	GetInstance(ctx context.Context, id int) (*model.WorkorderInstance, error)
	ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) (*model.ListResp[*model.WorkorderInstance], error)
	SubmitInstance(ctx context.Context, id int, operatorID int, operatorName string) error
	AssignInstance(ctx context.Context, id int, assigneeID int, operatorID int, operatorName string) error
	ApproveInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	RejectInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	GetAvailableActions(ctx context.Context, instanceID int, operatorID int) ([]string, error)
	GetCurrentStep(ctx context.Context, instanceID int) (*model.ProcessStep, error)
}

type instanceService struct {
	dao           dao.WorkorderInstanceDAO
	flowDao       dao.WorkorderInstanceFlowDAO
	timelineDao   dao.WorkorderInstanceTimelineDAO
	commentDao    dao.WorkorderInstanceCommentDAO
	processDao    dao.WorkorderProcessDAO
	formDesignDao dao.WorkorderFormDesignDAO
	templateDao   dao.WorkorderTemplateDAO
	logger        *zap.Logger
}

func NewInstanceService(
	dao dao.WorkorderInstanceDAO,
	flowDao dao.WorkorderInstanceFlowDAO,
	timelineDao dao.WorkorderInstanceTimelineDAO,
	commentDao dao.WorkorderInstanceCommentDAO,
	processDao dao.WorkorderProcessDAO,
	formDesignDao dao.WorkorderFormDesignDAO,
	templateDao dao.WorkorderTemplateDAO,
	logger *zap.Logger,
) InstanceService {
	return &instanceService{
		dao:           dao,
		flowDao:       flowDao,
		timelineDao:   timelineDao,
		commentDao:    commentDao,
		processDao:    processDao,
		formDesignDao: formDesignDao,
		templateDao:   templateDao,
		logger:        logger,
	}
}

// CreateInstance 创建工单实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) error {
	if req.Status < model.InstanceStatusDraft || req.Status > model.InstanceStatusCancelled {
		return fmt.Errorf("工单状态无效")
	}
	if req.Priority < model.PriorityHigh || req.Priority > model.PriorityLow {
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

	// 验证流程是否存在并获取关联的表单设计
	process, err := s.processDao.GetProcessByID(ctx, req.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return fmt.Errorf("流程不存在或已停用")
	}

	// 验证流程状态
	if process.Status != model.ProcessStatusPublished {
		return fmt.Errorf("只能使用已发布的流程创建工单")
	}

	// 验证表单数据
	if err := s.validateFormData(ctx, process.FormDesignID, req.FormData); err != nil {
		s.logger.Error("表单数据验证失败", zap.Error(err), zap.Int("formDesignID", process.FormDesignID))
		return fmt.Errorf("表单数据验证失败: %w", err)
	}

	// 生成工单编号
	serialNumber, err := s.dao.GenerateSerialNumber(ctx)
	if err != nil {
		s.logger.Error("生成工单编号失败", zap.Error(err))
		return err
	}

	// 根据流程定义设置初始当前步骤
	var initialStepID *string
	if process.Definition != nil {
		var definition model.ProcessDefinition
		definitionBytes, _ := json.Marshal(process.Definition)
		if json.Unmarshal(definitionBytes, &definition) == nil && len(definition.Steps) > 0 {
			// 如果是草稿状态，设置为开始步骤；否则设置为第一个非开始步骤
			if req.Status == model.InstanceStatusDraft {
				for _, step := range definition.Steps {
					if step.Type == model.ProcessStepTypeStart {
						initialStepID = &step.ID
						break
					}
				}
			} else {
				// 对于已提交的工单，设置为第一个非开始步骤
				for _, step := range definition.Steps {
					if step.Type != model.ProcessStepTypeStart {
						initialStepID = &step.ID
						break
					}
				}
			}
		}
	}

	instance := &model.WorkorderInstance{
		Title:         req.Title,
		SerialNumber:  serialNumber,
		ProcessID:     req.ProcessID,
		CurrentStepID: initialStepID,
		FormData:      req.FormData,
		Status:        req.Status,
		Priority:      req.Priority,
		OperatorID:    req.OperatorID,
		OperatorName:  req.OperatorName,
		AssigneeID:    req.AssigneeID,
		Description:   req.Description,
		Tags:          req.Tags,
		DueDate:       req.DueDate,
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

// CreateInstanceFromTemplate 从模板创建工单实例
func (s *instanceService) CreateInstanceFromTemplate(ctx context.Context, templateID int, req *model.CreateWorkorderInstanceFromTemplateReq) error {
	if req.Priority < model.PriorityHigh || req.Priority > model.PriorityLow {
		return fmt.Errorf("优先级无效")
	}

	// 获取模板信息
	template, err := s.templateDao.GetTemplate(ctx, templateID)
	if err != nil {
		s.logger.Error("获取工单模板失败", zap.Error(err), zap.Int("templateID", templateID))
		return fmt.Errorf("工单模板不存在或已禁用")
	}

	// 验证模板状态
	if template.Status != model.TemplateStatusEnabled {
		return fmt.Errorf("只能使用启用状态的模板创建工单")
	}

	// 合并表单数据：模板默认值 + 用户提交的数据
	formData := make(model.JSONMap)

	// 先使用模板的默认值
	if template.DefaultValues != nil {
		for key, value := range template.DefaultValues {
			formData[key] = value
		}
	}

	// 用户提交的数据覆盖默认值
	if req.FormData != nil {
		for key, value := range req.FormData {
			formData[key] = value
		}
	}

	// 验证流程是否存在并获取关联的表单设计
	process, err := s.processDao.GetProcessByID(ctx, template.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", template.ProcessID))
		return fmt.Errorf("流程不存在或已停用")
	}

	// 验证流程状态
	if process.Status != model.ProcessStatusPublished {
		return fmt.Errorf("只能使用已发布的流程创建工单")
	}

	// 验证表单数据
	if err := s.validateFormData(ctx, process.FormDesignID, formData); err != nil {
		s.logger.Error("表单数据验证失败", zap.Error(err), zap.Int("formDesignID", process.FormDesignID))
		return fmt.Errorf("表单数据验证失败: %w", err)
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

	// 生成工单编号
	serialNumber, err := s.dao.GenerateSerialNumber(ctx)
	if err != nil {
		s.logger.Error("生成工单编号失败", zap.Error(err))
		return err
	}

	// 创建工单实例
	instance := &model.WorkorderInstance{
		Title:        req.Title,
		SerialNumber: serialNumber,
		ProcessID:    template.ProcessID,
		FormData:     formData,
		Status:       model.InstanceStatusDraft, // 从模板创建的工单默认为草稿状态
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
	s.createFlowRecord(ctx, instance.ID, model.FlowActionSubmit, req.OperatorID, req.OperatorName, 0, model.InstanceStatusDraft, "", 1)

	// 创建时间线记录
	s.createTimelineRecord(ctx, instance.ID, model.TimelineActionCreate, req.OperatorID, req.OperatorName, fmt.Sprintf("从模板 %s 创建工单", template.Name))

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

	// 只允许删除草稿、已完成、已拒绝状态的工单
	if instance.Status != model.InstanceStatusDraft &&
		instance.Status != model.InstanceStatusCompleted &&
		instance.Status != model.InstanceStatusRejected {
		return fmt.Errorf("只有草稿、已完成或已拒绝状态的工单可以删除")
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

	// 验证当前状态是否允许提交
	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以提交")
	}

	// 检查操作权限
	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionSubmit {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限提交此工单")
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusPending

	// 获取流程定义，确定提交后的第一个实际步骤
	process, err := s.processDao.GetProcessByID(ctx, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", instance.ProcessID))
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	if process.Definition != nil {
		var definition model.ProcessDefinition
		definitionBytes, _ := json.Marshal(process.Definition)
		if json.Unmarshal(definitionBytes, &definition) == nil {
			// 找到第一个非开始步骤作为提交后的当前步骤
			for _, step := range definition.Steps {
				if step.Type != model.ProcessStepTypeStart {
					instance.CurrentStepID = &step.ID
					s.logger.Info("设置提交后的当前步骤",
						zap.Int("instanceID", id),
						zap.String("stepID", step.ID),
						zap.String("stepName", step.Name))
					break
				}
			}
		}
	}

	// 更新状态和当前步骤
	instance.Status = toStatus
	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
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

	// 验证当前状态是否允许指派
	if instance.Status != model.InstanceStatusPending {
		return fmt.Errorf("只有待处理状态的工单可以指派")
	}

	// 检查操作权限
	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionAssign {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限指派此工单")
	}

	// 验证受理人是否有效
	if assigneeID <= 0 {
		return fmt.Errorf("无效的受理人ID")
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

	// 验证当前状态是否允许审批
	if instance.Status != model.InstanceStatusPending && instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有待处理或处理中状态的工单可以审批")
	}

	// 检查操作权限
	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionApprove {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限审批此工单")
	}

	// 获取当前步骤
	currentStep, err := s.GetCurrentStep(ctx, id)
	if err != nil {
		s.logger.Error("获取当前步骤失败", zap.Error(err), zap.Int("instanceID", id))
		return fmt.Errorf("获取当前步骤失败: %w", err)
	}

	// 获取流程定义以查找下一个步骤
	process, err := s.processDao.GetProcessByID(ctx, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", instance.ProcessID))
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 解析流程定义
	var definition model.ProcessDefinition
	definitionBytes, err := json.Marshal(process.Definition)
	if err != nil {
		s.logger.Error("流程定义序列化失败", zap.Error(err))
		return fmt.Errorf("流程定义序列化失败: %w", err)
	}

	if err := json.Unmarshal(definitionBytes, &definition); err != nil {
		s.logger.Error("流程定义解析失败", zap.Error(err))
		return fmt.Errorf("流程定义解析失败: %w", err)
	}

	// 获取下一个步骤
	nextStep := s.getNextStep(currentStep, definition)

	fromStatus := instance.Status
	var toStatus int8
	var completedAt *time.Time

	if nextStep == nil || nextStep.Type == model.ProcessStepTypeEnd {
		// 没有下一个步骤或下一个步骤是结束节点，标记为完成
		toStatus = model.InstanceStatusCompleted
		now := time.Now()
		completedAt = &now
		s.logger.Info("工单审批完成，进入结束状态", zap.Int("instanceID", id))
	} else {
		// 有下一个步骤，进入下一个步骤对应的状态
		toStatus = s.getStatusForStep(nextStep)
		s.logger.Info("工单审批通过，进入下一步骤",
			zap.Int("instanceID", id),
			zap.String("nextStepID", nextStep.ID),
			zap.String("nextStepType", nextStep.Type),
			zap.Int8("nextStatus", toStatus))
	}

	// 更新工单状态和当前步骤
	instance.Status = toStatus
	if completedAt != nil {
		instance.CompletedAt = completedAt
	}

	// 更新当前步骤ID
	if nextStep != nil {
		instance.CurrentStepID = &nextStep.ID
		s.logger.Info("更新工单当前步骤",
			zap.Int("instanceID", id),
			zap.String("nextStepID", nextStep.ID),
			zap.String("nextStepName", nextStep.Name))
	} else {
		// 如果没有下一个步骤（流程结束），清空CurrentStepID
		instance.CurrentStepID = nil
		s.logger.Info("流程结束，清空当前步骤ID", zap.Int("instanceID", id))
	}

	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	// 创建流转记录
	s.createFlowRecord(ctx, id, model.FlowActionApprove, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	// 创建时间线记录
	timelineComment := fmt.Sprintf("工单审批通过: %s", comment)
	if nextStep != nil && nextStep.Type != model.ProcessStepTypeEnd {
		timelineComment += fmt.Sprintf("，进入步骤: %s", nextStep.Name)
	}
	s.createTimelineRecord(ctx, id, model.TimelineActionApprove, operatorID, operatorName, timelineComment)

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

	// 验证当前状态是否允许拒绝
	if instance.Status != model.InstanceStatusPending && instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有待处理或处理中状态的工单可以拒绝")
	}

	// 检查操作权限
	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionReject {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限拒绝此工单")
	}

	// 验证拒绝理由
	if comment == "" {
		return fmt.Errorf("拒绝工单必须提供理由")
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

// validateFormData 验证表单数据
func (s *instanceService) validateFormData(ctx context.Context, formDesignID int, formData model.JSONMap) error {
	if formDesignID <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	// 获取表单设计
	formDesign, err := s.formDesignDao.GetFormDesign(ctx, formDesignID)
	if err != nil {
		return fmt.Errorf("获取表单设计失败: %w", err)
	}

	// 验证表单设计状态
	if formDesign.Status != model.FormDesignStatusPublished {
		return fmt.Errorf("只能使用已发布的表单设计")
	}

	// 解析表单Schema
	var schema model.FormSchema
	schemaBytes, err := json.Marshal(formDesign.Schema)
	if err != nil {
		return fmt.Errorf("表单Schema序列化失败: %w", err)
	}

	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return fmt.Errorf("表单Schema解析失败: %w", err)
	}

	// 验证每个字段
	for _, field := range schema.Fields {
		if err := s.validateFormField(field, formData); err != nil {
			return fmt.Errorf("字段 %s 验证失败: %w", field.Label, err)
		}
	}

	return nil
}

// validateFormField 验证单个表单字段
func (s *instanceService) validateFormField(field model.FormField, formData model.JSONMap) error {
	value, exists := formData[field.ID]

	// 检查必填字段
	if field.Required == 1 && (!exists || s.isEmptyValue(value)) {
		return fmt.Errorf("字段 %s 为必填项", field.Label)
	}

	// 如果字段不存在或为空，且不是必填的，则跳过验证
	if !exists || s.isEmptyValue(value) {
		return nil
	}

	// 根据字段类型进行验证
	switch field.Type {
	case model.FormFieldTypeText, model.FormFieldTypePassword, model.FormFieldTypeTextarea:
		return s.validateStringField(field, value)
	case model.FormFieldTypeNumber:
		return s.validateNumberField(field, value)
	case model.FormFieldTypeSelect, model.FormFieldTypeRadio:
		return s.validateSelectField(field, value)
	case model.FormFieldTypeCheckbox:
		return s.validateCheckboxField(field, value)
	case model.FormFieldTypeDate:
		return s.validateDateField(field, value)
	case model.FormFieldTypeSwitch:
		return s.validateSwitchField(field, value)
	default:
		s.logger.Warn("未知的字段类型", zap.String("type", field.Type), zap.String("fieldID", field.ID))
		return nil
	}
}

// isEmptyValue 检查值是否为空
func (s *instanceService) isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

// validateStringField 验证字符串字段
func (s *instanceService) validateStringField(field model.FormField, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("期望字符串类型，实际类型: %T", value)
	}

	// 这里可以添加更多的字符串验证逻辑，如长度限制等
	if len(str) > 2000 { // 假设最大长度为2000
		return fmt.Errorf("字符串长度超过限制")
	}

	return nil
}

// validateNumberField 验证数字字段
func (s *instanceService) validateNumberField(field model.FormField, value interface{}) error {
	switch v := value.(type) {
	case float64, int, int64:
		return nil
	case string:
		// 尝试转换字符串到数字
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return fmt.Errorf("无法解析为数字: %s", v)
		}
		return nil
	default:
		return fmt.Errorf("期望数字类型，实际类型: %T", value)
	}
}

// validateSelectField 验证选择字段
func (s *instanceService) validateSelectField(field model.FormField, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("期望字符串类型，实际类型: %T", value)
	}

	// 检查值是否在选项列表中
	if len(field.Options) > 0 {
		for _, option := range field.Options {
			if option == str {
				return nil
			}
		}
		return fmt.Errorf("值 %s 不在可选项中", str)
	}

	return nil
}

// validateCheckboxField 验证复选框字段
func (s *instanceService) validateCheckboxField(field model.FormField, value interface{}) error {
	// 复选框可以是数组或单个值
	switch v := value.(type) {
	case []interface{}:
		// 验证数组中的每个值
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("期望字符串数组，实际包含类型: %T", item)
			}

			// 检查值是否在选项列表中
			if len(field.Options) > 0 {
				found := false
				for _, option := range field.Options {
					if option == str {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("值 %s 不在可选项中", str)
				}
			}
		}
		return nil
	case string:
		// 单个值的情况
		return s.validateSelectField(field, v)
	default:
		return fmt.Errorf("期望字符串或字符串数组类型，实际类型: %T", value)
	}
}

// validateDateField 验证日期字段
func (s *instanceService) validateDateField(field model.FormField, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("期望字符串类型，实际类型: %T", value)
	}

	// 尝试解析日期
	if _, err := time.Parse("2006-01-02", str); err != nil {
		if _, err := time.Parse("2006-01-02T15:04:05Z07:00", str); err != nil {
			return fmt.Errorf("无法解析日期: %s", str)
		}
	}

	return nil
}

// validateSwitchField 验证开关字段
func (s *instanceService) validateSwitchField(field model.FormField, value interface{}) error {
	switch v := value.(type) {
	case bool:
		return nil
	case string:
		if v == "true" || v == "false" || v == "1" || v == "0" {
			return nil
		}
		return fmt.Errorf("无效的开关值: %s", v)
	case int, int64, float64:
		return nil
	default:
		return fmt.Errorf("期望布尔或数字类型，实际类型: %T", value)
	}
}

// GetCurrentStep 获取工单当前步骤
func (s *instanceService) GetCurrentStep(ctx context.Context, instanceID int) (*model.ProcessStep, error) {
	instance, err := s.dao.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	s.logger.Debug("获取到工单实例", zap.Int("processID", instance.ProcessID), zap.Int8("status", instance.Status))

	process, err := s.processDao.GetProcessByID(ctx, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", instance.ProcessID))
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	s.logger.Debug("获取到流程定义", zap.String("processName", process.Name))

	// 检查流程定义是否为空
	if process.Definition == nil {
		s.logger.Error("流程定义为空", zap.Int("processID", instance.ProcessID))
		return nil, fmt.Errorf("流程定义为空")
	}

	// 解析流程定义
	var definition model.ProcessDefinition
	definitionBytes, err := json.Marshal(process.Definition)
	if err != nil {
		s.logger.Error("流程定义序列化失败", zap.Error(err))
		return nil, fmt.Errorf("流程定义序列化失败: %w", err)
	}

	if err := json.Unmarshal(definitionBytes, &definition); err != nil {
		s.logger.Error("流程定义解析失败", zap.Error(err))
		return nil, fmt.Errorf("流程定义解析失败: %w", err)
	}

	s.logger.Debug("解析流程定义成功", zap.Int("stepCount", len(definition.Steps)))

	var currentStep *model.ProcessStep

	// 优先使用CurrentStepID精确查找
	if instance.CurrentStepID != nil && *instance.CurrentStepID != "" {
		s.logger.Debug("使用CurrentStepID查找当前步骤", zap.String("currentStepID", *instance.CurrentStepID))
		for i := range definition.Steps {
			if definition.Steps[i].ID == *instance.CurrentStepID {
				currentStep = &definition.Steps[i]
				s.logger.Debug("通过CurrentStepID找到当前步骤",
					zap.String("stepID", currentStep.ID),
					zap.String("stepName", currentStep.Name),
					zap.String("stepType", currentStep.Type))
				break
			}
		}
	}

	// 如果通过CurrentStepID没找到，则回退到状态映射方式
	if currentStep == nil {
		s.logger.Debug("CurrentStepID未找到步骤，使用状态映射查找", zap.Int8("status", instance.Status))
		currentStep = s.findStepByStatus(definition.Steps, instance.Status)
		if currentStep == nil {
			s.logger.Error("未找到匹配状态的流程步骤", zap.Int8("status", instance.Status), zap.Int("stepCount", len(definition.Steps)))
			return nil, fmt.Errorf("未找到匹配状态的流程步骤")
		}

		// 如果CurrentStepID为空，更新它
		if instance.CurrentStepID == nil || *instance.CurrentStepID == "" {
			s.logger.Info("更新工单CurrentStepID",
				zap.Int("instanceID", instanceID),
				zap.String("stepID", currentStep.ID))
			instance.CurrentStepID = &currentStep.ID
			if updateErr := s.dao.UpdateInstance(ctx, instance); updateErr != nil {
				s.logger.Error("更新CurrentStepID失败", zap.Error(updateErr))
			}
		}
	}

	s.logger.Debug("找到当前步骤", zap.String("stepID", currentStep.ID),
		zap.String("stepName", currentStep.Name), zap.String("stepType", currentStep.Type))

	return currentStep, nil
}

// GetAvailableActions 获取可执行的动作
func (s *instanceService) GetAvailableActions(ctx context.Context, instanceID int, operatorID int) ([]string, error) {
	instance, err := s.dao.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	s.logger.Debug("工单实例信息", zap.Int8("status", instance.Status), zap.Int("processID", instance.ProcessID))

	currentStep, err := s.GetCurrentStep(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取当前步骤失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取当前步骤失败: %w", err)
	}

	if currentStep == nil {
		s.logger.Warn("当前步骤为空", zap.Int("instanceID", instanceID))
		return []string{}, nil
	}

	s.logger.Debug("当前步骤信息", zap.String("stepID", currentStep.ID), zap.String("stepType", currentStep.Type), zap.String("assigneeType", currentStep.AssigneeType))

	// 检查操作权限
	canOperate := s.canUserOperate(currentStep, operatorID, instance.AssigneeID)
	s.logger.Debug("权限检查结果", zap.Bool("canOperate", canOperate))

	if !canOperate {
		s.logger.Info("用户无权限操作此工单", zap.Int("operatorID", operatorID), zap.Int("instanceID", instanceID))
		return []string{}, nil // 无权限操作
	}

	// 根据工单状态和步骤类型返回可用动作
	actions := s.getActionsForStep(currentStep, instance.Status)
	s.logger.Debug("获取到的可用动作", zap.Strings("actions", actions))

	return actions, nil
}

// findStepByStatus 根据状态找到对应的流程步骤
func (s *instanceService) findStepByStatus(steps []model.ProcessStep, status int8) *model.ProcessStep {
	if len(steps) == 0 {
		s.logger.Warn("流程步骤为空")
		return nil
	}

	// 简化的状态映射逻辑，实际应该根据具体业务需求调整
	switch status {
	case model.InstanceStatusDraft:
		// 草稿状态对应开始步骤
		for i := range steps {
			if steps[i].Type == model.ProcessStepTypeStart {
				return &steps[i]
			}
		}
		s.logger.Debug("未找到开始步骤，使用第一个步骤", zap.Int8("status", status))
	case model.InstanceStatusPending, model.InstanceStatusProcessing:
		// 待处理/处理中状态对应审批或任务步骤
		for i := range steps {
			if steps[i].Type == model.ProcessStepTypeApproval || steps[i].Type == model.ProcessStepTypeTask {
				return &steps[i]
			}
		}
		s.logger.Debug("未找到审批或任务步骤，使用第一个步骤", zap.Int8("status", status))
	case model.InstanceStatusCompleted, model.InstanceStatusRejected, model.InstanceStatusCancelled:
		// 完成状态对应结束步骤
		for i := range steps {
			if steps[i].Type == model.ProcessStepTypeEnd {
				return &steps[i]
			}
		}
		s.logger.Debug("未找到结束步骤，使用第一个步骤", zap.Int8("status", status))
	}

	// 如果没找到合适的步骤，返回第一个步骤作为默认
	s.logger.Info("使用默认步骤", zap.Int8("status", status), zap.String("stepType", steps[0].Type))
	return &steps[0]
}

// canUserOperate 检查用户是否可以操作当前步骤
func (s *instanceService) canUserOperate(step *model.ProcessStep, operatorID int, assigneeID *int) bool {
	if step == nil {
		s.logger.Warn("步骤为空，拒绝操作")
		return false
	}

	// 如果工单已指派，只有指派人可以操作（优先级最高）
	if assigneeID != nil && *assigneeID == operatorID {
		return true
	}

	// 检查受理人类型
	switch step.AssigneeType {
	case model.AssigneeTypeUser:
		// 用户类型：检查操作人是否在受理人列表中
		for _, id := range step.AssigneeIDs {
			if id == operatorID {
				return true
			}
		}
		// 如果没有配置受理人列表，允许任何用户操作（兼容性处理）
		if len(step.AssigneeIDs) == 0 {
			s.logger.Debug("步骤未配置受理人列表，允许操作", zap.String("stepID", step.ID))
			return true
		}
		return false
	case model.AssigneeTypeGroup, "":
		// 系统类型或未配置类型：允许操作（兼容性处理）
		return true
	default:
		s.logger.Warn("未知的受理人类型", zap.String("assigneeType", step.AssigneeType))
		return true // 默认允许操作，避免阻塞
	}
}

// getActionsForStep 获取步骤可执行的动作
func (s *instanceService) getActionsForStep(step *model.ProcessStep, currentStatus int8) []string {
	var actions []string

	// 基础动作：从步骤定义中获取
	actions = append(actions, step.Actions...)

	// 根据当前状态添加额外的动作
	switch currentStatus {
	case model.InstanceStatusDraft:
		actions = append(actions, model.FlowActionSubmit)
	case model.InstanceStatusPending:
		actions = append(actions, model.FlowActionAssign, model.FlowActionApprove, model.FlowActionReject)
	case model.InstanceStatusProcessing:
		actions = append(actions, model.FlowActionComplete, model.FlowActionReturn, model.FlowActionApprove, model.FlowActionReject)
	}

	// 去重
	actionSet := make(map[string]bool)
	var uniqueActions []string
	for _, action := range actions {
		if !actionSet[action] {
			actionSet[action] = true
			uniqueActions = append(uniqueActions, action)
		}
	}

	return uniqueActions
}

// getNextStep 获取下一个流程步骤
func (s *instanceService) getNextStep(currentStep *model.ProcessStep, definition model.ProcessDefinition) *model.ProcessStep {
	if currentStep == nil {
		s.logger.Warn("当前步骤为空，无法获取下一个步骤")
		return nil
	}

	// 查找从当前步骤出发的连接
	var nextStepID string
	for _, connection := range definition.Connections {
		if connection.From == currentStep.ID {
			nextStepID = connection.To
			break
		}
	}

	if nextStepID == "" {
		s.logger.Info("未找到当前步骤的下一个步骤", zap.String("currentStepID", currentStep.ID))
		return nil
	}

	// 查找下一个步骤的详细信息
	for i := range definition.Steps {
		if definition.Steps[i].ID == nextStepID {
			s.logger.Debug("找到下一个步骤",
				zap.String("currentStepID", currentStep.ID),
				zap.String("nextStepID", nextStepID),
				zap.String("nextStepType", definition.Steps[i].Type))
			return &definition.Steps[i]
		}
	}

	s.logger.Warn("未找到下一个步骤的详细信息",
		zap.String("currentStepID", currentStep.ID),
		zap.String("nextStepID", nextStepID))
	return nil
}

// getStatusForStep 根据步骤类型获取对应的工单状态
func (s *instanceService) getStatusForStep(step *model.ProcessStep) int8 {
	if step == nil {
		s.logger.Warn("步骤为空，返回默认状态")
		return model.InstanceStatusPending
	}

	switch step.Type {
	case model.ProcessStepTypeStart:
		return model.InstanceStatusDraft
	case model.ProcessStepTypeApproval:
		return model.InstanceStatusPending
	case model.ProcessStepTypeTask:
		return model.InstanceStatusProcessing
	case model.ProcessStepTypeEnd:
		return model.InstanceStatusCompleted
	default:
		s.logger.Warn("未知的步骤类型，返回默认状态",
			zap.String("stepType", step.Type),
			zap.String("stepID", step.ID))
		return model.InstanceStatusPending
	}
}
