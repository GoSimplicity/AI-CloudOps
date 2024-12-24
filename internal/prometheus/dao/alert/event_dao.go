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
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"net/http"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerEventDAO interface {
	GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, error)
	GetMonitorAlertEventList(ctx context.Context) ([]*model.MonitorAlertEvent, error)
	EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error
	GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error
	SendMessageToGroup(ctx context.Context, url string, message string) error
}

type alertManagerEventDAO struct {
	db         *gorm.DB
	l          *zap.Logger
	userDao    userDao.UserDAO
	httpClient *http.Client
}

func NewAlertManagerEventDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerEventDAO {
	return &alertManagerEventDAO{
		db:         db,
		l:          l,
		userDao:    userDao,
		httpClient: &http.Client{},
	}
}

func (a *alertManagerEventDAO) GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		a.l.Error("GetMonitorAlertEventById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent
	if err := a.db.WithContext(ctx).First(&alertEvent, id).Error; err != nil {
		a.l.Error("获取 MonitorAlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

func (a *alertManagerEventDAO) SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, error) {
	var alertEvents []*model.MonitorAlertEvent

	if err := a.db.WithContext(ctx).
		Where("alert_name LIKE ?", "%"+name+"%").
		Find(&alertEvents).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertEvent 失败", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return alertEvents, nil
}

func (a *alertManagerEventDAO) GetMonitorAlertEventList(ctx context.Context) ([]*model.MonitorAlertEvent, error) {
	var alertEvents []*model.MonitorAlertEvent

	if err := a.db.WithContext(ctx).Find(&alertEvents).Error; err != nil {
		a.l.Error("获取 MonitorAlertEvent 列表失败", zap.Error(err))
		return nil, err
	}

	return alertEvents, nil
}

func (a *alertManagerEventDAO) EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error {
	if event == nil {
		a.l.Error("EventAlertClaim 失败: event 为 nil")
		return fmt.Errorf("event 不能为空")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("id = ?", event.ID).
		Updates(event).Error; err != nil {
		a.l.Error("EventAlertClaim 更新失败", zap.Error(err), zap.Int("id", event.ID))
		return err
	}

	return nil
}

func (a *alertManagerEventDAO) GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		a.l.Error("GetAlertEventByID 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent
	if err := a.db.WithContext(ctx).First(&alertEvent, id).Error; err != nil {
		a.l.Error("获取 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

func (a *alertManagerEventDAO) UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error {
	if alertEvent == nil {
		a.l.Error("UpdateAlertEvent 失败: alertEvent 为 nil")
		return fmt.Errorf("alertEvent 不能为空")
	}

	if err := a.db.WithContext(ctx).Save(alertEvent).Error; err != nil {
		a.l.Error("更新 AlertEvent 失败", zap.Error(err), zap.Int("id", alertEvent.ID))
		return err
	}

	return nil
}

// SendMessageToGroup 发送飞书群聊消息
func (a *alertManagerEventDAO) SendMessageToGroup(ctx context.Context, url string, message string) error {
	// 拼接发送内容
	content := fmt.Sprintf(`{"msg_type":"text","content":{"text":"%s"}}`, message)

	// 发送消息到群组
	body, err := pkg.PostWithJson(ctx, a.httpClient, a.l, url, content, nil, nil)
	if err != nil {
		a.l.Error("发送飞书群聊消息失败",
			zap.Error(err),
			zap.Any("结果", string(body)),
		)
		return fmt.Errorf("发送飞书群聊消息失败: %w", err)
	}

	a.l.Info("发送飞书群聊消息成功",
		zap.Any("结果", string(body)),
	)

	return nil
}
