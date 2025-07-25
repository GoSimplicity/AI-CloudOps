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

// MonitorOnDutyGroup 值班组的配置
type MonitorOnDutyGroup struct {
	Model
	Name                      string  `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:值班组名称"`
	UserID                    int     `json:"user_id" gorm:"index;comment:创建该值班组的用户ID"`
	ShiftDays                 int     `json:"shift_days" gorm:"type:int;not null;default:7;comment:轮班周期，以天为单位"`
	YesterdayNormalDutyUserID int     `json:"yesterday_normal_duty_user_id" gorm:"comment:昨天的正常排班值班人ID，由cron任务设置"`
	CreateUserName            string  `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	Users                     []*User `json:"users" gorm:"many2many:cl_monitor_on_duty_group_users;comment:值班组成员列表，多对多关系"`
	DutyPlans                 []*User `json:"duty_plans" gorm:"-;comment:值班计划列表"`
	Enable                    int8    `json:"enable" gorm:"type:tinyint(1);not null;default:1;comment:是否启用 1-启用 2-禁用"`
	Description               string  `json:"description" gorm:"type:varchar(255);comment:值班组描述"`
	TodayDutyUser             *User   `json:"today_duty_user" gorm:"-;comment:今日值班人"`
}

func (m *MonitorOnDutyGroup) TableName() string {
	return "cl_monitor_on_duty_groups"
}

// MonitorOnDutyChange 值班换班记录
type MonitorOnDutyChange struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	UserID         int    `json:"user_id" gorm:"index;comment:创建者ID"`
	Date           string `json:"date" gorm:"type:varchar(10);not null;comment:换班日期"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原值班人ID"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:新值班人ID"`
	CreateUserName string `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	Reason         string `json:"reason" gorm:"type:varchar(255);comment:换班原因"`
}

func (m *MonitorOnDutyChange) TableName() string {
	return "cl_monitor_on_duty_changes"
}

// MonitorOnDutyHistory 值班历史记录
type MonitorOnDutyHistory struct {
	Model
	OnDutyGroupID int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	DateString    string `json:"date_string" gorm:"type:varchar(10);not null;comment:值班日期"`
	OnDutyUserID  int    `json:"on_duty_user_id" gorm:"index;comment:当天值班人员ID"`
	OriginUserID  int    `json:"origin_user_id" gorm:"index;comment:原计划值班人员ID"`
}

func (m *MonitorOnDutyHistory) TableName() string {
	return "cl_monitor_on_duty_histories"
}

// MonitorOnDutyOne 单日值班信息
type MonitorOnDutyOne struct {
	Date       string `json:"date"`        // 值班日期
	User       *User  `json:"user"`        // 值班人信息
	OriginUser string `json:"origin_user"` // 原始值班人姓名
}

// GetMonitorOnDutyGroupListReq 获取值班组列表请求
type GetMonitorOnDutyGroupListReq struct {
	ListReq
	Enable *int8 `json:"enable" form:"enable" binding:"omitempty"`
}

// CreateMonitorOnDutyGroupReq 创建值班组请求
type CreateMonitorOnDutyGroupReq struct {
	Name           string `json:"name" binding:"required,min=1,max=50"`
	UserID         int    `json:"user_id" binding:"required"`
	UserIDs        []int  `json:"user_ids" binding:"required,min=1"`
	ShiftDays      int    `json:"shift_days" binding:"required,min=1"`
	CreateUserName string `json:"create_user_name"`
	Description    string `json:"description"`
}

// CreateMonitorOnDutyGroupChangeReq 创建值班组换班记录请求
type CreateMonitorOnDutyGroupChangeReq struct {
	OnDutyGroupID  int    `json:"on_duty_group_id" binding:"required"`
	Date           string `json:"date" binding:"required"`
	OriginUserID   int    `json:"origin_user_id" binding:"required"`
	OnDutyUserID   int    `json:"on_duty_user_id" binding:"required"`
	UserID         int    `json:"user_id" binding:"required"`
	CreateUserName string `json:"create_user_name"`
	Reason         string `json:"reason"`
}

// CreateMonitorOnDutyPlanReq 创建值班计划请求
type CreateMonitorOnDutyPlanReq struct {
	OnDutyGroupID  int    `json:"on_duty_group_id" binding:"required"`
	Date           string `json:"date" binding:"required"`
	OnDutyUserID   int    `json:"on_duty_user_id" binding:"required"`
	IsAdjusted     bool   `json:"is_adjusted"`
	OriginalUserID int    `json:"original_user_id"`
	CreateUserID   int    `json:"create_user_id" binding:"required"`
	CreateUserName string `json:"create_user_name"`
	Remark         string `json:"remark"`
}

// UpdateMonitorOnDutyGroupReq 更新值班组信息请求
type UpdateMonitorOnDutyGroupReq struct {
	ID          int    `json:"id" form:"id" binding:"required"`
	Name        string `json:"name" binding:"required,min=1,max=50"`
	ShiftDays   int    `json:"shift_days" binding:"required,min=1"`
	UserIDs     []int  `json:"user_ids" binding:"required,min=1"`
	Description string `json:"description"`
	Enable      *int8  `json:"enable" binding:"omitempty,oneof=1 2"`
}

// DeleteMonitorOnDutyGroupReq 删除值班组请求
type DeleteMonitorOnDutyGroupReq struct {
	ID int `json:"id" binding:"required"`
}

// GetMonitorOnDutyGroupReq 获取指定值班组信息请求
type GetMonitorOnDutyGroupReq struct {
	ID int `json:"id" binding:"required"`
}

// GetMonitorOnDutyGroupFuturePlanReq 获取值班组未来计划请求
type GetMonitorOnDutyGroupFuturePlanReq struct {
	ID        int    `json:"id" form:"id" binding:"required"`
	StartTime string `json:"start_time" form:"start_time" binding:"required"`
	EndTime   string `json:"end_time" form:"end_time" binding:"required"`
}

// GetMonitorOnDutyHistoryReq 获取值班历史记录请求
type GetMonitorOnDutyHistoryReq struct {
	ListReq
	OnDutyGroupID int    `json:"on_duty_group_id" binding:"required"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
}
