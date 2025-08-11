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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	ErrNoUsers = errors.New("值班组没有成员")
)

const (
	OnDutyBatchSize     = 100
	HostCheckBatchSize  = 100
	K8sMaxConcurrency   = 5
	OnDutyCheckInterval = 10 * time.Second
	// Prometheus 刷新默认间隔，支持通过配置覆盖
	DefaultPrometheusConfigRefreshInterval = 15 * time.Second
	HostCheckInterval                      = 30 * time.Second
	K8sCheckInterval                       = 60 * time.Second
	MaxRetries                             = 3
	RetryDelay                             = 5 * time.Second
)

type CronManager interface {
	StartOnDutyHistoryManager(ctx context.Context) error
	// StartCheckHostStatusManager(ctx context.Context) error
	StartCheckK8sStatusManager(ctx context.Context) error
	StartPrometheusConfigRefreshManager(ctx context.Context) error
}

type cronManager struct {
	logger          *zap.Logger
	onDutyDao       alert.AlertManagerOnDutyDAO
	k8sDao          admin.ClusterDAO
	k8sClient       client.K8sClient
	promConfigCache cache.MonitorCache
	clusterMgr      manager.ClusterManager
}

func NewCronManager(logger *zap.Logger, onDutyDao alert.AlertManagerOnDutyDAO, k8sDao admin.ClusterDAO, k8sClient client.K8sClient, clusterMgr manager.ClusterManager, promConfigCache cache.MonitorCache) CronManager {
	return &cronManager{
		logger:          logger,
		onDutyDao:       onDutyDao,
		k8sDao:          k8sDao,
		k8sClient:       k8sClient,
		clusterMgr:      clusterMgr,
		promConfigCache: promConfigCache,
	}
}

// StartOnDutyHistoryManager 启动值班历史记录填充任务
func (cm *cronManager) StartOnDutyHistoryManager(ctx context.Context) error {
	cm.logger.Info("启动值班历史记录填充任务")

	// 使用 wait.UntilWithContext 确保周期性执行，并添加 panic 恢复
	go func() {
		defer func() {
			if r := recover(); r != nil {
				cm.logger.Error("值班历史记录填充任务发生 panic，正在重启", zap.Any("panic", r))
				// 重启任务
				time.Sleep(RetryDelay)
				go cm.StartOnDutyHistoryManager(ctx)
			}
		}()

		wait.UntilWithContext(ctx, func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					cm.logger.Error("值班历史记录填充任务执行时发生 panic", zap.Any("panic", r))
				}
			}()

			// 添加重试机制
			var lastErr error
			for attempt := 1; attempt <= MaxRetries; attempt++ {
				if err := cm.fillOnDutyHistoryWithRetry(ctx); err != nil {
					lastErr = err
					if attempt < MaxRetries {
						cm.logger.Warn("值班历史记录填充任务执行失败，准备重试",
							zap.Int("attempt", attempt),
							zap.Int("maxRetries", MaxRetries),
							zap.Error(err))
						time.Sleep(RetryDelay)
						continue
					}
				} else {
					if attempt > 1 {
						cm.logger.Info("值班历史记录填充任务重试成功",
							zap.Int("attempt", attempt))
					}
					return
				}
			}

			if lastErr != nil {
				cm.logger.Error("值班历史记录填充任务重试失败，已达到最大重试次数",
					zap.Int("maxRetries", MaxRetries),
					zap.Error(lastErr))
			}
		}, OnDutyCheckInterval)
	}()

	<-ctx.Done()
	cm.logger.Info("值班历史记录填充任务已停止")
	return nil
}

// fillOnDutyHistory 填充所有值班组的历史记录
func (cm *cronManager) fillOnDutyHistory(ctx context.Context) {
	allGroups, err := cm.fetchAllEnabledGroups(ctx)
	if err != nil {
		cm.logger.Error("获取启用的值班组失败", zap.Error(err))
		return
	}

	if len(allGroups) == 0 {
		cm.logger.Debug("没有找到需要处理的值班组")
		return
	}

	cm.processGroupsInParallel(ctx, allGroups)
}

// fillOnDutyHistoryWithRetry 带重试机制的值班历史记录填充
func (cm *cronManager) fillOnDutyHistoryWithRetry(ctx context.Context) error {
	allGroups, err := cm.fetchAllEnabledGroups(ctx)
	if err != nil {
		return fmt.Errorf("获取启用的值班组失败: %w", err)
	}

	if len(allGroups) == 0 {
		cm.logger.Debug("没有找到需要处理的值班组")
		return nil
	}

	cm.processGroupsInParallel(ctx, allGroups)
	return nil
}

// fetchAllEnabledGroups 获取所有启用的值班组
func (cm *cronManager) fetchAllEnabledGroups(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
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
func (cm *cronManager) filterValidGroups(groups []*model.MonitorOnDutyGroup) []*model.MonitorOnDutyGroup {
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

// processGroupsInParallel 并行处理值班组
func (cm *cronManager) processGroupsInParallel(ctx context.Context, groups []*model.MonitorOnDutyGroup) {
	errChan := make(chan error, len(groups))
	var wg sync.WaitGroup

	for _, group := range groups {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wg.Add(1)
		go func(g *model.MonitorOnDutyGroup) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					cm.logger.Error("处理值班组时发生 panic", zap.Any("panic", r), zap.String("group", g.Name))
					errChan <- fmt.Errorf("处理值班组 %s 时发生 panic: %v", g.Name, r)
				}
			}()

			if err := cm.processOnDutyHistoryForGroup(ctx, g); err != nil {
				errChan <- fmt.Errorf("处理值班组 %s(ID:%d) 失败: %w", g.Name, g.ID, err)
			}
		}(group)
	}

	wg.Wait()
	close(errChan)

	cm.logProcessResults(errChan, len(groups))
}

// logProcessResults 记录处理结果
func (cm *cronManager) logProcessResults(errChan <-chan error, totalGroups int) {
	errCount := 0
	for err := range errChan {
		errCount++
		cm.logger.Error("处理值班历史记录时发生错误", zap.Error(err))
	}

	if errCount > 0 {
		cm.logger.Warn("值班历史记录填充任务完成，但有错误",
			zap.Int("errorCount", errCount),
			zap.Int("totalGroups", totalGroups))
	} else {
		cm.logger.Info("值班历史记录填充任务成功完成", zap.Int("totalGroups", totalGroups))
	}
}

// processOnDutyHistoryForGroup 填充单个值班组的历史记录
func (cm *cronManager) processOnDutyHistoryForGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
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

	// 检查今天是否有换班记录
	changes, _, err := cm.onDutyDao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, group.ID, todayStr, todayStr)
	if err != nil {
		cm.logger.Error("获取换班记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	// 优先处理换班记录
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
func (cm *cronManager) isShiftNeeded(ctx context.Context, group *model.MonitorOnDutyGroup, lastHistory *model.MonitorOnDutyHistory) (bool, error) {
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

// getMemberIndex 获取成员在值班组中的索引
func (cm *cronManager) getMemberIndex(group *model.MonitorOnDutyGroup, userID int) int {
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

// // StartCheckHostStatusManager 定期检查ecs主机状态
// func (cm *cronManager) StartCheckHostStatusManager(ctx context.Context) error {
// 	cm.logger.Info("启动主机状态检查任务")

// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				cm.logger.Error("主机状态检查任务发生 panic，正在重启", zap.Any("panic", r))
// 				// 重启任务
// 				time.Sleep(RetryDelay)
// 				go cm.StartCheckHostStatusManager(ctx)
// 			}
// 		}()

// 		wait.UntilWithContext(ctx, func(ctx context.Context) {
// 			defer func() {
// 				if r := recover(); r != nil {
// 					cm.logger.Error("主机状态检查任务执行时发生 panic", zap.Any("panic", r))
// 				}
// 			}()

// 			// 添加重试机制
// 			var lastErr error
// 			for attempt := 1; attempt <= MaxRetries; attempt++ {
// 				if err := cm.checkHostStatusWithRetry(ctx); err != nil {
// 					lastErr = err
// 					if attempt < MaxRetries {
// 						cm.logger.Warn("主机状态检查任务执行失败，准备重试",
// 							zap.Int("attempt", attempt),
// 							zap.Int("maxRetries", MaxRetries),
// 							zap.Error(err))
// 						time.Sleep(RetryDelay)
// 						continue
// 					}
// 				} else {
// 					if attempt > 1 {
// 						cm.logger.Info("主机状态检查任务重试成功",
// 							zap.Int("attempt", attempt))
// 					}
// 					return
// 				}
// 			}

// 			if lastErr != nil {
// 				cm.logger.Error("主机状态检查任务重试失败，已达到最大重试次数",
// 					zap.Int("maxRetries", MaxRetries),
// 					zap.Error(lastErr))
// 			}
// 		}, HostCheckInterval)
// 	}()

// 	<-ctx.Done()
// 	cm.logger.Info("主机状态检查任务已停止")
// 	return nil
// }

// // checkHostStatusWithRetry 带重试机制的主机状态检查
// func (cm *cronManager) checkHostStatusWithRetry(ctx context.Context) error {
// 	cm.logger.Info("开始检查ecs主机状态")

// 	const batchSize = HostCheckBatchSize
// 	offset := 0

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			cm.logger.Info("主机状态检查任务被取消", zap.Int("processed", offset))
// 			return ctx.Err()
// 		default:
// 		}

// 		ecss, _, err := cm.ecsDao.ListEcsResources(ctx, &model.ListEcsResourcesReq{
// 			ListReq: model.ListReq{
// 				Page: offset/batchSize + 1,
// 				Size: batchSize,
// 			},
// 		})
// 		if err != nil {
// 			cm.logger.Error("获取ecs主机失败", zap.Error(err), zap.Int("offset", offset))
// 			return fmt.Errorf("获取ecs主机失败: %w", err)
// 		}

// 		if len(ecss) == 0 {
// 			break
// 		}

// 		var wg sync.WaitGroup
// 		errChan := make(chan error, len(ecss))

// 		for _, ecs := range ecss {
// 			select {
// 			case <-ctx.Done():
// 				return ctx.Err()
// 			default:
// 			}

// 			wg.Add(1)
// 			go func(ecs *model.ResourceEcs) {
// 				defer wg.Done()
// 				defer func() {
// 					if r := recover(); r != nil {
// 						cm.logger.Error("检查主机状态时发生 panic",
// 							zap.Any("panic", r),
// 							zap.String("hostname", ecs.HostName))
// 						errChan <- fmt.Errorf("检查主机状态时发生 panic: %v", r)
// 					}
// 				}()

// 				if ecs.IpAddr == "" {
// 					cm.logger.Warn("目标ecs没有绑定公网ip",
// 						zap.String("hostname", ecs.HostName),
// 						zap.Int("id", ecs.ID))
// 					return
// 				}

// 				status := "RUNNING"
// 				if !utils.Ping(ecs.IpAddr) {
// 					cm.logger.Debug("ping请求失败",
// 						zap.String("ip", ecs.IpAddr),
// 						zap.String("hostname", ecs.HostName))
// 					status = "ERROR"
// 				}

// 				if err := cm.ecsDao.UpdateEcsStatus(ctx, strconv.Itoa(ecs.ID), status); err != nil {
// 					cm.logger.Error("更新主机状态失败",
// 						zap.Error(err),
// 						zap.String("hostname", ecs.HostName),
// 						zap.String("status", status))
// 					errChan <- err
// 				}
// 			}(ecs)
// 		}

// 		wg.Wait()
// 		close(errChan)

// 		for err := range errChan {
// 			cm.logger.Error("处理主机状态时发生错误", zap.Error(err), zap.Int("batch_offset", offset))
// 		}

// 		offset += len(ecss)

// 		if len(ecss) < batchSize {
// 			break
// 		}
// 	}

// 	cm.logger.Info("完成ecs主机状态检查", zap.Int("total_processed", offset))
// 	return nil
// }

// StartCheckK8sStatusManager 启动k8s状态检查任务
func (cm *cronManager) StartCheckK8sStatusManager(ctx context.Context) error {
	cm.logger.Info("启动k8s状态检查任务")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				cm.logger.Error("k8s状态检查任务发生 panic，正在重启", zap.Any("panic", r))
				// 重启任务
				time.Sleep(RetryDelay)
				go cm.StartCheckK8sStatusManager(ctx)
			}
		}()

		wait.UntilWithContext(ctx, func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					cm.logger.Error("k8s状态检查任务执行时发生 panic", zap.Any("panic", r))
				}
			}()

			// 添加重试机制
			var lastErr error
			for attempt := 1; attempt <= MaxRetries; attempt++ {
				if err := cm.checkK8sStatusWithRetry(ctx); err != nil {
					lastErr = err
					if attempt < MaxRetries {
						cm.logger.Warn("k8s状态检查任务执行失败，准备重试",
							zap.Int("attempt", attempt),
							zap.Int("maxRetries", MaxRetries),
							zap.Error(err))
						time.Sleep(RetryDelay)
						continue
					}
				} else {
					if attempt > 1 {
						cm.logger.Info("k8s状态检查任务重试成功",
							zap.Int("attempt", attempt))
					}
					return
				}
			}

			if lastErr != nil {
				cm.logger.Error("k8s状态检查任务重试失败，已达到最大重试次数",
					zap.Int("maxRetries", MaxRetries),
					zap.Error(lastErr))
			}
		}, K8sCheckInterval)
	}()

	<-ctx.Done()
	cm.logger.Info("k8s状态检查任务已停止")
	return nil
}

// checkK8sStatusWithRetry 带重试机制的k8s状态检查
func (cm *cronManager) checkK8sStatusWithRetry(ctx context.Context) error {
	cm.logger.Info("开始检查k8s状态")

	clusters, err := cm.k8sDao.ListAllClusters(ctx)
	if err != nil {
		cm.logger.Error("获取k8s集群列表失败", zap.Error(err))
		return fmt.Errorf("获取k8s集群列表失败: %w", err)
	}

	if len(clusters) == 0 {
		cm.logger.Debug("没有找到k8s集群")
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(clusters))
	semaphore := make(chan struct{}, K8sMaxConcurrency)

	for _, cluster := range clusters {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(cluster *model.K8sCluster) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					cm.logger.Error("检查集群状态时发生 panic",
						zap.Any("panic", r),
						zap.String("cluster", cluster.Name))
					errChan <- fmt.Errorf("检查集群状态时发生 panic: %v", r)
				}
			}()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := cm.checkClusterStatus(ctx, cluster); err != nil {
				errChan <- err
			}
		}(cluster)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		cm.logger.Error("k8s集群状态检查失败", zap.Errors("errors", errs))
		return fmt.Errorf("k8s集群状态检查失败: %v", errs)
	} else {
		cm.logger.Info("完成k8s集群状态检查")
	}
	return nil
}

// checkClusterStatus 检查单个集群状态
func (cm *cronManager) checkClusterStatus(ctx context.Context, cluster *model.K8sCluster) error {
	if err := cm.clusterMgr.CheckClusterStatus(ctx, cluster.ID); err != nil {
		cm.logger.Warn("集群连接检查失败",
			zap.Error(err),
			zap.String("cluster", cluster.Name))
		cluster.Status = "ERROR"
	} else {
		cluster.Status = "RUNNING"
	}

	if err := cm.k8sDao.UpdateClusterStatus(ctx, cluster.ID, cluster.Status); err != nil {
		cm.logger.Error("更新集群状态失败",
			zap.Error(err),
			zap.String("cluster", cluster.Name),
			zap.String("status", cluster.Status))
		return fmt.Errorf("更新集群[%s]状态失败: %w", cluster.Name, err)
	}

	return nil
}

// StartPrometheusConfigRefreshManager 启动Prometheus配置刷新任务
func (cm *cronManager) StartPrometheusConfigRefreshManager(ctx context.Context) error {
	cm.logger.Info("启动Prometheus配置刷新任务")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				cm.logger.Error("Prometheus配置刷新任务发生 panic，正在重启", zap.Any("panic", r))
				// 重启任务
				time.Sleep(RetryDelay)
				go cm.StartPrometheusConfigRefreshManager(ctx)
			}
		}()

		wait.UntilWithContext(ctx, func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					cm.logger.Error("Prometheus配置刷新任务执行时发生 panic", zap.Any("panic", r))
				}
			}()

			// 添加重试机制
			var lastErr error
			for attempt := 1; attempt <= MaxRetries; attempt++ {
				if err := cm.promConfigCache.MonitorCacheManager(ctx); err != nil {
					lastErr = err
					if attempt < MaxRetries {
						cm.logger.Warn("Prometheus配置定时刷新失败，准备重试",
							zap.Int("attempt", attempt),
							zap.Int("maxRetries", MaxRetries),
							zap.Error(err))
						time.Sleep(RetryDelay)
						continue
					}
				} else {
					if attempt > 1 {
						cm.logger.Info("Prometheus配置定时刷新重试成功",
							zap.Int("attempt", attempt))
					} else {
						cm.logger.Info("Prometheus配置定时刷新成功")
					}
					return
				}
			}

			if lastErr != nil {
				cm.logger.Error("Prometheus配置定时刷新重试失败，已达到最大重试次数",
					zap.Int("maxRetries", MaxRetries),
					zap.Error(lastErr))
			}
		}, cm.getPrometheusRefreshInterval())
	}()

	<-ctx.Done()
	cm.logger.Info("Prometheus配置定时刷新任务已停止")
	return nil
}

// getPrometheusRefreshInterval 从配置读取刷新间隔，支持两种格式：
// 1) "@every 15s"（与常见的 cron 语法一致，仅支持 @every 前缀）
// 2) "15s"（直接 time.ParseDuration 支持的时长表示）
// 解析失败时回退到 DefaultPrometheusConfigRefreshInterval。
func (cm *cronManager) getPrometheusRefreshInterval() time.Duration {
	spec := strings.TrimSpace(viper.GetString("prometheus.refresh_cron"))
	if spec == "" {
		return DefaultPrometheusConfigRefreshInterval
	}

	// 支持 @every 前缀
	if strings.HasPrefix(spec, "@every") {
		durStr := strings.TrimSpace(strings.TrimPrefix(spec, "@every"))
		if d, err := time.ParseDuration(durStr); err == nil && d > 0 {
			cm.logger.Info("使用配置的 Prometheus 刷新间隔(@every)", zap.String("spec", spec), zap.Duration("interval", d))
			return d
		}
		cm.logger.Warn("解析 @every 刷新间隔失败，使用默认值",
			zap.String("spec", spec),
			zap.Duration("default", DefaultPrometheusConfigRefreshInterval))
		return DefaultPrometheusConfigRefreshInterval
	}

	// 尝试直接解析 duration
	if d, err := time.ParseDuration(spec); err == nil && d > 0 {
		cm.logger.Info("使用配置的 Prometheus 刷新间隔(duration)", zap.String("spec", spec), zap.Duration("interval", d))
		return d
	}

	cm.logger.Warn("刷新间隔配置无法解析，使用默认值",
		zap.String("spec", spec),
		zap.Duration("default", DefaultPrometheusConfigRefreshInterval))
	return DefaultPrometheusConfigRefreshInterval
}
