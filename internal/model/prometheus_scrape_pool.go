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

// MonitorScrapePool 采集池的配置
type MonitorScrapePool struct {
	Model
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:pool池名称"`
	UserID                int        `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	ScrapeInterval        int        `json:"scrape_interval" gorm:"default:30;type:smallint;not null;comment:采集间隔(秒)"`
	ScrapeTimeout         int        `json:"scrape_timeout" gorm:"default:10;type:smallint;not null;comment:采集超时(秒)"`
	RemoteTimeoutSeconds  int        `json:"remote_timeout_seconds" gorm:"default:5;type:smallint;not null;comment:远程写入超时(秒)"`
	SupportAlert          int8       `json:"support_alert" gorm:"type:tinyint(1);default:2;not null;comment:告警支持(1:启用,2:禁用)"`
	SupportRecord         int8       `json:"support_record" gorm:"type:tinyint(1);default:2;not null;comment:预聚合支持(1:启用,2:禁用)"`
	PrometheusInstances   StringList `json:"prometheus_instances" gorm:"type:text;comment:Prometheus实例ID列表"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;comment:AlertManager实例ID列表"`
	ExternalLabels        StringList `json:"external_labels" gorm:"type:text;comment:外部标签（格式：[key1=val1,key2=val2]）"`
	RemoteWriteUrl        string     `json:"remote_write_url" gorm:"size:512;comment:远程写入地址"`
	RemoteReadUrl         string     `json:"remote_read_url" gorm:"size:512;comment:远程读取地址"`
	AlertManagerUrl       string     `json:"alert_manager_url" gorm:"size:512;comment:AlertManager地址"`
	RuleFilePath          string     `json:"rule_file_path" gorm:"size:512;comment:告警规则文件路径"`
	RecordFilePath        string     `json:"record_file_path" gorm:"size:512;comment:记录规则文件路径"`
	CreateUserName        string     `json:"create_user_name" gorm:"type:varchar(50);comment:创建人名称"`
}

func (m *MonitorScrapePool) TableName() string {
	return "cl_monitor_scrape_pools"
}

type GetMonitorScrapePoolListReq struct {
	ListReq
	SupportAlert  *int8 `json:"support_alert" form:"support_alert" binding:"omitempty"`
	SupportRecord *int8 `json:"support_record" form:"support_record" binding:"omitempty"`
}

type CreateMonitorScrapePoolReq struct {
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:pool池名称"`
	UserID                int        `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	ScrapeInterval        int        `json:"scrape_interval" gorm:"default:30;type:smallint;not null;comment:采集间隔(秒)"`
	ScrapeTimeout         int        `json:"scrape_timeout" gorm:"default:10;type:smallint;not null;comment:采集超时(秒)"`
	RemoteTimeoutSeconds  int        `json:"remote_timeout_seconds" gorm:"default:5;type:smallint;not null;comment:远程写入超时(秒)"`
	SupportAlert          int8       `json:"support_alert" gorm:"type:tinyint(1);default:2;not null;comment:告警支持(1:启用,2:禁用)"`
	SupportRecord         int8       `json:"support_record" gorm:"type:tinyint(1);default:2;not null;comment:预聚合支持(1:启用,2:禁用)"`
	PrometheusInstances   StringList `json:"prometheus_instances" gorm:"type:text;comment:Prometheus实例ID列表"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;comment:AlertManager实例ID列表"`
	ExternalLabels        StringList `json:"external_labels" gorm:"type:text;comment:外部标签（格式：[key1=val1,key2=val2]）"`
	RemoteWriteUrl        string     `json:"remote_write_url" gorm:"size:512;comment:远程写入地址"`
	RemoteReadUrl         string     `json:"remote_read_url" gorm:"size:512;comment:远程读取地址"`
	AlertManagerUrl       string     `json:"alert_manager_url" gorm:"size:512;comment:AlertManager地址"`
	RuleFilePath          string     `json:"rule_file_path" gorm:"size:512;comment:告警规则文件路径"`
	RecordFilePath        string     `json:"record_file_path" gorm:"size:512;comment:记录规则文件路径"`
	CreateUserName        string     `json:"create_user_name" gorm:"type:varchar(50);comment:创建人名称"`
}

type UpdateMonitorScrapePoolReq struct {
	ID                    int        `json:"id" form:"id" binding:"required"`
	Name                  string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;not null;comment:pool池名称"`
	UserID                int        `json:"user_id" gorm:"index;not null;comment:所属用户ID"`
	ScrapeInterval        int        `json:"scrape_interval" gorm:"default:30;type:smallint;not null;comment:采集间隔(秒)"`
	ScrapeTimeout         int        `json:"scrape_timeout" gorm:"default:10;type:smallint;not null;comment:采集超时(秒)"`
	RemoteTimeoutSeconds  int        `json:"remote_timeout_seconds" gorm:"default:5;type:smallint;not null;comment:远程写入超时(秒)"`
	SupportAlert          int8       `json:"support_alert" gorm:"type:tinyint(1);default:2;not null;comment:告警支持(1:启用,2:禁用)"`
	SupportRecord         int8       `json:"support_record" gorm:"type:tinyint(1);default:2;not null;comment:预聚合支持(1:启用,2:禁用)"`
	PrometheusInstances   StringList `json:"prometheus_instances" gorm:"type:text;comment:Prometheus实例ID列表"`
	AlertManagerInstances StringList `json:"alert_manager_instances" gorm:"type:text;comment:AlertManager实例ID列表"`
	ExternalLabels        StringList `json:"external_labels" gorm:"type:text;comment:外部标签（格式：[key1=val1,key2=val2]）"`
	RemoteWriteUrl        string     `json:"remote_write_url" gorm:"size:512;comment:远程写入地址"`
	RemoteReadUrl         string     `json:"remote_read_url" gorm:"size:512;comment:远程读取地址"`
	AlertManagerUrl       string     `json:"alert_manager_url" gorm:"size:512;comment:AlertManager地址"`
	RuleFilePath          string     `json:"rule_file_path" gorm:"size:512;comment:告警规则文件路径"`
	RecordFilePath        string     `json:"record_file_path" gorm:"size:512;comment:记录规则文件路径"`
}

type DeleteMonitorScrapePoolReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type GetMonitorScrapePoolDetailReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
