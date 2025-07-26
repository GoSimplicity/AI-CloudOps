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

import "time"

// ResourceElb 负载均衡资源
type ResourceElb struct {
	Model

	InstanceName       string        `json:"instance_name" gorm:"type:varchar(100);comment:资源实例名称"`
	InstanceId         string        `json:"instance_id" gorm:"type:varchar(100);comment:资源实例ID"`
	Provider           CloudProvider `json:"cloud_provider" gorm:"type:varchar(50);comment:云厂商"`
	RegionId           string        `json:"region_id" gorm:"type:varchar(50);comment:地区，如cn-hangzhou"`
	ZoneId             string        `json:"zone_id" gorm:"type:varchar(100);comment:可用区ID"`
	VpcId              string        `json:"vpc_id" gorm:"type:varchar(100);comment:VPC ID"`
	Status             string        `json:"status" gorm:"type:varchar(50);comment:资源状态"`
	CreationTime       string        `json:"creation_time" gorm:"type:varchar(30);comment:创建时间,ISO8601格式"`
	Env                string        `json:"environment" gorm:"type:varchar(50);comment:环境标识,如dev,prod"`
	InstanceChargeType string        `json:"instance_charge_type" gorm:"type:varchar(50);comment:付费类型"`
	Description        string        `json:"description" gorm:"type:text;comment:资源描述"`
	Tags               StringList    `json:"tags" gorm:"type:varchar(500);comment:资源标签集合"`
	SecurityGroupIds   StringList    `json:"security_group_ids" gorm:"type:varchar(500);comment:安全组ID列表"`
	PrivateIpAddress   StringList    `json:"private_ip_address" gorm:"type:varchar(500);comment:私有IP地址"`
	PublicIpAddress    StringList    `json:"public_ip_address" gorm:"type:varchar(500);comment:公网IP地址"`

	// 资源创建和管理标志
	CreateByOrder      bool       `json:"create_by_order" gorm:"comment:是否由工单创建"`
	LastSyncTime       time.Time  `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID         int        `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
	LoadBalancerType   string     `json:"loadBalancerType" gorm:"type:varchar(50);comment:负载均衡类型,如nlb,alb,clb"`
	BandwidthCapacity  int        `json:"bandwidthCapacity" gorm:"comment:带宽容量上限,单位Mb"`
	AddressType        string     `json:"addressType" gorm:"type:varchar(50);comment:地址类型,公网或内网"`
	DNSName            string     `json:"dnsName" gorm:"type:varchar(255);comment:DNS解析地址"`
	BandwidthPackageId string     `json:"bandwidthPackageId" gorm:"type:varchar(100);comment:带宽包ID"`
	CrossZoneEnabled   bool       `json:"crossZoneEnabled" gorm:"comment:是否启用跨可用区"`
	ListenerPorts      StringList `json:"listenerPorts" gorm:"type:varchar(500);comment:监听端口列表"`
	BackendServers     StringList `json:"backendServers" gorm:"type:varchar(1000);comment:后端服务器列表"`

	// 多对多关系
	ElbTreeNodes []*TreeNode `json:"elbTreeNodes" gorm:"many2many:cl_elb_tree_nodes;comment:关联服务树节点"`
}

func (r *ResourceElb) TableName() string {
	return "cl_tree_elb"
}

// ListElbResourcesReq ELB资源列表查询参数
type ListElbResourcesReq struct {
	PageNumber int           `form:"pageNumber" json:"pageNumber" binding:"min=1"`
	PageSize   int           `form:"pageSize" json:"pageSize" binding:"min=1,max=100"`
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
	Status     string        `form:"status" json:"status"`
	Env        string        `form:"env" json:"env"`
	TreeNodeID int           `form:"treeNodeId" json:"treeNodeId"`
	Keyword    string        `form:"keyword" json:"keyword"` // 搜索关键字，可以是实例名称或ID
}

// GetElbDetailReq 获取ELB详情请求
type GetElbDetailReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// CreateElbResourceReq 创建ELB资源请求
type CreateElbResourceReq struct {
	InstanceName       string        `json:"instance_name" binding:"required,max=100"`
	Provider           CloudProvider `json:"provider" binding:"required"`
	RegionId           string        `json:"region_id" binding:"required"`
	ZoneId             string        `json:"zone_id" binding:"required"`
	VpcId              string        `json:"vpc_id" binding:"required"`
	LoadBalancerType   string        `json:"loadBalancerType" binding:"required,oneof=nlb alb clb"`
	AddressType        string        `json:"addressType" binding:"required,oneof=internet intranet"`
	BandwidthCapacity  int           `json:"bandwidthCapacity" binding:"min=1"`
	TreeNodeID         int           `json:"tree_node_id" binding:"required"`
	Description        string        `json:"description" binding:"max=500"`
	Tags               StringList    `json:"tags"`
	SecurityGroupIds   StringList    `json:"security_group_ids"`
	Env                string        `json:"environment" binding:"required"`
	InstanceChargeType string        `json:"instance_charge_type" binding:"required,oneof=PrePaid PostPaid"`
	CrossZoneEnabled   bool          `json:"crossZoneEnabled"`
	BandwidthPackageId string        `json:"bandwidthPackageId"`
}

// UpdateElbReq 更新ELB请求
type UpdateElbReq struct {
	ID                 int        `json:"id" uri:"id" binding:"required"`
	InstanceName       string     `json:"instance_name" binding:"max=100"`
	Description        string     `json:"description" binding:"max=500"`
	Tags               StringList `json:"tags"`
	SecurityGroupIds   StringList `json:"security_group_ids"`
	BandwidthCapacity  int        `json:"bandwidthCapacity" binding:"min=0"`
	CrossZoneEnabled   *bool      `json:"crossZoneEnabled"` // 使用指针以区分零值和未设置
	BandwidthPackageId string     `json:"bandwidthPackageId"`
}

// DeleteElbReq 删除ELB请求
type DeleteElbReq struct {
	ID    int  `json:"id" uri:"id" binding:"required"`
	Force bool `json:"force"` // 是否强制删除
}

// StartElbReq 启动ELB请求
type StartElbReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// StopElbReq 停止ELB请求
type StopElbReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// RestartElbReq 重启ELB请求
type RestartElbReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// ResizeElbReq 调整ELB规格请求
type ResizeElbReq struct {
	ID                int    `json:"id" uri:"id" binding:"required"`
	BandwidthCapacity int    `json:"bandwidthCapacity" binding:"required,min=1"`
	LoadBalancerType  string `json:"loadBalancerType" binding:"oneof=nlb alb clb"`
}

// BindServersToElbReq 绑定服务器到ELB请求
type BindServersToElbReq struct {
	ElbID     int    `json:"elb_id" binding:"required"`
	ServerIDs []int  `json:"server_ids" binding:"required,min=1"`
	Ports     []int  `json:"ports" binding:"required,min=1"` // 端口列表
	Weight    int    `json:"weight" binding:"min=1,max=100"` // 权重，默认为50
	Type      string `json:"type" binding:"oneof=ecs rds"`   // 服务器类型
}

// UnbindServersFromElbReq 从ELB解绑服务器请求
type UnbindServersFromElbReq struct {
	ElbID     int   `json:"elb_id" binding:"required"`
	ServerIDs []int `json:"server_ids" binding:"required,min=1"`
	Ports     []int `json:"ports"`
}

// ConfigureHealthCheckReq 配置健康检查请求
type ConfigureHealthCheckReq struct {
	ID                  int    `json:"id" uri:"id" binding:"required"`
	HealthCheckEnabled  bool   `json:"healthCheckEnabled"`
	HealthCheckType     string `json:"healthCheckType" binding:"oneof=tcp http https"`
	HealthCheckPort     int    `json:"healthCheckPort" binding:"min=1,max=65535"`
	HealthCheckPath     string `json:"healthCheckPath"`     // HTTP/HTTPS检查路径
	HealthCheckInterval int    `json:"healthCheckInterval"` // 检查间隔，单位秒
	HealthCheckTimeout  int    `json:"healthCheckTimeout"`  // 超时时间，单位秒
	HealthyThreshold    int    `json:"healthyThreshold"`    // 健康阈值
	UnhealthyThreshold  int    `json:"unhealthyThreshold"`  // 不健康阈值
	HealthCheckHttpCode string `json:"healthCheckHttpCode"` // HTTP状态码
	HealthCheckDomain   string `json:"healthCheckDomain"`   // 检查域名
}

// ElbListener 监听器信息
type ElbListener struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Status   string `json:"status"`
}

type ElbRule struct {
	ID            int    `json:"id"`
	ListenerID    int    `json:"listener_id"`
	RuleType      string `json:"rule_type"`
	RuleName      string `json:"rule_name"`
	RulePriority  int    `json:"rule_priority"`
	RuleCondition string `json:"rule_condition"`
	RuleAction    string `json:"rule_action"`
	RuleStatus    string `json:"rule_status"`
}

// ElbBackendInfo 后端服务器信息
type ElbBackendInfo struct {
	ServerID   int    `json:"serverId"`
	ServerName string `json:"serverName"`
	Port       int    `json:"port"`
	Weight     int    `json:"weight"`
	Status     string `json:"status"`
	Type       string `json:"type"`
}

// ElbHealthCheck 健康检查配置
type ElbHealthCheck struct {
	Enabled            bool   `json:"enabled"`
	Type               string `json:"type"`
	Port               int    `json:"port"`
	Path               string `json:"path"`
	Interval           int    `json:"interval"`
	Timeout            int    `json:"timeout"`
	HealthyThreshold   int    `json:"healthyThreshold"`
	UnhealthyThreshold int    `json:"unhealthyThreshold"`
	HttpCode           string `json:"httpCode"`
	Domain             string `json:"domain"`
}
