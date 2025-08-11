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

package model

// MonitorAlertEventStatus 告警事件状态
type MonitorAlertEventStatus int8

const (
	MonitorAlertEventStatusFiring   MonitorAlertEventStatus = iota + 1 // 告警触发
	MonitorAlertEventStatusSilenced                                    // 告警静默
	MonitorAlertEventStatusClaimed                                     // 告警认领
	MonitorAlertEventStatusResolved                                    // 告警恢复
	MonitorAlertEventStatusUpgraded                                    // 告警升级
)

// MonitorAlertEvent 告警事件模型
type MonitorAlertEvent struct {
	Model
	AlertName     string                  `json:"alert_name" binding:"required,min=1,max=200" gorm:"size:200;not null;comment:告警名称"`
	Fingerprint   string                  `json:"fingerprint" binding:"required,min=1,max=100" gorm:"size:100;not null;uniqueIndex:idx_monitor_alert_event_fingerprint;comment:告警唯一ID"`
	Status        MonitorAlertEventStatus `json:"status" gorm:"not null;default:1;index;comment:告警状态(1:触发 2:静默 3:认领 4:恢复)"`
	RuleID        int                     `json:"rule_id" gorm:"index;not null;comment:关联的告警规则ID"`
	SendGroupID   int                     `json:"send_group_id" gorm:"index;not null;comment:关联的发送组ID"`
	EventTimes    int                     `json:"event_times" gorm:"not null;default:1;comment:触发次数"`
	SilenceID     string                  `json:"silence_id" gorm:"size:100;comment:AlertManager返回的静默ID"`
	RenLingUserID int                     `json:"ren_ling_user_id" gorm:"index;comment:认领告警的用户ID"`
	Labels        StringList              `json:"labels" gorm:"type:text;not null;comment:标签组,格式为key=value"`
	SendGroup     *MonitorSendGroup       `json:"send_group" gorm:"-"`
	RenLingUser   *User                   `json:"ren_ling_user" gorm:"-"`
	LabelsMap     map[string]string       `json:"labels_map" gorm:"-"`
}

func (m *MonitorAlertEvent) TableName() string {
	return "cl_monitor_alert_events"
}

// GetMonitorAlertEventListReq 获取告警事件列表请求
type GetMonitorAlertEventListReq struct {
	ListReq
	Status    MonitorAlertEventStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4"`
	StartTime string                  `json:"start_time" form:"start_time" binding:"omitempty"`
	EndTime   string                  `json:"end_time" form:"end_time" binding:"omitempty"`
}

// EventAlertSilenceReq 告警静默请求
type EventAlertSilenceReq struct {
	ID      int    `json:"id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	UseName int8   `json:"use_name" binding:"omitempty,oneof=1 2"` // 是否启用名称静默
	Time    string `json:"time" binding:"required"`
}

// EventAlertClaimReq 告警认领请求
type EventAlertClaimReq struct {
	ID     int `json:"id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}

// EventAlertUnSilenceReq 告警取消静默请求
type EventAlertUnSilenceReq struct {
	ID     int `json:"id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}
