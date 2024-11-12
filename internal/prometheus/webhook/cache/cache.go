package cache

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

import (
	"context"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WebhookCache 定义了Webhook缓存的接口
type WebhookCache interface {
	RenewAllCaches(ctx context.Context) error            // 刷新所有缓存
	GetOnDutyGroupById(id int) *model.MonitorOnDutyGroup // 根据 ID 获取 OnDutyGroup 数据
	GetRuleById(id int) *model.MonitorAlertRule          // 根据 ID 获取 Rule 数据
	GetSendGroupById(id int) *model.MonitorSendGroup     // 根据 ID 获取 SendGroup 数据
	GetUserById(id int) *model.User                      // 根据 ID 获取 User 数据
}

// webhookCache 是 WebhookCache 接口的实现
type webhookCache struct {
	l              *zap.Logger
	dao            dao.WebhookDao
	robot          robot.WebhookRobot
	cacheOnce      sync.Once      // 确保 cacheHasSynced 只被关闭一次
	cacheHasSynced chan struct{}  // 用于通知缓存已同步完成
	initWG         sync.WaitGroup // 用于等待所有缓存初次同步完成

	// 各类缓存数据
	SendGroupMap   map[int]*model.MonitorSendGroup   // 发送组缓存
	UserMap        map[int]*model.User               // 用户缓存
	OnDutyGroupMap map[int]*model.MonitorOnDutyGroup // 值班组缓存
	RuleMap        map[int]*model.MonitorAlertRule   // 规则缓存

	// 各类缓存数据的读写锁
	SendGroupLock   sync.RWMutex
	UserLock        sync.RWMutex
	OnDutyGroupLock sync.RWMutex
	RuleLock        sync.RWMutex
}

// NewWebhookCache 创建一个新的 webhookCache 实例
func NewWebhookCache(l *zap.Logger, dao dao.WebhookDao, robot robot.WebhookRobot) WebhookCache {
	return &webhookCache{
		l:              l,
		dao:            dao,
		robot:          robot,
		cacheHasSynced: make(chan struct{}),
		SendGroupMap:   make(map[int]*model.MonitorSendGroup),
		UserMap:        make(map[int]*model.User),
		OnDutyGroupMap: make(map[int]*model.MonitorOnDutyGroup),
		RuleMap:        make(map[int]*model.MonitorAlertRule),
	}
}

func (wc *webhookCache) RenewAllCaches(ctx context.Context) error {
	renewInterval := time.Duration(viper.GetInt("webhook.common_map_renew_interval_seconds")) * time.Second

	// 在 goroutine 启动前调用 Add
	wc.initWG.Add(4) // 四个缓存需要初次同步

	// 启动定时刷新各类缓存
	wc.startCacheRefresh(ctx, wc.RenewMapSendGroup, renewInterval)
	wc.startCacheRefresh(ctx, wc.RenewMapUser, renewInterval)
	wc.startCacheRefresh(ctx, wc.RenewMapOnDutyGroup, renewInterval)
	wc.startCacheRefresh(ctx, wc.RenewMapRule, renewInterval)

	// 启动私有机器人令牌的定时刷新
	go wait.UntilWithContext(ctx, wc.robot.RefreshPrivateRobotToken, 5*time.Minute)

	// 等待所有缓存初次同步完成
	go func() {
		wc.initWG.Wait()
		wc.cacheOnce.Do(func() {
			close(wc.cacheHasSynced)
		})
	}()

	// 等待上下文取消
	<-ctx.Done()
	wc.l.Info("RenewAllCaches 收到退出信号，停止所有缓存刷新任务")
	return nil
}

func (wc *webhookCache) startCacheRefresh(ctx context.Context, renewFunc func(context.Context), interval time.Duration) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				wc.l.Error("缓存刷新过程中发生 panic", zap.Any("error", r))
				wc.initWG.Done() // 确保 Done 被调用
			}
		}()

		renewFunc(ctx)                                  // 初次刷新
		wc.initWG.Done()                                // 完成一次初次同步
		wait.UntilWithContext(ctx, renewFunc, interval) // 定时刷新
	}()
}

// RenewMapOnDutyGroup 刷新 OnDutyGroup 缓存
func (wc *webhookCache) RenewMapOnDutyGroup(ctx context.Context) {
	onDutyGroups, err := wc.dao.GetMonitorOnDutyGroupList(ctx)
	if err != nil {
		wc.l.Error("[缓存刷新模块] 获取 OnDutyGroup 列表失败", zap.Error(err))
		return
	}

	if len(onDutyGroups) == 0 {
		wc.l.Info("[缓存刷新模块] OnDutyGroup 列表为空")
	}

	tmpMap := make(map[int]*model.MonitorOnDutyGroup, len(onDutyGroups))
	for _, group := range onDutyGroups {
		tmpMap[group.ID] = group
	}

	// 加锁并更新缓存
	wc.OnDutyGroupLock.Lock()
	wc.OnDutyGroupMap = tmpMap
	wc.OnDutyGroupLock.Unlock()

	wc.logCacheRefreshResult("OnDutyGroup", len(wc.OnDutyGroupMap))
}

// GetOnDutyGroupById 根据 ID 获取 OnDutyGroup 数据
func (wc *webhookCache) GetOnDutyGroupById(id int) *model.MonitorOnDutyGroup {
	wc.OnDutyGroupLock.RLock()
	defer wc.OnDutyGroupLock.RUnlock()
	if group, exists := wc.OnDutyGroupMap[id]; exists {
		return group
	}
	return nil
}

// RenewMapRule 刷新 Rule 缓存
func (wc *webhookCache) RenewMapRule(ctx context.Context) {
	rules, err := wc.dao.GetMonitorAlertRuleList(ctx)
	if err != nil {
		wc.l.Error("[缓存刷新模块] 获取 Rule 列表失败", zap.Error(err))
		return
	}

	if len(rules) == 0 {
		wc.l.Info("[缓存刷新模块] Rule 列表为空")
		return
	}

	tmpMap := make(map[int]*model.MonitorAlertRule, len(rules))
	for _, rule := range rules {
		tmpMap[rule.ID] = rule
	}

	// 加锁并更新缓存
	wc.RuleLock.Lock()
	wc.RuleMap = tmpMap
	wc.RuleLock.Unlock()

	wc.logCacheRefreshResult("Rule", len(wc.RuleMap))
}

// GetRuleById 根据 ID 获取 Rule 数据
func (wc *webhookCache) GetRuleById(id int) *model.MonitorAlertRule {
	wc.RuleLock.RLock()
	defer wc.RuleLock.RUnlock()
	if rule, exists := wc.RuleMap[id]; exists {
		return rule
	}
	return nil
}

// RenewMapSendGroup 刷新 SendGroup 缓存
func (wc *webhookCache) RenewMapSendGroup(ctx context.Context) {
	sendGroups, err := wc.dao.GetMonitorSendGroupList(ctx)
	if err != nil {
		wc.l.Error("[缓存刷新模块] 获取 SendGroup 列表失败", zap.Error(err))
		return
	}

	if len(sendGroups) == 0 {
		wc.l.Info("[缓存刷新模块] SendGroup 列表为空")
		return
	}

	tmpMap := make(map[int]*model.MonitorSendGroup, len(sendGroups))
	for _, sendGroup := range sendGroups {
		tmpMap[sendGroup.ID] = sendGroup
	}

	// 加锁并更新缓存
	wc.SendGroupLock.Lock()
	wc.SendGroupMap = tmpMap
	wc.SendGroupLock.Unlock()

	wc.logCacheRefreshResult("SendGroup", len(wc.SendGroupMap))
}

// GetSendGroupById 根据 ID 获取 SendGroup 数据
func (wc *webhookCache) GetSendGroupById(id int) *model.MonitorSendGroup {
	wc.SendGroupLock.RLock()
	defer wc.SendGroupLock.RUnlock()
	if group, exists := wc.SendGroupMap[id]; exists {
		return group
	}
	return nil
}

// RenewMapUser 刷新 User 缓存
func (wc *webhookCache) RenewMapUser(ctx context.Context) {
	users, err := wc.dao.GetUserList(ctx)
	if err != nil {
		wc.l.Error("[缓存刷新模块] 获取 User 列表失败", zap.Error(err))
		return
	}

	if len(users) == 0 {
		wc.l.Info("[缓存刷新模块] User 列表为空")
		return
	}

	tmpMap := make(map[int]*model.User, len(users))
	for _, user := range users {
		tmpMap[user.ID] = user
	}

	// 加锁并更新缓存
	wc.UserLock.Lock()
	wc.UserMap = tmpMap
	wc.UserLock.Unlock()

	wc.logCacheRefreshResult("User", len(wc.UserMap))
}

// GetUserById 根据 ID 获取 User 数据
func (wc *webhookCache) GetUserById(id int) *model.User {
	wc.UserLock.RLock()
	defer wc.UserLock.RUnlock()
	if user, exists := wc.UserMap[id]; exists {
		return user
	}
	return nil
}

// logCacheRefreshResult 记录缓存刷新结果日志
func (wc *webhookCache) logCacheRefreshResult(cacheName string, newNum int) {
	wc.l.Info("刷新缓存结果",
		zap.String("缓存名称", cacheName),
		zap.Int("当前数量", newNum),
	)
}
