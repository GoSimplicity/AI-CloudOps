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

	// 流程操作
	ProcessInstanceFlow(ctx context.Context, req *model.InstanceActionReq, operatorID int, operatorName string) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error)
	GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error)

	// 评论功能
	CommentInstance(ctx context.Context, req *model.InstanceCommentReq, creatorID int, creatorName string) error
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error)

	// 附件功能
	UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachment, error)
	DeleteAttachment(ctx context.Context, instanceID int, attachmentID int, operatorID int) error
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error)
	BatchDeleteAttachments(ctx context.Context, instanceID int, attachmentIDs []int, operatorID int) error

	// 业务功能
	GetMyInstances(ctx context.Context, userID int, req *model.MyInstanceReq) (*model.ListResp[model.InstanceResp], error)
	GetOverdueInstances(ctx context.Context) ([]model.InstanceResp, error)
	TransferInstance(ctx context.Context, instanceID int, fromUserID int, toUserID int, comment string) error
}

type instanceService struct {
	dao        dao.InstanceDAO
	processDao dao.ProcessDAO
	userDao    userdao.UserDAO
	logger     *zap.Logger
}

func NewInstanceService(dao dao.InstanceDAO, processDao dao.ProcessDAO, userDao userdao.UserDAO, logger *zap.Logger) InstanceService {
	return &instanceService{
		dao:        dao,
		userDao:    userDao,
		processDao: processDao,
		logger:     logger,
	}
}

// CreateInstance 创建工单实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateInstanceReq, creatorID int, creatorName string) (*model.InstanceResp, error) {
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("参数验证失败: %w", err)
	}

	// 验证流程是否存在
	process, err := s.processDao.GetProcess(ctx, req.ProcessID)
	if err != nil {
		if errors.Is(err, dao.ErrProcessNotFound) {
			return nil, ErrProcessNotFound
		}
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 解析流程定义
	processDef, err := s.parseProcessDefinition(process.Definition)
	if err != nil {
		return nil, fmt.Errorf("流程定义解析失败: %w", err)
	}

	// 创建工单实例
	instance, err := s.buildInstanceFromRequest(req, creatorID, creatorName, processDef)
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
	if len(ids) == 0 {
		return ErrInvalidRequest
	}

	if operatorID <= 0 {
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

// ProcessInstanceFlow 处理工单流程
func (s *instanceService) ProcessInstanceFlow(ctx context.Context, req *model.InstanceActionReq, operatorID int, operatorName string) error {
	if err := s.validateActionRequest(req); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 获取工单实例
	instance, err := s.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	// 验证操作权限
	if err := s.validateOperationPermission(instance, operatorID, req.Action); err != nil {
		return err
	}

	// 获取流程定义
	process, err := s.processDao.GetProcess(ctx, instance.ProcessID)
	if err != nil {
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	processDef, err := s.parseProcessDefinition(process.Definition)
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
	if err := s.validateCommentRequest(req); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 验证工单是否存在
	if _, err := s.dao.GetInstance(ctx, req.InstanceID); err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	comment := &model.InstanceComment{
		InstanceID:  req.InstanceID,
		Content:     strings.TrimSpace(req.Content),
		UserID:      creatorID,
		CreatorName: creatorName,
		ParentID:    req.ParentID,
		IsSystem:    false,
	}

	if err := s.dao.CreateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("创建工单评论失败", zap.Error(err))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	s.logger.Info("创建工单评论成功",
		zap.Int("instanceID", req.InstanceID),
		zap.Int("creatorID", creatorID))

	return nil
}

// GetInstanceFlows 获取工单流程记录
func (s *instanceService) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error) {
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

// GetInstanceComments 获取工单评论
func (s *instanceService) GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error) {
	if instanceID <= 0 {
		return nil, ErrInvalidRequest
	}

	comments, err := s.dao.GetInstanceComments(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	// 构建评论树
	return s.buildCommentTree(comments, nil), nil
}

// UploadAttachment 上传附件
func (s *instanceService) UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachment, error) {
	if err := s.validateAttachmentParams(instanceID, fileName, fileSize, filePath, uploaderID); err != nil {
		return nil, err
	}

	// 验证工单是否存在
	if _, err := s.dao.GetInstance(ctx, instanceID); err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	attachment := &model.InstanceAttachment{
		InstanceID:   instanceID,
		FileName:     strings.TrimSpace(fileName),
		FileSize:     fileSize,
		FilePath:     strings.TrimSpace(filePath),
		FileType:     strings.TrimSpace(fileType),
		UploaderID:   uploaderID,
		UploaderName: uploaderName,
	}

	result, err := s.dao.CreateInstanceAttachment(ctx, attachment)
	if err != nil {
		s.logger.Error("创建工单附件失败", zap.Error(err))
		return nil, fmt.Errorf("创建工单附件失败: %w", err)
	}

	s.logger.Info("上传工单附件成功",
		zap.Int("instanceID", instanceID),
		zap.String("fileName", fileName),
		zap.Int("uploaderID", uploaderID))

	return result, nil
}

// DeleteAttachment 删除附件
func (s *instanceService) DeleteAttachment(ctx context.Context, instanceID int, attachmentID int, operatorID int) error {
	if instanceID <= 0 || attachmentID <= 0 || operatorID <= 0 {
		return ErrInvalidRequest
	}

	// 验证权限（简化实现，可根据需要扩展）
	if err := s.dao.DeleteInstanceAttachment(ctx, instanceID, attachmentID); err != nil {
		if errors.Is(err, dao.ErrAttachmentNotBelong) {
			return fmt.Errorf("附件不属于指定工单")
		}
		s.logger.Error("删除工单附件失败", zap.Error(err))
		return fmt.Errorf("删除工单附件失败: %w", err)
	}

	s.logger.Info("删除工单附件成功",
		zap.Int("instanceID", instanceID),
		zap.Int("attachmentID", attachmentID),
		zap.Int("operatorID", operatorID))

	return nil
}

// GetInstanceAttachments 获取工单附件列表
func (s *instanceService) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error) {
	if instanceID <= 0 {
		return nil, ErrInvalidRequest
	}

	attachments, err := s.dao.GetInstanceAttachments(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取工单附件列表失败: %w", err)
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
func (s *instanceService) BatchDeleteAttachments(ctx context.Context, instanceID int, attachmentIDs []int, operatorID int) error {
	if instanceID <= 0 || len(attachmentIDs) == 0 || operatorID <= 0 {
		return ErrInvalidRequest
	}

	// 验证附件ID
	for _, id := range attachmentIDs {
		if id <= 0 {
			return ErrInvalidRequest
		}
	}

	if err := s.dao.BatchDeleteInstanceAttachments(ctx, instanceID, attachmentIDs); err != nil {
		s.logger.Error("批量删除工单附件失败", zap.Error(err))
		return fmt.Errorf("批量删除工单附件失败: %w", err)
	}

	s.logger.Info("批量删除工单附件成功",
		zap.Int("instanceID", instanceID),
		zap.Ints("attachmentIDs", attachmentIDs),
		zap.Int("operatorID", operatorID))

	return nil
}

// GetProcessDefinition 获取流程定义
func (s *instanceService) GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error) {
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

	return s.parseProcessDefinition(process.Definition)
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

	// 验证目标用户是否存在
	if _, err := s.userDao.GetUserByID(ctx, toUserID); err != nil {
		return ErrUserNotFound
	}

	if err := s.dao.TransferInstance(ctx, instanceID, fromUserID, toUserID, comment); err != nil {
		s.logger.Error("转移工单失败", zap.Error(err))
		return fmt.Errorf("转移工单失败: %w", err)
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

// validateActionRequest 验证操作请求
func (s *instanceService) validateActionRequest(req *model.InstanceActionReq) error {
	if req == nil || req.InstanceID <= 0 {
		return ErrInvalidRequest
	}
	if req.Action == "" {
		return fmt.Errorf("操作类型不能为空")
	}
	validActions := []string{"approve", "reject", "cancel", "transfer", "revoke"}
	for _, action := range validActions {
		if req.Action == action {
			return nil
		}
	}
	return ErrInvalidAction
}

// validateCommentRequest 验证评论请求
func (s *instanceService) validateCommentRequest(req *model.InstanceCommentReq) error {
	if req == nil || req.InstanceID <= 0 {
		return ErrInvalidRequest
	}
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("评论内容不能为空")
	}
	if len(req.Content) > MaxCommentLength {
		return fmt.Errorf("评论内容长度不能超过%d个字符", MaxCommentLength)
	}
	return nil
}

// validateAttachmentParams 验证附件参数
func (s *instanceService) validateAttachmentParams(instanceID int, fileName string, fileSize int64, filePath string, uploaderID int) error {
	if instanceID <= 0 || uploaderID <= 0 {
		return ErrInvalidRequest
	}
	if strings.TrimSpace(fileName) == "" {
		return fmt.Errorf("文件名不能为空")
	}
	if len(fileName) > MaxFileNameLength {
		return fmt.Errorf("文件名长度不能超过%d个字符", MaxFileNameLength)
	}
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if fileSize <= 0 || fileSize > MaxFileSize {
		return fmt.Errorf("文件大小必须在1字节到%d字节之间", MaxFileSize)
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

// parseProcessDefinition 解析流程定义
func (s *instanceService) parseProcessDefinition(definition string) (*model.ProcessDefinition, error) {
	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(definition), &processDef); err != nil {
		return nil, ErrProcessDefinition
	}
	return &processDef, nil
}

// buildInstanceFromRequest 从请求构建工单实例
func (s *instanceService) buildInstanceFromRequest(req *model.CreateInstanceReq, creatorID int, creatorName string, processDef *model.ProcessDefinition) (*model.Instance, error) {
	// 序列化表单数据
	var formData model.JSONMap
	if req.FormData != nil {
		formData = model.JSONMap(req.FormData)
	}

	// 处理标签
	var tags model.StringSlice
	if len(req.Tags) > 0 {
		tags = model.StringSlice(req.Tags)
	}

	// 确定初始步骤和状态
	initialStep, initialStatus := s.determineInitialStepAndStatus(processDef)

	// 构建实例对象
	instance := &model.Instance{
		Title:       strings.TrimSpace(req.Title),
		TemplateID:  req.TemplateID,
		ProcessID:   req.ProcessID,
		FormData:    formData,
		Status:      initialStatus,
		Priority:    req.Priority,
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
		s.assignInitialHandler(instance, req.AssigneeID, initialStep)
	}

	return instance, nil
}

// determineInitialStepAndStatus 确定初始步骤和状态
func (s *instanceService) determineInitialStepAndStatus(processDef *model.ProcessDefinition) (*model.ProcessStep, int8) {
	var initialStep *model.ProcessStep
	var initialStatus int8 = model.InstanceStatusDraft

	// 查找开始步骤
	for i, step := range processDef.Steps {
		if step.Type == "start" {
			initialStep = &processDef.Steps[i]
			initialStatus = model.InstanceStatusProcessing
			break
		}
	}

	if initialStep == nil && len(processDef.Steps) > 0 {
		// 如果没有明确的开始步骤，使用第一个步骤
		initialStep = &processDef.Steps[0]
		initialStatus = model.InstanceStatusProcessing
	}

	return initialStep, initialStatus
}

// assignInitialHandler 分配初始处理人
func (s *instanceService) assignInitialHandler(instance *model.Instance, assigneeID *int, initialStep *model.ProcessStep) {
	if assigneeID != nil && *assigneeID > 0 {
		instance.AssigneeID = assigneeID
		if user, err := s.userDao.GetUserByID(context.Background(), *assigneeID); err == nil {
			instance.AssigneeName = user.Username
		}
	} else if len(initialStep.Users) > 0 {
		// 使用步骤定义中的第一个用户
		instance.AssigneeID = &initialStep.Users[0]
		if user, err := s.userDao.GetUserByID(context.Background(), initialStep.Users[0]); err == nil {
			instance.AssigneeName = user.Username
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

	return s.dao.CreateInstanceFlow(ctx, flow)
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
	if req.FormData != nil {
		instance.FormData = model.JSONMap(req.FormData)
	}
	if len(req.Tags) > 0 {
		instance.Tags = model.StringSlice(req.Tags)
	}
}

// buildFlowRecord 构建流程记录
func (s *instanceService) buildFlowRecord(req *model.InstanceActionReq, instance *model.Instance, operatorID int, operatorName string) *model.InstanceFlow {
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
func (s *instanceService) handleFlowAction(ctx context.Context, instance *model.Instance, processDef *model.ProcessDefinition, flow *model.InstanceFlow, req *model.InstanceActionReq) error {
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
				UserID:      comment.UserID,
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
func (s *instanceService) handleRevokeAction(ctx context.Context, instance *model.Instance, flow *model.InstanceFlow) error {
	instance.Status = model.InstanceStatusDraft
	instance.AssigneeID = nil
	instance.AssigneeName = ""
	flow.ToStepID = instance.CurrentStep
	return nil
}
