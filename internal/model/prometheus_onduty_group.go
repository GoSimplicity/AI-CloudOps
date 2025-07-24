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

// MonitorOnDutyChange 值班换班记录
type MonitorOnDutyChange struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	UserID         int    `json:"user_id" gorm:"index;comment:创建者ID"`
	Date           string `json:"date" gorm:"type:varchar(10);not null;comment:换班日期"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原值班人ID"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:新值班人ID"`
	CreateUserName string `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	TargetUserName string `json:"target_user_name" gorm:"-"`
	OriginUserName string `json:"origin_user_name" gorm:"-"`
	PoolName       string `json:"pool_name" gorm:"-"`
}

// MonitorOnDutyGroup 值班组的配置
type MonitorOnDutyGroup struct {
	Model
	Name                      string   `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:值班组名称"`
	UserID                    int      `json:"user_id" gorm:"comment:创建该值班组的用户ID"`
	Members                   []*User  `json:"members" gorm:"many2many:monitor_on_duty_users;comment:值班组成员列表，多对多关系"`
	ShiftDays                 int      `json:"shift_days" gorm:"type:int;comment:轮班周期，以天为单位"`
	YesterdayNormalDutyUserID int      `json:"yesterday_normal_duty_user_id" gorm:"comment:昨天的正常排班值班人ID，由cron任务设置"`
	CreateUserName            string   `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	TodayDutyUser             *User    `json:"today_duty_user" gorm:"-"`
	UserNames                 []string `json:"user_names" gorm:"-"`
}

// MonitorOnDutyHistory 值班历史记录
type MonitorOnDutyHistory struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	DateString     string `json:"date_string" gorm:"type:varchar(10);not null;comment:值班日期"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:当天值班人员ID"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原计划值班人员ID"`
	CreateUserName string `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	OnDutyUserName string `json:"on_duty_user_name" gorm:"-"`
	OriginUserName string `json:"origin_user_name" gorm:"-"`
	PoolName       string `json:"pool_name" gorm:"-"`
}

// GetMonitorOnDutyGroupListReq 获取值班组列表请求
type GetMonitorOnDutyGroupListReq struct {
	ListReq
	PoolID int   `json:"pool_id" form:"pool_id" binding:"omitempty"`
	Enable *int8 `json:"enable" form:"enable" binding:"omitempty"`
}

// CreateMonitorOnDutyGroupReq 创建值班组请求
type CreateMonitorOnDutyGroupReq struct {
	Name           string `json:"name" binding:"required,min=1,max=50"`
	UserID         int    `json:"user_id" binding:"required"`
	MemberIDs      []int  `json:"member_ids" binding:"required"`
	ShiftDays      int    `json:"shift_days" binding:"required"`
	CreateUserName string `json:"create_user_name"`
}

// CreateMonitorOnDutyGroupChangeReq 创建值班组换班记录请求
type CreateMonitorOnDutyGroupChangeReq struct {
	OnDutyGroupID  int    `json:"on_duty_group_id" binding:"required"`
	Date           string `json:"date" binding:"required"`
	OriginUserID   int    `json:"origin_user_id" binding:"required"`
	OnDutyUserID   int    `json:"on_duty_user_id" binding:"required"`
	UserID         int    `json:"user_id" binding:"required"`
	CreateUserName string `json:"create_user_name"`
}

// UpdateMonitorOnDutyGroupReq 更新值班组信息请求
type UpdateMonitorOnDutyGroupReq struct {
	ID        int    `json:"id" binding:"required" form:"id"`
	Name      string `json:"name" binding:"required,min=1,max=50"`
	ShiftDays int    `json:"shift_days" binding:"required"`
	MemberIDs []int  `json:"member_ids" binding:"required"`
}

// DeleteMonitorOnDutyGroupReq 删除值班组请求
type DeleteMonitorOnDutyGroupReq struct {
	ID int `json:"id" binding:"required" form:"id"`
}

// GetMonitorOnDutyGroupReq 获取指定值班组信息请求
type GetMonitorOnDutyGroupReq struct {
	ID int `json:"id" binding:"required" form:"id"`
}

// GetMonitorOnDutyGroupFuturePlanReq 获取值班组未来计划请求
type GetMonitorOnDutyGroupFuturePlanReq struct {
	ID        int    `json:"id" binding:"required" form:"id"`
	StartTime string `json:"start_time" binding:"required" form:"start_time"`
	EndTime   string `json:"end_time" binding:"required" form:"end_time"`
}

type OnDutyPlanResp struct {
	Details       []OnDutyOne       `json:"details"`
	Map           map[string]string `json:"map"`
	UserNameMap   map[string]string `json:"user_name_map"`
	OriginUserMap map[string]string `json:"origin_user_map"`
}

type OnDutyOne struct {
	Date       string `json:"date"`
	User       *User  `json:"user"`
	OriginUser string `json:"origin_user"` // 原始用户名
}
