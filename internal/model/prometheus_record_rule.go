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

// MonitorRecordRule 记录规则的配置
type MonitorRecordRule struct {
	Model
	Name           string     `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:记录规则名称"`
	UserID         int        `json:"user_id" gorm:"index;not null;comment:创建该记录规则的用户ID"`
	PoolID         int        `json:"pool_id" gorm:"index;not null;comment:关联的Prometheus实例池ID"`
	IpAddress      string     `json:"ip_address" gorm:"size:255;comment:IP地址"`
	Enable         int8       `json:"enable" gorm:"type:tinyint(1);default:1;not null;comment:是否启用记录规则 1:启用 2:禁用"`
	ForTime        string     `json:"for_time" gorm:"size:50;default:'5m';not null;comment:持续时间"`
	Expr           string     `json:"expr" gorm:"type:text;not null;comment:记录规则表达式"`
	Labels         StringList `json:"labels" gorm:"type:text;comment:标签组(key=value)"`
	Annotations    StringList `json:"annotations" gorm:"type:text;comment:注解(key=value)"`
	CreateUserName string     `json:"create_user_name" gorm:"type:varchar(100);comment:创建人"`
	PoolName       string     `json:"pool_name" gorm:"-"`
}

// GetMonitorRecordRuleListReq 获取预聚合规则列表请求
type GetMonitorRecordRuleListReq struct {
	ListReq
	PoolID *int  `json:"pool_id" form:"pool_id" binding:"omitempty"`
	Enable *int8 `json:"enable" form:"enable" binding:"omitempty"`
}

// CreateMonitorRecordRuleReq 创建新的预聚合规则请求
type CreateMonitorRecordRuleReq struct {
	Name           string     `json:"name" binding:"required,min=1,max=50"`
	UserID         int        `json:"user_id"`
	PoolID         int        `json:"pool_id" binding:"required"`
	IpAddress      string     `json:"ip_address"`
	Enable         int8       `json:"enable"`
	ForTime        string     `json:"for_time"`
	Expr           string     `json:"expr" binding:"required"`
	Labels         StringList `json:"labels"`
	Annotations    StringList `json:"annotations"`
	CreateUserName string     `json:"create_user_name"`
}

// UpdateMonitorRecordRuleReq 更新预聚合规则请求
type UpdateMonitorRecordRuleReq struct {
	ID          int        `json:"id" binding:"required"`
	Name        string     `json:"name" binding:"required,min=1,max=50"`
	PoolID      int        `json:"pool_id" binding:"required"`
	IpAddress   string     `json:"ip_address"`
	Enable      int8       `json:"enable"`
	ForTime     string     `json:"for_time"`
	Expr        string     `json:"expr" binding:"required"`
	Labels      StringList `json:"labels"`
	Annotations StringList `json:"annotations"`
}

// DeleteMonitorRecordRuleReq 删除预聚合规则请求
type DeleteMonitorRecordRuleReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// EnableSwitchMonitorRecordRuleReq 切换预聚合规则启用状态请求
type EnableSwitchMonitorRecordRuleReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type GetMonitorRecordRuleReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
