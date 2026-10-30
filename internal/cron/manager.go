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

package cron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/scheduler"
	crontask "github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/hibiken/asynq"
	robcron "github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNoUsers = errors.New("值班组没有成员")
)

const (
	OnDutyBatchSize                        = 100
	K8sMaxConcurrency                      = 5
	OnDutyCheckInterval                    = 10 * time.Second
	DefaultPrometheusConfigRefreshInterval = 15 * time.Second
	K8sCheckInterval                       = 60 * time.Second
	MaxRetries                             = 3
	RetryDelay                             = 5 * time.Second
)

// BuiltinRunner 使用与手动执行同一处理器和数据库执行租约
type BuiltinRunner interface {
	ProcessTask(context.Context, *asynq.Task) error
	IsTaskFailure(error) bool
}
type builtinEntry struct {
	id                robcron.EntryID
	schedule, command string
}

// CronManager 统一的 Cron 管理器接口 - 整合系统内置任务和用户自定义任务
type CronManager interface {
	// 统一管理方法
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// unifiedCronManager 统一的 Cron 管理器实现
type unifiedCronManager struct {
	logger *zap.Logger

	// 系统内置任务依赖 (从原 cron.go 迁移)
	onDutyDao       alert.AlertManagerOnDutyDAO
	k8sDao          dao.ClusterDAO
	k8sClient       client.K8sClient
	promConfigCache cache.MonitorCache
	clusterMgr      manager.ClusterManager

	cronScheduler *scheduler.CronScheduler

	builtinTaskMgr *BuiltinTaskManager
	builtinRunner  BuiltinRunner
	builtinCron    *robcron.Cron
	builtinMu      sync.Mutex
	builtinEntries map[int]builtinEntry
	builtinReady   atomic.Bool

	lifecycleMu sync.Mutex
	cancel      context.CancelFunc
	stopping    chan struct{}
	wg          sync.WaitGroup
}

// NewUnifiedCronManager 创建统一的 Cron 管理器
func NewUnifiedCronManager(
	logger *zap.Logger,
	onDutyDao alert.AlertManagerOnDutyDAO,
	k8sDao dao.ClusterDAO,
	k8sClient client.K8sClient,
	clusterMgr manager.ClusterManager,
	promConfigCache cache.MonitorCache,
	cronScheduler *scheduler.CronScheduler,
	builtinTaskMgr *BuiltinTaskManager,
	builtinRunner BuiltinRunner,
) CronManager {
	return &unifiedCronManager{
		logger:          logger,
		onDutyDao:       onDutyDao,
		k8sDao:          k8sDao,
		k8sClient:       k8sClient,
		clusterMgr:      clusterMgr,
		promConfigCache: promConfigCache,
		cronScheduler:   cronScheduler,
		builtinTaskMgr:  builtinTaskMgr,
		builtinRunner:   builtinRunner,
		builtinCron:     robcron.New(robcron.WithLocation(time.UTC)),
		builtinEntries:  make(map[int]builtinEntry),
	}
}

// === 统一管理方法 ===

func (cm *unifiedCronManager) Start(ctx context.Context) error {
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	if cm.cancel != nil {
		return errors.New("统一Cron管理器已启动或尚未停止")
	}
	cm.logger.Info("启动统一Cron管理器")

	ctx, cancel := context.WithCancel(ctx)
	cm.cancel = cancel
	started := false
	defer func() {
		if !started {
			cancel()
			cm.cancel = nil
		}
	}()

	if err := cm.builtinTaskMgr.InitializeBuiltinTasks(ctx); err != nil {
		cancel()
		return fmt.Errorf("初始化内置任务失败: %w", err)
	}

	// 启动系统内置任务（根据数据库状态）
	if err := cm.StartSystemTasks(ctx); err != nil {
		cm.logger.Error("启动系统内置任务失败", zap.Error(err))
		return err
	}

	if err := cm.StartUserTaskScheduler(ctx); err != nil {
		cm.logger.Error("启动用户任务调度器失败", zap.Error(err))
		return err
	}

	cm.logger.Info("统一Cron管理器启动成功")
	started = true
	return nil
}

// SchedulerReady 同时反映用户任务和内置任务是否完成有效配置加载
func (cm *unifiedCronManager) SchedulerReady() bool {
	if cm.cronScheduler == nil {
		return false
	}
	status := cm.cronScheduler.Status()
	return status.Running && status.Ready && cm.builtinReady.Load()
}
func supportedBuiltin(command string) bool {
	for _, definition := range GetBuiltinTasks() {
		if definition.TaskType == command {
			return true
		}
	}
	return false
}

// StartSystemTasks 注册内置调度并周期对账；数据库暂时失败不覆盖已有条目
func (cm *unifiedCronManager) StartSystemTasks(ctx context.Context) error {
	if cm.builtinCron == nil || cm.builtinRunner == nil || cm.builtinTaskMgr == nil {
		return errors.New("内置任务调度依赖不能为空")
	}
	if err := cm.reconcileBuiltinTasks(ctx); err != nil {
		cm.logger.Error("初次加载内置任务失败", zap.String("error_type", fmt.Sprintf("%T", err)))
	}
	cm.builtinCron.Start()
	// 旧版启动后会立即执行启用的内置任务，保留这一行为
	cm.builtinMu.Lock()
	for id, entry := range cm.builtinEntries {
		id, entry := id, entry
		cm.wg.Add(1)
		go func() { defer cm.wg.Done(); cm.runScheduledBuiltin(ctx, id, entry.schedule, entry.command) }()
	}
	cm.builtinMu.Unlock()
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := cm.reconcileBuiltinTasks(ctx); err != nil {
					cm.logger.Error("重载内置任务失败", zap.String("error_type", fmt.Sprintf("%T", err)))
				}
			}
		}
	}()
	return nil
}

func (cm *unifiedCronManager) reconcileBuiltinTasks(ctx context.Context) error {
	jobs, err := cm.builtinTaskMgr.GetEnabledBuiltinTasks(ctx)
	if err != nil {
		cm.builtinReady.Store(false)
		return err
	}
	cm.builtinMu.Lock()
	defer cm.builtinMu.Unlock()
	if err := ctx.Err(); err != nil {
		cm.builtinReady.Store(false)
		return err
	}
	if cm.builtinEntries == nil {
		cm.builtinEntries = make(map[int]builtinEntry)
	}
	desired := make(map[int]bool, len(jobs))
	var failures []error
	for _, job := range jobs {
		desired[job.ID] = true
		if !supportedBuiltin(job.Command) {
			failures = append(failures, fmt.Errorf("未知内置任务类型 %s", job.Command))
			continue
		}
		if current, ok := cm.builtinEntries[job.ID]; ok && current.schedule == job.Schedule && current.command == job.Command {
			continue
		}
		id, schedule, command := job.ID, job.Schedule, job.Command
		parsed, err := crontask.ParseSchedule(schedule)
		if err != nil {
			failures = append(failures, fmt.Errorf("内置任务 %d 表达式无效: %w", job.ID, err))
			continue
		}
		entry := cm.builtinCron.Schedule(parsed, robcron.FuncJob(func() { cm.runScheduledBuiltin(ctx, id, schedule, command) }))
		if old, ok := cm.builtinEntries[id]; ok {
			cm.builtinCron.Remove(old.id)
		}
		cm.builtinEntries[id] = builtinEntry{id: entry, schedule: schedule, command: command}
	}
	for id, entry := range cm.builtinEntries {
		if !desired[id] {
			cm.builtinCron.Remove(entry.id)
			delete(cm.builtinEntries, id)
		}
	}
	if err := ctx.Err(); err != nil {
		cm.builtinReady.Store(false)
		return err
	}
	cm.builtinReady.Store(len(failures) == 0)
	return errors.Join(failures...)
}
func (cm *unifiedCronManager) runScheduledBuiltin(ctx context.Context, id int, schedule, command string) {
	defer func() {
		if r := recover(); r != nil {
			cm.logger.Error("内置任务异常", zap.Int("job_id", id), zap.String("panic_type", fmt.Sprintf("%T", r)))
		}
	}()
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			return
		}
		job, err := cm.builtinTaskMgr.cronDAO.GetCronJob(ctx, id)
		if err != nil {
			cm.logger.Warn("无法确认内置任务状态", zap.Int("job_id", id))
			return
		}
		if job == nil || job.IsBuiltIn != 1 || job.JobType != model.CronJobTypeSystem || !supportedBuiltin(job.Command) || job.Schedule != schedule || job.Command != command || (job.Status != model.CronJobStatusEnabled && job.Status != model.CronJobStatusRunning) {
			return
		}
		payload, err := json.Marshal(crontask.CronTaskPayload{JobID: id, JobName: job.Name, TaskType: job.JobType, TriggerBy: "builtin_scheduled", Data: map[string]interface{}{"schedule": schedule, "command": command}})
		if err != nil {
			return
		}
		err = cm.builtinRunner.ProcessTask(ctx, asynq.NewTask("cron:task", payload))
		if err == nil || !cm.builtinRunner.IsTaskFailure(err) || attempt >= job.MaxRetry {
			return
		}
		timer := time.NewTimer(RetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (cm *unifiedCronManager) StartUserTaskScheduler(ctx context.Context) error {
	cm.logger.Info("启动用户自定义任务调度器")

	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		if err := cm.cronScheduler.StartScheduler(ctx); err != nil {
			cm.logger.Error("用户任务调度器异常退出", zap.Error(err))
		}
	}()

	return nil
}

func (cm *unifiedCronManager) Stop(ctx context.Context) error {
	cm.logger.Info("停止统一Cron管理器")

	cm.lifecycleMu.Lock()
	if cm.cancel == nil {
		cm.lifecycleMu.Unlock()
		return nil
	}
	if cm.stopping == nil {
		cm.cancel()
		cm.builtinReady.Store(false)
		var stopped context.Context
		if cm.builtinCron != nil {
			stopped = cm.builtinCron.Stop()
		}
		done := make(chan struct{})
		cm.stopping = done
		go func() {
			if stopped != nil {
				<-stopped.Done()
			}
			cm.wg.Wait()
			cm.lifecycleMu.Lock()
			cm.cancel = nil
			cm.stopping = nil
			cm.builtinReady.Store(false)
			close(done)
			cm.lifecycleMu.Unlock()
		}()
	}
	done := cm.stopping
	cm.lifecycleMu.Unlock()

	// 等待停止或超时
	select {
	case <-done:
		cm.logger.Info("统一Cron管理器已停止")
		return nil
	case <-ctx.Done():
		cm.logger.Warn("统一Cron管理器停止超时")
		return ctx.Err()
	}
}

// === 以下是从原 cron.go 迁移的辅助方法实现 ===

// ExecuteBuiltinOnce 是定时和手动触发共用的单次真实业务执行入口
func ExecuteBuiltinOnce(ctx context.Context, command string, logger *zap.Logger, onDutyDao alert.AlertManagerOnDutyDAO, k8sDao dao.ClusterDAO, clusterMgr manager.ClusterManager, promConfigCache cache.MonitorCache) (string, error) {
	if ctx == nil || logger == nil {
		return "", errors.New("内置任务缺少上下文或日志")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cm := &unifiedCronManager{logger: logger, onDutyDao: onDutyDao, k8sDao: k8sDao, clusterMgr: clusterMgr, promConfigCache: promConfigCache}
	switch command {
	case "on_duty_history":
		if onDutyDao == nil {
			return "", errors.New("值班任务依赖不可用")
		}
		if err := cm.fillOnDutyHistoryWithRetry(ctx); err != nil {
			return "", err
		}
		return "值班历史处理完成", nil
	case "k8s_status_check":
		if k8sDao == nil || clusterMgr == nil {
			return "", errors.New("集群检查依赖不可用")
		}
		if err := cm.checkK8sStatusWithRetry(ctx); err != nil {
			return "", err
		}
		return "集群状态检查完成", nil
	case "prometheus_config_refresh":
		if promConfigCache == nil {
			return "", errors.New("监控配置刷新依赖不可用")
		}
		if err := promConfigCache.MonitorCacheManager(ctx); err != nil {
			return "", fmt.Errorf("监控配置刷新失败: %w", err)
		}
		return "监控配置刷新完成", nil
	default:
		return "", fmt.Errorf("未知的系统任务类型: %s", command)
	}
}

// fillOnDutyHistoryWithRetry 带重试机制的值班历史记录填充
func (cm *unifiedCronManager) fillOnDutyHistoryWithRetry(ctx context.Context) error {
	allGroups, err := cm.fetchAllEnabledGroups(ctx)
	if err != nil {
		return fmt.Errorf("获取启用的值班组失败: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(allGroups) == 0 {
		cm.logger.Debug("没有找到需要处理的值班组")
		return nil
	}

	return cm.processGroupsInParallel(ctx, allGroups)
}

func (cm *unifiedCronManager) fetchAllEnabledGroups(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var allGroups []*model.MonitorOnDutyGroup
	page := 1
	enable := int8(1)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		groups, total, err := cm.onDutyDao.GetMonitorOnDutyList(ctx, &model.GetMonitorOnDutyGroupListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: OnDutyBatchSize,
			},
			Enable: &enable,
		})
		if err != nil {
			return nil, fmt.Errorf("获取值班组失败 page=%d: %w", page, err)
		}

		validGroups := cm.filterValidGroups(groups)
		allGroups = append(allGroups, validGroups...)

		if int64(len(allGroups)) >= total || len(groups) == 0 {
			break
		}
		page++
	}

	return allGroups, nil
}

// filterValidGroups 过滤有效的值班组
func (cm *unifiedCronManager) filterValidGroups(groups []*model.MonitorOnDutyGroup) []*model.MonitorOnDutyGroup {
	var validGroups []*model.MonitorOnDutyGroup
	for _, group := range groups {
		if group.Enable == 2 {
			cm.logger.Debug("跳过未启用的值班组", zap.String("group", group.Name))
			continue
		}
		if len(group.Users) == 0 {
			cm.logger.Warn("跳过无成员的值班组", zap.String("group", group.Name), zap.Int("id", group.ID))
			continue
		}
		validGroups = append(validGroups, group)
	}
	return validGroups
}

// 并发有界，取消后仍等待已启动的任务退出，任何组失败均不伪装成整体成功
func (cm *unifiedCronManager) processGroupsInParallel(ctx context.Context, groups []*model.MonitorOnDutyGroup) error {
	errChan := make(chan error, len(groups)+1)
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
loop:
	for _, group := range groups {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			errChan <- ctx.Err()
			break loop
		}
		wg.Add(1)
		go func(g *model.MonitorOnDutyGroup) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("处理值班组异常: %T", r)
				}
			}()
			if err := cm.processOnDutyHistoryForGroup(ctx, g); err != nil {
				errChan <- fmt.Errorf("处理值班组 %d 失败: %w", g.ID, err)
			}
		}(group)
	}
	wg.Wait()
	close(errChan)
	var failures []error
	for err := range errChan {
		failures = append(failures, err)
	}
	if err := ctx.Err(); err != nil {
		failures = append(failures, err)
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	return nil
}

func (cm *unifiedCronManager) processOnDutyHistoryForGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	if len(group.Users) == 0 {
		return ErrNoUsers
	}

	todayStr := time.Now().Format("2006-01-02")

	// 检查今天是否已经有值班历史记录
	exists, err := cm.onDutyDao.ExistsMonitorOnDutyHistory(ctx, group.ID, todayStr)
	if err != nil {
		cm.logger.Error("检查值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	changes, _, err := cm.onDutyDao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, group.ID, todayStr, todayStr)
	if err != nil {
		cm.logger.Error("获取换班记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	if len(changes) > 0 {
		cm.logger.Info("发现换班记录，优先处理", zap.String("group", group.Name), zap.Int("groupID", group.ID))
		latestChange := changes[len(changes)-1]

		if exists {
			history, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, todayStr)
			if err != nil {
				cm.logger.Error("获取今日值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
				return err
			}
			history.OnDutyUserID = latestChange.OnDutyUserID
			history.OriginUserID = latestChange.OriginUserID
			if err := cm.onDutyDao.CreateMonitorOnDutyHistory(ctx, history); err != nil {
				cm.logger.Error("更新值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
				return err
			}
			cm.logger.Info("成功更新今日值班历史记录（换班）",
				zap.String("group", group.Name),
				zap.Int("groupID", group.ID),
				zap.String("date", todayStr),
				zap.Int("originUserID", latestChange.OriginUserID),
				zap.Int("onDutyUserID", latestChange.OnDutyUserID))
			return nil
		}

		history := &model.MonitorOnDutyHistory{
			OnDutyGroupID: group.ID,
			DateString:    todayStr,
			OnDutyUserID:  latestChange.OnDutyUserID,
			OriginUserID:  latestChange.OriginUserID,
		}
		if err := cm.onDutyDao.CreateMonitorOnDutyHistory(ctx, history); err != nil {
			cm.logger.Error("创建值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
			return err
		}
		cm.logger.Info("成功创建今日值班历史记录（换班）",
			zap.String("group", group.Name),
			zap.Int("groupID", group.ID),
			zap.String("date", todayStr),
			zap.Int("fromUserID", latestChange.OriginUserID),
			zap.Int("toUserID", latestChange.OnDutyUserID))
		return nil
	}

	// 如果今天已经有值班历史记录且没有换班记录，则跳过
	if exists {
		cm.logger.Debug("今日值班记录已存在，跳过", zap.String("group", group.Name), zap.Int("groupID", group.ID), zap.String("date", todayStr))
		return nil
	}

	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	yesterdayHistory, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, yesterdayStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		cm.logger.Error("获取昨天的值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	var onDutyUserID, originUserID int
	if yesterdayHistory == nil {
		onDutyUserID = group.Users[0].ID
		originUserID = group.Users[0].ID
		cm.logger.Debug("未找到昨日值班记录，使用第一位成员",
			zap.String("group", group.Name),
			zap.Int("groupID", group.ID),
			zap.Int("userID", onDutyUserID))
	} else {
		userStillExists := false
		for _, user := range group.Users {
			if user.ID == yesterdayHistory.OnDutyUserID {
				userStillExists = true
				break
			}
		}
		if !userStillExists {
			onDutyUserID = group.Users[0].ID
			originUserID = group.Users[0].ID
			cm.logger.Warn("昨日值班用户已不在值班组中，使用第一位成员",
				zap.String("group", group.Name),
				zap.Int("groupID", group.ID),
				zap.Int("oldUserID", yesterdayHistory.OnDutyUserID),
				zap.Int("newUserID", onDutyUserID))
		} else {
			shiftNeeded, err := cm.isShiftNeeded(ctx, group, yesterdayHistory)
			if err != nil {
				cm.logger.Error("判断是否需要轮换值班人失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
				return err
			}
			if shiftNeeded {
				nextUserIndex := (cm.getMemberIndex(group, yesterdayHistory.OnDutyUserID) + 1) % len(group.Users)
				originUserID = yesterdayHistory.OnDutyUserID
				onDutyUserID = group.Users[nextUserIndex].ID
				cm.logger.Debug("轮换值班人",
					zap.String("group", group.Name),
					zap.Int("groupID", group.ID),
					zap.Int("oldUserID", yesterdayHistory.OnDutyUserID),
					zap.Int("newUserID", onDutyUserID))
			} else {
				onDutyUserID = yesterdayHistory.OnDutyUserID
				originUserID = yesterdayHistory.OriginUserID
				if originUserID == 0 {
					originUserID = onDutyUserID
				}
				cm.logger.Debug("继续使用昨日值班人",
					zap.String("group", group.Name),
					zap.Int("groupID", group.ID),
					zap.Int("userID", onDutyUserID))
			}
		}
	}

	history := &model.MonitorOnDutyHistory{
		OnDutyGroupID: group.ID,
		DateString:    todayStr,
		OnDutyUserID:  onDutyUserID,
		OriginUserID:  originUserID,
	}
	if err := cm.onDutyDao.CreateMonitorOnDutyHistory(ctx, history); err != nil {
		cm.logger.Error("创建值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	cm.logger.Info("成功创建值班历史记录",
		zap.String("group", group.Name),
		zap.Int("groupID", group.ID),
		zap.String("date", todayStr),
		zap.Int("userID", onDutyUserID),
		zap.Int("originUserID", originUserID))
	return nil
}

// isShiftNeeded 判断是否需要轮换值班人
func (cm *unifiedCronManager) isShiftNeeded(ctx context.Context, group *model.MonitorOnDutyGroup, lastHistory *model.MonitorOnDutyHistory) (bool, error) {
	if group == nil || lastHistory == nil {
		return false, errors.New("group or lastHistory cannot be nil")
	}
	if group.ShiftDays <= 0 {
		return false, fmt.Errorf("invalid ShiftDays value: %d", group.ShiftDays)
	}

	startDate := time.Now().AddDate(0, 0, -group.ShiftDays).Format("2006-01-02")
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	histories, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx, group.ID, startDate, yesterdayStr)
	if err != nil {
		return false, fmt.Errorf("获取历史记录失败: %w", err)
	}

	consecutiveDays := 0
	for _, history := range histories {
		if history.OnDutyUserID == lastHistory.OnDutyUserID {
			consecutiveDays++
		}
	}

	cm.logger.Debug("检查是否需要轮换值班人",
		zap.String("group", group.Name),
		zap.Int("groupID", group.ID),
		zap.Int("userID", lastHistory.OnDutyUserID),
		zap.Int("consecutiveDays", consecutiveDays),
		zap.Int("shiftDays", group.ShiftDays))

	return consecutiveDays >= group.ShiftDays, nil
}

func (cm *unifiedCronManager) getMemberIndex(group *model.MonitorOnDutyGroup, userID int) int {
	if group == nil || len(group.Users) == 0 {
		return 0
	}

	for index, member := range group.Users {
		if member.ID == userID {
			return index
		}
	}

	cm.logger.Warn("在值班组中未找到指定用户，将使用第一位成员",
		zap.String("group", group.Name),
		zap.Int("groupID", group.ID),
		zap.Int("userID", userID))
	return 0
}

// checkK8sStatusWithRetry 带重试机制的k8s状态检查
func (cm *unifiedCronManager) checkK8sStatusWithRetry(ctx context.Context) error {
	const pageSize = 100
	var clusters []*model.K8sCluster
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		batch, total, err := cm.k8sDao.GetClusterList(ctx, &model.ListClustersReq{ListReq: model.ListReq{Page: page, Size: pageSize}})
		if err != nil {
			return fmt.Errorf("获取集群列表失败: %w", err)
		}
		clusters = append(clusters, batch...)
		if len(batch) == 0 || len(batch) < pageSize || int64(len(clusters)) >= total {
			break
		}
	}
	errChan := make(chan error, len(clusters)+1)
	sem := make(chan struct{}, K8sMaxConcurrency)
	var wg sync.WaitGroup
loop:
	for _, cluster := range clusters {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			errChan <- ctx.Err()
			break loop
		}
		wg.Add(1)
		go func(cluster *model.K8sCluster) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("集群状态检查异常: %T", r)
				}
			}()
			if err := cm.checkClusterStatus(ctx, cluster); err != nil {
				errChan <- err
			}
		}(cluster)
	}
	wg.Wait()
	close(errChan)
	var failures []error
	for err := range errChan {
		failures = append(failures, err)
	}
	if err := ctx.Err(); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

func (cm *unifiedCronManager) checkClusterStatus(ctx context.Context, cluster *model.K8sCluster) error {
	startTime := time.Now()
	var newStatus model.ClusterStatus

	checkErr := cm.clusterMgr.CheckClusterStatus(ctx, cluster.ID)
	if err := checkErr; err != nil {
		// 分类处理不同类型的连接错误
		if strings.Contains(err.Error(), "connection refused") {
			cm.logger.Debug("集群连接被拒绝（可能是正常的离线状态）",
				zap.String("cluster", cluster.Name),
				zap.String("endpoint", "检测到连接被拒绝"))
		} else if strings.Contains(err.Error(), "timeout") {
			cm.logger.Warn("集群连接超时",
				zap.Error(err),
				zap.String("cluster", cluster.Name))
		} else {
			cm.logger.Warn("集群连接检查失败",
				zap.Error(err),
				zap.String("cluster", cluster.Name))
		}
		newStatus = model.StatusError
	} else {
		newStatus = model.StatusRunning
		cm.logger.Debug("集群状态检查成功",
			zap.String("cluster", cluster.Name),
			zap.Duration("duration", time.Since(startTime)))
	}

	// 只有状态发生变化时才更新数据库，减少不必要的写操作
	if cluster.Status != newStatus {
		if err := cm.k8sDao.UpdateClusterStatus(ctx, cluster.ID, newStatus); err != nil {
			cm.logger.Error("更新集群状态失败",
				zap.Error(err),
				zap.String("cluster", cluster.Name),
				zap.Int8("oldStatus", int8(cluster.Status)),
				zap.Int8("newStatus", int8(newStatus)))
			return fmt.Errorf("更新集群[%s]状态失败: %w", cluster.Name, err)
		}
		cm.logger.Info("集群状态已更新",
			zap.String("cluster", cluster.Name),
			zap.Int8("oldStatus", int8(cluster.Status)),
			zap.Int8("newStatus", int8(newStatus)))
	}

	if checkErr != nil {
		return fmt.Errorf("集群 %d 状态检查失败", cluster.ID)
	}
	return nil
}
