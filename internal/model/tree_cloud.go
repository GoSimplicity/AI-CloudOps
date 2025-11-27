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

// ChangeType 变更类型常量
const (
	ChangeTypeCreated       = "created"        // 创建
	ChangeTypeUpdated       = "updated"        // 更新
	ChangeTypeDeleted       = "deleted"        // 删除
	ChangeTypeStatusChanged = "status_changed" // 状态变更
)

// ChangeSource 变更来源常量
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

// SyncStatus 同步状态
type SyncStatus string

const (
	SyncStatusSuccess SyncStatus = "success" // 成功
	SyncStatusFailed  SyncStatus = "failed"  // 失败
	SyncStatusPartial SyncStatus = "partial" // 部分成功
)

// TreeCloudResource 云资源管理
type TreeCloudResource struct {
	Model
	Name                 string              `json:"name" gorm:"type:varchar(100);not null;index;comment:资源名称"`
	ResourceType         CloudResourceType   `json:"resource_type" gorm:"type:tinyint(1);not null;index;comment:资源类型;default:1"`
	Status               CloudResourceStatus `json:"status" gorm:"type:tinyint(1);not null;index;comment:资源状态;default:1"`
	Environment          string              `json:"environment" gorm:"type:varchar(50);index;comment:环境标识(dev/test/prod)"`
	Description          string              `json:"description" gorm:"type:text;comment:资源描述"`
	Tags                 KeyValueList        `json:"tags" gorm:"type:text;serializer:json;comment:资源标签集合"`
	CreateUserID         int                 `json:"create_user_id" gorm:"index;comment:创建者ID;default:0"`
	CreateUserName       string              `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"`
	CloudAccountID       int                 `json:"cloud_account_id" gorm:"not null;index;comment:云账户ID"`
	CloudAccount         *CloudAccount       `json:"cloud_account,omitempty" gorm:"foreignKey:CloudAccountID"`
	CloudAccountRegionID int                 `json:"cloud_account_region_id" gorm:"not null;index;comment:云账户区域ID"`
	CloudAccountRegion   *CloudAccountRegion `json:"cloud_account_region,omitempty" gorm:"foreignKey:CloudAccountRegionID"`
	Region               string              `json:"region" gorm:"type:varchar(50);index;comment:区域,如cn-hangzhou"`
	InstanceID           string              `json:"instance_id" gorm:"type:varchar(100);uniqueIndex:idx_account_instance;comment:云资源实例ID"`
	InstanceType         string              `json:"instance_type" gorm:"type:varchar(100);comment:实例规格(如ecs.g6.large)"`
	Cpu                  int                 `json:"cpu" gorm:"comment:CPU核数;default:0"`
	Memory               int                 `json:"memory" gorm:"comment:内存大小(GiB);default:0"`
	Disk                 int                 `json:"disk" gorm:"comment:磁盘大小(GiB);default:0"`
	PublicIP             string              `json:"public_ip" gorm:"type:varchar(45);index;comment:公网IP"`
	PrivateIP            string              `json:"private_ip" gorm:"type:varchar(45);index;comment:私网IP"`
	VpcID                string              `json:"vpc_id" gorm:"type:varchar(100);index;comment:VPC ID"`
	ZoneID               string              `json:"zone_id" gorm:"type:varchar(50);index;comment:可用区ID"`
	ChargeType           ChargeType          `json:"charge_type" gorm:"type:varchar(50);index;comment:计费方式(PostPaid/PrePaid)"`
	ExpireTime           *time.Time          `json:"expire_time" gorm:"type:datetime;index;comment:到期时间"`
	MonthlyCost          float64             `json:"monthly_cost" gorm:"type:decimal(10,2);comment:月度成本;default:0"`
	Currency             Currency            `json:"currency" gorm:"type:varchar(10);not null;comment:货币单位;default:'CNY'"`
	OSType               string              `json:"os_type" gorm:"type:varchar(50);comment:操作系统类型(linux/windows)"`
	OSName               string              `json:"os_name" gorm:"type:varchar(100);comment:操作系统名称"`
	ImageID              string              `json:"image_id" gorm:"type:varchar(100);comment:镜像ID"`
	ImageName            string              `json:"image_name" gorm:"type:varchar(100);comment:镜像名称"`
	Port                 int                 `json:"port" gorm:"comment:SSH端口号;default:22"`
	Username             string              `json:"username" gorm:"type:varchar(100);comment:SSH用户名"`
	Password             string              `json:"-" gorm:"type:varchar(500);comment:SSH密码(加密存储)"`
	Key                  string              `json:"-" gorm:"type:text;comment:SSH密钥"`
	AuthMode             AuthMode            `json:"auth_mode" gorm:"type:tinyint(1);comment:SSH认证方式(1:密码,2:密钥);default:1"`
	LastSyncTime         *time.Time          `json:"last_sync_time" gorm:"type:datetime;comment:最后同步时间"`
	TreeNodes            []*TreeNode         `json:"tree_nodes,omitempty" gorm:"many2many:cl_tree_node_cloud"`
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
	Environment    string              `json:"environment" form:"environment" binding:"omitempty,max=50"`
	Region         string              `json:"region" form:"region" binding:"omitempty,max=50"`
	InstanceID     string              `json:"instance_id" form:"instance_id" binding:"omitempty,max=100"`
	Keyword        string              `json:"keyword" form:"keyword" binding:"omitempty,max=100"` // 搜索关键词(名称、IP等)
}

// GetTreeCloudResourceDetailReq 获取云资源详情请求
type GetTreeCloudResourceDetailReq struct {
	ID int `json:"id" form:"id" binding:"required,gt=0"`
}

// UpdateTreeCloudResourceReq 更新云资源本地元数据请求（不影响云上资源）
type UpdateTreeCloudResourceReq struct {
	ID           int          `json:"id" binding:"required,gt=0"`
	Environment  string       `json:"environment" binding:"omitempty,max=50"`   // 环境标识
	Description  string       `json:"description" binding:"omitempty,max=500"`  // 资源描述
	Tags         KeyValueList `json:"tags"`                                     // 自定义标签
	Port         int          `json:"port" binding:"omitempty,gte=1,lte=65535"` // SSH端口
	Username     string       `json:"username" binding:"omitempty,max=100"`     // SSH用户名
	Password     string       `json:"password" binding:"omitempty,max=500"`     // SSH密码
	Key          string       `json:"key" binding:"omitempty"`                  // SSH密钥
	AuthMode     AuthMode     `json:"auth_mode" binding:"omitempty,oneof=1 2"`  // SSH认证方式
	OperatorID   int          `json:"operator_id"`                              // 操作人ID
	OperatorName string       `json:"operator_name"`                            // 操作人姓名
}

// DeleteTreeCloudResourceReq 删除云资源请求（仅从平台删除，不影响云上资源）
type DeleteTreeCloudResourceReq struct {
	ID           int    `json:"id" binding:"required,gt=0"`
	OperatorID   int    `json:"operator_id"`   // 操作人ID
	OperatorName string `json:"operator_name"` // 操作人姓名
}

// BatchDeleteTreeCloudResourceReq 批量删除云资源请求
type BatchDeleteTreeCloudResourceReq struct {
	IDs          []int  `json:"ids" binding:"required,min=1,max=100,dive,gt=0"`
	OperatorID   int    `json:"operator_id"`   // 操作人ID
	OperatorName string `json:"operator_name"` // 操作人姓名
}

// SyncTreeCloudResourceReq 从云厂商同步资源请求
type SyncTreeCloudResourceReq struct {
	CloudAccountID        int                 `json:"cloud_account_id" binding:"required,gt=0"`
	CloudAccountRegionIDs []int               `json:"cloud_account_region_ids" binding:"omitempty,max=100,dive,gt=0"` // 指定同步的账号区域ID列表，为空则同步账号的所有区域
	ResourceTypes         []CloudResourceType `json:"resource_types" binding:"omitempty,max=10,dive,oneof=1 2 3 4 5 6"`
	InstanceIDs           []string            `json:"instance_ids" binding:"omitempty,max=100,dive,min=1"`  // 指定同步的实例ID列表，为空则同步所有
	SyncMode              SyncMode            `json:"sync_mode" binding:"omitempty,oneof=full incremental"` // 同步模式: full-全量, incremental-增量
	AutoBind              bool                `json:"auto_bind"`                                            // 是否自动绑定到服务树节点
	BindNodeID            int                 `json:"bind_node_id" binding:"omitempty,gt=0"`                // 自动绑定的目标节点ID
	OperatorID            int                 `json:"operator_id"`                                          // 操作人ID
	OperatorName          string              `json:"operator_name"`                                        // 操作人姓名
}

// VerifyCloudCredentialsReq 验证云厂商凭证请求
type VerifyCloudCredentialsReq struct {
	Provider  CloudProvider `json:"provider" binding:"required,oneof=1 2 3 4 5 6"`
	Region    string        `json:"region" binding:"required,min=1,max=50"`
	AccessKey string        `json:"access_key" binding:"required,min=10,max=500"`
	SecretKey string        `json:"secret_key" binding:"required,min=10,max=500"`
}

// GetTreeNodeCloudResourcesReq 获取树节点下的云资源请求
type GetTreeNodeCloudResourcesReq struct {
	NodeID         int                 `json:"node_id" form:"node_id" binding:"required,gt=0"`
	CloudAccountID int                 `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	ResourceType   CloudResourceType   `json:"resource_type" form:"resource_type" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Status         CloudResourceStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Page           int                 `json:"page" form:"page" binding:"omitempty,gte=1"`
	PageSize       int                 `json:"page_size" form:"page_size" binding:"omitempty,gte=1,lte=100"`
}

// BindTreeCloudResourceReq 绑定云资源到树节点请求
type BindTreeCloudResourceReq struct {
	ID          int   `json:"id" binding:"required,gt=0"`
	TreeNodeIDs []int `json:"tree_node_ids" binding:"required,min=1,max=100,dive,gt=0"`
}

// UnBindTreeCloudResourceReq 解绑云资源与树节点请求
type UnBindTreeCloudResourceReq struct {
	ID          int   `json:"id" binding:"required,gt=0"`
	TreeNodeIDs []int `json:"tree_node_ids" binding:"required,min=1,max=100,dive,gt=0"`
}

// BatchBindTreeCloudResourceReq 批量绑定云资源到树节点请求
type BatchBindTreeCloudResourceReq struct {
	IDs        []int `json:"ids" binding:"required,min=1,max=100,dive,gt=0"`
	TreeNodeID int   `json:"tree_node_id" binding:"required,gt=0"`
}

// BatchUnBindTreeCloudResourceReq 批量解绑云资源与树节点请求
type BatchUnBindTreeCloudResourceReq struct {
	IDs        []int `json:"ids" binding:"required,min=1,max=100,dive,gt=0"`
	TreeNodeID int   `json:"tree_node_id" binding:"required,gt=0"`
}

// ConnectTreeCloudResourceTerminalReq 连接云资源终端请求（针对ECS）
type ConnectTreeCloudResourceTerminalReq struct {
	ID     int `json:"id" form:"id" binding:"required,gt=0"`
	UserID int `json:"user_id" binding:"omitempty,gt=0"`
}

// UpdateCloudResourceStatusReq 更新云资源状态请求
type UpdateCloudResourceStatusReq struct {
	ID     int                 `json:"id" binding:"required,gt=0"`
	Status CloudResourceStatus `json:"status" binding:"required,oneof=1 2 3 4 5 6"`
}

// BatchUpdateCloudResourceStatusReq 批量更新云资源状态请求
type BatchUpdateCloudResourceStatusReq struct {
	IDs          []int               `json:"ids" binding:"required,min=1,max=100,dive,gt=0"`
	Status       CloudResourceStatus `json:"status" binding:"required,oneof=1 2 3 4 5 6"`
	OperatorID   int                 `json:"operator_id"`   // 操作人ID
	OperatorName string              `json:"operator_name"` // 操作人姓名
}

// CloudResourceSyncHistory 云资源同步历史
type CloudResourceSyncHistory struct {
	Model
	CloudAccountID  int           `json:"cloud_account_id" gorm:"not null;index;comment:云账户ID"`
	CloudAccount    *CloudAccount `json:"cloud_account,omitempty" gorm:"foreignKey:CloudAccountID"`
	SyncMode        SyncMode      `json:"sync_mode" gorm:"type:varchar(20);index;comment:同步模式"`
	TotalCount      int           `json:"total_count" gorm:"comment:同步总数;default:0"`
	NewCount        int           `json:"new_count" gorm:"comment:新增数量;default:0"`
	UpdateCount     int           `json:"update_count" gorm:"comment:更新数量;default:0"`
	DeleteCount     int           `json:"delete_count" gorm:"comment:删除数量;default:0"`
	FailedCount     int           `json:"failed_count" gorm:"comment:失败数量;default:0"`
	FailedInstances string        `json:"failed_instances" gorm:"type:text;comment:失败的实例ID列表(JSON)"`
	SyncStatus      SyncStatus    `json:"sync_status" gorm:"type:varchar(20);index;comment:同步状态(success/failed/partial)"`
	ErrorMessage    string        `json:"error_message" gorm:"type:text;comment:错误信息"`
	StartTime       time.Time     `json:"start_time" gorm:"type:datetime;index;comment:开始时间"`
	EndTime         *time.Time    `json:"end_time" gorm:"type:datetime;comment:结束时间"`
	Duration        int           `json:"duration" gorm:"comment:同步耗时(秒);default:0"`
	OperatorID      int           `json:"operator_id" gorm:"index;comment:操作人ID"`
	OperatorName    string        `json:"operator_name" gorm:"type:varchar(100);comment:操作人姓名"`
}

func (c *CloudResourceSyncHistory) TableName() string {
	return "cl_tree_cloud_resource_sync_history"
}

// GetCloudResourceSyncHistoryReq 获取同步历史请求
type GetCloudResourceSyncHistoryReq struct {
	ListReq
	CloudAccountID int        `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	SyncStatus     SyncStatus `json:"sync_status" form:"sync_status" binding:"omitempty,oneof=success failed partial"`
	SyncMode       SyncMode   `json:"sync_mode" form:"sync_mode" binding:"omitempty,oneof=full incremental"`
}

// CloudResourceChangeLog 云资源变更日志
type CloudResourceChangeLog struct {
	Model
	ResourceID     int                `json:"resource_id" gorm:"not null;index;comment:云资源ID"`
	CloudResource  *TreeCloudResource `json:"cloud_resource,omitempty" gorm:"foreignKey:ResourceID"`
	InstanceID     string             `json:"instance_id" gorm:"type:varchar(100);index;comment:实例ID"`
	ChangeType     string             `json:"change_type" gorm:"type:varchar(20);index;comment:变更类型(created/updated/deleted/status_changed)"`
	FieldName      string             `json:"field_name" gorm:"type:varchar(100);comment:变更字段名"`
	OldValue       string             `json:"old_value" gorm:"type:text;comment:旧值"`
	NewValue       string             `json:"new_value" gorm:"type:text;comment:新值"`
	ChangeSource   string             `json:"change_source" gorm:"type:varchar(50);index;comment:变更来源(sync/manual)"`
	OperatorID     int                `json:"operator_id" gorm:"index;comment:操作人ID"`
	OperatorName   string             `json:"operator_name" gorm:"type:varchar(100);comment:操作人姓名"`
	ChangeTime     time.Time          `json:"change_time" gorm:"type:datetime;index;comment:变更时间"`
	CloudAccountID int                `json:"cloud_account_id" gorm:"index;comment:云账户ID"`
}

func (c *CloudResourceChangeLog) TableName() string {
	return "cl_tree_cloud_resource_change_log"
}

// GetCloudResourceChangeLogReq 获取资源变更日志请求
type GetCloudResourceChangeLogReq struct {
	ListReq
	ResourceID     int    `json:"resource_id" form:"resource_id" binding:"omitempty,gt=0"`
	CloudAccountID int    `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	ChangeType     string `json:"change_type" form:"change_type" binding:"omitempty,oneof=created updated deleted status_changed"`
	ChangeSource   string `json:"change_source" form:"change_source" binding:"omitempty,oneof=sync manual"`
}

// ExportCloudResourceReq 导出云资源请求
type ExportCloudResourceReq struct {
	CloudAccountID int                 `json:"cloud_account_id" form:"cloud_account_id" binding:"omitempty,gt=0"`
	ResourceType   CloudResourceType   `json:"resource_type" form:"resource_type" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Status         CloudResourceStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Environment    string              `json:"environment" form:"environment" binding:"omitempty,max=50"`
	Format         string              `json:"format" form:"format" binding:"omitempty,oneof=json csv excel"`
	IDs            []int               `json:"ids" binding:"omitempty,max=1000,dive,gt=0"` // 指定导出的资源ID
}

// SyncCloudResourceResp 云资源同步响应
type SyncCloudResourceResp struct {
	TotalCount      int       `json:"total_count"`      // 总数
	NewCount        int       `json:"new_count"`        // 新增数量
	UpdateCount     int       `json:"update_count"`     // 更新数量
	DeleteCount     int       `json:"delete_count"`     // 删除数量
	FailedCount     int       `json:"failed_count"`     // 失败数量
	FailedInstances []string  `json:"failed_instances"` // 失败的实例ID列表
	SyncTime        time.Time `json:"sync_time"`        // 同步时间
	Message         string    `json:"message"`          // 同步消息
}

// String CloudProvider转字符串方法
func (p CloudProvider) String() string {
	switch p {
	case ProviderAliyun:
		return "aliyun"
	case ProviderTencent:
		return "tencent"
	case ProviderAWS:
		return "aws"
	case ProviderHuawei:
		return "huawei"
	case ProviderAzure:
		return "azure"
	case ProviderGCP:
		return "gcp"
	default:
		return "unknown"
	}
}
