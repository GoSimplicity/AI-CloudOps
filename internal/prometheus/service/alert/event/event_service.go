package event

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/event"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/pool"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"github.com/prometheus/alertmanager/types"
	promModel "github.com/prometheus/common/model"
	"go.uber.org/zap"
	"sync"
	"time"
)

type AlertManagerEventService interface {
	GetMonitorAlertEventList(ctx context.Context, searchName *string) ([]*model.MonitorAlertEvent, error)
	EventAlertSilence(ctx context.Context, id int, event *model.AlertEventSilenceRequest, userId int) error
	EventAlertClaim(ctx context.Context, id int, userId int) error
	BatchEventAlertSilence(ctx context.Context, request *model.BatchEventAlertSilenceRequest, userId int) error
}

type alertManagerEventService struct {
	dao     event.AlertManagerEventDAO
	poolDao pool.AlertManagerPoolDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerEventService(dao event.AlertManagerEventDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerEventService {
	return &alertManagerEventService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerEventService) GetMonitorAlertEventList(ctx context.Context, searchName *string) ([]*model.MonitorAlertEvent, error) {
	return pkg.HandleList(ctx, searchName,
		a.dao.SearchMonitorAlertEventByName,
		a.dao.GetMonitorAlertEventList)
}

func (a *alertManagerEventService) EventAlertSilence(ctx context.Context, id int, event *model.AlertEventSilenceRequest, userId int) error {
	// 验证 ID 是否有效
	if id <= 0 {
		a.l.Error("设置静默失败：无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	// 获取 AlertEvent
	alertEvent, err := a.dao.GetAlertEventByID(ctx, id)
	if err != nil {
		a.l.Error("设置静默失败：无法获取 AlertEvent", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, userId)
	if err != nil {
		a.l.Error("设置静默失败：无效的 userId", zap.Int("userId", userId))
		return fmt.Errorf("无效的 userId: %d", userId)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(event.Time)
	if err != nil {
		a.l.Error("设置静默失败：解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 构建匹配器
	matchers, err := pkg.BuildMatchers(alertEvent, a.l, event.UseName)
	if err != nil {
		a.l.Error("设置静默失败：构建匹配器错误", zap.Error(err))
		return err
	}

	// 创建 Silence 对象
	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(duration)),
		CreatedBy: user.RealName,
		Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
	}

	// 序列化 Silence 对象为 JSON
	silenceData, err := json.Marshal(silence)
	if err != nil {
		a.l.Error("设置静默失败：序列化 Silence 对象失败", zap.Error(err))
		return fmt.Errorf("序列化失败: %v", err)
	}

	// 获取 AlertManager 地址
	alertPool, err := a.poolDao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
	if err != nil {
		a.l.Error("设置静默失败：无法获取 AlertPool", zap.Error(err))
		return err
	}

	if len(alertPool.AlertManagerInstances) == 0 {
		a.l.Error("设置静默失败：AlertManager 实例为空", zap.Int("poolID", alertPool.ID))
		return fmt.Errorf("AlertManager 实例为空")
	}

	alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
	alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

	// 发送 Silence 请求到 AlertManager
	silenceID, err := pkg.SendSilenceRequest(ctx, a.l, alertUrl, silenceData)
	if err != nil {
		a.l.Error("设置静默失败：发送 Silence 请求失败", zap.Error(err))
		return fmt.Errorf("发送 Silence 请求失败: %v", err)
	}

	// 更新 AlertEvent 状态为已静默
	alertEvent.Status = "已屏蔽"
	alertEvent.SilenceID = silenceID
	if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		a.l.Error("设置静默失败：更新 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("更新 AlertEvent 失败: %v", err)
	}

	a.l.Info("设置静默成功", zap.Int("id", id), zap.String("silenceID", silenceID))
	return nil
}

func (a *alertManagerEventService) EventAlertClaim(ctx context.Context, id int, userId int) error {
	// 获取告警事件
	event, err := a.dao.GetMonitorAlertEventById(ctx, id)
	if err != nil {
		a.l.Error("认领告警事件失败：获取告警事件时出错", zap.Error(err))
		return err
	}

	// 更新认领用户
	event.RenLingUserID = userId

	// 更新数据库
	if err := a.dao.EventAlertClaim(ctx, event); err != nil {
		a.l.Error("认领告警事件失败：更新告警事件时出错", zap.Error(err))
		return err
	}

	a.l.Info("认领告警事件成功", zap.Int("id", id), zap.Int("userId", userId))
	return nil
}

func (a *alertManagerEventService) BatchEventAlertSilence(ctx context.Context, request *model.BatchEventAlertSilenceRequest, userId int) error {
	// 输入验证
	if request == nil || len(request.IDs) == 0 {
		a.l.Error("批量设置静默失败：未提供事件ID")
		return fmt.Errorf("未提供事件ID")
	}

	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, userId)
	if err != nil {
		a.l.Error("批量设置静默失败：无效的 userId", zap.Int("userId", userId), zap.Error(err))
		return fmt.Errorf("无效的 userId: %d", userId)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(request.Time)
	if err != nil {
		a.l.Error("批量设置静默失败：解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 初始化等待组和错误收集
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	// 定义信号量以限制并发数量（例如，最多 10 个并发 goroutine）
	sem := make(chan struct{}, 10)

	for _, id := range request.IDs {
		if id <= 0 {
			a.l.Error("批量设置静默跳过：无效的 ID", zap.Int("id", id))
			mu.Lock()
			errs = append(errs, fmt.Errorf("无效的 ID: %d", id))
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		go func(eventID int) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 获取 AlertEvent
			alertEvent, err := a.dao.GetAlertEventByID(ctx, eventID)
			if err != nil {
				a.l.Error("批量设置静默失败：无法获取 AlertEvent", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 构建匹配器
			matchers, err := pkg.BuildMatchers(alertEvent, a.l, request.UseName)
			if err != nil {
				a.l.Error("批量设置静默失败：构建匹配器错误", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 创建 Silence 对象
			silence := types.Silence{
				Matchers:  matchers,
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(time.Duration(duration)),
				CreatedBy: user.RealName,
				Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
			}

			// 序列化 Silence 对象为 JSON
			silenceData, err := json.Marshal(silence)
			if err != nil {
				a.l.Error("批量设置静默失败：序列化 Silence 对象失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 获取 AlertManager 地址
			alertPool, err := a.poolDao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
			if err != nil {
				a.l.Error("批量设置静默失败：无法获取 AlertPool", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			if len(alertPool.AlertManagerInstances) == 0 {
				a.l.Error("批量设置静默失败：AlertManager 实例为空", zap.Int("poolID", alertPool.ID), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: AlertManager 实例为空", eventID))
				mu.Unlock()
				return
			}

			alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
			alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

			// 发送 Silence 请求到 AlertManager
			silenceID, err := pkg.SendSilenceRequest(ctx, a.l, alertUrl, silenceData)
			if err != nil {
				a.l.Error("批量设置静默失败：发送 Silence 请求失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 更新 AlertEvent 状态为已静默
			alertEvent.Status = "已屏蔽"
			alertEvent.SilenceID = silenceID
			if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
				a.l.Error("批量设置静默失败：更新 AlertEvent 失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			a.l.Info("批量设置静默成功", zap.Int("id", eventID), zap.String("silenceID", silenceID))
		}(id)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 关闭信号量通道
	close(sem)

	if len(errs) > 0 {
		// 聚合错误
		errMsg := "批量设置静默过程中遇到以下错误："
		for _, e := range errs {
			errMsg += "\n" + e.Error()
		}
		a.l.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	a.l.Info("批量设置静默成功处理所有事件")
	return nil
}
