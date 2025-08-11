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

type AlertRuleSeverity int8

const (
	AlertRuleSeverityInfo AlertRuleSeverity = iota + 1
	AlertRuleSeverityWarning
	AlertRuleSeverityCritical
)

// MonitorAlertRule 告警规则的配置
type MonitorAlertRule struct {
	Model
	Name           string            `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:告警规则名称"`
	UserID         int               `json:"user_id" gorm:"index;not null;comment:创建该告警规则的用户ID"`
	PoolID         int               `json:"pool_id" gorm:"index;not null;comment:关联的Prometheus实例池ID"`
	SendGroupID    int               `json:"send_group_id" gorm:"index;not null;comment:关联的发送组ID"`
	IpAddress      string            `json:"ip_address" gorm:"size:255;comment:IP地址"`
	Enable         int8              `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用告警规则(1:启用,2:禁用)"`
	Expr           string            `json:"expr" gorm:"type:text;not null;comment:告警规则表达式"`
	Severity       AlertRuleSeverity `json:"severity" gorm:"type:tinyint(1);default:2;not null;comment:告警级别(1:info,2:warning,3:critical)"`
	GrafanaLink    string            `json:"grafana_link" gorm:"type:text;comment:Grafana大盘链接"`
	ForTime        string            `json:"for_time" gorm:"size:50;default:'5m';not null;comment:持续时间"`
	Labels         StringList        `json:"labels" gorm:"type:text;comment:标签组(key=value)"`
	Annotations    StringList        `json:"annotations" gorm:"type:text;comment:注解(key=value)"`
	CreateUserName string            `json:"create_user_name" gorm:"type:varchar(100);not null;comment:创建者名称"`
	PoolName       string            `json:"pool_name" gorm:"-"`
	SendGroupName  string            `json:"send_group_name" gorm:"-"`
}

func (m *MonitorAlertRule) TableName() string {
	return "cl_monitor_alert_rules"
}

// GetMonitorAlertRuleListReq 获取告警规则列表的请求
type GetMonitorAlertRuleListReq struct {
	ListReq
	Enable   *int8              `json:"enable" form:"enable" binding:"omitempty"`
	Severity *AlertRuleSeverity `json:"severity" form:"severity" binding:"omitempty"`
}

// CreateMonitorAlertRuleReq 创建告警规则请求
type CreateMonitorAlertRuleReq struct {
	Name           string            `json:"name" binding:"required,min=1,max=50"`
	UserID         int               `json:"user_id"`
	PoolID         int               `json:"pool_id" binding:"required"`
	SendGroupID    int               `json:"send_group_id" binding:"required"`
	IpAddress      string            `json:"ip_address"`
	Enable         int8              `json:"enable"`
	Expr           string            `json:"expr" binding:"required"`
	Severity       AlertRuleSeverity `json:"severity" binding:"omitempty"`
	GrafanaLink    string            `json:"grafana_link"`
	ForTime        string            `json:"for_time" binding:"required"`
	Labels         StringList        `json:"labels"`
	Annotations    StringList        `json:"annotations"`
	CreateUserName string            `json:"create_user_name"`
}

// UpdateMonitorAlertRuleReq 更新告警规则请求
type UpdateMonitorAlertRuleReq struct {
	ID          int               `json:"id" form:"id" binding:"required"`
	Name        string            `json:"name" binding:"required,min=1,max=50"`
	PoolID      int               `json:"pool_id" binding:"required"`
	SendGroupID int               `json:"send_group_id" binding:"required"`
	IpAddress   string            `json:"ip_address"`
	Enable      int8              `json:"enable"`
	Expr        string            `json:"expr" binding:"required"`
	Severity    AlertRuleSeverity `json:"severity" binding:"omitempty"`
	GrafanaLink string            `json:"grafana_link"`
	ForTime     string            `json:"for_time" binding:"required"`
	Labels      StringList        `json:"labels"`
	Annotations StringList        `json:"annotations"`
}

// DeleteMonitorAlertRuleReq 删除告警规则请求
type DeleteMonitorAlertRuleReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// PromqlAlertRuleExprCheckReq PromQL表达式检查请求
type PromqlAlertRuleExprCheckReq struct {
	PromqlExpr string `json:"promql_expr" binding:"required"`
}

// GetMonitorAlertRuleReq 获取告警规则请求
type GetMonitorAlertRuleReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
