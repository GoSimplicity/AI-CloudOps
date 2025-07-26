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

// 配置类型常量
const (
	ConfigTypePrometheus   int8 = 1 // Prometheus 主配置
	ConfigTypeAlertManager int8 = 2 // AlertManager 主配置
	ConfigTypeAlertRule    int8 = 3 // 告警规则配置
	ConfigTypeRecordRule   int8 = 4 // 预聚合规则配置
	ConfigTypeWebhookFile  int8 = 5 // webhook file
)

// 配置状态常量
const (
	ConfigStatusActive   int8 = 1 // 激活状态
	ConfigStatusInactive int8 = 2 // 非激活状态
)

// MonitorConfig 监控配置模型 - 由 cache 自动生成和管理
type MonitorConfig struct {
	Model
	Name              string `json:"name" gorm:"size:100;not null;comment:配置名称"`
	PoolID            int    `json:"pool_id" gorm:"index;not null;comment:关联的池ID"`
	InstanceIP        string `json:"instance_ip" gorm:"size:45;not null;index;comment:实例IP地址"`
	ConfigType        int8   `json:"config_type" gorm:"type:tinyint;not null;index;comment:配置类型(1:Prometheus主配置,2:AlertManager主配置,3:告警规则,4:预聚合规则,5:webhook file)"`
	ConfigContent     string `json:"config_content" gorm:"type:longtext;not null;comment:配置内容(YAML格式)"`
	ConfigHash        string `json:"config_hash" gorm:"size:64;not null;index;comment:配置内容的哈希值"`
	Status            int8   `json:"status" gorm:"type:tinyint;default:1;not null;comment:配置状态(1:激活,2:非激活)"`
	LastGeneratedTime int64  `json:"last_generated_time" gorm:"not null;comment:最后生成时间(Unix时间戳)"`
}

func (m *MonitorConfig) TableName() string {
	return "cl_monitor_configs"
}

// GetMonitorConfigListReq 获取监控配置列表请求
type GetMonitorConfigListReq struct {
	ListReq
	PoolID     *int   `json:"pool_id" form:"pool_id" binding:"omitempty"`
	InstanceIP string `json:"instance_ip" form:"instance_ip" binding:"omitempty"`
	ConfigType *int8  `json:"config_type" form:"config_type" binding:"omitempty,oneof=1 2 3 4"`
	Status     *int8  `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}

// GetMonitorConfigReq 获取单个监控配置请求
type GetMonitorConfigReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetMonitorConfigByInstanceReq 通过实例获取监控配置请求
type GetMonitorConfigByInstanceReq struct {
	InstanceIP string `json:"instance_ip" form:"instance_ip" binding:"required"`
	ConfigType int8   `json:"config_type" form:"config_type" binding:"required,oneof=1 2 3 4 5"`
}

// CreateMonitorConfigReq 创建监控配置请求
type CreateMonitorConfigReq struct {
	Name          string `json:"name" binding:"required,min=1,max=100"`
	PoolID        int    `json:"pool_id" binding:"required"`
	InstanceIP    string `json:"instance_ip" binding:"required"`
	ConfigType    int8   `json:"config_type" binding:"required,oneof=1 2 3 4 5"`
	ConfigContent string `json:"config_content" binding:"required"`
	Status        int8   `json:"status" binding:"omitempty,oneof=1 2"`
}

// UpdateMonitorConfigReq 更新监控配置请求
type UpdateMonitorConfigReq struct {
	ID            int    `json:"id" binding:"required"`
	Name          string `json:"name" binding:"required,min=1,max=100"`
	PoolID        int    `json:"pool_id" binding:"omitempty"`
	InstanceIP    string `json:"instance_ip" binding:"omitempty"`
	ConfigType    int8   `json:"config_type" binding:"omitempty,oneof=1 2 3 4 5"`
	ConfigContent string `json:"config_content" binding:"omitempty"`
	Status        int8   `json:"status" binding:"omitempty,oneof=1 2"`
}

// DeleteMonitorConfigReq 删除监控配置请求
type DeleteMonitorConfigReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
