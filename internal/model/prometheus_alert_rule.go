package model

// MonitorAlertRule 告警规则的配置
type MonitorAlertRule struct {
	Model
	Name           string            `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:告警规则名称"`
	UserID         int               `json:"user_id" gorm:"index;not null;comment:创建该告警规则的用户ID"`
	PoolID         int               `json:"pool_id" gorm:"index;not null;comment:关联的Prometheus实例池ID"`
	SendGroupID    int               `json:"send_group_id" gorm:"index;not null;comment:关联的发送组ID"`
	IpAddress      string            `json:"ip_address" gorm:"size:255;comment:IP地址"`
	Enable         bool              `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用告警规则"`
	Expr           string            `json:"expr" gorm:"type:text;not null;comment:告警规则表达式"`
	Severity       string            `json:"severity" gorm:"size:50;default:'warning';not null;comment:告警级别(critical/warning/info)"`
	GrafanaLink    string            `json:"grafana_link" gorm:"type:text;comment:Grafana大盘链接"`
	ForTime        string            `json:"for_time" gorm:"size:50;default:'5m';not null;comment:持续时间"`
	Labels         StringList        `json:"labels" gorm:"type:text;comment:标签组(key=value)"`
	Annotations    StringList        `json:"annotations" gorm:"type:text;comment:注解(key=value)"`
	NodePath       string            `json:"node_path" gorm:"-"`
	PoolName       string            `json:"pool_name" gorm:"-"`
	SendGroupName  string            `json:"send_group_name" gorm:"-"`
	CreateUserName string            `json:"create_user_name" gorm:"-"`
	LabelsMap      map[string]string `json:"labels_map" gorm:"-"`
	AnnotationsMap map[string]string `json:"annotations_map" gorm:"-"`
}

type DeleteMonitorAlertRuleRequest struct {
	ID int `json:"id" binding:"required"`
}
