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

type WebhookDao interface {
	GetOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	GetRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	GetMonitorAlertRuleList(ctx context.Context) ([]*model.MonitorAlertRule, error)
	GetSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error)
	GetUserById(ctx context.Context, id int) (*model.User, error)
	GetUserList(ctx context.Context) ([]*model.User, error)
	GetMonitorOnDutyGroupList(ctx context.Context) ([]*model.MonitorOnDutyGroup, error)
	GetMonitorSendGroupList(ctx context.Context) ([]*model.MonitorSendGroup, error)
	GetMonitorAlertEventByFingerprintId(ctx context.Context, fingerprintId string) (*model.MonitorAlertEvent, error)

	CreateOrUpdateEvent(ctx context.Context, event *model.MonitorAlertEvent) error
	UpdateMonitorAlertEvent(ctx context.Context, event *model.MonitorAlertEvent) error

	FillTodayOnDutyUser(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (*model.MonitorOnDutyGroup, error)
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
	if err := wd.db.WithContext(ctx).Where("id = ?", id).Preload("Members").First(&onDutyGroup).Error; err != nil {
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
	if err := wd.db.WithContext(ctx).Where("id = ?", id).Preload("FirstUpgradeUsers").First(&sendGroup).Error; err != nil {
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
	// 使用事务确保操作的原子性
	return wd.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingEvent model.MonitorAlertEvent

		// 根据 fingerprint 查询是否存在该事件
		err := tx.Where("fingerprint = ?", event.Fingerprint).First(&existingEvent).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 记录不存在，创建新事件
				if err := tx.Create(event).Error; err != nil {
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
		if err := tx.Model(&existingEvent).Updates(event).Error; err != nil {
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
	})
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

// FillTodayOnDutyUser 为指定的值班组填充当天的值班用户
func (wd *webhookDao) FillTodayOnDutyUser(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (*model.MonitorOnDutyGroup, error) {
	// 获取当前日期的字符串表示，格式为 "YYYY-MM-DD"
	today := time.Now().Format("2006-01-02")

	// 查询当天的值班历史记录
	history, err := wd.getTodayOnDutyHistory(ctx, today, onDutyGroup.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果当天没有值班历史记录，分配默认值班用户
			return wd.assignDefaultDutyUser(ctx, onDutyGroup, today)
		}
		// 其他错误，记录并返回
		wd.l.Error("获取值班历史失败",
			zap.Error(err),
			zap.String("dateString", today),
			zap.Int("onDutyGroupId", onDutyGroup.ID),
		)
		return nil, err
	}

	// 查询对应的值班用户
	user, err := wd.getUserByID(ctx, history.OnDutyUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果指定的值班用户不存在，分配默认值班用户
			wd.l.Warn("指定的值班用户不存在，分配默认值班用户",
				zap.Int("onDutyUserID", history.OnDutyUserID),
				zap.Int("onDutyGroupId", onDutyGroup.ID),
			)
			return wd.assignDefaultDutyUser(ctx, onDutyGroup, today)
		}
		// 其他错误，记录并返回
		wd.l.Error("获取值班人员失败",
			zap.Error(err),
			zap.Int("onDutyUserID", history.OnDutyUserID),
		)
		return nil, err
	}

	// 设置今天的值班用户
	if user.ID > 0 {
		onDutyGroup.TodayDutyUser = user
	} else {
		wd.l.Warn("获取到的用户 ID 不合法，分配默认值班用户",
			zap.Int("onDutyUserID", history.OnDutyUserID),
			zap.Int("onDutyGroupId", onDutyGroup.ID),
		)
		return wd.assignDefaultDutyUser(ctx, onDutyGroup, today)
	}

	return onDutyGroup, nil
}

// getTodayOnDutyHistory 查询当天的值班历史记录
func (wd *webhookDao) getTodayOnDutyHistory(ctx context.Context, dateStr string, groupID int) (*model.MonitorOnDutyHistory, error) {
	var history model.MonitorOnDutyHistory
	err := wd.db.WithContext(ctx).
		Where("date_string = ? AND on_duty_group_id = ?", dateStr, groupID).
		First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

// getUserByID 根据用户ID查询用户信息
func (wd *webhookDao) getUserByID(ctx context.Context, userID int) (*model.User, error) {
	var user model.User
	err := wd.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// assignDefaultDutyUser 为值班组分配默认的值班用户（成员列表中的第一个成员）
func (wd *webhookDao) assignDefaultDutyUser(_ context.Context, onDutyGroup *model.MonitorOnDutyGroup, dateStr string) (*model.MonitorOnDutyGroup, error) {
	if len(onDutyGroup.Members) == 0 {
		wd.l.Warn("onDutyGroup.Members 为空，无法分配 TodayDutyUser",
			zap.Int("onDutyGroupId", onDutyGroup.ID),
		)
		return nil, fmt.Errorf("onDutyGroup ID %d 的成员列表为空，无法分配 TodayDutyUser", onDutyGroup.ID)
	}

	onDutyGroup.TodayDutyUser = onDutyGroup.Members[0]
	wd.l.Info("分配默认值班用户",
		zap.String("dateString", dateStr),
		zap.Int("onDutyGroupId", onDutyGroup.ID),
		zap.Int("assignedUserID", onDutyGroup.TodayDutyUser.ID),
	)
	return onDutyGroup, nil
}

// UpdateMonitorAlertEvent 更新 MonitorAlertEvent
func (wd *webhookDao) UpdateMonitorAlertEvent(ctx context.Context, event *model.MonitorAlertEvent) error {
	if err := wd.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("id = ?", event.ID).
		Updates(event).Error; err != nil {
		wd.l.Error("更新 MonitorAlertEvent 失败",
			zap.Error(err),
			zap.Any("event", event),
		)
		return fmt.Errorf("failed to update MonitorAlertEvent: %w", err)
	}

	return nil
}

// GetMonitorOnDutyGroupList 获取所有 MonitorOnDutyGroup
func (wd *webhookDao) GetMonitorOnDutyGroupList(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var onDutyGroups []*model.MonitorOnDutyGroup

	if err := wd.db.WithContext(ctx).
		Preload("Members").
		Find(&onDutyGroups).Error; err != nil {
		wd.l.Error("获取值班组列表失败",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get onDutyGroups: %w", err)
	}

	return onDutyGroups, nil
}

// GetMonitorSendGroupList 获取所有 MonitorSendGroup
func (wd *webhookDao) GetMonitorSendGroupList(ctx context.Context) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	if err := wd.db.WithContext(ctx).
		Preload("FirstUpgradeUsers").
		Find(&sendGroups).Error; err != nil {
		wd.l.Error("获取发送组列表失败",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get sendGroups: %w", err)
	}

	return sendGroups, nil
}

// GetUserList 获取所有用户
func (wd *webhookDao) GetUserList(ctx context.Context) ([]*model.User, error) {
	var users []*model.User

	if err := wd.db.WithContext(ctx).Find(&users).Error; err != nil {
		wd.l.Error("获取用户列表失败", zap.Error(err))
		return nil, fmt.Errorf("failed to get user list: %w", err)
	}

	return users, nil
}

// GetMonitorAlertRuleList 获取所有 MonitorAlertRule
func (wd *webhookDao) GetMonitorAlertRuleList(ctx context.Context) ([]*model.MonitorAlertRule, error) {
	var rules []*model.MonitorAlertRule

	if err := wd.db.WithContext(ctx).Find(&rules).Error; err != nil {
		wd.l.Error("获取 MonitorAlertRule 失败", zap.Error(err))
		return nil, err
	}

	return rules, nil
}
