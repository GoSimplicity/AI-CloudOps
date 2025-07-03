package model

// MonitorSendGroup 发送组的配置
type MonitorSendGroup struct {
	Model
	Name                   string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:发送组英文名称"`
	NameZh                 string     `json:"name_zh" binding:"required,min=1,max=50" gorm:"size:100;comment:发送组中文名称"`
	Enable                 bool       `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用发送组"`
	UserID                 int        `json:"user_id" gorm:"index;not null;comment:创建该发送组的用户ID"`
	PoolID                 int        `json:"pool_id" gorm:"index;not null;comment:关联的AlertManager实例ID"`
	OnDutyGroupID          int        `json:"on_duty_group_id" gorm:"index;comment:值班组ID"`
	StaticReceiveUsers     []*User    `json:"static_receive_users" gorm:"many2many:monitor_send_group_static_receive_users;comment:静态配置的接收人列表"`
	FeiShuQunRobotToken    string     `json:"fei_shu_qun_robot_token" gorm:"size:255;comment:飞书机器人Token"`
	RepeatInterval         string     `json:"repeat_interval" gorm:"size:50;default:'4h';comment:重复发送时间间隔"`
	SendResolved           bool       `json:"send_resolved" gorm:"type:tinyint(1);default:1;not null;comment:是否发送恢复通知"`
	NotifyMethods          StringList `json:"notify_methods" gorm:"type:text;comment:通知方法列表"` // 例如: ["email", "feishu", "dingtalk"]
	NeedUpgrade            bool       `json:"need_upgrade" gorm:"type:tinyint(1);default:0;not null;comment:是否需要告警升级"`
	FirstUpgradeUsers      []*User    `json:"monitor_send_group_first_upgrade_users" gorm:"many2many:monitor_send_group_first_upgrade_users;comment:第一级升级人列表"`
	UpgradeMinutes         int        `json:"upgrade_minutes" gorm:"default:30;comment:告警升级等待时间(分钟)"`
	SecondUpgradeUsers     []*User    `json:"second_upgrade_users" gorm:"many2many:monitor_send_group_second_upgrade_users;comment:第二级升级人列表"`
	StaticReceiveUserNames []string   `json:"static_receive_user_names" gorm:"-"`
	FirstUserNames         []string   `json:"first_user_names" gorm:"-"`
	SecondUserNames        []string   `json:"second_user_names" gorm:"-"`
	PoolName               string     `json:"pool_name" gorm:"-"`
	OnDutyGroupName        string     `json:"on_duty_group_name" gorm:"-"`
	CreateUserName         string     `json:"create_user_name" gorm:"-"`
}

type DeleteMonitorSendGroupRequest struct {
	ID int `json:"id" binding:"required"`
}

type GetMonitorSendGroupRequest struct {
	ID int `json:"id" binding:"required"`
}
