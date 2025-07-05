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

type MonitorAlertManagerPool struct {
	Model
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:AlertManager实例名称"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;not null;comment:AlertManager实例列表"`
	UserID                int        `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	ResolveTimeout        string     `json:"resolve_timeout" gorm:"size:50;default:'5m';not null;comment:告警恢复超时时间"`
	GroupWait             string     `json:"group_wait" gorm:"size:50;default:'30s';not null;comment:首次告警等待时间"`
	GroupInterval         string     `json:"group_interval" gorm:"size:50;default:'5m';not null;comment:告警分组间隔时间"`
	RepeatInterval        string     `json:"repeat_interval" gorm:"size:50;default:'4h';not null;comment:重复告警间隔"`
	GroupBy               StringList `json:"group_by" gorm:"type:text;not null;comment:告警分组标签列表"`
	Receiver              string     `json:"receiver" gorm:"size:100;not null;comment:默认接收者"`
	CreatorName           string     `json:"creator_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
}

// CreateMonitorAlertManagerPoolReq 创建 AlertManager 实例池请求
type CreateMonitorAlertManagerPoolReq struct {
	Name                  string     `json:"name" binding:"required,min=1,max=50"`
	AlertManagerInstances StringList `json:"alert_manager_instances" binding:"required"`
	UserID                int        `json:"user_id" binding:"required"`
	ResolveTimeout        string     `json:"resolve_timeout"`
	GroupWait             string     `json:"group_wait"`
	GroupInterval         string     `json:"group_interval"`
	RepeatInterval        string     `json:"repeat_interval"`
	GroupBy               StringList `json:"group_by"`
	Receiver              string     `json:"receiver" binding:"required"`
	CreatorName           string     `json:"creator_name" binding:"required"`
}

// UpdateMonitorAlertManagerPoolReq 更新 AlertManager 实例池请求
type UpdateMonitorAlertManagerPoolReq struct {
	ID                    int        `json:"id" binding:"required"`
	Name                  string     `json:"name" binding:"required,min=1,max=50"`
	AlertManagerInstances StringList `json:"alert_manager_instances" binding:"required"`
	ResolveTimeout        string     `json:"resolve_timeout"`
	GroupWait             string     `json:"group_wait"`
	GroupInterval         string     `json:"group_interval"`
	RepeatInterval        string     `json:"repeat_interval"`
	GroupBy               StringList `json:"group_by"`
	Receiver              string     `json:"receiver" binding:"required"`
}

// DeleteMonitorAlertManagerPoolReq 删除 AlertManager 实例池请求
type DeleteMonitorAlertManagerPoolReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetMonitorAlertManagerPoolListReq 获取 AlertManager 实例池列表请求
type GetMonitorAlertManagerPoolListReq struct {
	ListReq
	PoolID int `json:"pool_id" form:"pool_id" binding:"omitempty"`
}

// GetMonitorAlertManagerPoolReq 获取 AlertManager 实例池请求
type GetMonitorAlertManagerPoolReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
