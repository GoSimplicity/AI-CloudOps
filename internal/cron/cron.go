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
	"strconv"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	ErrNoUsers = errors.New("值班组没有成员")
)

type CronManager interface {
	StartOnDutyHistoryManager(ctx context.Context) error
	StartCheckHostStatusManager(ctx context.Context) error
	StartCheckK8sStatusManager(ctx context.Context) error
}

type cronManager struct {
	logger     *zap.Logger
	onDutyDao  alert.AlertManagerOnDutyDAO
	k8sDao     admin.ClusterDAO
	k8sClient  client.K8sClient
	clusterMgr manager.ClusterManager
	ecsDao     dao.TreeEcsDAO
}

func NewCronManager(logger *zap.Logger, onDutyDao alert.AlertManagerOnDutyDAO, k8sDao admin.ClusterDAO, k8sClient client.K8sClient, clusterMgr manager.ClusterManager, ecsDao dao.TreeEcsDAO) CronManager {
	return &cronManager{
		logger:     logger,
		onDutyDao:  onDutyDao,
		k8sDao:     k8sDao,
		k8sClient:  k8sClient,
		clusterMgr: clusterMgr,
		ecsDao:     ecsDao,
	}
}

// StartOnDutyHistoryManager 启动值班历史记录填充任务
func (cm *cronManager) StartOnDutyHistoryManager(ctx context.Context) error {
	// 每隔 5 分钟执行一次 fillOnDutyHistory，直到 ctx.Done
	go wait.UntilWithContext(ctx, cm.fillOnDutyHistory, 10*time.Second)
	<-ctx.Done()
	cm.logger.Info("值班历史记录填充任务已停止")
	return nil
}

// fillOnDutyHistory 填充所有值班组的历史记录
func (cm *cronManager) fillOnDutyHistory(ctx context.Context) {
	const batchSize = 100
	page := 1
	enable := int8(1)
	var allGroups []*model.MonitorOnDutyGroup

	// 分批获取所有值班组
	for {
		groups, total, err := cm.onDutyDao.GetMonitorOnDutyList(ctx, &model.GetMonitorOnDutyGroupListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: batchSize,
			},
			Enable: &enable, // 只获取启用的值班组
		})
		if err != nil {
			cm.logger.Error("获取值班组失败", zap.Error(err), zap.Int("page", page))
			return
		}

		allGroups = append(allGroups, groups...)

		// 如果已经获取了所有数据，则退出循环
		if int64(len(allGroups)) >= total || len(groups) == 0 {
			break
		}

		page++
	}

	if len(allGroups) == 0 {
		cm.logger.Info("没有找到需要处理的值班组")
		return
	}

	errChan := make(chan error, len(allGroups))
	var wg sync.WaitGroup

	for _, group := range allGroups {
		if group.Enable == 2 {
			cm.logger.Debug("跳过未启用的值班组", zap.String("group", group.Name))
			continue
		}

		if len(group.Users) == 0 {
			cm.logger.Warn("跳过无成员的值班组", zap.String("group", group.Name), zap.Int("id", group.ID))
			continue
		}

		wg.Add(1)
		go func(g *model.MonitorOnDutyGroup) {
			defer wg.Done()
			if err := cm.processOnDutyHistoryForGroup(ctx, g); err != nil {
				errChan <- fmt.Errorf("处理值班组 %s(ID:%d) 失败: %w", g.Name, g.ID, err)
			}
		}(group)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)

	// 收集错误
	errCount := 0
	for err := range errChan {
		errCount++
		cm.logger.Error("处理值班历史记录时发生错误", zap.Error(err))
	}

	if errCount > 0 {
		cm.logger.Warn("值班历史记录填充任务完成，但有错误", zap.Int("errorCount", errCount), zap.Int("totalGroups", len(allGroups)))
	} else {
		cm.logger.Info("值班历史记录填充任务成功完成", zap.Int("totalGroups", len(allGroups)))
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
	changes, err := cm.onDutyDao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, group.ID, todayStr, todayStr)
	if err != nil {
		cm.logger.Error("获取换班记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	// 如果今天有换班记录，需要更新或创建今天的值班历史
	if len(changes) > 0 {
		cm.logger.Info("发现今日换班记录", zap.String("group", group.Name), zap.Int("groupID", group.ID), zap.Int("changeCount", len(changes)))

		// 获取最新的换班记录（如果有多条，取最后一条）
		latestChange := changes[len(changes)-1]

		if exists {
			// 如果今天已有值班记录，则更新
			history, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, todayStr)
			if err != nil {
				cm.logger.Error("获取今日值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
				return err
			}

			// 更新值班人员
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

		// 如果今天没有值班记录，则创建一条新的，使用换班记录中的目标用户
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

	// 获取昨天的日期字符串
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 获取昨天的值班历史记录
	yesterdayHistory, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIDAndDay(ctx, group.ID, yesterdayStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		cm.logger.Error("获取昨天的值班历史记录失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
		return err
	}

	var onDutyUserID int
	var originUserID int
	if yesterdayHistory == nil {
		// 如果昨天没有记录，默认取成员列表的第一个用户
		onDutyUserID = group.Users[0].ID
		originUserID = group.Users[0].ID // 初始值班时，原始用户和值班用户相同
		cm.logger.Debug("未找到昨日值班记录，使用第一位成员",
			zap.String("group", group.Name),
			zap.Int("groupID", group.ID),
			zap.Int("userID", onDutyUserID))
	} else {
		// 检查昨天的值班用户是否仍在值班组中
		userStillExists := false
		for _, user := range group.Users {
			if user.ID == yesterdayHistory.OnDutyUserID {
				userStillExists = true
				break
			}
		}

		if !userStillExists {
			// 如果昨天的值班用户不在当前值班组中，使用第一个用户
			onDutyUserID = group.Users[0].ID
			originUserID = group.Users[0].ID // 重置为第一个用户时，原始用户和值班用户相同
			cm.logger.Warn("昨日值班用户已不在值班组中，使用第一位成员",
				zap.String("group", group.Name),
				zap.Int("groupID", group.ID),
				zap.Int("oldUserID", yesterdayHistory.OnDutyUserID),
				zap.Int("newUserID", onDutyUserID))
		} else {
			// 计算是否需要轮换值班人
			shiftNeeded, err := cm.isShiftNeeded(ctx, group, yesterdayHistory)
			if err != nil {
				cm.logger.Error("判断是否需要轮换值班人失败", zap.Error(err), zap.String("group", group.Name), zap.Int("groupID", group.ID))
				return err
			}
			if shiftNeeded {
				// 获取下一个值班人的索引
				nextUserIndex := (cm.getMemberIndex(group, yesterdayHistory.OnDutyUserID) + 1) % len(group.Users)
				originUserID = yesterdayHistory.OnDutyUserID // 记录原始值班人
				onDutyUserID = group.Users[nextUserIndex].ID
				cm.logger.Debug("轮换值班人",
					zap.String("group", group.Name),
					zap.Int("groupID", group.ID),
					zap.Int("oldUserID", yesterdayHistory.OnDutyUserID),
					zap.Int("newUserID", onDutyUserID))
			} else {
				// 继续昨天的值班人
				onDutyUserID = yesterdayHistory.OnDutyUserID
				originUserID = yesterdayHistory.OriginUserID // 保持原始值班人不变
				if originUserID == 0 {                       // 如果历史记录中没有原始值班人，则使用当前值班人
					originUserID = onDutyUserID
				}
				cm.logger.Debug("继续使用昨日值班人",
					zap.String("group", group.Name),
					zap.Int("groupID", group.ID),
					zap.Int("userID", onDutyUserID))
			}
		}
	}

	// 创建今天的值班历史记录
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

	// 计算开始日期，向前推移 shiftDays 天
	startDate := time.Now().AddDate(0, 0, -group.ShiftDays).Format("2006-01-02")
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 获取最近 shiftDays 天的值班历史记录
	histories, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx, group.ID, startDate, yesterdayStr)
	if err != nil {
		return false, fmt.Errorf("获取历史记录失败: %w", err)
	}

	// 统计连续值班天数
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

	// 如果连续值班天数达到 shiftDays，则需要轮换
	return consecutiveDays >= group.ShiftDays, nil
}

// getMemberIndex 获取成员在成员列表中的索引
func (cm *cronManager) getMemberIndex(group *model.MonitorOnDutyGroup, userID int) int {
	if group == nil || len(group.Users) == 0 {
		return 0
	}

	for index, member := range group.Users {
		if member.ID == userID {
			return index
		}
	}

	// 如果找不到该用户，记录警告并返回0
	cm.logger.Warn("在值班组中未找到指定用户，将使用第一位成员",
		zap.String("group", group.Name),
		zap.Int("groupID", group.ID),
		zap.Int("userID", userID))
	return 0
}

// StartCheckHostStatusManager 定期检查ecs主机状态
func (cm *cronManager) StartCheckHostStatusManager(ctx context.Context) error {
	cm.logger.Info("开始检查ecs主机状态")

	const batchSize = 100 // 每批处理的主机数量
	offset := 0

	for {
		// 分批获取ECS主机
		ecss, _, err := cm.ecsDao.ListEcsResources(ctx, &model.ListEcsResourcesReq{
			ListReq: model.ListReq{
				Page: offset/batchSize + 1, // 计算当前页码
				Size: batchSize,
			},
		})
		if err != nil {
			cm.logger.Error("获取ecs主机失败", zap.Error(err), zap.Int("offset", offset))
			return err
		}

		// 如果没有更多数据，则退出循环
		if len(ecss) == 0 {
			break
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(ecss))

		for _, ecs := range ecss {
			wg.Add(1)
			go func(ecs *model.ResourceEcs) {
				defer wg.Done()

				// 检查IP地址
				if ecs.IpAddr == "" {
					cm.logger.Warn("目标ecs没有绑定公网ip",
						zap.String("hostname", ecs.HostName),
						zap.Int("id", ecs.ID))
					return
				}

				// 发送ping请求检查状态
				status := "RUNNING"
				if ok := utils.Ping(ecs.IpAddr); !ok {
					cm.logger.Debug("ping请求失败",
						zap.String("ip", ecs.IpAddr),
						zap.String("hostname", ecs.HostName))
					status = "ERROR"
				}

				// 更新主机状态
				// 捕获可能的错误
				func() {
					defer func() {
						if r := recover(); r != nil {
							cm.logger.Error("更新主机状态时发生panic",
								zap.Any("recover", r),
								zap.String("hostname", ecs.HostName),
								zap.String("status", status))
							errChan <- fmt.Errorf("更新主机状态时发生panic: %v", r)
						}
					}()

					if err := cm.ecsDao.UpdateEcsStatus(ctx, strconv.Itoa(ecs.ID), status); err != nil {
						cm.logger.Error("更新主机状态失败",
							zap.Error(err),
							zap.String("hostname", ecs.HostName),
							zap.String("status", status))
						errChan <- err
					}
				}()
			}(ecs)
		}

		// 等待当前批次所有检查完成
		wg.Wait()
		close(errChan)

		// 检查是否有错误发生
		for err := range errChan {
			cm.logger.Error("处理主机状态时发生错误", zap.Error(err), zap.Int("batch_offset", offset))
		}

		// 更新偏移量，处理下一批
		offset += len(ecss)

		// 检查是否需要继续
		if len(ecss) < batchSize {
			break // 如果返回的数据少于请求的数量，说明已经到达末尾
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			cm.logger.Info("主机状态检查任务被取消", zap.Int("processed", offset))
			return ctx.Err()
		default:
			// 继续处理
		}
	}

	cm.logger.Info("完成ecs主机状态检查", zap.Int("total_processed", offset))
	return nil
}

// StartCheckK8sStatusManager 启动k8s状态检查任务
func (cm *cronManager) StartCheckK8sStatusManager(ctx context.Context) error {
	cm.logger.Info("开始检查k8s状态")

	// 获取所有k8s集群
	clusters, err := cm.k8sDao.ListAllClusters(ctx)
	if err != nil {
		cm.logger.Error("获取k8s集群列表失败", zap.Error(err))
		return fmt.Errorf("获取k8s集群列表失败: %w", err)
	}

	if len(clusters) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(clusters))

	// 限制并发数
	semaphore := make(chan struct{}, 5)

	for _, cluster := range clusters {
		wg.Add(1)
		go func(cluster *model.K8sCluster) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := cm.checkClusterStatus(ctx, cluster); err != nil {
				errChan <- err
			}
		}(cluster)
	}

	// 等待所有检查完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误发生
	if len(errChan) > 0 {
		var errs []error
		for err := range errChan {
			errs = append(errs, err)
		}
		return fmt.Errorf("k8s集群状态检查失败: %v", errs)
	}

	cm.logger.Info("完成k8s集群状态检查")
	return nil
}

// checkClusterStatus 检查单个集群状态
func (cm *cronManager) checkClusterStatus(ctx context.Context, cluster *model.K8sCluster) error {
	// 使用集群管理器检查集群状态
	if err := cm.clusterMgr.CheckClusterStatus(ctx, cluster.ID); err != nil {
		cm.logger.Warn("集群连接检查失败",
			zap.Error(err),
			zap.String("cluster", cluster.Name))
		cluster.Status = "ERROR"
	} else {
		cluster.Status = "RUNNING"
	}

	// 更新集群状态
	if err := cm.k8sDao.UpdateClusterStatus(ctx, cluster.ID, cluster.Status); err != nil {
		cm.logger.Error("更新集群状态失败",
			zap.Error(err),
			zap.String("cluster", cluster.Name),
			zap.String("status", cluster.Status))
		return fmt.Errorf("更新集群[%s]状态失败: %w", cluster.Name, err)
	}

	return nil
}
