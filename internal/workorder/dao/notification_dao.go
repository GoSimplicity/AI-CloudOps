package dao

import (
	"context"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type NotificationDAO interface {
	CreateNotification(ctx context.Context, req *model.CreateNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListNotificationReq) (model.ListResp[*model.Notification], error)
	DetailNotification(ctx context.Context, req *model.DetailNotificationReq) (*model.Notification, error)
	UpdateStatus(ctx context.Context, req *model.UpdateStatusReq) error
	GetStatistics(ctx context.Context) (*model.NotificationStats, error)
	GetNotificationByID(ctx context.Context, id int) (*model.Notification, error)
	AddSendLog(ctx context.Context, log *model.NotificationLog) error
	GetSendLogs(ctx context.Context, req *model.ListSendLogReq) (model.ListResp[*model.NotificationLog], error)
	IncrementSentCount(ctx context.Context, id int) error
}

type notificationDAO struct {
	db *gorm.DB
}

func NewNotificationDAO(db *gorm.DB) NotificationDAO {
	return &notificationDAO{db: db}
}

// CreateNotification 创建通知配置
func (n *notificationDAO) CreateNotification(ctx context.Context, req *model.CreateNotificationReq) error {
	notification := &model.Notification{
		FormID:          req.FormID,
		Channels:        model.StringList(req.Channels),
		Recipients:      model.StringList(req.Recipients),
		MessageTemplate: req.MessageTemplate,
		TriggerType:     req.TriggerType,
		ScheduledTime:   req.ScheduledTime,
		Status:          model.NotificationStatusEnabled,
		CreatorID:       req.UserID,
		FormUrl:         req.FormUrl,
	}

	return n.db.WithContext(ctx).Create(notification).Error
}

// UpdateNotification 更新通知配置
func (n *notificationDAO) UpdateNotification(ctx context.Context, req *model.UpdateNotificationReq) error {
	notification := &model.Notification{
		Model: model.Model{
			ID: req.ID,
		},
		FormID:          req.FormID,
		Channels:        model.StringList(req.Channels),
		Recipients:      model.StringList(req.Recipients),
		MessageTemplate: req.MessageTemplate,
		TriggerType:     req.TriggerType,
		ScheduledTime:   req.ScheduledTime,
		FormUrl:         req.FormUrl,
	}

	// 如果提供了状态参数，则更新状态
	if req.Status == model.NotificationStatusEnabled || req.Status == model.NotificationStatusDisabled {
		notification.Status = req.Status
	}

	return n.db.WithContext(ctx).Model(&model.Notification{}).Where("id = ?", req.ID).
		Updates(notification).Error
}

// DeleteNotification 删除通知配置
func (n *notificationDAO) DeleteNotification(ctx context.Context, req *model.DeleteNotificationReq) error {
	return n.db.WithContext(ctx).Delete(&model.Notification{}, req.ID).Error
}

// ListNotification 获取通知配置列表
func (n *notificationDAO) ListNotification(ctx context.Context, req *model.ListNotificationReq) (model.ListResp[*model.Notification], error) {
	var result model.ListResp[*model.Notification]
	var total int64
	var notifications []*model.Notification

	db := n.db.WithContext(ctx).Model(&model.Notification{})

	// 添加查询条件
	if req.Channel != nil {
		db = db.Where("JSON_CONTAINS(channels, JSON_QUOTE(?))", *req.Channel)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.FormID != nil {
		db = db.Where("form_id = ?", *req.FormID)
	}
	// 关键词搜索，如果ListReq中有Keyword字段
	if req.Search != "" {
		searchPattern := "%" + sanitizeSearchInput(req.Search) + "%"
		db = db.Where("message_template LIKE ?", searchPattern)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return result, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := db.Order("id DESC").Offset(int(offset)).Limit(int(req.Size)).Find(&notifications).Error; err != nil {
		return result, err
	}

	// 设置返回结果
	result.Total = total
	result.Items = notifications
	return result, nil
}

// DetailNotification 获取通知配置详情
func (n *notificationDAO) DetailNotification(ctx context.Context, req *model.DetailNotificationReq) (*model.Notification, error) {
	var notification model.Notification
	err := n.db.WithContext(ctx).Where("id = ?", req.ID).First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// GetNotificationByID 根据ID获取通知配置
func (n *notificationDAO) GetNotificationByID(ctx context.Context, id int) (*model.Notification, error) {
	var notification model.Notification
	err := n.db.WithContext(ctx).Where("id = ?", id).First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// UpdateStatus 更新通知配置状态
func (n *notificationDAO) UpdateStatus(ctx context.Context, req *model.UpdateStatusReq) error {
	return n.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ?", req.ID).
		Update("status", req.Status).Error
}

// GetStatistics 获取通知统计信息
func (n *notificationDAO) GetStatistics(ctx context.Context) (*model.NotificationStats, error) {
	var stats model.NotificationStats
	var enabled, disabled, todaySent int64

	// 获取启用状态的通知数量
	if err := n.db.WithContext(ctx).Model(&model.Notification{}).
		Where("status = ?", model.NotificationStatusEnabled).
		Count(&enabled).Error; err != nil {
		return nil, err
	}
	stats.Enabled = int(enabled)

	// 获取禁用状态的通知数量
	if err := n.db.WithContext(ctx).Model(&model.Notification{}).
		Where("status = ?", model.NotificationStatusDisabled).
		Count(&disabled).Error; err != nil {
		return nil, err
	}
	stats.Disabled = int(disabled)

	// 获取今日发送数量
	today := time.Now().Format("2006-01-02")
	if err := n.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("DATE(created_at) = ?", today).
		Count(&todaySent).Error; err != nil {
		return nil, err
	}
	stats.TodaySent = int(todaySent)

	return &stats, nil
}

// AddSendLog 添加发送日志
func (n *notificationDAO) AddSendLog(ctx context.Context, log *model.NotificationLog) error {
	return n.db.WithContext(ctx).Create(log).Error
}

// GetSendLogs 获取发送日志
func (n *notificationDAO) GetSendLogs(ctx context.Context, req *model.ListSendLogReq) (model.ListResp[*model.NotificationLog], error) {
	var result model.ListResp[*model.NotificationLog]
	var total int64
	var logs []*model.NotificationLog

	db := n.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("notification_id = ?", req.NotificationID)

	// 添加查询条件
	if req.Channel != nil {
		db = db.Where("channel = ?", *req.Channel)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return result, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := db.Order("id DESC").Offset(int(offset)).Limit(int(req.Size)).Find(&logs).Error; err != nil {
		return result, err
	}

	result.Total = total
	result.Items = logs

	return result, nil
}

// IncrementSentCount 增加发送次数并更新最后发送时间
func (n *notificationDAO) IncrementSentCount(ctx context.Context, id int) error {
	now := time.Now()
	return n.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sent_count": gorm.Expr("sent_count + 1"),
			"last_sent":  now,
		}).Error
}
