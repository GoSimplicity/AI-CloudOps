package cron

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/onduty"
	"gorm.io/gorm"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CronManager 定义计划任务管理接口
type CronManager interface {
	StartOnDutyHistoryManager(ctx context.Context) error
	fillOnDutyHistory(ctx context.Context)
	processOnDutyHistoryForGroup(ctx context.Context, group *model.MonitorOnDutyGroup)
}

// cronManager 实现 CronManager 接口
type cronManager struct {
	logger    *zap.Logger
	onDutyDao onduty.AlertManagerOnDutyDAO
	sync.RWMutex
}

// NewCronManager 创建一个新的 CronManager 实例
func NewCronManager(logger *zap.Logger, onDutyDao onduty.AlertManagerOnDutyDAO) CronManager {
	return &cronManager{
		logger:    logger,
		onDutyDao: onDutyDao,
	}
}

// StartOnDutyHistoryManager 启动值班历史记录填充任务
func (cm *cronManager) StartOnDutyHistoryManager(ctx context.Context) error {
	// 每隔 5 分钟执行一次 fillOnDutyHistory，直到 ctx.Done
	go wait.UntilWithContext(ctx, cm.fillOnDutyHistory, 5*time.Minute)
	<-ctx.Done()
	cm.logger.Info("值班历史记录填充任务已停止")
	return nil
}

// fillOnDutyHistory 填充所有值班组的历史记录
func (cm *cronManager) fillOnDutyHistory(ctx context.Context) {
	// 获取所有的值班组
	groups, err := cm.onDutyDao.GetAllMonitorOnDutyGroup(ctx)
	if err != nil {
		cm.logger.Error("获取值班组失败", zap.Error(err))
		return
	}

	var wg sync.WaitGroup
	for _, group := range groups {
		if len(group.Members) == 0 {
			continue
		}
		wg.Add(1)
		go func(g *model.MonitorOnDutyGroup) {
			defer wg.Done()
			cm.processOnDutyHistoryForGroup(ctx, g)
		}(group)
	}
	wg.Wait()
}

// processOnDutyHistoryForGroup 填充单个值班组的历史记录
func (cm *cronManager) processOnDutyHistoryForGroup(ctx context.Context, group *model.MonitorOnDutyGroup) {
	todayStr := time.Now().Format("2006-01-02")

	// 检查今天是否已经有值班历史记录
	exists, err := cm.onDutyDao.ExistsMonitorOnDutyHistory(ctx, group.ID, todayStr)
	if err != nil {
		cm.logger.Error("检查值班历史记录失败", zap.Error(err), zap.String("group", group.Name))
		return
	}
	if exists {
		return
	}

	// 获取昨天的日期字符串
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 获取昨天的值班历史记录
	yesterdayHistory, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIdAndDay(ctx, group.ID, yesterdayStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		cm.logger.Error("获取昨天的值班历史记录失败", zap.Error(err), zap.String("group", group.Name))
		return
	}

	var onDutyUserID int
	if yesterdayHistory == nil {
		// 如果昨天没有记录，默认取成员列表的第一个用户
		onDutyUserID = group.Members[0].ID
	} else {
		// 计算是否需要轮换值班人
		shiftNeeded, err := cm.isShiftNeeded(ctx, group, yesterdayHistory)
		if err != nil {
			cm.logger.Error("判断是否需要轮换值班人失败", zap.Error(err), zap.String("group", group.Name))
			return
		}
		if shiftNeeded {
			// 获取下一个值班人的索引
			nextUserIndex := (cm.getMemberIndex(group, yesterdayHistory.OnDutyUserID) + 1) % len(group.Members)
			onDutyUserID = group.Members[nextUserIndex].ID
		} else {
			// 继续昨天的值班人
			onDutyUserID = yesterdayHistory.OnDutyUserID
		}
	}

	// 创建今天的值班历史记录
	history := &model.MonitorOnDutyHistory{
		OnDutyGroupID: group.ID,
		DateString:    todayStr,
		OnDutyUserID:  onDutyUserID,
	}
	if err := cm.onDutyDao.CreateMonitorOnDutyHistory(ctx, history); err != nil {
		cm.logger.Error("创建值班历史记录失败", zap.Error(err), zap.String("group", group.Name))
		return
	}
}

// isShiftNeeded 判断是否需要轮换值班人
func (cm *cronManager) isShiftNeeded(ctx context.Context, group *model.MonitorOnDutyGroup, lastHistory *model.MonitorOnDutyHistory) (bool, error) {
	// 计算开始日期，向前推移 shiftDays 天
	startDate := time.Now().AddDate(0, 0, -group.ShiftDays).Format("2006-01-02")
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 获取最近 shiftDays 天的值班历史记录
	histories, err := cm.onDutyDao.GetMonitorOnDutyHistoryByGroupIdAndTimeRange(ctx, group.ID, startDate, yesterdayStr)
	if err != nil {
		return false, err
	}

	// 统计连续值班天数
	consecutiveDays := 0
	for _, history := range histories {
		if history.OnDutyUserID == lastHistory.OnDutyUserID {
			consecutiveDays++
		}
	}

	// 如果连续值班天数达到 shiftDays，则需要轮换
	return consecutiveDays >= group.ShiftDays, nil
}

// getMemberIndex 获取成员在成员列表中的索引
func (cm *cronManager) getMemberIndex(group *model.MonitorOnDutyGroup, userID int) int {
	for index, member := range group.Members {
		if member.ID == userID {
			return index
		}
	}
	return 0 // 默认返回第一个成员
}
