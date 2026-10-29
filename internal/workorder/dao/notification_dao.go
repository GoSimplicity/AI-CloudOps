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
	"time"

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
	GetActiveNotificationsByEventType(ctx context.Context, eventType string, processID int) ([]*model.WorkorderNotification, error)
	GetInstanceByID(ctx context.Context, instanceID int) (*model.WorkorderInstance, error)
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

func (d *notificationDAO) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	maxRetries := 3
	if req.MaxRetries != nil {
		maxRetries = *req.MaxRetries
	}
	notification := &model.WorkorderNotification{
		Name:               req.Name,
		Description:        req.Description,
		ProcessID:          req.ProcessID,
		TemplateID:         req.TemplateID,
		CategoryID:         req.CategoryID,
		EventTypes:         req.EventTypes,
		TriggerType:        req.TriggerType,
		TriggerCondition:   req.TriggerCondition,
		Channels:           req.Channels,
		RecipientTypes:     req.RecipientTypes,
		RecipientUsers:     req.RecipientUsers,
		RecipientRoles:     req.RecipientRoles,
		RecipientDepts:     req.RecipientDepts,
		RecipientAddresses: req.RecipientAddresses,
		MessageTemplate:    req.MessageTemplate,
		SubjectTemplate:    req.SubjectTemplate,
		ScheduledTime:      req.ScheduledTime,
		RepeatInterval:     req.RepeatInterval,
		MaxRetries:         maxRetries,
		RetryInterval:      req.RetryInterval,
		Status:             req.Status,
		Priority:           req.Priority,
		OperatorID:         req.UserID,
		IsDefault:          req.IsDefault,
		Settings:           req.Settings,
	}

	if err := DBWithContext(ctx, d.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notification).Error; err != nil {
			return err
		}
		// GORM会以模型default:3替换零值，在同一事务中恢复客户端显式的0
		if maxRetries == 0 {
			return tx.Model(notification).UpdateColumn("max_retries", 0).Error
		}
		return nil
	}); err != nil {
		d.logger.Error("创建通知配置失败", zap.Error(err), zap.String("name", req.Name))
		return fmt.Errorf("创建通知配置失败: %w", err)
	}

	return nil
}

func (d *notificationDAO) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
	updateData := map[string]any{}

	if req.Name != "" {
		updateData["name"] = req.Name
	}
	if req.Description != nil {
		updateData["description"] = *req.Description
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
	if req.RecipientAddresses != nil {
		updateData["recipient_addresses"] = req.RecipientAddresses
	}
	if req.RecipientDepts != nil {
		updateData["recipient_depts"] = req.RecipientDepts
	}
	if req.MessageTemplate != nil {
		updateData["message_template"] = *req.MessageTemplate
	}
	if req.SubjectTemplate != nil {
		updateData["subject_template"] = *req.SubjectTemplate
	}
	if req.ScheduledTime != nil {
		updateData["scheduled_time"] = req.ScheduledTime
	}
	if req.RepeatInterval != nil {
		updateData["repeat_interval"] = req.RepeatInterval
	}
	if req.MaxRetries != nil {
		updateData["max_retries"] = *req.MaxRetries
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
	if req.IsDefault != nil {
		updateData["is_default"] = *req.IsDefault
	}
	if req.Settings != nil {
		updateData["settings"] = req.Settings
	}

	result := DBWithContext(ctx, d.db).Model(&model.WorkorderNotification{}).
		Where("id = ?", req.ID).
		Updates(updateData)

	if result.Error != nil {
		d.logger.Error("更新通知配置失败", zap.Error(result.Error), zap.Int("id", req.ID))
		return fmt.Errorf("更新通知配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		// MySQL同值写可返回零变更行，必须单独区分幂等更新和不存在
		var count int64
		if err := DBWithContext(ctx, d.db).Model(&model.WorkorderNotification{}).Where("id = ?", req.ID).Count(&count).Error; err != nil {
			return fmt.Errorf("确认通知配置是否存在失败: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("通知配置不存在")
		}
	}

	return nil
}

func (d *notificationDAO) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
	result := DBWithContext(ctx, d.db).Delete(&model.WorkorderNotification{}, req.ID)
	if result.Error != nil {
		d.logger.Error("删除通知配置失败", zap.Error(result.Error), zap.Int("id", req.ID))
		return fmt.Errorf("删除通知配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("通知配置不存在")
	}

	return nil
}

func (d *notificationDAO) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error) {
	var notifications []*model.WorkorderNotification
	var total int64

	req.Page, req.PageSize = ValidatePagination(req.Page, req.PageSize)

	db := DBWithContext(ctx, d.db).Model(&model.WorkorderNotification{})

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

	filtered, filterErr := notificationChannelFilter(db, req.Channel)
	if filterErr != nil {
		return nil, filterErr
	}
	db = filtered

	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取通知配置总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.PageSize
	err := db.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&notifications).Error
	if err != nil {
		d.logger.Error("获取通知配置列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置列表失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderNotification]{
		Items: notifications,
		Total: total,
	}, nil
}

func (d *notificationDAO) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	var notification model.WorkorderNotification
	err := DBWithContext(ctx, d.db).First(&notification, req.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("通知配置不存在")
		}
		d.logger.Error("获取通知配置详情失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("获取通知配置详情失败: %w", err)
	}
	return &notification, nil
}

// GetNotificationByID 根据ID获取通知配置
func (d *notificationDAO) GetNotificationByID(ctx context.Context, id int) (*model.WorkorderNotification, error) {
	var notification model.WorkorderNotification
	err := DBWithContext(ctx, d.db).First(&notification, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("通知配置不存在")
		}
		d.logger.Error("根据ID获取通知配置失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("根据ID获取通知配置失败: %w", err)
	}
	return &notification, nil
}

// AddSendLog 添加发送日志
func (d *notificationDAO) AddSendLog(ctx context.Context, log *model.WorkorderNotificationLog) error {
	if err := DBWithContext(ctx, d.db).Create(log).Error; err != nil {
		d.logger.Error("添加发送日志失败", zap.Error(err))
		return fmt.Errorf("添加发送日志失败: %w", err)
	}
	return nil
}

func (d *notificationDAO) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error) {
	var logs []*model.WorkorderNotificationLog
	var total int64

	req.Page, req.PageSize = ValidatePagination(req.Page, req.PageSize)

	db := DBWithContext(ctx, d.db).Model(&model.WorkorderNotificationLog{})

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
		d.logger.Error("获取发送日志总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志总数失败: %w", err)
	}

	offset := (req.Page - 1) * req.PageSize
	err := db.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&logs).Error
	if err != nil {
		d.logger.Error("获取发送日志列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志列表失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderNotificationLog]{
		Items: logs,
		Total: total,
	}, nil
}

// IncrementSentCount 增加发送计数
func (d *notificationDAO) IncrementSentCount(ctx context.Context, id int) error {
	err := DBWithContext(ctx, d.db).Model(&model.WorkorderNotification{}).
		Where("id = ?", id).
		Update("updated_at", time.Now()).Error
	if err != nil {
		d.logger.Error("更新发送计数失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("更新发送计数失败: %w", err)
	}
	return nil
}

// GetActiveNotificationsByEventType 根据事件类型和流程ID获取活跃的通知配置
func (d *notificationDAO) GetActiveNotificationsByEventType(ctx context.Context, eventType string, processID int) ([]*model.WorkorderNotification, error) {
	var notifications []*model.WorkorderNotification

	db := DBWithContext(ctx, d.db).Model(&model.WorkorderNotification{}).
		Where("status = ?", 1) // 1-启用状态

	db = db.Where("(process_id IS NULL OR process_id = ?)", processID)
	db = db.Where("event_types IS NOT NULL AND event_types != '' AND JSON_VALID(event_types) = 1 AND JSON_CONTAINS(event_types, ?)", fmt.Sprintf(`"%s"`, eventType))

	err := db.Order("priority ASC, id ASC").Find(&notifications).Error
	if err != nil {
		d.logger.Error("获取活跃通知配置失败",
			zap.Error(err),
			zap.String("event_type", eventType),
			zap.Int("process_id", processID))
		return nil, fmt.Errorf("获取活跃通知配置失败: %w", err)
	}

	return notifications, nil
}

// GetInstanceByID 根据ID获取工单实例信息
func (d *notificationDAO) GetInstanceByID(ctx context.Context, instanceID int) (*model.WorkorderInstance, error) {
	var instance model.WorkorderInstance
	err := DBWithContext(ctx, d.db).First(&instance, instanceID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("工单实例不存在")
		}
		d.logger.Error("根据ID获取工单实例失败", zap.Error(err), zap.Int("instance_id", instanceID))
		return nil, fmt.Errorf("根据ID获取工单实例失败: %w", err)
	}
	return &instance, nil
}
