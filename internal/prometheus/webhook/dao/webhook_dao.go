package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WebhookDao interface {
	// GetOnDutyGroupById 根据ID获取MonitorOnDutyGroup
	GetOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	// GetRuleById 根据ID获取MonitorAlertRule
	GetRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	// GetSendGroupById 根据ID获取MonitorSendGroup
	GetSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error)
	// GetUserById 根据ID获取User
	GetUserById(ctx context.Context, id int) (*model.User, error)

	// CreateOrUpdateEvent 创建或更新MonitorAlertEvent
	CreateOrUpdateEvent(ctx context.Context, event *model.MonitorAlertEvent) error
	// GetMonitorAlertEventByFingerprintId 根据fingerprintId获取MonitorAlertEvent
	GetMonitorAlertEventByFingerprintId(ctx context.Context, fingerprintId string) (*model.MonitorAlertEvent, error)
}

type webhookDao struct {
	l  *zap.Logger
	db *gorm.DB
}

func NewWebhookDao(l *zap.Logger, db *gorm.DB) WebhookDao {
	return &webhookDao{
		l:  l,
		db: db,
	}
}

// GetOnDutyGroupById 根据ID获取MonitorOnDutyGroup
func (wd *webhookDao) GetOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	var onDutyGroup model.MonitorOnDutyGroup

	// 执行查询
	if err := wd.db.WithContext(ctx).Where("id = ?", id).First(&onDutyGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wd.l.Warn("MonitorOnDutyGroup 未找到", zap.Int("id", id))
			return nil, nil
		}
		wd.l.Error("获取 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("failed to get MonitorOnDutyGroup by id %d: %w", id, err)
	}

	return &onDutyGroup, nil
}

// GetRuleById 根据ID获取MonitorAlertRule
func (wd *webhookDao) GetRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error) {
	var rule model.MonitorAlertRule

	// 执行查询
	if err := wd.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wd.l.Warn("MonitorAlertRule 未找到", zap.Int("id", id))
			return nil, nil
		}
		wd.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("failed to get MonitorAlertRule by id %d: %w", id, err)
	}

	return &rule, nil
}

// GetSendGroupById 根据ID获取MonitorSendGroup
func (wd *webhookDao) GetSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error) {
	var sendGroup model.MonitorSendGroup

	// 执行查询
	if err := wd.db.WithContext(ctx).Where("id = ?", id).First(&sendGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wd.l.Warn("MonitorSendGroup 未找到", zap.Int("id", id))
			return nil, nil
		}
		wd.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("failed to get MonitorSendGroup by id %d: %w", id, err)
	}

	return &sendGroup, nil
}

// GetUserById 根据ID获取User
func (wd *webhookDao) GetUserById(ctx context.Context, id int) (*model.User, error) {
	var user model.User

	// 执行查询
	if err := wd.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wd.l.Warn("User 未找到", zap.Int("id", id))
			return nil, nil
		}
		wd.l.Error("获取 User 失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("failed to get User by id %d: %w", id, err)
	}

	return &user, nil
}

// CreateOrUpdateEvent 创建或更新 MonitorAlertEvent
func (wd *webhookDao) CreateOrUpdateEvent(ctx context.Context, event *model.MonitorAlertEvent) error {
	var existingEvent model.MonitorAlertEvent

	// 根据 fingerprint 查询是否存在该事件
	err := wd.db.WithContext(ctx).
		Where("fingerprint = ?", event.Fingerprint).
		First(&existingEvent).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录不存在，创建新事件
			if err := wd.db.WithContext(ctx).Create(event).Error; err != nil {
				wd.l.Error("创建 MonitorAlertEvent 失败",
					zap.Error(err),
					zap.Any("event", event),
				)
				return fmt.Errorf("failed to create MonitorAlertEvent: %w", err)
			}
			wd.l.Info("成功创建 MonitorAlertEvent",
				zap.String("fingerprint", event.Fingerprint),
			)
			return nil
		}
		// 其他错误
		wd.l.Error("查询 MonitorAlertEvent 失败",
			zap.Error(err),
			zap.String("fingerprint", event.Fingerprint),
		)
		return fmt.Errorf("failed to query MonitorAlertEvent by fingerprint %s: %w", event.Fingerprint, err)
	}

	// 记录存在，执行更新
	if err := wd.db.WithContext(ctx).
		Model(&existingEvent).
		Updates(event).Error; err != nil {
		wd.l.Error("更新 MonitorAlertEvent 失败",
			zap.Error(err),
			zap.Any("event", event),
		)
		return fmt.Errorf("failed to update MonitorAlertEvent: %w", err)
	}

	wd.l.Info("成功更新 MonitorAlertEvent",
		zap.String("fingerprint", event.Fingerprint),
	)
	return nil
}

// GetMonitorAlertEventByFingerprintId 根据fingerprintId获取MonitorAlertEvent
func (wd *webhookDao) GetMonitorAlertEventByFingerprintId(ctx context.Context, fingerprintId string) (*model.MonitorAlertEvent, error) {
	var alertEvent model.MonitorAlertEvent

	// 执行查询
	if err := wd.db.WithContext(ctx).Where("fingerprint = ?", fingerprintId).First(&alertEvent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wd.l.Warn("MonitorAlertEvent 未找到", zap.String("fingerprintId", fingerprintId))
			return nil, nil
		}
		wd.l.Error("通过 fingerprint 查询 MonitorAlertEvent 失败", zap.Error(err), zap.String("fingerprintId", fingerprintId))
		return nil, fmt.Errorf("failed to get MonitorAlertEvent by fingerprint %s: %w", fingerprintId, err)
	}

	return &alertEvent, nil
}
