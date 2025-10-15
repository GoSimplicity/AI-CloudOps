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

// 变更类型常量
const (
	ChangeTypeCreated       = "created"        // 创建
	ChangeTypeUpdated       = "updated"        // 更新
	ChangeTypeDeleted       = "deleted"        // 删除
	ChangeTypeStatusChanged = "status_changed" // 状态变更
)

// 变更来源常量
const (
	ChangeSourceManual = "manual" // 手动操作
	ChangeSourceSync   = "sync"   // 同步操作
)

// CloudProvider 云厂商类型
type CloudProvider int8

const (
	ProviderAliyun  CloudProvider = iota + 1 // 阿里云
	ProviderTencent                          // 腾讯云
	ProviderAWS                              // AWS
	ProviderHuawei                           // 华为云
	ProviderAzure                            // Azure
	ProviderGCP                              // Google Cloud
)

// String 返回云厂商的字符串表示（用于日志和调试）
func (p CloudProvider) String() string {
	switch p {
	case ProviderAliyun:
		return "阿里云"
	case ProviderTencent:
		return "腾讯云"
	case ProviderAWS:
		return "AWS"
	case ProviderHuawei:
		return "华为云"
	case ProviderAzure:
		return "Azure"
	case ProviderGCP:
		return "Google Cloud"
	default:
		return "未知"
	}
}

// CloudResourceType 云资源类型
type CloudResourceType int8

const (
	ResourceTypeECS   CloudResourceType = iota + 1 // 云服务器
	ResourceTypeRDS                                // 云数据库
	ResourceTypeSLB                                // 负载均衡
	ResourceTypeOSS                                // 对象存储
	ResourceTypeVPC                                // 虚拟私有云
	ResourceTypeOther                              // 其他资源
)

// String 返回资源类型的字符串表示（用于日志和调试）
func (r CloudResourceType) String() string {
	switch r {
	case ResourceTypeECS:
		return "云服务器"
	case ResourceTypeRDS:
		return "云数据库"
	case ResourceTypeSLB:
		return "负载均衡"
	case ResourceTypeOSS:
		return "对象存储"
	case ResourceTypeVPC:
		return "虚拟私有云"
	case ResourceTypeOther:
		return "其他"
	default:
		return "未知"
	}
}

// CloudResourceStatus 云资源状态
type CloudResourceStatus int8

const (
	CloudResourceRunning  CloudResourceStatus = iota + 1 // 运行中
	CloudResourceStopped                                 // 已停止
	CloudResourceStarting                                // 启动中
	CloudResourceStopping                                // 停止中
	CloudResourceDeleted                                 // 已删除
	CloudResourceUnknown                                 // 未知状态
)

// String 返回资源状态的字符串表示（用于日志和调试）
func (s CloudResourceStatus) String() string {
	switch s {
	case CloudResourceRunning:
		return "运行中"
	case CloudResourceStopped:
		return "已停止"
	case CloudResourceStarting:
		return "启动中"
	case CloudResourceStopping:
		return "停止中"
	case CloudResourceDeleted:
		return "已删除"
	case CloudResourceUnknown:
		return "未知"
	default:
		return "未知"
	}
}

// Currency 货币单位
type Currency string

const (
	CurrencyCNY Currency = "CNY" // 人民币
	CurrencyUSD Currency = "USD" // 美元
)

// ChargeType 计费方式
type ChargeType string

const (
	ChargeTypePostPaid ChargeType = "PostPaid" // 按量付费
	ChargeTypePrePaid  ChargeType = "PrePaid"  // 包年包月
)

// SyncMode 同步模式
type SyncMode string

const (
	SyncModeFull        SyncMode = "full"        // 全量同步
	SyncModeIncremental SyncMode = "incremental" // 增量同步
)

// TreeCloudResource 云资源管理
type TreeCloudResource struct {
	Model

	Name           string              `json:"name" gorm:"type:varchar(100);not null;comment:资源名称"`
	ResourceType   CloudResourceType   `json:"resource_type" gorm:"type:tinyint(1);not null;comment:资源类型;default:1"`
	Status         CloudResourceStatus `json:"status" gorm:"type:tinyint(1);not null;comment:资源状态;default:1"`
	Environment    string              `json:"environment" gorm:"type:varchar(50);comment:环境标识(dev/test/prod)"`
	Description    string              `json:"description" gorm:"type:text;comment:资源描述"`
	Tags           KeyValueList        `json:"tags" gorm:"type:text;serializer:json;comment:资源标签集合"`
	CreateUserID   int                 `json:"create_user_id" gorm:"comment:创建者ID;default:0"`
	CreateUserName string              `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"`
	CloudAccountID int                 `json:"cloud_account_id" gorm:"not null;comment:云账户ID"`
	CloudAccount   *CloudAccount       `json:"cloud_account,omitempty" gorm:"foreignKey:CloudAccountID"`
	Region         string              `json:"region" gorm:"type:varchar(50);comment:区域,如cn-hangzhou"`
	InstanceID     string              `json:"instance_id" gorm:"type:varchar(100);comment:云资源实例ID"`
	InstanceType   string              `json:"instance_type" gorm:"type:varchar(100);comment:实例规格(如ecs.g6.large)"`
	Cpu            int                 `json:"cpu" gorm:"comment:CPU核数;default:0"`
	Memory         int                 `json:"memory" gorm:"comment:内存大小(GiB);default:0"`
	Disk           int                 `json:"disk" gorm:"comment:磁盘大小(GiB);default:0"`
	PublicIP       string              `json:"public_ip" gorm:"type:varchar(45);comment:公网IP"`
	PrivateIP      string              `json:"private_ip" gorm:"type:varchar(45);comment:私网IP"`
	VpcID          string              `json:"vpc_id" gorm:"type:varchar(100);comment:VPC ID"`
	ZoneID         string              `json:"zone_id" gorm:"type:varchar(50);comment:可用区ID"`
	ChargeType     ChargeType          `json:"charge_type" gorm:"type:varchar(50);comment:计费方式(PostPaid/PrePaid)"`
	ExpireTime     *time.Time          `json:"expire_time" gorm:"type:datetime;comment:到期时间"`
	MonthlyCost    float64             `json:"monthly_cost" gorm:"type:decimal(10,2);comment:月度成本;default:0"`
	Currency       Currency            `json:"currency" gorm:"type:varchar(10);not null;comment:货币单位;default:'CNY'"`
	OSType         string              `json:"os_type" gorm:"type:varchar(50);comment:操作系统类型(linux/windows)"`
	OSName         string              `json:"os_name" gorm:"type:varchar(100);comment:操作系统名称"`
	ImageID        string              `json:"image_id" gorm:"type:varchar(100);comment:镜像ID"`
	ImageName      string              `json:"image_name" gorm:"type:varchar(100);comment:镜像名称"`
	Port           int                 `json:"port" gorm:"comment:SSH端口号;default:22"`
	Username       string              `json:"username" gorm:"type:varchar(100);comment:SSH用户名"`
	Password       string              `json:"-" gorm:"type:varchar(500);comment:SSH密码(加密存储)"`
	Key            string              `json:"-" gorm:"type:text;comment:SSH密钥"`
	AuthMode       AuthMode            `json:"auth_mode" gorm:"type:tinyint(1);comment:SSH认证方式(1:密码,2:密钥);default:1"`
	TreeNodes      []*TreeNode         `json:"tree_nodes" gorm:"many2many:cl_tree_node_cloud"`
}

func (t *TreeCloudResource) TableName() string {
	return "cl_tree_cloud_resource"
}

// GetTreeCloudResourceListReq 获取云资源列表请求
type GetTreeCloudResourceListReq struct {
	ListReq
	CloudAccountID int                 `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	ResourceType   CloudResourceType   `json:"resource_type" form:"resource_type" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Status         CloudResourceStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Environment    string              `json:"environment" form:"environment"`
}

// GetTreeCloudResourceDetailReq 获取云资源详情请求
type GetTreeCloudResourceDetailReq struct {
	ID int `json:"id" form:"id" binding:"required,gt=0"`
}

// UpdateTreeCloudResourceReq 更新云资源本地元数据请求（不影响云上资源）
type UpdateTreeCloudResourceReq struct {
	ID           int          `json:"id" binding:"required,gt=0"`
	Environment  string       `json:"environment"`                              // 环境标识
	Description  string       `json:"description"`                              // 资源描述
	Tags         KeyValueList `json:"tags"`                                     // 自定义标签
	Port         int          `json:"port" binding:"omitempty,gte=1,lte=65535"` // SSH端口
	Username     string       `json:"username"`                                 // SSH用户名
	Password     string       `json:"password"`                                 // SSH密码
	Key          string       `json:"key"`                                      // SSH密钥
	AuthMode     AuthMode     `json:"auth_mode" binding:"omitempty,oneof=1 2"`  // SSH认证方式
	OperatorID   int          `json:"-"`                                        // 操作人ID (不通过JSON传递)
	OperatorName string       `json:"-"`                                        // 操作人姓名 (不通过JSON传递)
}

// DeleteTreeCloudResourceReq 删除云资源请求（仅从平台删除，不影响云上资源）
type DeleteTreeCloudResourceReq struct {
	ID           int    `json:"id" binding:"required,gt=0"`
	OperatorID   int    `json:"-"` // 操作人ID (不通过JSON传递)
	OperatorName string `json:"-"` // 操作人姓名 (不通过JSON传递)
}

// SyncTreeCloudResourceReq 从云厂商同步资源请求
type SyncTreeCloudResourceReq struct {
	CloudAccountID int                 `json:"cloud_account_id" binding:"required,gt=0"`
	ResourceTypes  []CloudResourceType `json:"resource_types" binding:"omitempty"`                   // 同步的资源类型列表，为空则同步所有
	Regions        []string            `json:"regions"`                                              // 指定同步的区域列表，为空则同步账号配置的区域
	InstanceIDs    []string            `json:"instance_ids"`                                         // 指定同步的实例ID列表，为空则同步所有
	SyncMode       SyncMode            `json:"sync_mode" binding:"omitempty,oneof=full incremental"` // 同步模式: full-全量, incremental-增量
	AutoBind       bool                `json:"auto_bind"`                                            // 是否自动绑定到服务树节点
	BindNodeID     int                 `json:"bind_node_id" binding:"omitempty,gt=0"`                // 自动绑定的目标节点ID
	OperatorID     int                 `json:"-"`                                                    // 操作人ID (不通过JSON传递)
	OperatorName   string              `json:"-"`                                                    // 操作人姓名 (不通过JSON传递)
}

// SyncCloudResourceResp 同步云资源响应
type SyncCloudResourceResp struct {
	TotalCount      int       `json:"total_count"`      // 总共同步的资源数量
	NewCount        int       `json:"new_count"`        // 新增的资源数量
	UpdateCount     int       `json:"update_count"`     // 更新的资源数量
	DeleteCount     int       `json:"delete_count"`     // 删除的资源数量（全量同步时）
	FailedCount     int       `json:"failed_count"`     // 同步失败的数量
	FailedInstances []string  `json:"failed_instances"` // 同步失败的实例ID列表
	SyncTime        time.Time `json:"sync_time"`        // 同步时间
}

// VerifyCloudCredentialsReq 验证云厂商凭证请求
// Deprecated: 使用 cloud_account.go 中的 VerifyCloudAccountReq
type VerifyCloudCredentialsReq struct {
	Provider  CloudProvider `json:"provider" binding:"required,oneof=1 2 3 4 5 6"`
	Region    string        `json:"region" binding:"required"`
	AccessKey string        `json:"access_key" binding:"required"`
	SecretKey string        `json:"secret_key" binding:"required"`
}

// GetTreeNodeCloudResourcesReq 获取树节点下的云资源请求
type GetTreeNodeCloudResourcesReq struct {
	NodeID         int                 `json:"node_id" form:"node_id" binding:"required,gt=0"`
	CloudAccountID int                 `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	ResourceType   CloudResourceType   `json:"resource_type" form:"resource_type" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Status         CloudResourceStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
}

// BindTreeCloudResourceReq 绑定云资源到树节点请求
type BindTreeCloudResourceReq struct {
	ID          int   `json:"id" binding:"required,gt=0"`
	TreeNodeIDs []int `json:"tree_node_ids" binding:"required,min=1,dive,gt=0"`
}

// UnBindTreeCloudResourceReq 解绑云资源与树节点请求
type UnBindTreeCloudResourceReq struct {
	ID          int   `json:"id" binding:"required,gt=0"`
	TreeNodeIDs []int `json:"tree_node_ids" binding:"required,min=1,dive,gt=0"`
}

// ConnectTreeCloudResourceTerminalReq 连接云资源终端请求（针对ECS）
type ConnectTreeCloudResourceTerminalReq struct {
	ID     int `json:"id" form:"id" binding:"required,gt=0"`
	UserID int `json:"user_id"`
}

// UpdateCloudResourceStatusReq 更新云资源状态请求
type UpdateCloudResourceStatusReq struct {
	ID     int                 `json:"id" binding:"required,gt=0"`
	Status CloudResourceStatus `json:"status" binding:"required,oneof=1 2 3 4 5 6"`
}

// CloudResourceSyncHistory 云资源同步历史
type CloudResourceSyncHistory struct {
	Model
	CloudAccountID  int       `json:"cloud_account_id" gorm:"not null;comment:云账户ID"`
	SyncMode        SyncMode  `json:"sync_mode" gorm:"type:varchar(20);comment:同步模式"`
	TotalCount      int       `json:"total_count" gorm:"comment:同步总数"`
	NewCount        int       `json:"new_count" gorm:"comment:新增数量"`
	UpdateCount     int       `json:"update_count" gorm:"comment:更新数量"`
	DeleteCount     int       `json:"delete_count" gorm:"comment:删除数量"`
	FailedCount     int       `json:"failed_count" gorm:"comment:失败数量"`
	FailedInstances string    `json:"failed_instances" gorm:"type:text;comment:失败的实例ID列表(JSON)"`
	SyncStatus      string    `json:"sync_status" gorm:"type:varchar(20);comment:同步状态(success/failed/partial)"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text;comment:错误信息"`
	StartTime       time.Time `json:"start_time" gorm:"comment:开始时间"`
	EndTime         time.Time `json:"end_time" gorm:"comment:结束时间"`
	Duration        int       `json:"duration" gorm:"comment:同步耗时(秒)"`
}

func (c *CloudResourceSyncHistory) TableName() string {
	return "cl_cloud_resource_sync_history"
}

// GetCloudResourceSyncHistoryReq 获取同步历史请求
type GetCloudResourceSyncHistoryReq struct {
	ListReq
	CloudAccountID int    `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	SyncStatus     string `json:"sync_status" form:"sync_status"`
}

// CloudResourceChangeLog 云资源变更日志
type CloudResourceChangeLog struct {
	Model
	ResourceID   int       `json:"resource_id" gorm:"not null;comment:云资源ID"`
	InstanceID   string    `json:"instance_id" gorm:"type:varchar(100);comment:实例ID"`
	ChangeType   string    `json:"change_type" gorm:"type:varchar(20);comment:变更类型(created/updated/deleted/status_changed)"`
	FieldName    string    `json:"field_name" gorm:"type:varchar(100);comment:变更字段名"`
	OldValue     string    `json:"old_value" gorm:"type:text;comment:旧值"`
	NewValue     string    `json:"new_value" gorm:"type:text;comment:新值"`
	ChangeSource string    `json:"change_source" gorm:"type:varchar(50);comment:变更来源(sync/manual)"`
	OperatorID   int       `json:"operator_id" gorm:"comment:操作人ID"`
	OperatorName string    `json:"operator_name" gorm:"type:varchar(100);comment:操作人姓名"`
	ChangeTime   time.Time `json:"change_time" gorm:"comment:变更时间"`
}

func (c *CloudResourceChangeLog) TableName() string {
	return "cl_cloud_resource_change_log"
}

// GetCloudResourceChangeLogReq 获取资源变更日志请求
type GetCloudResourceChangeLogReq struct {
	ListReq
	ResourceID int    `json:"resource_id" form:"resource_id" binding:"omitempty,gt=0"`
	ChangeType string `json:"change_type" form:"change_type"`
}
