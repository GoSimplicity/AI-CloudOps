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

// AlertEventSilenceState is the two-field state compared and written by a
// silence operation. Generic event updates retain their separate full contract.
type AlertEventSilenceState struct {
	Status    model.MonitorAlertEventStatus
	SilenceID string
}

var ErrAlertSilenceConflict = errors.New("alert silence state missing or changed")
var ErrAlertSilenceStore = errors.New("alert silence persistence outcome unconfirmed")

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
	UpdateMonitorAlertEventSilence(ctx context.Context, id int, expected, desired AlertEventSilenceState) error

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

	if err := wd.db.WithContext(ctx).Where("id = ?", id).Preload("Users").First(&onDutyGroup).Error; err != nil {
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

	if err := wd.db.WithContext(ctx).Where("fingerprint = ?", fingerprintId).First(&alertEvent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wd.l.Warn("MonitorAlertEvent 未找到")
			return nil, nil
		}
		wd.l.Error("通过 fingerprint 查询 MonitorAlertEvent 失败", zap.String("operation", "alert_event_lookup"))
		return nil, errors.New("failed to get MonitorAlertEvent")
	}

	return &alertEvent, nil
}

// FillTodayOnDutyUser 为指定的值班组填充当天的值班用户
func (wd *webhookDao) FillTodayOnDutyUser(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (*model.MonitorOnDutyGroup, error) {
	// 获取当前日期的字符串表示，格式为 "YYYY-MM-DD"
	today := time.Now().Format("2006-01-02")

	history, err := wd.getTodayOnDutyHistory(ctx, today, onDutyGroup.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果当天没有值班历史记录，分配默认值班用户
			return wd.assignDefaultDutyUser(ctx, onDutyGroup, today)
		}
		wd.l.Error("获取值班历史失败",
			zap.Error(err),
			zap.String("dateString", today),
			zap.Int("onDutyGroupId", onDutyGroup.ID),
		)
		return nil, err
	}

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
		wd.l.Error("获取值班人员失败",
			zap.Error(err),
			zap.Int("onDutyUserID", history.OnDutyUserID),
		)
		return nil, err
	}

	if user.ID > 0 {
		// 转换为 MonitorOnDutyUser 类型
		onDutyUser := &model.User{
			Model:        model.Model{ID: user.ID},
			RealName:     user.RealName,
			Username:     user.Username,
			FeiShuUserId: user.FeiShuUserId,
		}
		onDutyGroup.TodayDutyUser = onDutyUser
	} else {
		wd.l.Warn("获取到的用户 ID 不合法，分配默认值班用户",
			zap.Int("onDutyUserID", history.OnDutyUserID),
			zap.Int("onDutyGroupId", onDutyGroup.ID),
		)
		return wd.assignDefaultDutyUser(ctx, onDutyGroup, today)
	}

	return onDutyGroup, nil
}

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
	if len(onDutyGroup.Users) == 0 {
		wd.l.Warn("onDutyGroup.Users 为空，无法分配 TodayDutyUser",
			zap.Int("onDutyGroupId", onDutyGroup.ID),
		)
		return nil, fmt.Errorf("onDutyGroup ID %d 的成员列表为空，无法分配 TodayDutyUser", onDutyGroup.ID)
	}

	onDutyGroup.TodayDutyUser = onDutyGroup.Users[0]
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
		wd.l.Error("更新 MonitorAlertEvent 失败", zap.String("operation", "alert_event_update"))
		return errors.New("failed to update MonitorAlertEvent")
	}

	return nil
}

// UpdateMonitorAlertEventSilence atomically changes only the requested silence
// state. An old HTTP response cannot overwrite a newer local silence ID. NULL
// old IDs compare as empty, matching GORM's string decoding of legacy rows.
func (wd *webhookDao) UpdateMonitorAlertEventSilence(ctx context.Context, id int, expected, desired AlertEventSilenceState) error {
	if id <= 0 {
		return ErrAlertSilenceConflict
	}
	result := wd.db.WithContext(ctx).Model(&model.MonitorAlertEvent{}).
		// HEX preserves byte equality under case-insensitive MySQL collations.
		Where("id = ? AND status = ? AND HEX(COALESCE(silence_id, '')) = HEX(?)", id, expected.Status, expected.SilenceID).
		Updates(map[string]any{"status": desired.Status, "silence_id": desired.SilenceID})
	if result.Error != nil {
		wd.l.Error("保存静默状态失败", zap.String("operation", "alert_silence_update"))
		return ErrAlertSilenceStore
	}
	if result.RowsAffected == 1 {
		return nil
	}
	if result.RowsAffected != 0 {
		return ErrAlertSilenceStore
	}
	// MySQL can report zero for an unchanged row; concurrent identical requests
	// can also miss the old-state predicate. Read back the exact desired state.
	var current AlertEventSilenceState
	err := wd.db.WithContext(ctx).Model(&model.MonitorAlertEvent{}).Select("status", "silence_id").Where("id = ?", id).Take(&current).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAlertSilenceConflict
	}
	if err != nil {
		wd.l.Error("核对静默状态失败", zap.String("operation", "alert_silence_readback"))
		return ErrAlertSilenceStore
	}
	if current != desired {
		return ErrAlertSilenceConflict
	}
	return nil
}

// GetMonitorOnDutyGroupList 获取所有 MonitorOnDutyGroup
func (wd *webhookDao) GetMonitorOnDutyGroupList(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var onDutyGroups []*model.MonitorOnDutyGroup

	if err := wd.db.WithContext(ctx).
		Preload("Users").
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
