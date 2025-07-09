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

// CloudProvider 云厂商类型枚举
type CloudProvider string

const (
	CloudProviderAliyun CloudProvider = "aliyun" // 阿里云
	CloudProviderLocal  CloudProvider = "local"  // 本地环境
	CloudProviderHuawei CloudProvider = "huawei" // 华为云
	CloudProviderAWS    CloudProvider = "aws"    // AWS
	// CloudProviderTencent CloudProvider = "tencent" // 腾讯云
	// CloudProviderAzure   CloudProvider = "azure"   // Azure
	// CloudProviderGCP     CloudProvider = "gcp"     // Google Cloud
)

// CloudAccount 云账户信息
type CloudAccount struct {
	Model
	Name            string        `json:"name" gorm:"type:varchar(100);comment:账户名称"`
	Provider        CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	AccountId       string        `json:"accountId" gorm:"type:varchar(100);comment:账户ID"`
	AccessKey       string        `json:"accessKey" gorm:"type:varchar(100);comment:访问密钥ID"`
	EncryptedSecret string        `json:"encryptedSecret" gorm:"type:varchar(500);comment:加密的访问密钥"`
	Regions         StringList    `json:"regions" gorm:"type:varchar(500);comment:可用区域列表"`
	IsEnabled       bool          `json:"isEnabled" gorm:"comment:是否启用"`
	LastSyncTime    time.Time     `json:"lastSyncTime" gorm:"comment:最后同步时间"`
	Description     string        `json:"description" gorm:"type:text;comment:账户描述"`
}

// TableName 指定表名
func (CloudAccount) TableName() string {
	return "cloud_accounts"
}

// CloudAccountSyncStatus 云账户同步状态
type CloudAccountSyncStatus struct {
	Model
	AccountId    int       `json:"accountId" gorm:"comment:云账户ID"`
	ResourceType string    `json:"resourceType" gorm:"type:varchar(50);comment:资源类型"`
	Region       string    `json:"region" gorm:"type:varchar(50);comment:区域"`
	Status       string    `json:"status" gorm:"type:varchar(20);comment:同步状态"`
	LastSyncTime time.Time `json:"lastSyncTime" gorm:"comment:最后同步时间"`
	ErrorMessage string    `json:"errorMessage" gorm:"type:text;comment:错误信息"`
	SyncCount    int64     `json:"syncCount" gorm:"comment:同步资源数量"`
}

// TableName 指定表名
func (CloudAccountSyncStatus) TableName() string {
	return "cloud_account_sync_status"
}

// CloudAccountAuditLog 云账户审计日志
type CloudAccountAuditLog struct {
	Model
	AccountId int    `json:"accountId" gorm:"comment:云账户ID"`
	Operation string `json:"operation" gorm:"type:varchar(50);comment:操作类型"`
	Operator  string `json:"operator" gorm:"type:varchar(100);comment:操作人"`
	Details   string `json:"details" gorm:"type:text;comment:操作详情"`
	IPAddress string `json:"ipAddress" gorm:"type:varchar(50);comment:IP地址"`
	UserAgent string `json:"userAgent" gorm:"type:varchar(500);comment:用户代理"`
}

// TableName 指定表名
func (CloudAccountAuditLog) TableName() string {
	return "cloud_account_audit_logs"
}

// CreateCloudAccountReq 创建云账号请求
type CreateCloudAccountReq struct {
	Name        string        `json:"name" binding:"required" validate:"max=100"`
	Provider    CloudProvider `json:"provider" binding:"required"`
	AccountId   string        `json:"accountId" binding:"required" validate:"max=100"`
	AccessKey   string        `json:"accessKey" binding:"required" validate:"max=100"`
	SecretKey   string        `json:"secretKey" binding:"required"`
	Regions     []string      `json:"regions"`
	IsEnabled   bool          `json:"isEnabled"`
	Description string        `json:"description" validate:"max=500"`
}

// UpdateCloudAccountReq 更新云账号请求
type UpdateCloudAccountReq struct {
	ID          int           `json:"id"`
	Name        string        `json:"name" validate:"max=100"`
	Provider    CloudProvider `json:"provider"`
	AccountId   string        `json:"accountId" validate:"max=100"`
	AccessKey   string        `json:"accessKey" validate:"max=100"`
	SecretKey   string        `json:"secretKey"`
	Regions     []string      `json:"regions"`
	IsEnabled   bool          `json:"isEnabled"`
	Description string        `json:"description" validate:"max=500"`
}

type DeleteCloudAccountReq struct {
	ID int `json:"id"`
}

// GetCloudAccountReq 获取云账号详情请求
type GetCloudAccountReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// ListCloudAccountsReq 获取云账号列表请求
type ListCloudAccountsReq struct {
	ListReq
	Provider CloudProvider `json:"provider" form:"provider"`
	Enabled  bool          `json:"enabled" form:"enabled"`
}

// TestCloudAccountReq 测试云账号连接请求
type TestCloudAccountReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// SyncCloudReq 同步云资源请求
type SyncCloudReq struct {
	AccountIds   []int    `json:"accountIds"`   // 要同步的账号ID列表，为空则同步所有启用的账号
	ResourceType string   `json:"resourceType"` // 资源类型：ecs,vpc,sg等，为空则同步所有类型
	Regions      []string `json:"regions"`      // 要同步的区域列表，为空则同步所有区域
	Force        bool     `json:"force"`        // 是否强制重新同步
}

// GetSyncStatusReq 获取同步状态请求
type GetSyncStatusReq struct {
	AccountId    int    `json:"accountId" form:"accountId"`       // 云账户ID
	ResourceType string `json:"resourceType" form:"resourceType"` // 资源类型
	Region       string `json:"region" form:"region"`             // 区域
}

// BatchSyncAccountsReq 批量同步账户请求
type BatchSyncAccountsReq struct {
	AccountIds   []int    `json:"accountIds" binding:"required"` // 要同步的账户ID列表
	ResourceType string   `json:"resourceType"`                  // 资源类型
	Regions      []string `json:"regions"`                       // 区域列表
	Force        bool     `json:"force"`                         // 是否强制同步
}

// BatchTestAccountsReq 批量测试账户请求
type BatchTestAccountsReq struct {
	AccountIds []int `json:"accountIds" binding:"required"` // 要测试的账户ID列表
}

// EnableCloudAccountReq 启用云账户请求
type EnableCloudAccountReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// DisableCloudAccountReq 禁用云账户请求
type DisableCloudAccountReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// BatchDeleteCloudAccountsReq 批量删除云账号请求
type BatchDeleteCloudAccountsReq struct {
	AccountIDs []int `json:"accountIds" binding:"required"` // 要删除的账号ID列表
}

// BatchTestCloudAccountsReq 批量测试云账号请求
type BatchTestCloudAccountsReq struct {
	AccountIDs []int `json:"accountIds" binding:"required"` // 要测试的账号ID列表
}

// SyncCloudResourcesReq 同步云资源请求
type SyncCloudResourcesReq struct {
	AccountIds   []int    `json:"accountIds"`   // 要同步的账号ID列表，为空则同步所有启用的账号
	ResourceType string   `json:"resourceType"` // 资源类型：ecs,vpc,sg等，为空则同步所有类型
	Regions      []string `json:"regions"`      // 要同步的区域列表，为空则同步所有区域
	Force        bool     `json:"force"`        // 是否强制重新同步
}

// SyncCloudAccountResourcesReq 同步指定云账号资源请求
type SyncCloudAccountResourcesReq struct {
	ID           int      `json:"id"`           // 云账号ID
	ResourceType string   `json:"resourceType"` // 资源类型
	Regions      []string `json:"regions"`      // 区域列表
	Force        bool     `json:"force"`        // 是否强制同步
}

// GetCloudAccountStatisticsReq 获取云账号统计信息请求
type GetCloudAccountStatisticsReq struct {
	ListReq
	Provider CloudProvider `json:"provider"`
	Enabled  bool          `json:"enabled"`
}

// CloudAccountStatistics 云账号统计信息
type CloudAccountStatistics struct {
	TotalAccounts    int64                   `json:"totalAccounts"`    // 总账号数
	EnabledAccounts  int64                   `json:"enabledAccounts"`  // 启用账号数
	DisabledAccounts int64                   `json:"disabledAccounts"` // 禁用账号数
	ProviderStats    map[string]int64        `json:"providerStats"`    // 各云厂商账号数统计
	RegionStats      map[string]int64        `json:"regionStats"`      // 各区域账号数统计
	SyncStatus       map[string]int64        `json:"syncStatus"`       // 同步状态统计
	RecentActivities []*CloudAccountAuditLog `json:"recentActivities"` // 最近活动
}

// 同步状态常量
const (
	SyncStatusPending   = "pending"   // 等待中
	SyncStatusRunning   = "running"   // 同步中
	SyncStatusSuccess   = "success"   // 成功
	SyncStatusFailed    = "failed"    // 失败
	SyncStatusCancelled = "cancelled" // 已取消
)

// 操作类型常量
const (
	OperationCreate  = "create"  // 创建
	OperationUpdate  = "update"  // 更新
	OperationDelete  = "delete"  // 删除
	OperationEnable  = "enable"  // 启用
	OperationDisable = "disable" // 禁用
	OperationTest    = "test"    // 测试
	OperationSync    = "sync"    // 同步
)

// 资源类型常量
const (
	ResourceTypeECS           = "ecs"            // ECS实例
	ResourceTypeVPC           = "vpc"            // VPC网络
	ResourceTypeSecurityGroup = "security_group" // 安全组
	ResourceTypeDisk          = "disk"           // 磁盘
	ResourceTypeLoadBalancer  = "load_balancer"  // 负载均衡
	ResourceTypeRDS           = "rds"            // 数据库
	ResourceTypeAll           = "all"            // 所有资源
)
