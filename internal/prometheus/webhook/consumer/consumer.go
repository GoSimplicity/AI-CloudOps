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

package consumer

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/content"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/prometheus/alertmanager/template"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// WebhookConsumer 定义了Webhook消费者的接口
type WebhookConsumer interface {
	// AlertReceiveConsumerManager 管理告警接收的消费者
	AlertReceiveConsumerManager(ctx context.Context) error
	// HandleAlert 处理单个告警接收
	HandleAlert(ctx context.Context, alert template.Alert)
}

// webhookConsumer 是 WebhookConsumer 接口的实现
type webhookConsumer struct {
	alertReceiveQueue chan template.Alert // 告警接收队列
	cache             cache.WebhookCache
	dao               dao.WebhookDao
	content           content.WebhookContent
	logger            *zap.Logger
	workerCount       int           // 固定的工作者数量
	exitWorkerChan    chan struct{} // 退出信号通道

	wg     sync.WaitGroup // 用于等待所有工作者完成
	mu     sync.Mutex     // 保护资源
	closed bool           // 标记消费者是否已关闭
}

// NewWebhookConsumer 创建一个新的WebhookConsumer实例
func NewWebhookConsumer(logger *zap.Logger, cache cache.WebhookCache, dao dao.WebhookDao, content content.WebhookContent, alertReceiveQueue chan template.Alert) WebhookConsumer {
	return &webhookConsumer{
		logger:            logger,
		cache:             cache,
		dao:               dao,
		content:           content,
		alertReceiveQueue: alertReceiveQueue,
		exitWorkerChan:    make(chan struct{}),
		workerCount:       viper.GetInt("webhook.fixed_workers"), // 从配置中获取固定工作者数量
	}
}

// AlertReceiveConsumerManager 启动消费者管理器，启动固定数量的工作者并监听告警接收队列
func (wc *webhookConsumer) AlertReceiveConsumerManager(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			wc.logger.Info("AlertReceiveConsumerManager 收到其他任务退出信号 退出")
			return nil
		case alert := <-wc.alertReceiveQueue:
			go wc.HandleAlert(ctx, alert)
		}

	}
}

// HandleAlert 处理单个告警接收
func (wc *webhookConsumer) HandleAlert(ctx context.Context, alert template.Alert) {
	// 提取 send_group_id
	sendGroupIDStr, exists := alert.Labels["alert_send_group"]
	if !exists {
		wc.logger.Info("告警信息缺少 send_group_id", zap.Any("alert", alert))
		return
	}

	// 提取 rule_id
	ruleIDStr, exists := alert.Labels["alert_rule_id"]
	if !exists {
		wc.logger.Info("告警信息缺少 rule_id", zap.Any("alert", alert))
		return
	}

	// 转换 send_group_id 和 rule_id 为整数
	sendGroupID, err := strconv.Atoi(sendGroupIDStr)
	if err != nil {
		wc.logger.Error("转换 send_group_id 失败",
			zap.String("sendGroupIDStr", sendGroupIDStr),
			zap.Error(err),
		)
		return
	}

	ruleID, err := strconv.Atoi(ruleIDStr)
	if err != nil {
		wc.logger.Error("转换 rule_id 失败",
			zap.String("ruleIDStr", ruleIDStr),
			zap.Error(err),
		)
		return
	}

	// 从缓存中获取 sendGroup
	sendGroup := wc.cache.GetSendGroupById(sendGroupID)
	if sendGroup == nil {
		wc.logger.Info("缓存中不存在对应的 sendGroup",
			zap.Int("sendGroupID", sendGroupID),
			zap.Any("alert", alert),
		)
		return
	}

	// 从缓存中获取用户信息
	createUser := wc.cache.GetUserById(sendGroup.UserID)
	if createUser == nil {
		wc.logger.Info("缓存中不存在对应的用户",
			zap.Int("userID", sendGroup.UserID),
			zap.Any("alert", alert),
		)
		return
	}

	// 从缓存中获取规则
	rule := wc.cache.GetRuleById(ruleID)
	if rule == nil {
		wc.logger.Info("缓存中不存在对应的规则",
			zap.Int("ruleID", ruleID),
			zap.Any("alert", alert),
		)
		return
	}

	wc.logger.Debug("收到告警信息，准备处理",
		zap.Any("alert", alert),
		zap.Time("start_time", alert.StartsAt),
		zap.Time("end_time", alert.EndsAt),
		zap.Int("sendGroupID", sendGroupID),
		zap.Int("ruleID", ruleID),
	)

	// 判断是否需要升级
	upgradeNeed := false
	if alert.Status == "firing" && sendGroup.FirstUpgradeUsers != nil && len(sendGroup.FirstUpgradeUsers) > 0 {
		upgradeNeed = true
	}

	status := alert.Status
	if upgradeNeed {
		status = "upgraded"
	}

	// 构造标签
	var labels []string
	for key, val := range alert.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, val))
	}

	// 构造 MonitorAlertEvent
	event := &model.MonitorAlertEvent{
		AlertName:   alert.Labels["alertname"],
		Fingerprint: alert.Fingerprint,
		Status:      status,
		RuleID:      ruleID,
		Labels:      labels,
		SendGroupID: sendGroupID,
	}

	// 创建或更新事件
	if err := wc.dao.CreateOrUpdateEvent(ctx, event); err != nil {
		wc.logger.Error("创建或更新 MonitorAlertEvent 失败",
			zap.Error(err),
			zap.Any("event", event),
			zap.Any("alert", alert),
		)
		return
	}

	// 获取更新后的事件
	updatedEvent, err := wc.dao.GetMonitorAlertEventByFingerprintId(ctx, alert.Fingerprint)
	if err != nil {
		wc.logger.Error("查询 MonitorAlertEvent 失败",
			zap.Error(err),
			zap.String("fingerprint", alert.Fingerprint),
		)
		return
	}

	// 生成飞书卡片内容
	if err := wc.content.GenerateFeishuCardContentOneAlert(ctx, alert, updatedEvent, rule, sendGroup); err != nil {
		wc.logger.Error("生成飞书卡片内容失败",
			zap.Error(err),
			zap.Any("alert", alert),
			zap.Any("event", updatedEvent),
			zap.Any("rule", rule),
			zap.Any("sendGroup", sendGroup),
		)
		return
	}

	wc.logger.Info("成功处理告警",
		zap.String("fingerprint", alert.Fingerprint),
	)
}
