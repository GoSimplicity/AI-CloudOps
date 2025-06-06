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
	"time"
)

// ResourceRds 数据库资源实体
type ResourceRds struct {
	Model

	// 基础信息
	InstanceName       string        `json:"instance_name" gorm:"type:varchar(100);comment:资源实例名称"`
	InstanceId         string        `json:"instance_id" gorm:"type:varchar(100);unique;comment:资源实例ID"`
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

	// 网络信息
	SecurityGroupIds StringList `json:"security_group_ids" gorm:"type:varchar(500);comment:安全组ID列表"`
	PrivateIpAddress StringList `json:"private_ip_address" gorm:"type:varchar(500);comment:私有IP地址"`
	PublicIpAddress  StringList `json:"public_ip_address" gorm:"type:varchar(500);comment:公网IP地址"`

	// RDS特有属性
	Engine              string `json:"engine" gorm:"type:varchar(50);comment:数据库引擎类型,如mysql,postgresql"`
	EngineVersion       string `json:"engine_version" gorm:"type:varchar(50);comment:数据库版本,如8.0,5.7"`
	DBInstanceClass     string `json:"db_instance_class" gorm:"type:varchar(100);comment:实例规格"`
	DBInstanceType      string `json:"db_instance_type" gorm:"type:varchar(50);comment:实例类型,如Primary,Readonly"`
	DBInstanceNetType   string `json:"db_instance_net_type" gorm:"type:varchar(50);comment:实例网络类型"`
	MasterInstanceId    string `json:"master_instance_id" gorm:"type:varchar(100);comment:主实例ID"`
	ReplicateId         string `json:"replicate_id" gorm:"type:varchar(100);comment:复制实例ID"`
	DBStatus            string `json:"db_status" gorm:"type:varchar(50);comment:数据库状态"`
	Port                int    `json:"port" gorm:"comment:数据库端口;default:3306"`
	ConnectionString    string `json:"connection_string" gorm:"type:varchar(255);comment:连接字符串"`
	AllocatedStorage    int    `json:"allocated_storage" gorm:"comment:分配存储空间(GB)"`
	MaxConnections      int    `json:"max_connections" gorm:"comment:最大连接数"`
	BackupRetentionDays int    `json:"backup_retention_days" gorm:"comment:备份保留天数"`
	PreferredBackupTime string `json:"preferred_backup_time" gorm:"type:varchar(20);comment:首选备份时间"`
	MaintenanceWindow   string `json:"maintenance_window" gorm:"type:varchar(50);comment:维护时间窗口"`

	// 管理信息
	CreateByOrder bool      `json:"create_by_order" gorm:"comment:是否由工单创建"`
	LastSyncTime  time.Time `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID    int       `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`

	// 多对多关系
	RdsTreeNodes []*TreeNode `json:"rds_tree_nodes" gorm:"many2many:resource_rds_tree_nodes;comment:关联服务树节点"`
}

// TableName 指定表名
func (ResourceRds) TableName() string {
	return "resource_rds"
}

// ====================== 请求结构体定义 ======================

// ListRdsResourcesReq 获取RDS实例列表请求
type ListRdsResourcesReq struct {
	PageNumber   int           `form:"pageNumber" json:"pageNumber" binding:"min=1"`
	PageSize     int           `form:"pageSize" json:"pageSize" binding:"min=1,max=100"`
	Provider     CloudProvider `form:"provider" json:"provider"`
	Region       string        `form:"region" json:"region"`
	ZoneId       string        `form:"zoneId" json:"zoneId"`
	Status       string        `form:"status" json:"status"`
	Engine       string        `form:"engine" json:"engine"`
	TreeNodeId   int           `form:"treeNodeId" json:"treeNodeId"`
	InstanceName string        `form:"instanceName" json:"instanceName"`
	Environment  string        `form:"environment" json:"environment"`
}

// GetRdsDetailReq 获取RDS实例详情请求
type GetRdsDetailReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// CreateRdsResourceReq 创建RDS实例请求
type CreateRdsResourceReq struct {
	InstanceName        string            `json:"instanceName" binding:"required,min=2,max=100"`
	Provider            CloudProvider     `json:"provider" binding:"required"`
	Region              string            `json:"region" binding:"required"`
	ZoneId              string            `json:"zoneId" binding:"required"`
	Engine              string            `json:"engine" binding:"required"`
	EngineVersion       string            `json:"engineVersion" binding:"required"`
	DBInstanceClass     string            `json:"dbInstanceClass" binding:"required"`
	VpcId               string            `json:"vpcId" binding:"required"`
	DBInstanceNetType   string            `json:"dbInstanceNetType" binding:"required"`
	InstanceChargeType  string            `json:"instanceChargeType" binding:"required"`
	TreeNodeId          int               `json:"treeNodeId" binding:"required"`
	Description         string            `json:"description"`
	Tags                map[string]string `json:"tags"`
	SecurityGroupIds    []string          `json:"securityGroupIds"`
	AllocatedStorage    int               `json:"allocatedStorage" binding:"min=20"`
	BackupRetentionDays int               `json:"backupRetentionDays" binding:"min=1,max=30"`
	PreferredBackupTime string            `json:"preferredBackupTime"`
	MaintenanceWindow   string            `json:"maintenanceWindow"`
	Environment         string            `json:"environment" binding:"required"`
}

// UpdateRdsReq 更新RDS实例请求
type UpdateRdsReq struct {
	ID                  int               `json:"id" uri:"id" binding:"required"`
	InstanceName        string            `json:"instanceName" binding:"omitempty,min=2,max=100"`
	Description         string            `json:"description"`
	Tags                map[string]string `json:"tags"`
	BackupRetentionDays int               `json:"backupRetentionDays" binding:"omitempty,min=1,max=30"`
	PreferredBackupTime string            `json:"preferredBackupTime"`
	MaintenanceWindow   string            `json:"maintenanceWindow"`
	TreeNodeId          int               `json:"treeNodeId"`
}

// StartRdsReq 启动RDS实例请求
type StartRdsReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// StopRdsReq 停止RDS实例请求
type StopRdsReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// RestartRdsReq 重启RDS实例请求
type RestartRdsReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// DeleteRdsReq 删除RDS实例请求
type DeleteRdsReq struct {
	ID           int  `json:"id" uri:"id" binding:"required"`
	ForceDelete  bool `json:"forceDelete"`
	DeleteBackup bool `json:"deleteBackup"`
}

// ResizeRdsReq 调整RDS实例规格请求
type ResizeRdsReq struct {
	ID               int    `json:"id" uri:"id" binding:"required"`
	DBInstanceClass  string `json:"dbInstanceClass" binding:"required"`
	AllocatedStorage int    `json:"allocatedStorage" binding:"omitempty,min=20"`
	ApplyImmediately bool   `json:"applyImmediately"`
}

// BackupRdsReq 备份RDS实例请求
type BackupRdsReq struct {
	ID          int    `json:"id" uri:"id" binding:"required"`
	BackupName  string `json:"backupName" binding:"required,min=2,max=100"`
	BackupType  string `json:"backupType" binding:"required,oneof=Full Incremental"`
	Description string `json:"description"`
}

// RestoreRdsReq 恢复RDS实例请求
type RestoreRdsReq struct {
	ID               int    `json:"id" uri:"id" binding:"required"`
	BackupId         string `json:"backupId"`
	RestoreTime      string `json:"restoreTime"`
	RestoreType      string `json:"restoreType" binding:"required,oneof=backup time"`
	TargetInstanceId string `json:"targetInstanceId"`
	NewInstanceName  string `json:"newInstanceName"`
}

// ResetRdsPasswordReq 重置RDS实例密码请求
type ResetRdsPasswordReq struct {
	ID          int    `json:"id" uri:"id" binding:"required"`
	Username    string `json:"username" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=32"`
}

// RenewRdsReq 续费RDS实例请求
type RenewRdsReq struct {
	ID          int    `json:"id" uri:"id" binding:"required"`
	Period      int    `json:"period" binding:"required,min=1,max=36"`
	PeriodUnit  string `json:"periodUnit" binding:"required,oneof=Month Year"`
	AutoRenew   bool   `json:"autoRenew"`
	ClientToken string `json:"clientToken"`
}
