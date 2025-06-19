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

package alert

import (
	"context"
	"fmt"
	"net/http"
	"time"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerEventDAO interface {
	GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, int64, error)
	GetMonitorAlertEventList(ctx context.Context, offset, limit int) ([]*model.MonitorAlertEvent, int64, error)
	EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error
	GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error
	SendMessageToGroup(ctx context.Context, url string, message string) error
	GetMonitorAlertEventTotal(ctx context.Context) (int, error)
}

type alertManagerEventDAO struct {
	db         *gorm.DB
	l          *zap.Logger
	userDao    userDao.UserDAO
	httpClient *http.Client
}

func NewAlertManagerEventDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerEventDAO {
	return &alertManagerEventDAO{
		db:      db,
		l:       l,
		userDao: userDao,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// 获取当前时间戳
func getTime() int64 {
	return time.Now().Unix()
}

// GetMonitorAlertEventById 获取告警事件
func (a *alertManagerEventDAO) GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		a.l.Error("GetMonitorAlertEventById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent

	if err := a.db.WithContext(ctx).Where("deleted_at = ?", 0).First(&alertEvent, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到ID为 %d 的告警事件", id)
		}
		a.l.Error("获取 MonitorAlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

// SearchMonitorAlertEventByName 通过名称搜索告警事件
func (a *alertManagerEventDAO) SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, int64, error) {
	if name == "" {
		return nil, 0, fmt.Errorf("搜索名称不能为空")
	}

	var alertEvents []*model.MonitorAlertEvent
	var total int64

	// 先获取符合条件的记录总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("deleted_at = ?", 0).
		Where("alert_name LIKE ?", "%"+name+"%").
		Count(&total).Error; err != nil {
		a.l.Error("获取搜索记录总数失败", zap.Error(err), zap.String("name", name))
		return nil, 0, err
	}

	// 获取符合条件的记录列表
	if err := a.db.WithContext(ctx).
		Where("deleted_at = ?", 0).
		Where("alert_name LIKE ?", "%"+name+"%").
		Find(&alertEvents).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertEvent 失败", zap.Error(err), zap.String("name", name))
		return nil, 0, err
	}

	return alertEvents, total, nil
}

// GetMonitorAlertEventList 获取告警事件列表
func (a *alertManagerEventDAO) GetMonitorAlertEventList(ctx context.Context, offset, limit int) ([]*model.MonitorAlertEvent, int64, error) {
	if offset < 0 {
		return nil, 0, fmt.Errorf("offset不能为负数")
	}
	if limit <= 0 {
		return nil, 0, fmt.Errorf("limit必须大于0")
	}

	var alertEvents []*model.MonitorAlertEvent
	var total int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("deleted_at = ?", 0).
		Count(&total).Error; err != nil {
		a.l.Error("获取 MonitorAlertEvent 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 获取分页数据
	if err := a.db.WithContext(ctx).
		Where("deleted_at = ?", 0).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&alertEvents).Error; err != nil {
		a.l.Error("获取 MonitorAlertEvent 列表失败", zap.Error(err))
		return nil, 0, err
	}

	return alertEvents, total, nil
}

// EventAlertClaim 认领告警事件
func (a *alertManagerEventDAO) EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error {
	if event.ID <= 0 {
		return fmt.Errorf("无效的事件ID")
	}

	result := a.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("id = ? AND deleted_at = ?", event.ID, 0).
		Updates(event)

	if result.Error != nil {
		a.l.Error("EventAlertClaim 更新失败", zap.Error(result.Error), zap.Int("id", event.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到ID为 %d 的告警事件或已被删除", event.ID)
	}

	return nil
}

// GetAlertEventByID 通过ID获取告警事件
func (a *alertManagerEventDAO) GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		a.l.Error("GetAlertEventByID 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent

	if err := a.db.WithContext(ctx).Where("deleted_at = ?", 0).First(&alertEvent, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到ID为 %d 的告警事件", id)
		}
		a.l.Error("获取 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

// UpdateAlertEvent 更新告警事件
func (a *alertManagerEventDAO) UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error {
	if alertEvent.ID <= 0 {
		return fmt.Errorf("无效的事件ID")
	}

	result := a.db.WithContext(ctx).
		Where("id = ? AND deleted_at = ?", alertEvent.ID, 0).
		Updates(map[string]interface{}{
			"alert_name":       alertEvent.AlertName,
			"fingerprint":      alertEvent.Fingerprint,
			"status":           alertEvent.Status,
			"rule_id":          alertEvent.RuleID,
			"send_group_id":    alertEvent.SendGroupID,
			"event_times":      alertEvent.EventTimes,
			"silence_id":       alertEvent.SilenceID,
			"ren_ling_user_id": alertEvent.RenLingUserID,
			"labels":           alertEvent.Labels,
			"updated_at":       getTime(),
		})

	if result.Error != nil {
		a.l.Error("更新 AlertEvent 失败", zap.Error(result.Error), zap.Int("id", alertEvent.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到ID为 %d 的告警事件或已被删除", alertEvent.ID)
	}

	return nil
}

// SendMessageToGroup 发送飞书群聊消息
func (a *alertManagerEventDAO) SendMessageToGroup(ctx context.Context, url string, message string) error {
	if url == "" {
		return fmt.Errorf("url不能为空")
	}
	if message == "" {
		return fmt.Errorf("message不能为空")
	}

	// 拼接发送内容
	content := fmt.Sprintf(`{"msg_type":"text","content":{"text":"%s"}}`, message)

	// 发送消息到群组
	body, err := pkg.PostWithJson(ctx, a.httpClient, a.l, url, content, nil, nil)
	if err != nil {
		a.l.Error("发送飞书群聊消息失败",
			zap.Error(err),
			zap.String("url", url),
			zap.String("message", message),
			zap.Any("结果", string(body)),
		)
		return fmt.Errorf("发送飞书群聊消息失败: %w", err)
	}

	a.l.Info("发送飞书群聊消息成功",
		zap.String("url", url),
		zap.String("message", message),
		zap.Any("结果", string(body)),
	)

	return nil
}

// GetMonitorAlertEventTotal 获取监控告警事件总数
func (a *alertManagerEventDAO) GetMonitorAlertEventTotal(ctx context.Context) (int, error) {
	var count int64

	if err := a.db.WithContext(ctx).Model(&model.MonitorAlertEvent{}).Where("deleted_at = ?", 0).Count(&count).Error; err != nil {
		a.l.Error("获取监控告警事件总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
