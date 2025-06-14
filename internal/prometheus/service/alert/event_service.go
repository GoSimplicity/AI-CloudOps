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
	"encoding/json"
	"fmt"
	"sync"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/domain"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

// AlertManagerEventService 定义告警事件管理服务接口
type AlertManagerEventService interface {
	GetMonitorAlertEventList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorAlertEvent, error)
	EventAlertSilence(ctx context.Context, req *model.AlertEventSilenceRequest) error
	EventAlertClaim(ctx context.Context, req *model.AlertEventClaimRequest) error
	EventAlertUnSilence(ctx context.Context, req *model.AlertEventUnSilenceRequest) error
	BatchEventAlertSilence(ctx context.Context, req *model.BatchEventAlertSilenceRequest) error
	GetMonitorAlertEventTotal(ctx context.Context) (int, error)
}

// alertManagerEventService 实现告警事件管理服务
type alertManagerEventService struct {
	dao     alert.AlertManagerEventDAO
	sendDao alert.AlertManagerSendDAO
	poolDao alert.AlertManagerPoolDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

// NewAlertManagerEventService 创建告警事件管理服务实例
func NewAlertManagerEventService(dao alert.AlertManagerEventDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO, sendDao alert.AlertManagerSendDAO) AlertManagerEventService {
	return &alertManagerEventService{
		dao:     dao,
		userDao: userDao,
		sendDao: sendDao,
		l:       l,
		cache:   cache,
	}
}

// GetMonitorAlertEventList 获取告警事件列表
func (a *alertManagerEventService) GetMonitorAlertEventList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorAlertEvent, error) {
	if listReq.Search != "" {
		events, err := a.dao.SearchMonitorAlertEventByName(ctx, listReq.Search)
		if err != nil {
			a.l.Error("搜索告警事件失败", zap.String("search", listReq.Search), zap.Error(err))
			return nil, err
		}
		return events, nil
	}

	offset := (listReq.Page - 1) * listReq.Size
	limit := listReq.Size

	events, err := a.dao.GetMonitorAlertEventList(ctx, offset, limit)
	if err != nil {
		a.l.Error("获取告警事件列表失败", zap.Error(err))
		return nil, err
	}

	return events, nil
}

// EventAlertSilence 设置告警事件静默
func (a *alertManagerEventService) EventAlertSilence(ctx context.Context, req *model.AlertEventSilenceRequest) error {
	// 参数校验
	if req.ID <= 0 {
		a.l.Error("设置静默失败: 无效的 ID", zap.Int("id", req.ID))
		return fmt.Errorf("无效的 ID: %d", req.ID)
	}

	// 获取告警事件信息
	alertEvent, err := a.dao.GetAlertEventByID(ctx, req.ID)
	if err != nil {
		a.l.Error("设置静默失败: 无法获取告警事件", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("获取告警事件失败: %v", err)
	}

	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, req.UserID)
	if err != nil {
		a.l.Error("设置静默失败: 无效的用户ID", zap.Int("userId", req.UserID), zap.Error(err))
		return fmt.Errorf("无效的用户ID: %d, %v", req.UserID, err)
	}

	// 创建领域对象
	eventDomain := domain.NewAlertEventDomain(alertEvent, user, a.l)

	// 构建静默对象
	silence, err := eventDomain.BuildSilence(ctx, req)
	if err != nil {
		return err
	}

	// 序列化静默规则
	silenceData, err := json.Marshal(silence)
	if err != nil {
		a.l.Error("设置静默失败: 序列化静默规则失败", zap.Error(err))
		return fmt.Errorf("序列化静默规则失败: %v", err)
	}

	// 获取告警管理器实例
	alertPool, err := a.poolDao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
	if err != nil {
		a.l.Error("设置静默失败: 无法获取告警管理器实例", zap.Error(err))
		return fmt.Errorf("获取告警管理器实例失败: %v", err)
	}

	if len(alertPool.AlertManagerInstances) == 0 {
		a.l.Error("设置静默失败: 告警管理器实例为空", zap.Int("poolID", alertPool.ID))
		return fmt.Errorf("告警管理器实例未配置")
	}

	// 构建请求URL
	alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
	alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

	// 发送静默请求
	silenceID, err := pkg.SendSilenceRequest(ctx, a.l, alertUrl, silenceData)
	if err != nil {
		a.l.Error("设置静默失败: 发送静默请求失败", zap.Error(err))
		return fmt.Errorf("发送静默请求失败: %v", err)
	}

	// 标记为已静默
	eventDomain.MarkAsSilenced(silenceID)

	// 更新告警事件状态
	if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		a.l.Error("设置静默失败: 更新告警事件状态失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新告警事件状态失败: %v", err)
	}

	a.l.Info("设置静默成功", zap.Int("id", req.ID), zap.String("silenceID", silenceID))
	return nil
}

// EventAlertClaim 认领告警事件
func (a *alertManagerEventService) EventAlertClaim(ctx context.Context, req *model.AlertEventClaimRequest) error {
	// 获取告警事件
	event, err := a.dao.GetMonitorAlertEventById(ctx, req.ID)
	if err != nil {
		a.l.Error("认领告警事件失败: 获取告警事件失败", zap.Error(err))
		return fmt.Errorf("获取告警事件失败: %v", err)
	}

	// 获取发送组信息
	sendGroup, err := a.sendDao.GetMonitorSendGroupById(ctx, event.SendGroupID)
	if err != nil {
		a.l.Error("认领告警事件失败: 获取发送组失败", zap.Error(err))
		return fmt.Errorf("获取发送组失败: %v", err)
	}

	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, req.UserID)
	if err != nil {
		a.l.Error("认领告警事件失败: 获取用户信息失败", zap.Error(err))
		return fmt.Errorf("获取用户信息失败: %v", err)
	}

	// 创建领域对象
	eventDomain := domain.NewAlertEventDomain(event, user, a.l)

	// 标记为已认领
	eventDomain.MarkAsClaimed()

	// 更新数据库
	if err := a.dao.EventAlertClaim(ctx, event); err != nil {
		a.l.Error("认领告警事件失败: 更新告警事件失败", zap.Error(err))
		return fmt.Errorf("更新告警事件失败: %v", err)
	}

	// 构建通知内容
	content := eventDomain.BuildClaimMessage()

	// 发送飞书通知
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/bot/v2/hook/%s", sendGroup.FeiShuQunRobotToken)
	if err = a.dao.SendMessageToGroup(ctx, url, content); err != nil {
		a.l.Error("发送飞书通知失败", zap.Error(err))
		// 不影响主流程,仅记录日志
	}

	a.l.Info("认领告警事件成功", zap.Int("id", req.ID), zap.Int("userId", req.UserID))
	return nil
}

// EventAlertUnSilence 取消告警事件静默
func (a *alertManagerEventService) EventAlertUnSilence(ctx context.Context, req *model.AlertEventUnSilenceRequest) error {
	// 参数校验
	if req.ID <= 0 {
		a.l.Error("取消静默失败: 无效的 ID", zap.Int("id", req.ID))
		return fmt.Errorf("无效的 ID: %d", req.ID)
	}

	// 获取告警事件信息
	alertEvent, err := a.dao.GetAlertEventByID(ctx, req.ID)
	if err != nil {
		a.l.Error("取消静默失败: 无法获取告警事件", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("获取告警事件失败: %v", err)
	}

	// 检查是否已静默
	if alertEvent.SilenceID == "" {
		a.l.Error("取消静默失败: 该告警事件未被静默", zap.Int("id", req.ID))
		return fmt.Errorf("该告警事件未被静默")
	}

	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, req.UserID)
	if err != nil {
		a.l.Error("取消静默失败: 无效的用户ID", zap.Int("userId", req.UserID), zap.Error(err))
		return fmt.Errorf("无效的用户ID: %d, %v", req.UserID, err)
	}

	// 获取告警管理器实例
	alertPool, err := a.poolDao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
	if err != nil {
		a.l.Error("取消静默失败: 无法获取告警管理器实例", zap.Error(err))
		return fmt.Errorf("获取告警管理器实例失败: %v", err)
	}

	if len(alertPool.AlertManagerInstances) == 0 {
		a.l.Error("取消静默失败: 告警管理器实例为空", zap.Int("poolID", alertPool.ID))
		return fmt.Errorf("告警管理器实例未配置")
	}

	// 构建请求URL
	alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
	_ = fmt.Sprintf("%s/api/v1/silence/%s", alertAddr, alertEvent.SilenceID)

	// 发送删除静默请求
	// TODO: 暂时不删除静默，因为删除静默后，告警事件会重新触发
	// if err := pkg.DeleteSilenceRequest(ctx, a.l, alertUrl); err != nil {
	// 	a.l.Error("取消静默失败: 发送删除静默请求失败", zap.Error(err))
	// 	return fmt.Errorf("发送删除静默请求失败: %v", err)
	// }

	// 创建领域对象
	eventDomain := domain.NewAlertEventDomain(alertEvent, user, a.l)

	// 标记为已取消静默
	eventDomain.MarkAsUnSilenced()

	// 更新告警事件状态
	if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		a.l.Error("取消静默失败: 更新告警事件状态失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新告警事件状态失败: %v", err)
	}

	a.l.Info("取消静默成功", zap.Int("id", req.ID), zap.String("silenceID", alertEvent.SilenceID))
	return nil
}

// BatchEventAlertSilence 批量设置告警事件静默
func (a *alertManagerEventService) BatchEventAlertSilence(ctx context.Context, req *model.BatchEventAlertSilenceRequest) error {
	// 参数校验
	if req == nil || len(req.IDs) == 0 {
		a.l.Error("批量设置静默失败: 未提供事件ID")
		return fmt.Errorf("未提供有效的事件ID列表")
	}

	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, req.UserID)
	if err != nil {
		a.l.Error("批量设置静默失败: 无效的用户ID", zap.Int("userId", req.UserID), zap.Error(err))
		return fmt.Errorf("无效的用户ID: %d, %v", req.UserID, err)
	}

	// 并发控制
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
		sem  = make(chan struct{}, 10) // 限制最大并发数为10
	)

	// 并发处理每个告警事件
	for _, id := range req.IDs {
		if id <= 0 {
			a.l.Error("批量设置静默跳过: 无效的ID", zap.Int("id", id))
			mu.Lock()
			errs = append(errs, fmt.Errorf("无效的ID: %d", id))
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(eventID int) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			if err := a.processSingleSilence(ctx, eventID, req, user); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件ID %d: %v", eventID, err))
				mu.Unlock()
			}
		}(id)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(sem)

	// 处理错误
	if len(errs) > 0 {
		var errMsg string
		for _, e := range errs {
			if errMsg != "" {
				errMsg += "\n"
			}
			errMsg += e.Error()
		}
		a.l.Error(errMsg)
		return fmt.Errorf("批量设置静默失败: %s", errMsg)
	}

	a.l.Info("批量设置静默成功完成")
	return nil
}

// processSingleSilence 处理单个告警事件的静默
func (a *alertManagerEventService) processSingleSilence(ctx context.Context, eventID int, request *model.BatchEventAlertSilenceRequest, user *model.User) error {
	// 获取告警事件
	alertEvent, err := a.dao.GetAlertEventByID(ctx, eventID)
	if err != nil {
		a.l.Error("处理单个静默失败: 获取告警事件失败", zap.Error(err), zap.Int("id", eventID))
		return fmt.Errorf("获取告警事件失败: %v", err)
	}

	// 创建领域对象
	eventDomain := domain.NewAlertEventDomain(alertEvent, user, a.l)

	// 构建静默对象
	silence, err := eventDomain.BuildSilence(ctx, &model.AlertEventSilenceRequest{
		Time:    request.Time,
		UseName: request.UseName,
	})
	if err != nil {
		return err
	}

	// 序列化静默规则
	silenceData, err := json.Marshal(silence)
	if err != nil {
		return fmt.Errorf("序列化静默规则失败: %v", err)
	}

	// 获取告警管理器实例
	alertPool, err := a.poolDao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
	if err != nil {
		return fmt.Errorf("获取告警管理器实例失败: %v", err)
	}

	// 发送静默请求
	silenceID, err := a.sendSilenceRequest(ctx, alertPool, silenceData)
	if err != nil {
		return err
	}

	// 标记为已静默
	eventDomain.MarkAsSilenced(silenceID)

	// 更新告警事件状态
	if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		return fmt.Errorf("更新告警事件状态失败: %v", err)
	}

	a.l.Info("处理单个静默成功", zap.Int("id", eventID), zap.String("silenceID", silenceID))
	return nil
}

// sendSilenceRequest 发送静默请求到AlertManager
func (a *alertManagerEventService) sendSilenceRequest(ctx context.Context, alertPool *model.MonitorAlertManagerPool, silenceData []byte) (string, error) {
	if len(alertPool.AlertManagerInstances) == 0 {
		return "", fmt.Errorf("告警管理器实例未配置")
	}

	alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
	alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

	silenceID, err := pkg.SendSilenceRequest(ctx, a.l, alertUrl, silenceData)
	if err != nil {
		return "", fmt.Errorf("发送静默请求失败: %v", err)
	}

	return silenceID, nil
}

// GetMonitorAlertEventTotal 获取监控告警事件总数
func (a *alertManagerEventService) GetMonitorAlertEventTotal(ctx context.Context) (int, error) {
	return a.dao.GetMonitorAlertEventTotal(ctx)
}
