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
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

// 错误定义
var (
	ErrInvalidRequest        = errors.New("请求参数无效")
	ErrInstanceNotFound      = errors.New("工单实例不存在")
	ErrProcessNotFound       = errors.New("流程定义不存在")
	ErrUnauthorized          = errors.New("无权限执行此操作")
	ErrInvalidStatus         = errors.New("工单状态无效")
	ErrInvalidAction         = errors.New("操作类型无效")
	ErrUserNotFound          = errors.New("用户不存在")
	ErrInstanceStatusChanged = errors.New("工单状态已变更，无法操作")
	ErrProcessDefinition     = errors.New("流程定义解析失败")
)

// 常量定义
const (
	DefaultPageSize      = 20
	MaxPageSize          = 100
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MaxCommentLength     = 1000
	MaxFileNameLength    = 255
	MaxFileSize          = 100 * 1024 * 1024 // 100MB
)

type InstanceService interface {
	// 基础CRUD操作
	CreateInstance(ctx context.Context, req *model.CreateInstanceReq, creatorID int, creatorName string) (*model.InstanceResp, error)
	UpdateInstance(ctx context.Context, req *model.UpdateInstanceReq, operatorID int) error
	DeleteInstance(ctx context.Context, id int, operatorID int) error
	GetInstance(ctx context.Context, id int) (*model.InstanceResp, error)
	ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.InstanceResp], error)
	BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8, operatorID int) error

	// 业务功能
	GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.InstanceResp], error)
	GetOverdueInstances(ctx context.Context) ([]model.InstanceResp, error)
	TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error
}

type instanceService struct {
	dao        dao.InstanceDAO
	processDao dao.ProcessDAO
	flowDao    dao.InstanceFlowDAO
	userDao    userdao.UserDAO
	logger     *zap.Logger
}

func NewInstanceService(
	dao dao.InstanceDAO,
	processDao dao.ProcessDAO,
	flowDao dao.InstanceFlowDAO,
	userDao userdao.UserDAO,
	logger *zap.Logger,
) InstanceService {
	return &instanceService{
		dao:        dao,
		userDao:    userDao,
		processDao: processDao,
		flowDao:    flowDao,
		logger:     logger,
	}
}

// CreateInstance 创建工单实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateInstanceReq, creatorID int, creatorName string) (*model.InstanceResp, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("参数验证失败: %w", err)
	}

	// 验证流程是否存在并取出表单数据
	process, err := s.processDao.GetProcess(ctx, req.ProcessID)
	if err != nil {
		if errors.Is(err, dao.ErrProcessNotFound) {
			return nil, ErrProcessNotFound
		}
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 创建工单实例
	instance, err := s.buildInstanceFromRequest(ctx, req, creatorID, creatorName, process)
	if err != nil {
		return nil, fmt.Errorf("构建工单实例失败: %w", err)
	}

	// 保存工单实例
	if err := s.dao.CreateInstance(ctx, instance); err != nil {
		s.logger.Error("创建工单实例失败", zap.Error(err))
		return nil, fmt.Errorf("创建工单实例失败: %w", err)
	}

	// 创建初始流程记录
	if err := s.createInitialFlow(ctx, instance, creatorID, creatorName); err != nil {
		s.logger.Warn("创建初始流程记录失败", zap.Error(err), zap.Int("instanceID", instance.ID))
	}

	s.logger.Info("创建工单实例成功",
		zap.Int("instanceID", instance.ID),
		zap.String("title", instance.Title),
		zap.Int("creatorID", creatorID))

	return s.convertToInstanceResp(instance), nil
}

// UpdateInstance 更新工单实例
func (s *instanceService) UpdateInstance(ctx context.Context, req *model.UpdateInstanceReq, operatorID int) error {
	if err := s.validateUpdateRequest(req); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 获取当前实例
	instance, err := s.dao.GetInstance(ctx, req.ID)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	// 验证操作权限
	if err := s.validateUpdatePermission(instance, operatorID); err != nil {
		return err
	}

	// 只有草稿状态和待处理状态可以更新基本信息
	if !s.isEditableStatus(instance.Status) {
		return fmt.Errorf("当前状态的工单不允许修改")
	}

	// 更新实例字段
	s.updateInstanceFields(instance, req)

	// 保存更新
	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单实例失败", zap.Error(err), zap.Int("instanceID", req.ID))
		return fmt.Errorf("更新工单实例失败: %w", err)
	}

	s.logger.Info("更新工单实例成功",
		zap.Int("instanceID", req.ID),
		zap.Int("operatorID", operatorID))

	return nil
}

// DeleteInstance 删除工单实例
func (s *instanceService) DeleteInstance(ctx context.Context, id int, operatorID int) error {
	if id <= 0 {
		return ErrInvalidRequest
	}

	// 检查工单状态和权限
	instance, err := s.dao.GetInstance(ctx, id)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	// 验证删除权限
	if err := s.validateDeletePermission(instance, operatorID); err != nil {
		return err
	}

	// 只有草稿状态可以删除
	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以删除")
	}

	if err := s.dao.DeleteInstance(ctx, id); err != nil {
		s.logger.Error("删除工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return fmt.Errorf("删除工单实例失败: %w", err)
	}

	s.logger.Info("删除工单实例成功",
		zap.Int("instanceID", id),
		zap.Int("operatorID", operatorID))

	return nil
}

// GetInstance 获取工单实例详情
func (s *instanceService) GetInstance(ctx context.Context, id int) (*model.InstanceResp, error) {
	if id <= 0 {
		return nil, ErrInvalidRequest
	}

	instance, err := s.dao.GetInstanceWithRelations(ctx, id)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	return s.convertToInstanceResp(instance), nil
}

// ListInstance 获取工单实例列表
func (s *instanceService) ListInstance(ctx context.Context, req *model.ListInstanceReq) (*model.ListResp[model.InstanceResp], error) {
	if req == nil {
		req = &model.ListInstanceReq{}
	}

	// 标准化分页参数
	s.normalizePagination(&req.Page, &req.Size)

	result, err := s.dao.ListInstance(ctx, req)
	if err != nil {
		s.logger.Error("获取工单实例列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单实例列表失败: %w", err)
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
func (s *instanceService) BatchUpdateInstanceStatus(ctx context.Context, ids []int, status int8, operatorID int) error {
	if len(ids) == 0 || operatorID <= 0 {
		return ErrInvalidRequest
	}

	// 验证状态值
	if !s.isValidStatus(status) {
		return fmt.Errorf("无效的状态值: %d", status)
	}

	// 验证ID有效性
	for _, id := range ids {
		if id <= 0 {
			return ErrInvalidRequest
		}
	}

	if err := s.dao.BatchUpdateInstanceStatus(ctx, ids, status); err != nil {
		s.logger.Error("批量更新工单状态失败",
			zap.Error(err),
			zap.Ints("ids", ids),
			zap.Int8("status", status),
			zap.Int("operatorID", operatorID))
		return fmt.Errorf("批量更新工单状态失败: %w", err)
	}

	s.logger.Info("批量更新工单状态成功",
		zap.Ints("ids", ids),
		zap.Int8("status", status),
		zap.Int("operatorID", operatorID))

	return nil
}

// GetMyInstances 获取我的工单
func (s *instanceService) GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.InstanceResp], error) {
	if userID <= 0 {
		return nil, ErrInvalidRequest
	}

	if req == nil {
		req = &model.MyInstanceReq{}
	}

	// 标准化分页参数
	s.normalizePagination(&req.Page, &req.Size)

	result, err := s.dao.GetMyInstances(ctx, userID, req)
	if err != nil {
		s.logger.Error("获取我的工单失败", zap.Error(err), zap.Int("userID", userID))
		return nil, fmt.Errorf("获取我的工单失败: %w", err)
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
		s.logger.Error("获取超时工单失败", zap.Error(err))
		return nil, fmt.Errorf("获取超时工单失败: %w", err)
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
		return ErrInvalidRequest
	}

	if fromUserID == toUserID {
		return fmt.Errorf("转移目标用户不能是当前用户")
	}

	// 检查工单是否存在
	instance, err := s.dao.GetInstance(ctx, instanceID)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	// 验证转移权限
	if instance.AssigneeID == nil || *instance.AssigneeID != fromUserID {
		return ErrUnauthorized
	}

	// 验证目标用户是否存在
	toUser, err := s.userDao.GetUserByID(ctx, toUserID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := s.dao.TransferInstance(ctx, instanceID, fromUserID, toUserID, comment); err != nil {
		s.logger.Error("转移工单失败", zap.Error(err),
			zap.Int("instanceID", instanceID),
			zap.Int("fromUserID", fromUserID),
			zap.Int("toUserID", toUserID))
		return fmt.Errorf("转移工单失败: %w", err)
	}

	// 记录工单流转
	flow := &model.InstanceFlow{
		InstanceID:   instanceID,
		StepID:       instance.CurrentStep,
		StepName:     "转交",
		Action:       "transfer",
		OperatorID:   fromUserID,
		OperatorName: "", 
		Comment:      comment,
		FromUserID:   fromUserID,
		ToUserID:     toUserID,
		ToUserName:   toUser.Username,
	}

	if err := s.flowDao.CreateInstanceFlow(ctx, flow); err != nil {
		s.logger.Warn("记录工单转交流程失败", zap.Error(err), zap.Int("instanceID", instanceID))
	}

	s.logger.Info("转移工单成功",
		zap.Int("instanceID", instanceID),
		zap.Int("fromUserID", fromUserID),
		zap.Int("toUserID", toUserID))

	return nil
}

// 私有辅助方法

// validateCreateRequest 验证创建请求
func (s *instanceService) validateCreateRequest(req *model.CreateInstanceReq) error {
	if req == nil {
		return ErrInvalidRequest
	}
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("工单标题不能为空")
	}
	if len(req.Title) > MaxTitleLength {
		return fmt.Errorf("工单标题长度不能超过%d个字符", MaxTitleLength)
	}
	if req.ProcessID <= 0 {
		return fmt.Errorf("流程ID不能为空")
	}
	if len(req.Description) > MaxDescriptionLength {
		return fmt.Errorf("工单描述长度不能超过%d个字符", MaxDescriptionLength)
	}
	return nil
}

// validateUpdateRequest 验证更新请求
func (s *instanceService) validateUpdateRequest(req *model.UpdateInstanceReq) error {
	if req == nil || req.ID <= 0 {
		return ErrInvalidRequest
	}
	if req.Title != "" && len(req.Title) > MaxTitleLength {
		return fmt.Errorf("工单标题长度不能超过%d个字符", MaxTitleLength)
	}
	if len(req.Description) > MaxDescriptionLength {
		return fmt.Errorf("工单描述长度不能超过%d个字符", MaxDescriptionLength)
	}
	return nil
}

// validateUpdatePermission 验证更新权限
func (s *instanceService) validateUpdatePermission(instance *model.Instance, operatorID int) error {
	// 只有创建人或当前处理人可以更新
	if instance.CreatorID != operatorID {
		if instance.AssigneeID == nil || *instance.AssigneeID != operatorID {
			return ErrUnauthorized
		}
	}
	return nil
}

// validateDeletePermission 验证删除权限
func (s *instanceService) validateDeletePermission(instance *model.Instance, operatorID int) error {
	// 只有创建人可以删除
	if instance.CreatorID != operatorID {
		return ErrUnauthorized
	}
	return nil
}

// isEditableStatus 判断状态是否可编辑
func (s *instanceService) isEditableStatus(status int8) bool {
	return status == model.InstanceStatusDraft || status == model.InstanceStatusPending
}

// isValidStatus 验证状态值是否有效
func (s *instanceService) isValidStatus(status int8) bool {
	validStatuses := []int8{
		model.InstanceStatusDraft,
		model.InstanceStatusPending,
		model.InstanceStatusProcessing,
		model.InstanceStatusCompleted,
		model.InstanceStatusCancelled,
		model.InstanceStatusRejected,
	}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}

// normalizePagination 标准化分页参数
func (s *instanceService) normalizePagination(page, size *int) {
	if *page <= 0 {
		*page = 1
	}
	if *size <= 0 {
		*size = DefaultPageSize
	}
	if *size > MaxPageSize {
		*size = MaxPageSize
	}
}

// buildInstanceFromRequest 从请求构建工单实例
func (s *instanceService) buildInstanceFromRequest(ctx context.Context, req *model.CreateInstanceReq, creatorID int, creatorName string, process *model.Process) (*model.Instance, error) {
	var formData model.JSONMap

	if process.FormDesign != nil {
		// 将表单设计的schema转换为JSON字符串，再解析为表单数据
		schemaBytes, err := json.Marshal(process.FormDesign.Schema)
		if err != nil {
			return nil, fmt.Errorf("序列化表单设计schema失败: %w", err)
		}
		
		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			return nil, fmt.Errorf("解析表单设计schema失败: %w", err)
		}
		formData = schema
	}

	var processData model.JSONMap
	if process.Definition != "" {
		var definition map[string]interface{}
		if err := json.Unmarshal([]byte(process.Definition), &definition); err != nil {
			return nil, fmt.Errorf("解析流程定义失败: %w", err)
		}
		processData = definition
	}

	// 处理标签
	var tags model.StringSlice
	if len(req.Tags) > 0 {
		tags = model.StringSlice(req.Tags)
	}

	// 确定初始步骤和状态
	initialStep, initialStatus := s.determineInitialStepAndStatus(process.Definition)

	// 构建实例对象
	instance := &model.Instance{
		Title:       strings.TrimSpace(req.Title),
		TemplateID:  req.TemplateID,
		ProcessID:   req.ProcessID,
		FormData:    formData,
		Status:      initialStatus,
		Priority:    req.Priority,
		ProcessData: processData,
		CategoryID:  req.CategoryID,
		CreatorID:   creatorID,
		CreatorName: creatorName,
		Description: strings.TrimSpace(req.Description),
		DueDate:     req.DueDate,
		Tags:        tags,
	}

	// 设置初始步骤和处理人
	if initialStep != nil {
		instance.CurrentStep = initialStep.ID
		s.assignInitialHandler(ctx, instance, req.AssigneeID, initialStep)
	}

	return instance, nil
}

// determineInitialStepAndStatus 确定初始步骤和状态
func (s *instanceService) determineInitialStepAndStatus(processDefStr string) (*model.ProcessStep, int8) {
	if processDefStr == "" {
		return nil, model.InstanceStatusDraft
	}

	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(processDefStr), &processDef); err != nil {
		s.logger.Error("解析流程定义失败", zap.Error(err))
		return nil, model.InstanceStatusDraft
	}

	var initialStep *model.ProcessStep
	var initialStatus int8 = model.InstanceStatusDraft

	// 查找开始步骤
	for i, step := range processDef.Steps {
		if step.Type == "start" {
			initialStep = &processDef.Steps[i]
			break
		}
	}

	if initialStep == nil && len(processDef.Steps) > 0 {
		// 如果没有明确的开始步骤，使用第一个步骤
		initialStep = &processDef.Steps[0]
	}

	return initialStep, initialStatus
}

// assignInitialHandler 分配初始处理人
func (s *instanceService) assignInitialHandler(ctx context.Context, instance *model.Instance, assigneeID *int, initialStep *model.ProcessStep) {
	if assigneeID != nil && *assigneeID > 0 {
		instance.AssigneeID = assigneeID
		if user, err := s.userDao.GetUserByID(ctx, *assigneeID); err == nil {
			instance.AssigneeName = user.Username
		} else {
			instance.AssigneeName = "未知"
			s.logger.Warn("获取指定处理人信息失败", zap.Error(err), zap.Int("assigneeID", *assigneeID))
		}
	} else if initialStep != nil && len(initialStep.Users) > 0 {
		// 使用步骤定义中的第一个用户
		instance.AssigneeID = &initialStep.Users[0]
		if user, err := s.userDao.GetUserByID(ctx, initialStep.Users[0]); err == nil {
			instance.AssigneeName = user.Username
		} else {
			instance.AssigneeName = "未知"
			s.logger.Warn("获取步骤处理人信息失败", zap.Error(err), zap.Int("assigneeID", initialStep.Users[0]))
		}
	}
}

// createInitialFlow 创建初始流程记录
func (s *instanceService) createInitialFlow(ctx context.Context, instance *model.Instance, creatorID int, creatorName string) error {
	if instance.CurrentStep == "" {
		return nil
	}

	flow := &model.InstanceFlow{
		InstanceID:   instance.ID,
		StepID:       instance.CurrentStep,
		StepName:     "开始",
		Action:       "create",
		OperatorID:   creatorID,
		OperatorName: creatorName,
		Comment:      "工单创建",
		FromStepID:   "",
		ToStepID:     instance.CurrentStep,
	}

	return s.flowDao.CreateInstanceFlow(ctx, flow)
}

// updateInstanceFields 更新实例字段
func (s *instanceService) updateInstanceFields(instance *model.Instance, req *model.UpdateInstanceReq) {
	if req.Title != "" {
		instance.Title = strings.TrimSpace(req.Title)
	}
	if req.Description != "" {
		instance.Description = strings.TrimSpace(req.Description)
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
	if len(req.Tags) > 0 {
		instance.Tags = model.StringSlice(req.Tags)
	}
}

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
