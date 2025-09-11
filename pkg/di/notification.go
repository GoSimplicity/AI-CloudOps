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

package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// NotificationConfigAdapter 通知配置适配器
type NotificationConfigAdapter struct {
	config *NotificationConfig
}

// GetEmail 获取邮箱配置
func (a *NotificationConfigAdapter) GetEmail() notification.EmailConfig {
	emailConfig := a.config.GetEmail()
	if emailConfig == nil {
		return nil
	}
	return emailConfig
}

// GetFeishu 获取飞书配置
func (a *NotificationConfigAdapter) GetFeishu() notification.FeishuConfig {
	feishuConfig := a.config.GetFeishu()
	if feishuConfig == nil {
		return nil
	}
	return feishuConfig
}

// InitNotificationConfig 初始化通知配置
func InitNotificationConfig() notification.NotificationConfig {
	return &NotificationConfigAdapter{
		config: &GlobalConfig.Notification,
	}
}

// InitNotificationManager 初始化通知管理器
func InitNotificationManager(config notification.NotificationConfig, asynqClient *asynq.Client, logger *zap.Logger) *notification.Manager {
	manager, err := notification.NewManager(config, asynqClient, logger)
	if err != nil {
		panic(err)
	}
	return manager
}
