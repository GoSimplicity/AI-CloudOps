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

// MonitorSendGroup 发送组的配置
type MonitorSendGroup struct {
	Model
	Name                   string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:发送组英文名称"`
	NameZh                 string     `json:"name_zh" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:发送组中文名称"`
	Enable                 int8       `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用发送组 1:启用 2:禁用"`
	UserID                 int        `json:"user_id" gorm:"index;not null;comment:创建该发送组的用户ID"`
	PoolID                 int        `json:"pool_id" gorm:"index;not null;comment:关联的AlertManager实例ID"`
	OnDutyGroupID          int        `json:"on_duty_group_id" gorm:"index;comment:值班组ID"`
	FeiShuQunRobotToken    string     `json:"fei_shu_qun_robot_token" gorm:"size:255;comment:飞书机器人Token"`
	RepeatInterval         string     `json:"repeat_interval" gorm:"size:50;default:'4h';comment:重复发送时间间隔"`
	SendResolved           int8       `json:"send_resolved" gorm:"type:tinyint(1);default:1;not null;comment:是否发送恢复通知 1:发送 2:不发送"`
	NotifyMethods          StringList `json:"notify_methods" gorm:"type:text;comment:通知方法列表"` // 例如: ["email", "feishu", "dingtalk"]
	NeedUpgrade            int8       `json:"need_upgrade" gorm:"type:tinyint(1);default:0;not null;comment:是否需要告警升级 1:需要 2:不需要"`
	UpgradeMinutes         int        `json:"upgrade_minutes" gorm:"default:30;comment:告警升级等待时间(分钟)"`
	StaticReceiveUsers     []*User    `json:"static_receive_users" gorm:"many2many:cl_monitor_send_group_static_receive_users;comment:静态配置的接收人列表"`
	FirstUpgradeUsers      []*User    `json:"first_upgrade_users" gorm:"many2many:cl_monitor_send_group_first_upgrade_users;comment:第一级升级人列表"`
	SecondUpgradeUsers     []*User    `json:"second_upgrade_users" gorm:"many2many:cl_monitor_send_group_second_upgrade_users;comment:第二级升级人列表"`
	CreateUserName         string     `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建该发送组的用户名称"`
	StaticReceiveUserNames []string   `json:"static_receive_user_names" gorm:"-"`
	FirstUserNames         []string   `json:"first_user_names" gorm:"-"`
	SecondUserNames        []string   `json:"second_user_names" gorm:"-"`
}

func (m *MonitorSendGroup) TableName() string {
	return "cl_monitor_send_groups"
}

// CreateMonitorSendGroupReq 创建发送组请求
type CreateMonitorSendGroupReq struct {
	Name                string     `json:"name" binding:"required,min=1,max=50"`
	NameZh              string     `json:"name_zh" binding:"required,min=1,max=50"`
	Enable              int8       `json:"enable" binding:"omitempty,oneof=1 2"`
	UserID              int        `json:"user_id" binding:"required"`
	PoolID              int        `json:"pool_id" binding:"required"`
	OnDutyGroupID       int        `json:"on_duty_group_id"`
	StaticReceiveUsers  []*User    `json:"static_receive_users"`
	FeiShuQunRobotToken string     `json:"fei_shu_qun_robot_token" binding:"max=255"`
	RepeatInterval      string     `json:"repeat_interval" binding:"max=50"`
	SendResolved        int8       `json:"send_resolved" binding:"omitempty,oneof=1 2"`
	NotifyMethods       StringList `json:"notify_methods"`
	NeedUpgrade         int8       `json:"need_upgrade" binding:"omitempty,oneof=1 2"`
	FirstUpgradeUsers   []*User    `json:"first_upgrade_users"`
	UpgradeMinutes      int        `json:"upgrade_minutes" binding:"min=0"`
	SecondUpgradeUsers  []*User    `json:"second_upgrade_users"`
	CreateUserName      string     `json:"create_user_name"`
}

// UpdateMonitorSendGroupReq 更新发送组请求
type UpdateMonitorSendGroupReq struct {
	ID                  int        `json:"id" form:"id" binding:"required"`
	Name                string     `json:"name" binding:"required,min=1,max=50"`
	NameZh              string     `json:"name_zh" binding:"required,min=1,max=50"`
	Enable              int8       `json:"enable" binding:"omitempty,oneof=1 2"`
	PoolID              int        `json:"pool_id" binding:"required"`
	OnDutyGroupID       int        `json:"on_duty_group_id"`
	StaticReceiveUsers  []*User    `json:"static_receive_users"`
	FeiShuQunRobotToken string     `json:"fei_shu_qun_robot_token" binding:"max=255"`
	RepeatInterval      string     `json:"repeat_interval" binding:"max=50"`
	SendResolved        int8       `json:"send_resolved" binding:"omitempty,oneof=1 2"`
	NotifyMethods       StringList `json:"notify_methods"`
	NeedUpgrade         int8       `json:"need_upgrade" binding:"omitempty,oneof=1 2"`
	FirstUpgradeUsers   []*User    `json:"first_upgrade_users"`
	UpgradeMinutes      int        `json:"upgrade_minutes" binding:"min=0"`
	SecondUpgradeUsers  []*User    `json:"second_upgrade_users"`
}

// DeleteMonitorSendGroupReq 删除发送组请求
type DeleteMonitorSendGroupReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetMonitorSendGroupReq 获取发送组请求
type GetMonitorSendGroupReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetMonitorSendGroupListReq 获取发送组列表请求
type GetMonitorSendGroupListReq struct {
	ListReq
	PoolID        *int  `json:"pool_id" form:"pool_id"`
	Enable        *int8 `json:"enable" form:"enable" binding:"omitempty,oneof=1 2"`
	OnDutyGroupID *int  `json:"on_duty_group_id" form:"on_duty_group_id"`
}
