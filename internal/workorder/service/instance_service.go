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
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type InstanceService interface {
	// 基础CRUD操作
	CreateInstance(ctx context.Context, req *model.CreateInstanceReq, creatorID int, creatorName string) error
	UpdateInstance(ctx context.Context, req *model.UpdateInstanceReq) error
	DeleteInstance(ctx context.Context, id int) error
	GetInstance(ctx context.Context, id int) (*model.InstanceResp, error)
	ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.InstanceResp], error)
	BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8) error

	// 流程操作
	ProcessInstanceFlow(ctx context.Context, req *model.InstanceActionReq, operatorID int, operatorName string) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error)
	GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error)

	// 评论功能
	CommentInstance(ctx context.Context, req *model.InstanceCommentReq, creatorID int, creatorName string) error
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error)

	// 附件功能
	UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachment, error)
	DeleteAttachment(ctx context.Context, instanceID int, attachmentID int) error
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error)
	BatchDeleteAttachments(ctx context.Context, instanceID int, attachmentIDs []int) error

	// 统计分析
	GetInstanceStatistics(ctx context.Context, req *model.OverviewStatsReq) (*model.OverviewStatsResp, error)
	GetInstanceTrend(ctx context.Context, req *model.TrendStatsReq) (*model.TrendStatsResp, error)
	GetCategoryStatistics(ctx context.Context, req *model.CategoryStatsReq) (*model.CategoryStatsResp, error)
	GetUserPerformanceStatistics(ctx context.Context, req *model.PerformanceStatsReq) (*model.PerformanceStatsResp, error)

	// 业务功能
	GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.InstanceResp], error)
	GetOverdueInstances(ctx context.Context) ([]model.InstanceResp, error)
	TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error
}

type instanceService struct {
	dao     dao.InstanceDAO
	userDAO userdao.UserDAO
	logger  *zap.Logger
}

func NewInstanceService(dao dao.InstanceDAO, userDAO userdao.UserDAO, logger *zap.Logger) InstanceService {
	return &instanceService{
		dao:     dao,
		userDAO: userDAO,
		logger:  logger,
	}
}

// CreateInstance 创建工单实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateInstanceReq, creatorID int, creatorName string) error {
	if req == nil {
		return fmt.Errorf("创建请求不能为空")
	}

	// 验证必填字段
	if req.Title == "" {
		return fmt.Errorf("工单标题不能为空")
	}
	if req.ProcessID == 0 {
		return fmt.Errorf("流程ID不能为空")
	}

	// 验证流程是否存在
	process, err := s.dao.GetProcess(ctx, req.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return fmt.Errorf("流程不存在或已禁用")
	}

	// 序列化表单数据
	var formData model.JSONMap
	if req.FormData != nil {
		formDataBytes, err := json.Marshal(req.FormData)
		if err != nil {
			s.logger.Error("序列化表单数据失败", zap.Error(err))
			return fmt.Errorf("表单数据格式错误")
		}
		if err := json.Unmarshal(formDataBytes, &formData); err != nil {
			s.logger.Error("反序列化表单数据失败", zap.Error(err))
			return fmt.Errorf("表单数据格式错误")
		}
	}

	// 处理标签
	var tags model.StringSlice
	if len(req.Tags) > 0 {
		tags = make(model.StringSlice, len(req.Tags))
		copy(tags, req.Tags)
	}

	// 解析流程定义，确定初始步骤
	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &processDef); err != nil {
		s.logger.Error("解析流程定义失败", zap.Error(err))
		return fmt.Errorf("流程定义格式错误")
	}

	var initialStep *model.ProcessStep
	var initialStatus int8 = model.InstanceStatusDraft

	// 查找开始步骤
	for _, step := range processDef.Steps {
		if step.Type == "start" {
			initialStep = &step
			initialStatus = model.InstanceStatusProcessing
			break
		}
	}

	if initialStep == nil && len(processDef.Steps) > 0 {
		// 如果没有明确的开始步骤，使用第一个步骤
		initialStep = &processDef.Steps[0]
		initialStatus = model.InstanceStatusProcessing
	}

	// 构建实例对象
	instance := &model.Instance{
		Title:       req.Title,
		TemplateID:  req.TemplateID,
		ProcessID:   req.ProcessID,
		FormData:    formData,
		Status:      initialStatus,
		Priority:    req.Priority,
		CategoryID:  req.CategoryID,
		CreatorID:   creatorID,
		CreatorName: creatorName,
		Description: req.Description,
		DueDate:     req.DueDate,
		Tags:        tags,
	}

	// 设置初始步骤和处理人
	if initialStep != nil {
		instance.CurrentStep = initialStep.ID

		// 分配初始处理人
		if req.AssigneeID != nil {
			instance.AssigneeID = req.AssigneeID
			if user, err := s.userDAO.GetUserByID(ctx, *req.AssigneeID); err == nil {
				instance.AssigneeName = user.Username
			}
		} else if len(initialStep.Users) > 0 {
			// 使用步骤定义中的第一个用户
			instance.AssigneeID = &initialStep.Users[0]
			if user, err := s.userDAO.GetUserByID(ctx, initialStep.Users[0]); err == nil {
				instance.AssigneeName = user.Username
			}
		}
	}

	// 创建实例
	if err := s.dao.CreateInstance(ctx, instance); err != nil {
		s.logger.Error("创建工单实例失败", zap.Error(err))
		return err
	}

	// 创建初始流程记录
	if initialStep != nil {
		flow := &model.InstanceFlow{
			InstanceID:   instance.ID,
			StepID:       initialStep.ID,
			StepName:     initialStep.Name,
			Action:       "create",
			OperatorID:   creatorID,
			OperatorName: creatorName,
			Comment:      "工单创建",
			FromStepID:   "",
			ToStepID:     initialStep.ID,
		}

		if err := s.dao.CreateInstanceFlow(ctx, flow); err != nil {
			s.logger.Warn("创建初始流程记录失败", zap.Error(err), zap.Int("instanceID", instance.ID))
		}
	}

	s.logger.Info("创建工单实例成功", zap.Int("instanceID", instance.ID), zap.String("title", instance.Title))
	return nil
}

// UpdateInstance 更新工单实例
func (s *instanceService) UpdateInstance(ctx context.Context, req *model.UpdateInstanceReq) error {
	if req == nil || req.ID == 0 {
		return fmt.Errorf("更新请求无效")
	}

	// 获取当前实例
	instance, err := s.dao.GetInstance(ctx, req.ID)
	if err != nil {
		return err
	}

	// 只有草稿状态和待处理状态可以更新基本信息
	if instance.Status != model.InstanceStatusDraft && instance.Status != model.InstanceStatusPending {
		return fmt.Errorf("当前状态的工单不允许修改")
	}

	// 更新字段
	if req.Title != "" {
		instance.Title = req.Title
	}
	if req.Description != "" {
		instance.Description = req.Description
	}
	if req.Priority != 0 {
		instance.Priority = req.Priority
	}
	if req.CategoryID != nil {
		instance.CategoryID = req.CategoryID
	}
	if req.DueDate != nil {
		instance.DueDate = req.DueDate
	}

	// 更新表单数据 - FormData 是 JSONMap 类型，直接赋值
	if req.FormData != nil {
		instance.FormData = req.FormData
	}

	// 更新标签 - Tags 是 StringSlice 类型，直接赋值
	if len(req.Tags) > 0 {
		instance.Tags = req.Tags
	}

	return s.dao.UpdateInstance(ctx, instance)
}

// DeleteInstance 删除工单实例
func (s *instanceService) DeleteInstance(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("工单ID无效")
	}

	// 检查工单状态，只有草稿状态可以删除
	instance, err := s.dao.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以删除")
	}

	return s.dao.DeleteInstance(ctx, id)
}

// GetInstance 获取工单实例详情
func (s *instanceService) GetInstance(ctx context.Context, id int) (*model.InstanceResp, error) {
	if id <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	instance, err := s.dao.GetInstanceWithRelations(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.convertToInstanceResp(instance), nil
}

// ListInstance 获取工单实例列表
func (s *instanceService) ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.InstanceResp], error) {
	if req == nil {
		req = &model.ListInstanceReq{}
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Size > 100 {
		req.Size = 100
	}

	result, err := s.dao.ListInstance(ctx, req)
	if err != nil {
		return nil, err
	}

	// 转换响应
	respItems := make([]model.InstanceResp, 0, len(result.Items))
	for _, item := range result.Items {
		respItems = append(respItems, *s.convertToInstanceResp(&item))
	}

	return &model.ListResp[model.InstanceResp]{
		Items: respItems,
		Total: result.Total,
	}, nil
}

// BatchUpdateInstanceStatus 批量更新工单状态
func (s *instanceService) BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8) error {
	if len(ids) == 0 {
		return fmt.Errorf("工单ID列表不能为空")
	}

	// 验证状态值
	validStatuses := []int8{
		model.InstanceStatusDraft,
		model.InstanceStatusPending,
		model.InstanceStatusProcessing,
		model.InstanceStatusCompleted,
		model.InstanceStatusCancelled,
		model.InstanceStatusRejected,
	}

	isValidStatus := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		return fmt.Errorf("无效的状态值: %d", status)
	}

	return s.dao.BatchUpdateInstanceStatus(ctx, ids, status)
}

// ProcessInstanceFlow 处理工单流程
func (s *instanceService) ProcessInstanceFlow(ctx context.Context, req *model.InstanceActionReq, operatorID int, operatorName string) error {
	if req == nil {
		return fmt.Errorf("操作请求不能为空")
	}

	// 获取工单实例
	instance, err := s.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}

	// 验证操作权限
	if err := s.validateOperationPermission(instance, operatorID, req.Action); err != nil {
		return err
	}

	// 获取流程定义
	process, err := s.dao.GetProcess(ctx, instance.ProcessID)
	if err != nil {
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &processDef); err != nil {
		return fmt.Errorf("解析流程定义失败: %w", err)
	}

	// 处理表单数据
	var formData model.JSONMap
	if req.FormData != nil {
		formDataBytes, err := json.Marshal(req.FormData)
		if err != nil {
			return fmt.Errorf("序列化表单数据失败: %w", err)
		}
		if err := json.Unmarshal(formDataBytes, &formData); err != nil {
			return fmt.Errorf("反序列化表单数据失败: %w", err)
		}
	}

	// 创建流程记录
	flow := &model.InstanceFlow{
		InstanceID:   req.InstanceID,
		StepID:       instance.CurrentStep,
		Action:       req.Action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Comment:      req.Comment,
		FormData:     formData,
		FromStepID:   instance.CurrentStep,
	}

	// 根据操作类型处理
	switch req.Action {
	case "approve":
		err = s.handleApproveAction(ctx, instance, &processDef, flow, req)
	case "reject":
		err = s.handleRejectAction(ctx, instance, flow)
	case "cancel":
		err = s.handleCancelAction(ctx, instance, flow)
	case "transfer":
		err = s.handleTransferAction(ctx, instance, flow, req.AssigneeID)
	case "revoke":
		err = s.handleRevokeAction(ctx, instance, flow)
	default:
		return fmt.Errorf("不支持的操作类型: %s", req.Action)
	}

	if err != nil {
		return err
	}

	// 保存流程记录
	if err := s.dao.CreateInstanceFlow(ctx, flow); err != nil {
		s.logger.Error("创建流程记录失败", zap.Error(err))
		return fmt.Errorf("创建流程记录失败: %w", err)
	}

	// 更新实例
	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新实例失败", zap.Error(err))
		return fmt.Errorf("更新实例失败: %w", err)
	}

	s.logger.Info("工单流程处理成功",
		zap.Int("instanceID", req.InstanceID),
		zap.String("action", req.Action),
		zap.Int("operatorID", operatorID))

	return nil
}

// CommentInstance 添加工单评论
func (s *instanceService) CommentInstance(ctx context.Context, req *model.InstanceCommentReq, creatorID int, creatorName string) error {
	if req == nil {
		return fmt.Errorf("评论请求不能为空")
	}

	if req.Content == "" {
		return fmt.Errorf("评论内容不能为空")
	}

	// 验证工单是否存在
	_, err := s.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}

	comment := &model.InstanceComment{
		InstanceID:  req.InstanceID,
		Content:     req.Content,
		CreatorID:   creatorID,
		CreatorName: creatorName,
		ParentID:    req.ParentID,
		IsSystem:    false,
	}

	return s.dao.CreateInstanceComment(ctx, comment)
}

// GetInstanceFlows 获取工单流程记录
func (s *instanceService) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	flows, err := s.dao.GetInstanceFlows(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	respFlows := make([]model.InstanceFlowResp, 0, len(flows))
	for _, flow := range flows {
		respFlow := s.convertToFlowResp(&flow)
		respFlows = append(respFlows, *respFlow)
	}

	return respFlows, nil
}

// GetInstanceComments 获取工单评论
func (s *instanceService) GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	comments, err := s.dao.GetInstanceComments(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	// 构建评论树
	return s.buildCommentTree(comments, nil), nil
}

// UploadAttachment 上传附件
func (s *instanceService) UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachment, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	// 验证工单是否存在
	_, err := s.dao.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	attachment := &model.InstanceAttachment{
		InstanceID:   instanceID,
		FileName:     fileName,
		FileSize:     fileSize,
		FilePath:     filePath,
		FileType:     fileType,
		UploaderID:   uploaderID,
		UploaderName: uploaderName,
	}

	return s.dao.CreateInstanceAttachment(ctx, attachment)
}

// DeleteAttachment 删除附件
func (s *instanceService) DeleteAttachment(ctx context.Context, instanceID int, attachmentID int) error {
	if instanceID <= 0 || attachmentID <= 0 {
		return fmt.Errorf("工单ID或附件ID无效")
	}

	return s.dao.DeleteInstanceAttachment(ctx, instanceID, attachmentID)
}

// GetInstanceAttachments 获取工单附件列表
func (s *instanceService) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error) {
	if instanceID <= 0 {
		return nil, fmt.Errorf("工单ID无效")
	}

	attachments, err := s.dao.GetInstanceAttachments(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	respAttachments := make([]model.InstanceAttachmentResp, 0, len(attachments))
	for _, att := range attachments {
		respAttachments = append(respAttachments, model.InstanceAttachmentResp{
			ID:           att.ID,
			InstanceID:   att.InstanceID,
			FileName:     att.FileName,
			FileSize:     att.FileSize,
			FilePath:     att.FilePath,
			FileType:     att.FileType,
			UploaderID:   att.UploaderID,
			UploaderName: att.UploaderName,
			CreatedAt:    att.CreatedAt,
		})
	}

	return respAttachments, nil
}

// BatchDeleteAttachments 批量删除附件
func (s *instanceService) BatchDeleteAttachments(ctx context.Context, instanceID int, attachmentIDs []int) error {
	if instanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}
	if len(attachmentIDs) == 0 {
		return fmt.Errorf("附件ID列表不能为空")
	}

	return s.dao.BatchDeleteInstanceAttachments(ctx, instanceID, attachmentIDs)
}

// GetProcessDefinition 获取流程定义
func (s *instanceService) GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error) {
	if processID <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	process, err := s.dao.GetProcess(ctx, processID)
	if err != nil {
		return nil, err
	}

	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &processDef); err != nil {
		return nil, fmt.Errorf("解析流程定义失败: %w", err)
	}

	return &processDef, nil
}

// GetInstanceStatistics 获取工单统计信息
func (s *instanceService) GetInstanceStatistics(ctx context.Context, req *model.OverviewStatsReq) (*model.OverviewStatsResp, error) {
	if req == nil {
		req = &model.OverviewStatsReq{}
	}

	return s.dao.GetInstanceStatistics(ctx, req)
}

// GetInstanceTrend 获取工单趋势
func (s *instanceService) GetInstanceTrend(ctx context.Context, req *model.TrendStatsReq) (*model.TrendStatsResp, error) {
	if req == nil {
		return nil, fmt.Errorf("趋势统计请求不能为空")
	}

	// 验证时间范围
	if req.StartDate.After(req.EndDate) {
		return nil, fmt.Errorf("开始时间不能晚于结束时间")
	}

	return s.dao.GetInstanceTrend(ctx, req)
}

// GetCategoryStatistics 获取分类统计
func (s *instanceService) GetCategoryStatistics(ctx context.Context, req *model.CategoryStatsReq) (*model.CategoryStatsResp, error) {
	if req == nil {
		req = &model.CategoryStatsReq{}
	}

	return s.dao.GetCategoryStatistics(ctx, req)
}

// GetUserPerformanceStatistics 获取用户绩效统计
func (s *instanceService) GetUserPerformanceStatistics(ctx context.Context, req *model.PerformanceStatsReq) (*model.PerformanceStatsResp, error) {
	if req == nil {
		req = &model.PerformanceStatsReq{}
	}

	return s.dao.GetUserPerformanceStatistics(ctx, req)
}

// GetMyInstances 获取我的工单
func (s *instanceService) GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.InstanceResp], error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}

	if req == nil {
		req = &model.MyInstanceReq{}
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	result, err := s.dao.GetMyInstances(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// 转换响应
	respItems := make([]model.InstanceResp, 0, len(result.Items))
	for _, item := range result.Items {
		respItems = append(respItems, *s.convertToInstanceResp(&item))
	}

	return &model.ListResp[model.InstanceResp]{
		Items: respItems,
		Total: result.Total,
	}, nil
}

// GetOverdueInstances 获取超时工单
func (s *instanceService) GetOverdueInstances(ctx context.Context) ([]model.InstanceResp, error) {
	instances, err := s.dao.GetOverdueInstances(ctx)
	if err != nil {
		return nil, err
	}

	respInstances := make([]model.InstanceResp, 0, len(instances))
	for _, instance := range instances {
		respInstances = append(respInstances, *s.convertToInstanceResp(&instance))
	}

	return respInstances, nil
}

// TransferInstance 转移工单
func (s *instanceService) TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error {
	if instanceID <= 0 || fromUserID <= 0 || toUserID <= 0 {
		return fmt.Errorf("参数无效")
	}

	if fromUserID == toUserID {
		return fmt.Errorf("转移目标用户不能是当前用户")
	}

	// 验证目标用户是否存在
	_, err := s.userDAO.GetUserByID(ctx, toUserID)
	if err != nil {
		return fmt.Errorf("目标用户不存在")
	}

	return s.dao.TransferInstance(ctx, instanceID, fromUserID, toUserID, comment)
}

// 辅助方法

// convertToInstanceResp 转换实例为响应格式
func (s *instanceService) convertToInstanceResp(instance *model.Instance) *model.InstanceResp {
	if instance == nil {
		return nil
	}

	// 解析表单数据
	var formData map[string]interface{}
	if instance.FormData != nil {
		formData = map[string]interface{}(instance.FormData)
	}

	// 解析标签
	var tags []string
	if instance.Tags != nil {
		tags = []string(instance.Tags)
	}

	resp := &model.InstanceResp{
		ID:           instance.ID,
		Title:        instance.Title,
		TemplateID:   instance.TemplateID,
		ProcessID:    instance.ProcessID,
		FormData:     formData,
		CurrentStep:  instance.CurrentStep,
		Status:       instance.Status,
		Priority:     instance.Priority,
		CategoryID:   instance.CategoryID,
		CreatorID:    instance.CreatorID,
		CreatorName:  instance.CreatorName,
		Description:  instance.Description,
		AssigneeID:   instance.AssigneeID,
		AssigneeName: instance.AssigneeName,
		CompletedAt:  instance.CompletedAt,
		DueDate:      instance.DueDate,
		Tags:         tags,
		CreatedAt:    instance.CreatedAt,
		UpdatedAt:    instance.UpdatedAt,
	}

	// 判断是否超时
	if instance.DueDate != nil && instance.DueDate.Before(time.Now()) &&
		instance.Status != model.InstanceStatusCompleted &&
		instance.Status != model.InstanceStatusCancelled &&
		instance.Status != model.InstanceStatusRejected {
		resp.IsOverdue = true
	}

	return resp
}

// convertToFlowResp 转换流程记录为响应格式
func (s *instanceService) convertToFlowResp(flow *model.InstanceFlow) *model.InstanceFlowResp {
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

// buildCommentTree 构建评论树
func (s *instanceService) buildCommentTree(comments []model.InstanceComment, parentID *int) []model.InstanceCommentResp {
	tree := make([]model.InstanceCommentResp, 0)

	for _, comment := range comments {
		// 检查是否为当前层级的评论
		if (parentID == nil && comment.ParentID == nil) ||
			(parentID != nil && comment.ParentID != nil && *parentID == *comment.ParentID) {

			children := s.buildCommentTree(comments, &comment.ID)

			respComment := model.InstanceCommentResp{
				ID:          comment.ID,
				InstanceID:  comment.InstanceID,
				Content:     comment.Content,
				CreatorID:   comment.CreatorID,
				CreatorName: comment.CreatorName,
				ParentID:    comment.ParentID,
				IsSystem:    comment.IsSystem,
				CreatedAt:   comment.CreatedAt,
				Children:    children,
			}

			tree = append(tree, respComment)
		}
	}

	return tree
}

// validateOperationPermission 验证操作权限
func (s *instanceService) validateOperationPermission(instance *model.Instance, operatorID int, action string) error {
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

// handleApproveAction 处理审批操作
func (s *instanceService) handleApproveAction(ctx context.Context, instance *model.Instance, processDef *model.ProcessDefinition, flow *model.InstanceFlow, req *model.InstanceActionReq) error {
	// 查找当前步骤
	var currentStep *model.ProcessStep
	for _, step := range processDef.Steps {
		if step.ID == instance.CurrentStep {
			currentStep = &step
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
			// TODO: 这里可以添加条件判断逻辑
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
	for _, step := range processDef.Steps {
		if step.ID == nextStepID {
			nextStep = &step
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
			if user, err := s.userDAO.GetUserByID(ctx, nextStep.Users[0]); err == nil {
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
func (s *instanceService) handleRejectAction(ctx context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusRejected
	flow.ToStepID = instance.CurrentStep
	return nil
}

// handleCancelAction 处理取消操作
func (s *instanceService) handleCancelAction(ctx context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusCancelled
	flow.ToStepID = instance.CurrentStep
	return nil
}

// handleTransferAction 处理转移操作
func (s *instanceService) handleTransferAction(ctx context.Context, instance *model.Instance, flow *model.InstanceFlow, assigneeID *int) error {
	if assigneeID == nil || *assigneeID == 0 {
		return fmt.Errorf("转移操作需要指定有效的处理人")
	}

	// 验证目标用户是否存在
	user, err := s.userDAO.GetUserByID(ctx, *assigneeID)
	if err != nil {
		return fmt.Errorf("目标用户不存在")
	}

	instance.AssigneeID = assigneeID
	instance.AssigneeName = user.Username
	flow.ToStepID = instance.CurrentStep

	return nil
}

// handleRevokeAction 处理撤销操作
func (s *instanceService) handleRevokeAction(ctx context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusDraft
	instance.AssigneeID = nil
	instance.AssigneeName = ""
	flow.ToStepID = instance.CurrentStep
	return nil
}
