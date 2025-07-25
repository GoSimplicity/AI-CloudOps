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
	Name                      string               `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:值班组名称"`
	UserID                    int                  `json:"user_id" gorm:"index;comment:创建该值班组的用户ID"`
	ShiftDays                 int                  `json:"shift_days" gorm:"type:int;not null;default:7;comment:轮班周期，以天为单位"`
	YesterdayNormalDutyUserID int                  `json:"yesterday_normal_duty_user_id" gorm:"comment:昨天的正常排班值班人ID，由cron任务设置"`
	CreateUserName            string               `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	Members                   []*MonitorOnDutyUser `json:"members" gorm:"many2many:monitor_on_duty_users;comment:值班组成员列表，多对多关系"`
	TodayDutyUser             *MonitorOnDutyUser   `json:"today_duty_user" gorm:"-"`
	DutyPlans                 []*MonitorOnDutyPlan `json:"duty_plans" gorm:"foreignKey:OnDutyGroupID;references:ID;comment:值班计划列表"`
	Enable                    int8                 `json:"enable" gorm:"type:tinyint(1);not null;default:1;comment:是否启用 1-启用 0-禁用"`
	Description               string               `json:"description" gorm:"type:varchar(255);comment:值班组描述"`
}

// MonitorOnDutyPlan 值班计划表
type MonitorOnDutyPlan struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	Date           string `json:"date" gorm:"type:varchar(10);not null;comment:值班日期"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:值班人员ID"`
	IsAdjusted     bool   `json:"is_adjusted" gorm:"type:tinyint(1);not null;default:0;comment:是否为调整后的值班安排"`
	OriginalUserID int    `json:"original_user_id" gorm:"index;comment:原计划值班人员ID，仅当is_adjusted为true时有值"`
	Status         int    `json:"status" gorm:"type:int;not null;default:1;comment:计划状态 1-生效中 2-已过期 3-未开始"`
	CreateUserID   int    `json:"create_user_id" gorm:"index;comment:创建者ID"`
	CreateUserName string `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	UpdateUserID   int    `json:"update_user_id" gorm:"comment:更新者ID"`
	UpdateUserName string `json:"update_user_name" gorm:"type:varchar(100);comment:更新者名称"`
	Remark         string `json:"remark" gorm:"type:varchar(255);comment:备注信息"`
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

// MonitorOnDutyHistory 值班历史记录
type MonitorOnDutyHistory struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	DateString     string `json:"date_string" gorm:"type:varchar(10);not null;comment:值班日期"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:当天值班人员ID"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原计划值班人员ID"`
	CreateUserName string `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
}

// MonitorOnDutyOne 单日值班信息
type MonitorOnDutyOne struct {
	Date       string             `json:"date"`        // 值班日期
	User       *MonitorOnDutyUser `json:"user"`        // 值班人信息
	OriginUser string             `json:"origin_user"` // 原始值班人姓名
}

type MonitorOnDutyUser struct {
	ID           int    `json:"id" gorm:"index;comment:用户ID"`
	RealName     string `json:"real_name" gorm:"type:varchar(100);not null;comment:用户真实姓名"`
	Username     string `json:"username" gorm:"type:varchar(100);not null;comment:用户名"`
	FeiShuUserId string `json:"fei_shu_user_id" gorm:"type:varchar(100);comment:飞书用户ID"`
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
	MemberIDs      []int  `json:"member_ids" binding:"required,min=1"`
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
	ID          int    `json:"id" binding:"required" form:"id"`
	Name        string `json:"name" binding:"required,min=1,max=50"`
	ShiftDays   int    `json:"shift_days" binding:"required,min=1"`
	MemberIDs   []int  `json:"member_ids" binding:"required,min=1"`
	Description string `json:"description"`
	Enable      *int8  `json:"enable"`
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

// GetMonitorOnDutyHistoryReq 获取值班历史记录请求
type GetMonitorOnDutyHistoryReq struct {
	OnDutyGroupID int    `json:"on_duty_group_id" form:"on_duty_group_id" binding:"required"`
	StartDate     string `json:"start_date" form:"start_date" binding:"required"`
	EndDate       string `json:"end_date" form:"end_date" binding:"required"`
}
