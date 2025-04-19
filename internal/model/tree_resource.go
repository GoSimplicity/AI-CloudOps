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

// ResourceBase 资源基础信息
type ResourceBase struct {
	Model

	InstanceName       string        `json:"instance_name" gorm:"uniqueIndex;type:varchar(100);comment:资源实例名称"`
	InstanceId         string        `json:"instance_id" gorm:"uniqueIndex;type:varchar(100);comment:资源实例ID"`
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
	CreateByOrder bool      `json:"create_by_order" gorm:"comment:是否由工单创建"`
	LastSyncTime  time.Time `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID    uint      `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
}

// ComputeResource 计算资源通用属性
type ComputeResource struct {
	ResourceBase
	Cpu          int    `json:"cpu" gorm:"comment:CPU核数"`
	Memory       int    `json:"memory" gorm:"comment:内存大小,单位GiB"`
	InstanceType string `json:"instanceType" gorm:"type:varchar(100);comment:实例类型"`
	ImageId      string `json:"imageId" gorm:"type:varchar(100);comment:镜像ID"`
	IpAddr       string `json:"ipAddr" gorm:"type:varchar(45);uniqueIndex;comment:主IP地址"`
	Port         int    `json:"port" gorm:"comment:端口号;default:22"`
	HostName     string `json:"hostname" gorm:"comment:主机名"`
	Password     string `json:"password" gorm:"type:varchar(500);comment:密码"`
	Key          string `json:"key" gorm:"comment:密钥"`
	AuthMode     string `json:"authMode" gorm:"comment:认证方式;default:password"` // password或key
}

// ResourceCreationRequest 资源创建请求
type ResourceCreationRequest struct {
	Model

	RequestType    string        `json:"requestType" gorm:"type:varchar(50);comment:请求类型,如ecs,elb,rds"`
	Provider       CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	Region         string        `json:"region" gorm:"type:varchar(50);comment:地区"`
	Status         string        `json:"status" gorm:"type:varchar(50);comment:请求状态"`
	TreeNodeId     uint          `json:"treeNodeId" gorm:"comment:关联的服务树节点ID"`
	CreatedBy      uint          `json:"createdBy" gorm:"comment:创建者ID"`
	ApprovedBy     uint          `json:"approvedBy" gorm:"comment:审批者ID"`
	ApprovedAt     time.Time     `json:"approvedAt" gorm:"comment:审批时间"`
	ExecutedAt     time.Time     `json:"executedAt" gorm:"comment:执行时间"`
	RequestContent string        `json:"requestContent" gorm:"type:text;comment:请求详情JSON"`
	ResultContent  string        `json:"resultContent" gorm:"type:text;comment:结果详情JSON"`
}

// DiskCreationParams 磁盘创建参数
type DiskCreationParams struct {
	Provider     CloudProvider     `json:"provider" binding:"required"`
	Region       string            `json:"region" binding:"required"`
	ZoneId       string            `json:"zoneId" binding:"required"`
	DiskName     string            `json:"diskName" binding:"required"`
	DiskCategory string            `json:"diskCategory" binding:"required"`
	Size         int               `json:"size" binding:"required,min=20"`
	VpcId        string            `json:"vpcId" binding:"required"`
	InstanceId   string            `json:"instanceId"`
	PayType      string            `json:"payType" binding:"required"`
	TreeNodeId   uint              `json:"treeNodeId" binding:"required"`
	Description  string            `json:"description"`
	Tags         map[string]string `json:"tags"`
}

// ResourceBindingRequest 资源绑定请求
type ResourceBindingRequest struct {
	NodeId       uint   `json:"nodeId" binding:"required"`
	ResourceIds  []uint `json:"resourceIds" binding:"required,min=1"`
	ResourceType string `json:"resourceType" binding:"required,oneof=ecs elb rds"`
}

// SyncResourcesReq 同步资源请求
type SyncResourcesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

// PageResp 分页响应
type PageResp struct {
	Total int64       `json:"total"`
	Data  interface{} `json:"data"`
}
