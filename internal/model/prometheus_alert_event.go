package model

import "github.com/prometheus/alertmanager/template"

// MonitorAlertEvent 告警事件与相关实体的关系
type MonitorAlertEvent struct {
	Model
	AlertName      string            `json:"alert_name" binding:"required,min=1,max=200" gorm:"size:200;not null;comment:告警名称"`
	Fingerprint    string            `json:"fingerprint" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:告警唯一ID"`
	Status         string            `json:"status" gorm:"size:50;not null;default:'firing';comment:告警状态(firing/silenced/claimed/resolved)"`
	RuleID         int               `json:"rule_id" gorm:"index;not null;comment:关联的告警规则ID"`
	SendGroupID    int               `json:"send_group_id" gorm:"index;not null;comment:关联的发送组ID"`
	EventTimes     int               `json:"event_times" gorm:"not null;default:1;comment:触发次数"`
	SilenceID      string            `json:"silence_id" gorm:"size:100;comment:AlertManager返回的静默ID"`
	RenLingUserID  int               `json:"ren_ling_user_id" gorm:"index;comment:认领告警的用户ID"`
	Labels         StringList        `json:"labels" gorm:"type:text;not null;comment:标签组,格式为key=value"`
	AlertRuleName  string            `json:"alert_rule_name" gorm:"-"`
	SendGroupName  string            `json:"send_group_name" gorm:"-"`
	Alert          template.Alert    `json:"alert" gorm:"-"`
	SendGroup      *MonitorSendGroup `json:"send_group" gorm:"-"`
	RenLingUser    *User             `json:"ren_ling_user" gorm:"-"`
	Rule           *MonitorAlertRule `json:"rule" gorm:"-"`
	LabelsMap      map[string]string `json:"labels_map" gorm:"-"`
	AnnotationsMap map[string]string `json:"annotations_map" gorm:"-"`
}

type EventAlertSilenceReq struct {
	ID      int    `json:"id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	UseName bool   `json:"use_name"` // 是否启用名称静默
	Time    string `json:"time"`
}

type GetMonitorAlertEventListReq struct {
	ListReq
	Status string `json:"status" form:"status" binding:"omitempty,oneof=firing silenced claimed resolved"`
}

type EventAlertClaimReq struct {
	ID     int `json:"id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}

type EventAlertUnSilenceReq struct {
	ID     int `json:"id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}

type PromqlExprCheckReq struct {
	PromqlExpr string `json:"promql_expr" binding:"required"`
}

type EnableSwitchMonitorAlertRuleReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
