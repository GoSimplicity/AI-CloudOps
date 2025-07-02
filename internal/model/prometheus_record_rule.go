package model

// MonitorRecordRule 记录规则的配置
type MonitorRecordRule struct {
	Model
	Name           string            `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:记录规则名称"`
	UserID         int               `json:"user_id" gorm:"index;not null;comment:创建该记录规则的用户ID"`
	PoolID         int               `json:"pool_id" gorm:"index;not null;comment:关联的Prometheus实例池ID"`
	IpAddress      string            `json:"ip_address" gorm:"size:255;comment:IP地址"`
	Enable         bool              `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用记录规则"`
	ForTime        string            `json:"for_time" gorm:"size:50;default:'5m';not null;comment:持续时间"`
	Expr           string            `json:"expr" gorm:"type:text;not null;comment:记录规则表达式"`
	Labels         StringList        `json:"labels" gorm:"type:text;comment:标签组(key=value)"`
	Annotations    StringList        `json:"annotations" gorm:"type:text;comment:注解(key=value)"`
	NodePath       string            `json:"node_path" gorm:"-"`
	PoolName       string            `json:"pool_name" gorm:"-"`
	SendGroupName  string            `json:"send_group_name" gorm:"-"`
	CreateUserName string            `json:"create_user_name" gorm:"-"`
	LabelsMap      map[string]string `json:"labels_map" gorm:"-"`
	AnnotationsMap map[string]string `json:"annotations_map" gorm:"-"`
}

type DeleteMonitorRecordRuleRequest struct {
	ID int `json:"id" binding:"required"`
}

type EnableSwitchMonitorRecordRuleRequest struct {
	ID int `json:"id" binding:"required"`
}
