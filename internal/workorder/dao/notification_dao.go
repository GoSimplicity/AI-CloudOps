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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderNotificationDAO interface {
	CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error)
	DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error)
	GetNotificationByID(ctx context.Context, id int) (*model.WorkorderNotification, error)
	AddSendLog(ctx context.Context, log *model.WorkorderNotificationLog) error
	GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error)
	IncrementSentCount(ctx context.Context, id int) error
}

type notificationDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewNotificationDAO(db *gorm.DB, logger *zap.Logger) WorkorderNotificationDAO {
	return &notificationDAO{
		db:     db,
		logger: logger,
	}
}

// CreateNotification 创建通知配置
func (n *notificationDAO) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	notification := &model.WorkorderNotification{
		Name:             req.Name,
		Description:      req.Description,
		ProcessID:        req.ProcessID,
		TemplateID:       req.TemplateID,
		CategoryID:       req.CategoryID,
		EventTypes:       req.EventTypes,
		TriggerType:      req.TriggerType,
		TriggerCondition: req.TriggerCondition,
		Channels:         req.Channels,
		RecipientTypes:   req.RecipientTypes,
		RecipientUsers:   req.RecipientUsers,
		RecipientRoles:   req.RecipientRoles,
		RecipientDepts:   req.RecipientDepts,
		MessageTemplate:  req.MessageTemplate,
		SubjectTemplate:  req.SubjectTemplate,
		ScheduledTime:    req.ScheduledTime,
		RepeatInterval:   req.RepeatInterval,
		MaxRetries:       req.MaxRetries,
		RetryInterval:    req.RetryInterval,
		Status:           req.Status,
		Priority:         req.Priority,
		OperatorID:       req.UserID,
		IsDefault:        req.IsDefault,
		Settings:         req.Settings,
	}

	if err := n.db.WithContext(ctx).Create(notification).Error; err != nil {
		n.logger.Error("创建通知配置失败", zap.Error(err), zap.String("name", req.Name))
		return fmt.Errorf("创建通知配置失败: %w", err)
	}

	return nil
}

// UpdateNotification 更新通知配置
func (n *notificationDAO) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
	updateData := map[string]any{}

	if req.Name != "" {
		updateData["name"] = req.Name
	}
	if req.Description != "" {
		updateData["description"] = req.Description
	}
	if req.ProcessID != nil {
		updateData["process_id"] = req.ProcessID
	}
	if req.TemplateID != nil {
		updateData["template_id"] = req.TemplateID
	}
	if req.CategoryID != nil {
		updateData["category_id"] = req.CategoryID
	}
	if req.EventTypes != nil {
		updateData["event_types"] = req.EventTypes
	}
	if req.TriggerType != "" {
		updateData["trigger_type"] = req.TriggerType
	}
	if req.TriggerCondition != nil {
		updateData["trigger_condition"] = req.TriggerCondition
	}
	if req.Channels != nil {
		updateData["channels"] = req.Channels
	}
	if req.RecipientTypes != nil {
		updateData["recipient_types"] = req.RecipientTypes
	}
	if req.RecipientUsers != nil {
		updateData["recipient_users"] = req.RecipientUsers
	}
	if req.RecipientRoles != nil {
		updateData["recipient_roles"] = req.RecipientRoles
	}
	if req.RecipientDepts != nil {
		updateData["recipient_depts"] = req.RecipientDepts
	}
	if req.MessageTemplate != "" {
		updateData["message_template"] = req.MessageTemplate
	}
	if req.SubjectTemplate != "" {
		updateData["subject_template"] = req.SubjectTemplate
	}
	if req.ScheduledTime != nil {
		updateData["scheduled_time"] = req.ScheduledTime
	}
	if req.RepeatInterval != nil {
		updateData["repeat_interval"] = req.RepeatInterval
	}
	if req.MaxRetries > 0 {
		updateData["max_retries"] = req.MaxRetries
	}
	if req.RetryInterval > 0 {
		updateData["retry_interval"] = req.RetryInterval
	}
	if req.Status != 0 {
		updateData["status"] = req.Status
	}
	if req.Priority != 0 {
		updateData["priority"] = req.Priority
	}
	updateData["is_default"] = req.IsDefault
	if req.Settings != nil {
		updateData["settings"] = req.Settings
	}

	result := n.db.WithContext(ctx).Model(&model.WorkorderNotification{}).
		Where("id = ?", req.ID).
		Updates(updateData)

	if result.Error != nil {
		n.logger.Error("更新通知配置失败", zap.Error(result.Error), zap.Int("id", req.ID))
		return fmt.Errorf("更新通知配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("通知配置不存在")
	}

	return nil
}

// DeleteNotification 删除通知配置
func (n *notificationDAO) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
	result := n.db.WithContext(ctx).Delete(&model.WorkorderNotification{}, req.ID)
	if result.Error != nil {
		n.logger.Error("删除通知配置失败", zap.Error(result.Error), zap.Int("id", req.ID))
		return fmt.Errorf("删除通知配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("通知配置不存在")
	}

	return nil
}

// ListNotification 获取通知配置列表
func (n *notificationDAO) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error) {
	var notifications []*model.WorkorderNotification
	var total int64

	req.Page, req.PageSize = ValidatePagination(req.Page, req.PageSize)

	db := n.db.WithContext(ctx).Model(&model.WorkorderNotification{})

	if req.Name != "" {
		searchTerm := sanitizeSearchInput(req.Name)
		db = db.Where("name LIKE ?", "%"+searchTerm+"%")
	}
	if req.ProcessID != nil {
		db = db.Where("process_id = ?", *req.ProcessID)
	}
	if req.TemplateID != nil {
		db = db.Where("template_id = ?", *req.TemplateID)
	}
	if req.CategoryID != nil {
		db = db.Where("category_id = ?", *req.CategoryID)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.IsDefault != nil {
		db = db.Where("is_default = ?", *req.IsDefault)
	}

	if err := db.Count(&total).Error; err != nil {
		n.logger.Error("获取通知配置总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.PageSize
	err := db.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&notifications).Error
	if err != nil {
		n.logger.Error("获取通知配置列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置列表失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderNotification]{
		Items: notifications,
		Total: total,
	}, nil
}

// DetailNotification 获取通知配置详情
func (n *notificationDAO) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	var notification model.WorkorderNotification
	err := n.db.WithContext(ctx).First(&notification, req.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("通知配置不存在")
		}
		n.logger.Error("获取通知配置详情失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("获取通知配置详情失败: %w", err)
	}
	return &notification, nil
}

// GetNotificationByID 根据ID获取通知配置
func (n *notificationDAO) GetNotificationByID(ctx context.Context, id int) (*model.WorkorderNotification, error) {
	var notification model.WorkorderNotification
	err := n.db.WithContext(ctx).First(&notification, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("通知配置不存在")
		}
		n.logger.Error("根据ID获取通知配置失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("根据ID获取通知配置失败: %w", err)
	}
	return &notification, nil
}

// AddSendLog 添加发送日志
func (n *notificationDAO) AddSendLog(ctx context.Context, log *model.WorkorderNotificationLog) error {
	if err := n.db.WithContext(ctx).Create(log).Error; err != nil {
		n.logger.Error("添加发送日志失败", zap.Error(err))
		return fmt.Errorf("添加发送日志失败: %w", err)
	}
	return nil
}

// GetSendLogs 获取发送日志列表
func (n *notificationDAO) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error) {
	var logs []*model.WorkorderNotificationLog
	var total int64

	req.Page, req.PageSize = ValidatePagination(req.Page, req.PageSize)

	db := n.db.WithContext(ctx).Model(&model.WorkorderNotificationLog{})

	if req.NotificationID != nil {
		db = db.Where("notification_id = ?", *req.NotificationID)
	}
	if req.InstanceID != nil {
		db = db.Where("instance_id = ?", *req.InstanceID)
	}
	if req.EventType != "" {
		db = db.Where("event_type = ?", req.EventType)
	}
	if req.Channel != "" {
		db = db.Where("channel = ?", req.Channel)
	}
	if req.RecipientType != "" {
		db = db.Where("recipient_type = ?", req.RecipientType)
	}
	if req.RecipientID != "" {
		db = db.Where("recipient_id = ?", req.RecipientID)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		n.logger.Error("获取发送日志总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.PageSize
	err := db.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&logs).Error
	if err != nil {
		n.logger.Error("获取发送日志列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志列表失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderNotificationLog]{
		Items: logs,
		Total: total,
	}, nil
}

// IncrementSentCount 增加发送计数
func (n *notificationDAO) IncrementSentCount(ctx context.Context, id int) error {
	err := n.db.WithContext(ctx).Model(&model.WorkorderNotification{}).
		Where("id = ?", id).
		Update("updated_at", "NOW()").Error
	if err != nil {
		n.logger.Error("更新发送计数失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("更新发送计数失败: %w", err)
	}
	return nil
}
