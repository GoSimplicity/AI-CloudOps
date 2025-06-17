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

import (
	"github.com/prometheus/alertmanager/template"
)

// MonitorScrapePool 采集池的配置
type MonitorScrapePool struct {
	ID                    int        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt             int64      `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt             int64      `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt             int64      `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:pool池名称"`
	PrometheusInstances   StringList `json:"prometheus_instances" gorm:"type:text;comment:Prometheus实例ID列表"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;comment:AlertManager实例ID列表"`
	UserID                int        `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	ScrapeInterval        int        `json:"scrape_interval" gorm:"default:30;type:smallint;not null;comment:采集间隔(秒)"`
	ScrapeTimeout         int        `json:"scrape_timeout" gorm:"default:10;type:smallint;not null;comment:采集超时(秒)"`
	RemoteTimeoutSeconds  int        `json:"remote_timeout_seconds" gorm:"default:5;type:smallint;not null;comment:远程写入超时(秒)"`
	SupportAlert          bool       `json:"support_alert" gorm:"type:tinyint(1);default:0;not null;comment:告警支持(0:不支持,1:支持)"`
	SupportRecord         bool       `json:"support_record" gorm:"type:tinyint(1);default:0;not null;comment:预聚合支持(0:不支持,1:支持)"`
	ExternalLabels        StringList `json:"external_labels" gorm:"type:text;comment:外部标签（格式：[key1=val1,key2=val2]）"`
	RemoteWriteUrl        string     `json:"remote_write_url" gorm:"size:512;comment:远程写入地址"`
	RemoteReadUrl         string     `json:"remote_read_url" gorm:"size:512;comment:远程读取地址"`
	AlertManagerUrl       string     `json:"alert_manager_url" gorm:"size:512;comment:AlertManager地址"`
	RuleFilePath          string     `json:"rule_file_path" gorm:"size:512;comment:告警规则文件路径"`
	RecordFilePath        string     `json:"record_file_path" gorm:"size:512;comment:记录规则文件路径"`
	CreateUserName        string     `json:"create_user_name" gorm:"-"`
}

// MonitorScrapeJob 监控采集任务的配置
type MonitorScrapeJob struct {
	ID                       int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt                int64  `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt                int64  `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt                int64  `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
	Name                     string `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:采集任务名称"`
	UserID                   int    `json:"user_id" gorm:"index;not null;comment:任务关联的用户ID"`
	Enable                   bool   `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用采集任务"`
	ServiceDiscoveryType     string `json:"service_discovery_type" gorm:"size:50;not null;default:'http';comment:服务发现类型(k8s/http)"`
	MetricsPath              string `json:"metrics_path" gorm:"size:255;not null;default:'/metrics';comment:监控采集的路径"`
	Scheme                   string `json:"scheme" gorm:"size:10;not null;default:'http';comment:监控采集的协议方案(http/https)"`
	ScrapeInterval           int    `json:"scrape_interval" gorm:"default:30;not null;comment:采集的时间间隔(秒)"`
	ScrapeTimeout            int    `json:"scrape_timeout" gorm:"default:10;not null;comment:采集的超时时间(秒)"`
	PoolID                   int    `json:"pool_id" gorm:"index;not null;comment:关联的采集池ID"`
	RelabelConfigsYamlString string `json:"relabel_configs_yaml_string" gorm:"type:text;comment:relabel配置的YAML字符串"`
	RefreshInterval          int    `json:"refresh_interval" gorm:"default:300;not null;comment:刷新目标的时间间隔(秒)"`
	Port                     int    `json:"port" gorm:"default:9090;not null;comment:采集端口号"`
	IpAddress                string `json:"ip_address" gorm:"size:255;comment:IP地址"`
	KubeConfigFilePath       string `json:"kube_config_file_path" gorm:"size:255;comment:K8s配置文件路径"`
	TlsCaFilePath            string `json:"tls_ca_file_path" gorm:"size:255;comment:TLS CA证书文件路径"`
	TlsCaContent             string `json:"tls_ca_content" gorm:"type:text;comment:TLS CA证书内容"`
	BearerToken              string `json:"bearer_token" gorm:"type:text;comment:鉴权Token内容"`
	BearerTokenFile          string `json:"bearer_token_file" gorm:"size:255;comment:鉴权Token文件路径"`
	KubernetesSdRole         string `json:"kubernetes_sd_role" gorm:"size:50;default:'pod';comment:K8s服务发现角色"`
	CreateUserName           string `json:"create_user_name" gorm:"-"`
}

// MonitorAlertManagerPool AlertManager 实例池的配置
type MonitorAlertManagerPool struct {
	ID                    int        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt             int64      `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt             int64      `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt             int64      `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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

// MonitorAlertRule 告警规则的配置
type MonitorAlertRule struct {
	ID             int               `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt      int64             `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      int64             `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt      int64             `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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

// MonitorAlertEvent 告警事件与相关实体的关系
type MonitorAlertEvent struct {
	ID             int               `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt      int64             `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      int64             `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt      int64             `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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

// MonitorRecordRule 记录规则的配置
type MonitorRecordRule struct {
	ID             int               `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt      int64             `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      int64             `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt      int64             `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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

// MonitorSendGroup 发送组的配置
type MonitorSendGroup struct {
	ID                     int        `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt              int64      `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt              int64      `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt              int64      `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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

// MonitorOnDutyChange 值班换班记录
type MonitorOnDutyChange struct {
	ID             int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt      int64  `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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
	ID                        int      `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt                 int64    `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt                 int64    `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt                 int64    `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
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
	ID             int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	CreatedAt      int64  `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      int64  `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt      int64  `json:"deleted_at" gorm:"index:idx_deleted_at;default:0;comment:删除时间"`
	OnDutyGroupID  int    `json:"on_duty_group_id" gorm:"index:idx_group_date_deleted_at;comment:值班组ID"`
	DateString     string `json:"date_string" gorm:"type:varchar(10);not null;comment:值班日期"`
	OnDutyUserID   int    `json:"on_duty_user_id" gorm:"index;comment:当天值班人员ID"`
	OriginUserID   int    `json:"origin_user_id" gorm:"index;comment:原计划值班人员ID"`
	OnDutyUserName string `json:"on_duty_user_name" gorm:"-"`
	OriginUserName string `json:"origin_user_name" gorm:"-"`
	PoolName       string `json:"pool_name" gorm:"-"`
	CreateUserName string `json:"create_user_name" gorm:"-"`
}

type AlertEventSilenceRequest struct {
	ID      int    `json:"id" binding:"required"`
	UserID  int    `json:"user_id" binding:"required"`
	UseName bool   `json:"use_name"` // 是否启用名称静默
	Time    string `json:"time"`
}

type AlertEventClaimRequest struct {
	ID     int `json:"id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}

type AlertEventUnSilenceRequest struct {
	ID     int `json:"id" binding:"required"`
	UserID int `json:"user_id" binding:"required"`
}

type BatchEventAlertSilenceRequest struct {
	IDs    []int `json:"ids" binding:"required"`
	UserID int   `json:"user_id" binding:"required"`
	AlertEventSilenceRequest
}

type BatchRequest struct {
	IDs []int `json:"ids" binding:"required"`
}

type PromqlExprCheckReq struct {
	PromqlExpr string `json:"promql_expr" binding:"required"`
}

type IdRequest struct {
	ID int `json:"id" binding:"required"`
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

type DeleteMonitorAlertManagerPoolRequest struct {
	ID int `json:"id" binding:"required"`
}

type GetMonitorAlertManagerPoolTotalRequest struct {
	ID int `json:"id" binding:"required"`
}

type DeleteMonitorAlertRuleRequest struct {
	ID int `json:"id" binding:"required"`
}