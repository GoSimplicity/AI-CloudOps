package webhook

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"strconv"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/content"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/prometheus/alertmanager/template"
	"go.uber.org/zap"
)

type WebhookConsumer interface {
	// AlertReceiveConsumerManager 管理告警接收的消费者
	AlertReceiveConsumerManager(ctx context.Context) error
	// DealWithOneAlertReceive 处理单个告警接收
	DealWithOneAlertReceive(ctx context.Context, alert template.Alert)
}

type webhookConsumer struct {
	alertReceiveQueue chan template.Alert
	cache             cache.WebhookCache
	dao               dao.WebhookDao
	content           content.WebhookContent
	logger            *zap.Logger

	workerCount    int           // 当前工作协程数量
	minWorkers     int           // 最小工作协程数量
	maxWorkers     int           // 最大工作协程数量
	scaleInterval  time.Duration // 监控和调整工作协程的间隔时间
	scaleThreshold int           // 阈值，决定是否需要扩缩协程
	exitWorkerChan chan struct{} // 退出信号通道

	mu sync.Mutex // 保护 workerCount
}

func NewWebhookConsumer(logger *zap.Logger, cache cache.WebhookCache, alertReceiveQueue chan template.Alert, dao dao.WebhookDao, content content.WebhookContent, minWorkers, maxWorkers, scaleThreshold int, scaleInterval time.Duration) WebhookConsumer {
	return &webhookConsumer{
		logger:            logger,
		alertReceiveQueue: alertReceiveQueue,
		cache:             cache,
		dao:               dao,
		content:           content,
		minWorkers:        viper.GetInt("webhook.min_workers"),
		maxWorkers:        viper.GetInt("webhook.max_workers"),
		scaleThreshold:    viper.GetInt("webhook.scale_threshold"),
		scaleInterval:     viper.GetDuration("webhook.scale_interval"),
		exitWorkerChan:    make(chan struct{}, maxWorkers), // 设置缓冲区大小以避免阻塞
	}
}

// AlertReceiveConsumerManager 启动消费者管理器，监听告警接收队列并动态调整工作协程数量
func (wc *webhookConsumer) AlertReceiveConsumerManager(ctx context.Context) error {
	// 初始化工作协程数量为最小值
	wc.mu.Lock()
	wc.workerCount = wc.minWorkers
	wc.mu.Unlock()

	for i := 0; i < wc.minWorkers; i++ {
		go wc.worker(ctx, i)
	}

	// 启动一个独立的协程用于监控和调整工作协程数量
	go wc.scaleWorkers(ctx)

	// 等待上下文取消
	<-ctx.Done()
	wc.logger.Info("AlertReceiveConsumerManager 收到退出信号，等待工作协程退出")
	return nil
}

// worker 是一个工作协程，持续从告警接收队列中获取告警并处理
func (wc *webhookConsumer) worker(ctx context.Context, workerID int) {
	wc.logger.Info("启动一个工作协程", zap.Int("workerID", workerID))
	for {
		select {
		case <-ctx.Done():
			wc.logger.Info("工作协程收到上下文取消信号，退出", zap.Int("workerID", workerID))
			return
		case <-wc.exitWorkerChan:
			wc.logger.Info("工作协程收到缩减退出信号，退出", zap.Int("workerID", workerID))
			return
		case alert, ok := <-wc.alertReceiveQueue:
			if !ok {
				wc.logger.Info("告警接收队列已关闭，工作协程退出", zap.Int("workerID", workerID))
				return
			}
			wc.DealWithOneAlertReceive(ctx, alert)
		}
	}
}

// scaleWorkers 监控队列长度并动态调整工作协程数量
func (wc *webhookConsumer) scaleWorkers(ctx context.Context) {
	ticker := time.NewTicker(wc.scaleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wc.logger.Info("scaleWorkers 收到退出信号，停止动态扩缩")
			return
		case <-ticker.C:
			queueLength := len(wc.alertReceiveQueue)
			wc.logger.Debug("监控队列长度", zap.Int("queueLength", queueLength), zap.Int("currentWorkers", wc.getWorkerCount()))

			if queueLength > wc.scaleThreshold && wc.getWorkerCount() < wc.maxWorkers {
				// 扩展工作协程
				wc.mu.Lock()
				newWorkerID := wc.workerCount
				wc.workerCount++
				wc.mu.Unlock()
				go wc.worker(ctx, newWorkerID)
				wc.logger.Info("扩展工作协程", zap.Int("newWorkerID", newWorkerID))
			} else if queueLength < wc.scaleThreshold/2 && wc.getWorkerCount() > wc.minWorkers {
				// 缩减工作协程：发送退出信号
				select {
				case wc.exitWorkerChan <- struct{}{}:
					wc.mu.Lock()
					wc.workerCount--
					wc.mu.Unlock()
					wc.logger.Info("缩减工作协程", zap.Int("remainingWorkerCount", wc.getWorkerCount()))
				default:
					wc.logger.Warn("尝试发送退出信号失败，可能退出信号通道已满")
				}
			}
		}
	}
}

// getWorkerCount 安全地获取当前工作协程数量
func (wc *webhookConsumer) getWorkerCount() int {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return wc.workerCount
}

// DealWithOneAlertReceive 处理单个告警接收
func (wc *webhookConsumer) DealWithOneAlertReceive(ctx context.Context, alert template.Alert) {
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
	}

	// 从缓存中获取规则
	rule := wc.cache.GetRuleById(ruleID)
	if rule == nil {
		wc.logger.Info("缓存中不存在对应的规则",
			zap.Int("ruleID", ruleID),
			zap.Any("alert", alert),
		)
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
	if err := wc.content.GenerateFeishuCardContentOneAlert(alert, updatedEvent, rule, sendGroup); err != nil {
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
