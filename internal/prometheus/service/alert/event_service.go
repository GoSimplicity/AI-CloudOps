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
	"time"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/prometheus/alertmanager/types"
	promModel "github.com/prometheus/common/model"
	"go.uber.org/zap"
)

// AlertManagerEventService 定义告警事件管理服务接口
type AlertManagerEventService interface {
	GetMonitorAlertEventList(ctx context.Context, req *model.GetMonitorAlertEventListReq) (model.ListResp[*model.MonitorAlertEvent], error)
	EventAlertSilence(ctx context.Context, req *model.EventAlertSilenceReq) error
	EventAlertClaim(ctx context.Context, req *model.EventAlertClaimReq) error
	EventAlertUnSilence(ctx context.Context, req *model.EventAlertUnSilenceReq) error
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
func (a *alertManagerEventService) GetMonitorAlertEventList(ctx context.Context, req *model.GetMonitorAlertEventListReq) (model.ListResp[*model.MonitorAlertEvent], error) {
	events, total, err := a.dao.GetMonitorAlertEventList(ctx, req)
	if err != nil {
		a.l.Error("获取告警事件列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorAlertEvent]{}, err
	}

	return model.ListResp[*model.MonitorAlertEvent]{
		Total: total,
		Items: events,
	}, nil
}

// EventAlertSilence 设置告警事件静默
func (a *alertManagerEventService) EventAlertSilence(ctx context.Context, req *model.EventAlertSilenceReq) error {
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

	// 解析持续时间
	duration, err := promModel.ParseDuration(req.Time)
	if err != nil {
		a.l.Error("构建静默对象失败: 解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 构建匹配器
	matchers, err := utils.BuildMatchers(alertEvent, a.l, req.UseName)
	if err != nil {
		a.l.Error("构建静默对象失败: 构建匹配器错误", zap.Error(err))
		return err
	}

	// 创建 Silence 对象
	silence := &types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(duration)),
		CreatedBy: user.RealName,
		Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
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
	alertEvent.Status = model.MonitorAlertEventStatusSilenced
	alertEvent.SilenceID = silenceID

	// 更新告警事件状态
	if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		a.l.Error("设置静默失败: 更新告警事件状态失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新告警事件状态失败: %v", err)
	}

	a.l.Info("设置静默成功", zap.Int("id", req.ID), zap.String("silenceID", silenceID))
	return nil
}

// EventAlertClaim 认领告警事件
func (a *alertManagerEventService) EventAlertClaim(ctx context.Context, req *model.EventAlertClaimReq) error {
	// 获取告警事件
	event, err := a.dao.GetMonitorAlertEventById(ctx, req.ID)
	if err != nil {
		a.l.Error("认领告警事件失败: 获取告警事件失败", zap.Error(err))
		return fmt.Errorf("获取告警事件失败: %v", err)
	}

	// 获取发送组信息
	sendGroup, err := a.sendDao.GetMonitorSendGroupByID(ctx, event.SendGroupID)
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

	// 标记为已认领
	event.RenLingUserID = int(user.ID)
	event.Status = model.MonitorAlertEventStatusClaimed

	// 更新数据库
	if err := a.dao.EventAlertClaim(ctx, event); err != nil {
		a.l.Error("认领告警事件失败: 更新告警事件失败", zap.Error(err))
		return fmt.Errorf("更新告警事件失败: %v", err)
	}

	// 构建通知内容
	content := fmt.Sprintf(
		"**%s** 认领了告警事件: %s, 当前时间: %s",
		user.RealName,
		event.AlertName,
		time.Now().Format("2006-01-02 15:04:05"),
	)

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
func (a *alertManagerEventService) EventAlertUnSilence(ctx context.Context, req *model.EventAlertUnSilenceReq) error {
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

	// 标记为已取消静默
	alertEvent.SilenceID = ""
	alertEvent.Status = model.MonitorAlertEventStatusFiring

	// 更新告警事件状态
	if err := a.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		a.l.Error("取消静默失败: 更新告警事件状态失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新告警事件状态失败: %v", err)
	}

	a.l.Info("取消静默成功", zap.Int("id", req.ID), zap.String("silenceID", alertEvent.SilenceID))
	return nil
}

// processSingleSilence 处理单个告警事件的静默
func (a *alertManagerEventService) processSingleSilence(ctx context.Context, eventID int, request *model.EventAlertSilenceReq, user *model.User) error {
	// 获取告警事件
	alertEvent, err := a.dao.GetAlertEventByID(ctx, eventID)
	if err != nil {
		a.l.Error("处理单个静默失败: 获取告警事件失败", zap.Error(err), zap.Int("id", eventID))
		return fmt.Errorf("获取告警事件失败: %v", err)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(request.Time)
	if err != nil {
		a.l.Error("构建静默对象失败: 解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 构建匹配器
	matchers, err := utils.BuildMatchers(alertEvent, a.l, request.UseName)
	if err != nil {
		a.l.Error("构建静默对象失败: 构建匹配器错误", zap.Error(err))
		return err
	}

	// 创建 Silence 对象
	silence := &types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(duration)),
		CreatedBy: user.RealName,
		Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
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
	alertEvent.Status = model.MonitorAlertEventStatusSilenced
	alertEvent.SilenceID = silenceID

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
