package model

// MonitorOnDutyChange 值班换班记录
type MonitorOnDutyChange struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	UserID         int    `json:"user_id" gorm:"index;comment:创建者ID"`
	Date           string `json:"date" gorm:"type:varchar(10);not null;comment:换班日期"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原值班人ID"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:新值班人ID"`
	TargetUserName string `json:"target_user_name" gorm:"-"`
	OriginUserName string `json:"origin_user_name" gorm:"-"`
	PoolName       string `json:"pool_name" gorm:"-"`
	CreateUserName string `json:"create_user_name" gorm:"-"`
}

// MonitorOnDutyGroup 值班组的配置
type MonitorOnDutyGroup struct {
	Model
	Name                      string   `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:值班组名称"`
	UserID                    int      `json:"user_id" gorm:"comment:创建该值班组的用户ID"`
	Members                   []*User  `json:"members" gorm:"many2many:monitor_on_duty_users;comment:值班组成员列表，多对多关系"`
	ShiftDays                 int      `json:"shift_days" gorm:"type:int;comment:轮班周期，以天为单位"`
	YesterdayNormalDutyUserID int      `json:"yesterday_normal_duty_user_id" gorm:"comment:昨天的正常排班值班人ID，由cron任务设置"`
	TodayDutyUser             *User    `json:"today_duty_user" gorm:"-"`
	UserNames                 []string `json:"user_names" gorm:"-"`
	CreateUserName            string   `json:"create_user_name" gorm:"-"`
}

// MonitorOnDutyHistory 值班历史记录
type MonitorOnDutyHistory struct {
	Model
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	DateString     string `json:"date_string" gorm:"type:varchar(10);not null;comment:值班日期"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:当天值班人员ID"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原计划值班人员ID"`
	OnDutyUserName string `json:"on_duty_user_name" gorm:"-"`
	OriginUserName string `json:"origin_user_name" gorm:"-"`
	PoolName       string `json:"pool_name" gorm:"-"`
	CreateUserName string `json:"create_user_name" gorm:"-"`
}

type DeleteMonitorOnDutyGroupRequest struct {
	ID int `json:"id" binding:"required"`
}

type GetMonitorOnDutyGroupReq struct {
	ID int `json:"id" binding:"required"`
}

type GetMonitorOnDutyGroupFuturePlanReq struct {
	ID        int    `json:"id" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
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
