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
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type InstanceAttachmentService interface {
	// 附件功能
	UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachment, error)
	DeleteAttachment(ctx context.Context, instanceID int, attachmentID int, operatorID int) error
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error)
	BatchDeleteAttachments(ctx context.Context, instanceID int, attachmentIDs []int, operatorID int) error
}

type instanceAttachmentService struct {
	dao         dao.InstanceAttachmentDAO
	instanceDao dao.InstanceDAO
	logger      *zap.Logger
}

func NewInstanceAttachmentService(dao dao.InstanceAttachmentDAO, instanceDao dao.InstanceDAO, logger *zap.Logger) InstanceAttachmentService {
	return &instanceAttachmentService{
		dao:         dao,
		instanceDao: instanceDao,
		logger:      logger,
	}
}

// UploadAttachment 上传附件
func (s *instanceAttachmentService) UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachment, error) {
	if err := s.validateAttachmentParams(instanceID, fileName, fileSize, filePath, uploaderID); err != nil {
		return nil, err
	}

	// 验证工单是否存在
	if _, err := s.instanceDao.GetInstance(ctx, instanceID); err != nil {
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
func (s *instanceAttachmentService) DeleteAttachment(ctx context.Context, instanceID int, attachmentID int, operatorID int) error {
	if instanceID <= 0 || attachmentID <= 0 || operatorID <= 0 {
		return ErrInvalidRequest
	}

	// 验证附件是否存在且属于该工单
	attachment, err := s.dao.GetInstanceAttachment(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("获取附件信息失败: %w", err)
	}

	if attachment.InstanceID != instanceID {
		return fmt.Errorf("附件不属于指定工单")
	}

	// 验证操作权限 (通常只有上传者或管理员可以删除)
	if attachment.UploaderID != operatorID {
		// 获取工单信息，验证是否为工单创建者
		instance, err := s.instanceDao.GetInstance(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("获取工单信息失败: %w", err)
		}

		if instance.CreatorID != operatorID {
			return ErrUnauthorized
		}
	}

	if err := s.dao.DeleteInstanceAttachment(ctx, instanceID, attachmentID); err != nil {
		s.logger.Error("删除工单附件失败",
			zap.Error(err),
			zap.Int("attachmentID", attachmentID),
			zap.Int("operatorID", operatorID))
		return fmt.Errorf("删除工单附件失败: %w", err)
	}

	s.logger.Info("删除工单附件成功",
		zap.Int("instanceID", instanceID),
		zap.Int("attachmentID", attachmentID),
		zap.Int("operatorID", operatorID))

	return nil
}

// GetInstanceAttachments 获取工单附件列表
func (s *instanceAttachmentService) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error) {
	if instanceID <= 0 {
		return nil, ErrInvalidRequest
	}

	attachments, err := s.dao.GetInstanceAttachments(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取工单附件列表失败: %w", err)
	}

	respAttachments := make([]model.InstanceAttachmentResp, 0, len(attachments))
	for _, attachment := range attachments {
		respAttachments = append(respAttachments, *s.convertToAttachmentResp(&attachment))
	}

	return respAttachments, nil
}

// BatchDeleteAttachments 批量删除附件
func (s *instanceAttachmentService) BatchDeleteAttachments(ctx context.Context, instanceID int, attachmentIDs []int, operatorID int) error {
	if instanceID <= 0 || len(attachmentIDs) == 0 || operatorID <= 0 {
		return ErrInvalidRequest
	}

	// 获取工单信息，验证操作权限
	instance, err := s.instanceDao.GetInstance(ctx, instanceID)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单信息失败: %w", err)
	}

	// 验证所有附件是否属于该工单
	attachments, err := s.dao.GetInstanceAttachments(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("获取工单附件列表失败: %w", err)
	}

	validIDs := make(map[int]bool)
	for _, attachment := range attachments {
		validIDs[attachment.ID] = true
	}

	// 验证所有要删除的附件都属于该工单
	for _, id := range attachmentIDs {
		if !validIDs[id] {
			return fmt.Errorf("附件ID %d 不属于指定工单", id)
		}
	}

	// 检查权限：只有工单创建者可以批量删除附件
	if instance.CreatorID != operatorID {
		return ErrUnauthorized
	}

	if err := s.dao.BatchDeleteInstanceAttachments(ctx, instanceID, attachmentIDs); err != nil {
		s.logger.Error("批量删除工单附件失败",
			zap.Error(err),
			zap.Ints("attachmentIDs", attachmentIDs),
			zap.Int("operatorID", operatorID))
		return fmt.Errorf("批量删除工单附件失败: %w", err)
	}

	s.logger.Info("批量删除工单附件成功",
		zap.Int("instanceID", instanceID),
		zap.Ints("attachmentIDs", attachmentIDs),
		zap.Int("operatorID", operatorID))

	return nil
}

// 辅助方法

// validateAttachmentParams 验证附件参数
func (s *instanceAttachmentService) validateAttachmentParams(instanceID int, fileName string, fileSize int64, filePath string, uploaderID int) error {
	if instanceID <= 0 {
		return ErrInvalidRequest
	}

	if strings.TrimSpace(fileName) == "" {
		return fmt.Errorf("文件名不能为空")
	}

	if len(fileName) > MaxFileNameLength {
		return fmt.Errorf("文件名过长，最大长度为 %d", MaxFileNameLength)
	}

	if fileSize <= 0 {
		return fmt.Errorf("文件大小无效")
	}

	if fileSize > MaxFileSize {
		return fmt.Errorf("文件大小超过最大限制 %d", MaxFileSize)
	}

	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	if uploaderID <= 0 {
		return ErrInvalidRequest
	}

	return nil
}

// convertToAttachmentResp 转换附件对象为响应格式
func (s *instanceAttachmentService) convertToAttachmentResp(attachment *model.InstanceAttachment) *model.InstanceAttachmentResp {
	if attachment == nil {
		return nil
	}

	return &model.InstanceAttachmentResp{
		ID:           attachment.ID,
		InstanceID:   attachment.InstanceID,
		FileName:     attachment.FileName,
		FileSize:     attachment.FileSize,
		FilePath:     attachment.FilePath,
		FileType:     attachment.FileType,
		UploaderID:   attachment.UploaderID,
		UploaderName: attachment.UploaderName,
		CreatedAt:    attachment.CreatedAt,
	}
}
