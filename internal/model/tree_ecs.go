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

// ResourceEcs 表示云服务器实例资源
type ResourceEcs struct {
	Model

	// 基本实例信息
	InstanceName       string        `json:"instance_name" gorm:"type:varchar(100);comment:资源实例名称"`
	InstanceId         string        `json:"instance_id" gorm:"type:varchar(100);comment:资源实例ID"`
	Provider           CloudProvider `json:"cloud_provider" gorm:"type:varchar(50);comment:云厂商"`
	RegionId           string        `json:"region_id" gorm:"type:varchar(50);comment:地区，如cn-hangzhou"`
	ZoneId             string        `json:"zone_id" gorm:"type:varchar(100);comment:可用区ID"`
	VpcId              string        `json:"vpc_id" gorm:"type:varchar(100);comment:VPC ID"`
	Status             string        `json:"status" gorm:"type:varchar(50);comment:资源状态;default:RUNNING;enum:RUNNING,STOPPED,STARTING,STOPPING,RESTARTING,DELETING,ERROR"`
	CreationTime       string        `json:"creation_time" gorm:"type:varchar(30);comment:创建时间,ISO8601格式"`
	Env                string        `json:"environment" gorm:"type:varchar(50);comment:环境标识,如dev,prod"`
	InstanceChargeType string        `json:"instance_charge_type" gorm:"type:varchar(50);comment:付费类型"`
	Description        string        `json:"description" gorm:"type:text;comment:资源描述"`
	Tags               StringList    `json:"tags" gorm:"type:varchar(500);comment:资源标签集合"`
	SecurityGroupIds   StringList    `json:"security_group_ids" gorm:"type:varchar(500);comment:安全组ID列表"`
	PrivateIpAddress   StringList    `json:"private_ip_address" gorm:"type:varchar(500);comment:私有IP地址"`
	PublicIpAddress    StringList    `json:"public_ip_address" gorm:"type:varchar(500);comment:公网IP地址"`

	// 资源管理元数据
	CreateByOrder bool       `json:"create_by_order" gorm:"comment:是否由工单创建;default:false"`
	LastSyncTime  *time.Time `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID    int        `json:"tree_node_id" gorm:"comment:关联的服务树节点ID;default:0"`

	// 硬件规格信息
	Cpu               int        `json:"cpu" gorm:"comment:CPU核数"`
	Memory            int        `json:"memory" gorm:"comment:内存大小,单位GiB"`
	InstanceType      string     `json:"instance_type" gorm:"type:varchar(100);comment:实例类型"`
	ImageId           string     `json:"image_id" gorm:"type:varchar(100);comment:镜像ID"`
	IpAddr            string     `json:"ip_addr" gorm:"type:varchar(45);comment:主IP地址"`
	Port              int        `json:"port" gorm:"comment:端口号;default:22"`
	HostName          string     `json:"hostname" gorm:"comment:主机名"`
	Password          string     `json:"-" gorm:"type:varchar(500);comment:密码,加密存储"`
	Key               string     `json:"key" gorm:"comment:密钥"`
	AuthMode          string     `json:"auth_mode" gorm:"comment:认证方式;default:password"`
	OsType            string     `json:"os_type" gorm:"type:varchar(50);comment:操作系统类型,如win,linux"`
	VmType            int        `json:"vm_type" gorm:"default:1;comment:设备类型,1=虚拟设备,2=物理设备"`
	OSName            string     `json:"os_name" gorm:"type:varchar(100);comment:操作系统名称"`
	ImageName         string     `json:"image_name" gorm:"type:varchar(100);comment:镜像名称"`
	Disk              int        `json:"disk" gorm:"comment:系统盘大小,单位GiB"`
	NetworkInterfaces StringList `json:"network_interfaces" gorm:"type:varchar(500);comment:弹性网卡ID集合"`
	DiskIds           StringList `json:"disk_ids" gorm:"type:varchar(500);comment:云盘ID集合"`
	StartTime         string     `json:"start_time" gorm:"type:varchar(30);comment:最近启动时间"`
	AutoReleaseTime   string     `json:"auto_release_time" gorm:"type:varchar(30);comment:自动释放时间"`

	// 关联关系
	EcsTreeNodes []*TreeNode `json:"ecs_tree_nodes" gorm:"many2many:resource_ecs_tree_nodes;comment:关联服务树节点"`
}

// =============== 请求模型 ===============

// ListEcsResourcesReq 定义ECS资源列表查询参数
type ListEcsResourcesReq struct {
	ListReq
	Provider CloudProvider `form:"provider" json:"provider"`
	Status   string        `form:"status" json:"status"`
	Region   string        `form:"region" json:"region"`
}

// ListEcsResourceOptionsReq 定义实例选项列表请求参数
type ListEcsResourceOptionsReq struct {
	ListReq
	Provider           CloudProvider `json:"provider" binding:"required"` // 云提供商
	ResourceType       string        `json:"resource_type"`               // 资源类型
	PayType            string        `json:"pay_type"`                    // 付费类型
	Region             string        `json:"region"`                      // 区域
	Zone               string        `json:"zone"`                        // 可用区
	InstanceType       string        `json:"instance_type"`               // 实例类型
	ImageId            string        `json:"image_id"`                    // 镜像ID
	SystemDiskCategory string        `json:"system_disk_category"`        // 系统盘类型
	DataDiskCategory   string        `json:"data_disk_category"`          // 数据盘类型
}

// GetEcsDetailReq 定义获取ECS详情请求参数
type GetEcsDetailReq struct {
	ID         int           `json:"id" form:"id"`                  // 内部ID
	Provider   CloudProvider `json:"provider" form:"provider"`      // 云提供商
	Region     string        `json:"region" form:"region"`          // 区域
	InstanceId string        `json:"instance_id" form:"instanceId"` // 实例ID
}

// CreateEcsResourceReq 定义ECS创建请求参数
type CreateEcsResourceReq struct {
	AccountId          int           `json:"account_id"`                             // 云账号ID
	Provider           CloudProvider `json:"provider" binding:"required"`            // 云提供商
	Region             string        `json:"region"`                                 // 区域
	ZoneId             string        `json:"zone_id"`                                // 可用区ID
	InstanceType       string        `json:"instance_type"`                          // 实例类型
	ImageId            string        `json:"image_id"`                               // 镜像ID
	VSwitchId          string        `json:"v_switch_id"`                            // 交换机ID
	SecurityGroupIds   []string      `json:"security_group_ids"`                     // 安全组ID列表
	Amount             int           `json:"amount"`                                 // 创建数量
	Hostname           string        `json:"hostname" binding:"required"`            // 主机名
	Password           string        `json:"password" binding:"required"`            // 密码
	InstanceName       string        `json:"instance_name"`                          // 实例名称
	TreeNodeId         int           `json:"tree_node_id"`                           // 服务树节点ID
	Description        string        `json:"description"`                            // 描述
	SystemDiskCategory string        `json:"system_disk_category"`                   // 系统盘类型
	AutoRenewPeriod    int           `json:"auto_renew_period"`                      // 自动续费周期
	PeriodUnit         string        `json:"period_unit"`                            // 购买时长单位(Month/Year)
	Period             int           `json:"period"`                                 // 购买时长
	AutoRenew          bool          `json:"auto_renew"`                             // 是否自动续费
	InstanceChargeType string        `json:"instance_charge_type"`                   // 付费类型
	SpotStrategy       string        `json:"spot_strategy"`                          // 竞价策略
	SpotDuration       int           `json:"spot_duration"`                          // 竞价时长
	SystemDiskSize     int           `json:"system_disk_size"`                       // 系统盘大小
	DataDiskSize       int           `json:"data_disk_size"`                         // 数据盘大小
	DataDiskCategory   string        `json:"data_disk_category"`                     // 数据盘类型
	DryRun             bool          `json:"dry_run"`                                // 是否仅预览不创建
	Tags               StringList    `json:"tags"`                                   // 资源标签
	OsType             string        `json:"os_type"`                                // 操作系统类型
	ImageName          string        `json:"image_name"`                             // 镜像名称
	AuthMode           string        `json:"auth_mode"` // 认证方式
	Key                string        `json:"key"`                                    // 密钥内容
	IpAddr             string        `json:"ip_addr"`                                // IP地址
	Port               int           `json:"port"`                                   // SSH端口号
}

// UpdateEcsReq 定义ECS更新请求参数
type UpdateEcsReq struct {
	AccountId        int           `json:"account_id"`         // 云账号ID
	ID               int           `json:"id"`                 // 内部ID
	Provider         CloudProvider `json:"provider"`           // 云提供商
	Region           string        `json:"region"`             // 区域
	InstanceId       string        `json:"instance_id"`        // 实例ID
	InstanceName     string        `json:"instance_name"`      // 实例名称
	Description      string        `json:"description"`        // 描述
	Tags             StringList    `json:"tags"`               // 资源标签
	SecurityGroupIds []string      `json:"security_group_ids"` // 安全组ID列表
	Hostname         string        `json:"hostname"`           // 主机名
	Password         string        `json:"password"`           // 密码
	TreeNodeId       int           `json:"tree_node_id"`       // 服务树节点ID
	Env              string        `json:"environment"`        // 环境标识
	IpAddr           string        `json:"ip_addr"`            // IP地址
	Port             int           `json:"port"`               // SSH端口号
	AuthMode         string        `json:"auth_mode"`          // 认证方式
	Key              string        `json:"key"`                // 密钥内容
}

// DeleteEcsReq 定义ECS删除请求参数
type DeleteEcsReq struct {
	AccountId  int           `json:"account_id"`  // 云账号ID
	ID         int           `json:"id"`          // 内部ID
	Provider   CloudProvider `json:"provider"`    // 云提供商
	Region     string        `json:"region"`      // 区域
	InstanceId string        `json:"instance_id"` // 实例ID
	Force      bool          `json:"force"`       // 是否强制删除
}

// StartEcsReq 定义ECS启动请求参数
type StartEcsReq struct {
	ID         int           `json:"id"`          // 内部ID
	Provider   CloudProvider `json:"provider"`    // 云提供商
	Region     string        `json:"region"`      // 区域
	InstanceId string        `json:"instance_id"` // 实例ID
}

// StopEcsReq 定义ECS停止请求参数
type StopEcsReq struct {
	ID         int           `json:"id"`          // 内部ID
	Provider   CloudProvider `json:"provider"`    // 云提供商
	Region     string        `json:"region"`      // 区域
	InstanceId string        `json:"instance_id"` // 实例ID
	ForceStop  bool          `json:"force_stop"`  // 是否强制停止
}

// RestartEcsReq 定义ECS重启请求参数
type RestartEcsReq struct {
	ID         int           `json:"id"`          // 内部ID
	Provider   CloudProvider `json:"provider"`    // 云提供商
	Region     string        `json:"region"`      // 区域
	InstanceId string        `json:"instance_id"` // 实例ID
	ForceStop  bool          `json:"force_stop"`  // 是否强制重启
}

// ResizeEcsReq 定义ECS调整规格请求参数
type ResizeEcsReq struct {
	ID           int             `json:"id"`            // 内部ID
	Provider     CloudProvider   `json:"provider"`      // 云提供商
	Region       string          `json:"region"`        // 区域
	InstanceId   string          `json:"instance_id"`   // 实例ID
	InstanceType string          `json:"instance_type"` // 新的实例类型
	SystemDisk   ResizeDiskReq   `json:"system_disk"`   // 系统盘调整参数
	DataDisks    []ResizeDiskReq `json:"data_disks"`    // 数据盘调整参数
	DryRun       bool            `json:"dry_run"`       // 是否仅预览
}

// ResetEcsPasswordReq 定义ECS重置密码请求参数
type ResetEcsPasswordReq struct {
	ID          int           `json:"id"`            // 内部ID
	Provider    CloudProvider `json:"provider"`      // 云提供商
	Region      string        `json:"region"`        // 区域
	InstanceId  string        `json:"instance_id"`   // 实例ID
	NewPassword string        `json:"new_password"`  // 新密码
	KeyPairName string        `json:"key_pair_name"` // 密钥对名称
}

// RenewEcsReq 定义ECS续费请求参数
type RenewEcsReq struct {
	ID                int           `json:"id"`                  // 内部ID
	Provider          CloudProvider `json:"provider"`            // 云提供商
	Region            string        `json:"region"`              // 区域
	InstanceId        string        `json:"instance_id"`         // 实例ID
	Period            int           `json:"period"`              // 续费时长
	PeriodUnit        string        `json:"period_unit"`         // 时长单位(Month/Year)
	AutoRenew         bool          `json:"auto_renew"`          // 是否自动续费
	AutoRenewPeriod   int           `json:"auto_renew_period"`   // 自动续费周期
	ExpectedStartTime string        `json:"expected_start_time"` // 预期生效时间
}

// ResourceECSListResp 定义ECS资源列表响应
type ResourceECSListResp struct {
	Total int64          `json:"total"` // 总记录数
	Data  []*ResourceEcs `json:"data"`  // 数据列表
}

// ResourceECSDetailResp 定义ECS资源详情响应
type ResourceECSDetailResp struct {
	Data *ResourceEcs `json:"data"` // 详情数据
}

// ListEcsResourceOptionsResp 定义实例选项列表响应
type ListEcsResourceOptionsResp struct {
	Value              string `json:"value"`               // 选项值
	Label              string `json:"label"`               // 选项标签
	DataDiskCategory   string `json:"data_disk_category"`  // 数据盘类型
	SystemDiskCategory string `json:"system_disk_category"` // 系统盘类型
	InstanceType       string `json:"instance_type"`       // 实例类型
	Region             string `json:"region"`              // 区域
	Zone               string `json:"zone"`                // 可用区
	PayType            string `json:"pay_type"`            // 付费类型
	Valid              bool   `json:"valid"`               // 是否有效
	ImageId            string `json:"image_id"`            // 镜像ID
	OSName             string `json:"os_name"`             // 操作系统名称
	OSType             string `json:"os_type"`             // 操作系统类型
	Architecture       string `json:"architecture"`        // 架构
	Cpu                int    `json:"cpu"`                 // CPU核数
	Memory             int    `json:"memory"`              // 内存大小
}

// ListRegionsReq 定义区域列表请求参数
type ListRegionsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"` // 云提供商
}

// ListZonesReq 定义可用区列表请求参数
type ListZonesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"` // 云提供商
	Region   string        `json:"region" binding:"required"`   // 区域
}

// ListInstanceTypesReq 定义实例类型列表请求参数
type ListInstanceTypesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"` // 云提供商
	Region   string        `json:"region" binding:"required"`   // 区域
}

// ListImagesReq 定义镜像列表请求参数
type ListImagesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"` // 云提供商
	Region   string        `json:"region" binding:"required"`   // 区域
	OsType   string        `json:"os_type"`                     // 操作系统类型
}

// RegionResp 定义区域信息响应
type RegionResp struct {
	RegionId       string `json:"region_id"`       // 区域ID
	LocalName      string `json:"local_name"`      // 区域名称
	RegionEndpoint string `json:"region_endpoint"` // 区域终端节点
}

// ZoneResp 定义可用区信息响应
type ZoneResp struct {
	ZoneId    string `json:"zone_id"`    // 可用区ID
	LocalName string `json:"local_name"` // 可用区名称
}

// InstanceTypeResp 定义实例类型响应
type InstanceTypeResp struct {
	InstanceTypeId string `json:"instance_type_id"` // 实例类型ID
	CpuCoreCount   int    `json:"cpu_core_count"`   // CPU核心数
	MemorySize     int    `json:"memory_size"`      // 内存大小
	Description    string `json:"description"`      // 描述
}

// ImageResp 定义镜像响应
type ImageResp struct {
	ImageId     string `json:"image_id"`     // 镜像ID
	ImageName   string `json:"image_name"`   // 镜像名称
	OSType      string `json:"os_type"`      // 操作系统类型
	Description string `json:"description"`  // 描述
}

// SecurityGroupResp 定义安全组响应
type SecurityGroupResp struct {
	SecurityGroupId   string `json:"security_group_id"`   // 安全组ID
	SecurityGroupName string `json:"security_group_name"` // 安全组名称
	Description       string `json:"description"`         // 描述
}
