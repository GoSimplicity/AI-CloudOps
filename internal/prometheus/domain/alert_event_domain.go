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

package domain

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/prometheus/alertmanager/types"
	promModel "github.com/prometheus/common/model"
	"go.uber.org/zap"
)

type AlertEventDomain struct {
	Event      *model.MonitorAlertEvent
	User       *model.User
	SilenceReq *model.AlertEventSilenceRequest
	Logger     *zap.Logger
}

func NewAlertEventDomain(event *model.MonitorAlertEvent, user *model.User, logger *zap.Logger) *AlertEventDomain {
	return &AlertEventDomain{
		Event:  event,
		User:   user,
		Logger: logger,
	}
}

// BuildSilence 构建静默对象
func (d *AlertEventDomain) BuildSilence(ctx context.Context, silenceReq *model.AlertEventSilenceRequest) (*types.Silence, error) {
	// 解析持续时间
	duration, err := promModel.ParseDuration(silenceReq.Time)
	if err != nil {
		d.Logger.Error("构建静默对象失败: 解析持续时间错误", zap.Error(err))
		return nil, fmt.Errorf("无效的持续时间: %v", err)
	}

	// 构建匹配器
	matchers, err := utils.BuildMatchers(d.Event, d.Logger, silenceReq.UseName)
	if err != nil {
		d.Logger.Error("构建静默对象失败: 构建匹配器错误", zap.Error(err))
		return nil, err
	}

	// 创建 Silence 对象
	silence := &types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(duration)),
		CreatedBy: d.User.RealName,
		Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", d.Event.ID, d.User.RealName, duration),
	}

	return silence, nil
}

// MarkAsSilenced 标记为已静默
func (d *AlertEventDomain) MarkAsSilenced(silenceID string) {
	d.Event.Status = "已屏蔽"
	d.Event.SilenceID = silenceID
}

// MarkAsClaimed 标记为已认领
func (d *AlertEventDomain) MarkAsClaimed() {
	d.Event.RenLingUserID = int(d.User.ID)
	d.Event.Status = "已认领"
}

// BuildClaimMessage 构建认领消息
func (d *AlertEventDomain) BuildClaimMessage() string {
	return fmt.Sprintf(
		"**%s** 认领了告警事件: %s, 当前时间: %s",
		d.User.RealName,
		d.Event.AlertName,
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

// Validate 验证领域对象
func (d *AlertEventDomain) Validate() error {
	if d.Event == nil {
		return fmt.Errorf("告警事件不能为空")
	}
	if d.User == nil {
		return fmt.Errorf("用户信息不能为空")
	}
	return nil
}
