package model

// MonitorAlertManagerPool AlertManager 实例池的配置
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
	CreateUserName        string     `json:"create_user_name" gorm:"-"`
}

type CreateMonitorAlertManagerPoolReq struct {
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:AlertManager实例名称"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;not null;comment:AlertManager实例列表"`
	UserID                int        `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	ResolveTimeout        string     `json:"resolve_timeout" gorm:"size:50;default:'5m';not null;comment:告警恢复超时时间"`
	GroupWait             string     `json:"group_wait" gorm:"size:50;default:'30s';not null;comment:首次告警等待时间"`
	GroupInterval         string     `json:"group_interval" gorm:"size:50;default:'5m';not null;comment:告警分组间隔时间"`
	RepeatInterval        string     `json:"repeat_interval" gorm:"size:50;default:'4h';not null;comment:重复告警间隔"`
	GroupBy               StringList `json:"group_by" gorm:"type:text;not null;comment:告警分组标签列表"`
	Receiver              string     `json:"receiver" gorm:"size:100;not null;comment:默认接收者"`
}

type UpdateMonitorAlertManagerPoolReq struct {
	ID                    int        `json:"id" binding:"required"`
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:AlertManager实例名称"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;not null;comment:AlertManager实例列表"`
	ResolveTimeout        string     `json:"resolve_timeout" gorm:"size:50;default:'5m';not null;comment:告警恢复超时时间"`
	GroupWait             string     `json:"group_wait" gorm:"size:50;default:'30s';not null;comment:首次告警等待时间"`
	GroupInterval         string     `json:"group_interval" gorm:"size:50;default:'5m';not null;comment:告警分组间隔时间"`
	RepeatInterval        string     `json:"repeat_interval" gorm:"size:50;default:'4h';not null;comment:重复告警间隔"`
	GroupBy               StringList `json:"group_by" gorm:"type:text;not null;comment:告警分组标签列表"`
	Receiver              string     `json:"receiver" gorm:"size:100;not null;comment:默认接收者"`
}

type DeleteMonitorAlertManagerPoolReq struct {
	ID int `json:"id" binding:"required"`
}

type GetMonitorAlertManagerPoolListReq struct {
	ListReq
}
