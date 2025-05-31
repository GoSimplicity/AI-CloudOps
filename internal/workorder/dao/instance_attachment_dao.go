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

package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrAttachmentNotFound   = errors.New("附件不存在")
	ErrAttachmentNotBelong  = errors.New("附件不属于指定工单")
	ErrAttachmentNilPointer = errors.New("附件对象为空")
	ErrInvalidParameters    = errors.New("参数无效")
)

type InstanceAttachmentDAO interface {
	// 附件方法
	CreateInstanceAttachment(ctx context.Context, attachment *model.InstanceAttachment) (*model.InstanceAttachment, error)
	DeleteInstanceAttachment(ctx context.Context, instanceID int, attachmentID int) error
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachment, error)
	GetInstanceAttachment(ctx context.Context, attachmentID int) (*model.InstanceAttachment, error)
	BatchDeleteInstanceAttachments(ctx context.Context, instanceID int, attachmentIDs []int) error
}

type instanceAttachmentDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewInstanceAttachmentDAO(db *gorm.DB, logger *zap.Logger) InstanceAttachmentDAO {
	return &instanceAttachmentDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstanceAttachment 创建工单附件记录
func (d *instanceAttachmentDAO) CreateInstanceAttachment(ctx context.Context, attachment *model.InstanceAttachment) (*model.InstanceAttachment, error) {
	if attachment == nil {
		return nil, ErrAttachmentNilPointer
	}

	if err := d.validateAttachment(attachment); err != nil {
		return nil, fmt.Errorf("附件验证失败: %w", err)
	}

	d.logger.Debug("开始创建工单附件", zap.Int("instanceID", attachment.InstanceID), zap.String("fileName", attachment.FileName))

	if err := d.db.WithContext(ctx).Create(attachment).Error; err != nil {
		d.logger.Error("创建工单附件失败", zap.Error(err), zap.Int("instanceID", attachment.InstanceID))
		return nil, fmt.Errorf("创建工单附件失败: %w", err)
	}

	d.logger.Info("创建工单附件成功", zap.Int("id", attachment.ID), zap.String("fileName", attachment.FileName))
	return attachment, nil
}

// DeleteInstanceAttachment 删除工单附件记录
func (d *instanceAttachmentDAO) DeleteInstanceAttachment(ctx context.Context, instanceID int, attachmentID int) error {
	if instanceID <= 0 || attachmentID <= 0 {
		return ErrInvalidParameters
	}

	d.logger.Debug("开始删除工单附件", zap.Int("instanceID", instanceID), zap.Int("attachmentID", attachmentID))

	result := d.db.WithContext(ctx).
		Where("id = ? AND instance_id = ?", attachmentID, instanceID).
		Delete(&model.InstanceAttachment{})

	if result.Error != nil {
		d.logger.Error("删除工单附件失败", zap.Error(result.Error), zap.Int("attachmentID", attachmentID))
		return fmt.Errorf("删除工单附件失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		d.logger.Warn("附件不存在或不属于指定工单", zap.Int("attachmentID", attachmentID), zap.Int("instanceID", instanceID))
		return ErrAttachmentNotBelong
	}

	d.logger.Info("删除工单附件成功", zap.Int("attachmentID", attachmentID))
	return nil
}

// GetInstanceAttachments 获取工单附件列表
func (d *instanceAttachmentDAO) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachment, error) {
	if instanceID <= 0 {
		return nil, ErrInvalidParameters
	}

	d.logger.Debug("开始获取工单附件列表", zap.Int("instanceID", instanceID))

	var attachments []model.InstanceAttachment
	err := d.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at DESC").
		Find(&attachments).Error

	if err != nil {
		d.logger.Error("获取工单附件列表失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单附件列表失败: %w", err)
	}

	d.logger.Debug("获取工单附件列表成功", zap.Int("count", len(attachments)))
	return attachments, nil
}

// GetInstanceAttachment 获取工单附件
func (d *instanceAttachmentDAO) GetInstanceAttachment(ctx context.Context, attachmentID int) (*model.InstanceAttachment, error) {
	if attachmentID <= 0 {
		return nil, ErrInvalidParameters
	}

	d.logger.Debug("开始获取工单附件", zap.Int("attachmentID", attachmentID))

	var attachment model.InstanceAttachment
	result := d.db.WithContext(ctx).
		Where("id = ?", attachmentID).
		First(&attachment)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			d.logger.Warn("工单附件不存在", zap.Int("attachmentID", attachmentID))
			return nil, ErrAttachmentNotFound
		}
		d.logger.Error("获取工单附件失败", zap.Error(result.Error), zap.Int("attachmentID", attachmentID))
		return nil, fmt.Errorf("获取工单附件失败: %w", result.Error)
	}

	d.logger.Debug("获取工单附件成功", zap.Int("attachmentID", attachmentID))
	return &attachment, nil
}

// BatchDeleteInstanceAttachments 批量删除工单附件
func (d *instanceAttachmentDAO) BatchDeleteInstanceAttachments(ctx context.Context, instanceID int, attachmentIDs []int) error {
	if instanceID <= 0 || len(attachmentIDs) == 0 {
		return ErrInvalidParameters
	}

	// 验证附件ID的有效性
	for _, id := range attachmentIDs {
		if id <= 0 {
			return ErrInvalidParameters
		}
	}

	result := d.db.WithContext(ctx).
		Where("instance_id = ? AND id IN ?", instanceID, attachmentIDs).
		Delete(&model.InstanceAttachment{})

	if result.Error != nil {
		d.logger.Error("批量删除工单附件失败", zap.Error(result.Error), zap.Ints("attachmentIDs", attachmentIDs))
		return fmt.Errorf("批量删除工单附件失败: %w", result.Error)
	}

	d.logger.Info("批量删除工单附件成功", zap.Ints("attachmentIDs", attachmentIDs), zap.Int64("affected", result.RowsAffected))
	return nil
}

// validateAttachment 验证附件数据
func (d *instanceAttachmentDAO) validateAttachment(attachment *model.InstanceAttachment) error {
	if attachment.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}
	if strings.TrimSpace(attachment.FileName) == "" {
		return fmt.Errorf("文件名不能为空")
	}
	if strings.TrimSpace(attachment.FilePath) == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	return nil
}
