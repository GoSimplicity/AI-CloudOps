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

// ResourceEcs 服务器资源
type ResourceEcs struct {
	Model

	// 资源实例信息
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
	CreateByOrder bool       `json:"create_by_order" gorm:"comment:是否由工单创建;default:false"`
	LastSyncTime  *time.Time `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID    int        `json:"tree_node_id" gorm:"comment:关联的服务树节点ID;default:0"`

	// 资源规格信息
	Cpu               int        `json:"cpu" gorm:"comment:CPU核数"`
	Memory            int        `json:"memory" gorm:"comment:内存大小,单位GiB"`
	InstanceType      string     `json:"instanceType" gorm:"type:varchar(100);comment:实例类型"`
	ImageId           string     `json:"imageId" gorm:"type:varchar(100);comment:镜像ID"`
	IpAddr            string     `json:"ipAddr" gorm:"type:varchar(45);comment:主IP地址"`
	Port              int        `json:"port" gorm:"comment:端口号;default:22"`
	HostName          string     `json:"hostname" gorm:"comment:主机名"`
	Password          string     `json:"-" gorm:"type:varchar(500);comment:密码,加密存储"`
	Key               string     `json:"key" gorm:"comment:密钥"`
	AuthMode          string     `json:"authMode" gorm:"comment:认证方式;default:password"` // password或key
	OsType            string     `json:"osType" gorm:"type:varchar(50);comment:操作系统类型,如win,linux"`
	VmType            int        `json:"vmType" gorm:"default:1;comment:设备类型,1=虚拟设备,2=物理设备"`
	OSName            string     `json:"osName" gorm:"type:varchar(100);comment:操作系统名称"`
	ImageName         string     `json:"imageName" gorm:"type:varchar(100);comment:镜像名称"`
	Disk              int        `json:"disk" gorm:"comment:系统盘大小,单位GiB"`
	NetworkInterfaces StringList `json:"networkInterfaces" gorm:"type:varchar(500);comment:弹性网卡ID集合"`
	DiskIds           StringList `json:"diskIds" gorm:"type:varchar(500);comment:云盘ID集合"`
	StartTime         string     `json:"startTime" gorm:"type:varchar(30);comment:最近启动时间"`
	AutoReleaseTime   string     `json:"autoReleaseTime" gorm:"type:varchar(30);comment:自动释放时间"`

	// 多对多关系
	EcsTreeNodes []*TreeNode `json:"ecsTreeNodes" gorm:"many2many:resource_ecs_tree_nodes;comment:关联服务树节点"`
}

// =============== 请求模型 ===============

// ListEcsResourcesReq ECS资源列表查询参数
type ListEcsResourcesReq struct {
	ListReq
	Provider CloudProvider `form:"provider" json:"provider"`
	Region   string        `form:"region" json:"region"`
}

// ListEcsResourceOptionsReq 实例选项列表请求
type ListEcsResourceOptionsReq struct {
	ListReq
	Provider           CloudProvider `json:"provider" binding:"required"` // 云提供商
	ResourceType       string        `json:"resourceType"`                // 资源类型
	PayType            string        `json:"payType"`                     // 付费类型
	Region             string        `json:"region"`                      // 区域
	Zone               string        `json:"zone"`                        // 可用区
	InstanceType       string        `json:"instanceType"`                // 实例类型
	ImageId            string        `json:"imageId"`                     // 镜像ID
	SystemDiskCategory string        `json:"systemDiskCategory"`          // 系统盘类型
	DataDiskCategory   string        `json:"dataDiskCategory"`            // 数据盘类型
}

// GetEcsDetailReq 获取ECS详情请求
type GetEcsDetailReq struct {
	ID         int           `json:"id" form:"id"`                 // 内部ID（从URL参数获取）
	Provider   CloudProvider `json:"provider" form:"provider"`     // 云提供商
	Region     string        `json:"region" form:"region"`         // 区域
	InstanceId string        `json:"instanceId" form:"instanceId"` // 实例ID
}

// CreateEcsResourceReq ECS创建参数
type CreateEcsResourceReq struct {
	Provider           CloudProvider `json:"provider" binding:"required"`           // 云提供商
	Region             string        `json:"region"`                                // 区域
	ZoneId             string        `json:"zoneId"`                                // 可用区ID
	InstanceType       string        `json:"instanceType"`                          // 实例类型
	ImageId            string        `json:"imageId"`                               // 镜像ID
	VSwitchId          string        `json:"vSwitchId"`                             // 交换机ID
	SecurityGroupIds   []string      `json:"securityGroupIds"`                      // 安全组ID
	Amount             int           `json:"amount"`                                // 创建数量
	Hostname           string        `json:"hostname" binding:"required"`           // 主机名
	Password           string        `json:"password" binding:"required"`           // 密码
	InstanceName       string        `json:"instanceName"`                          // 实例名称
	TreeNodeId         int           `json:"treeNodeId"`                            // 服务树节点ID
	Description        string        `json:"description"`                           // 描述
	SystemDiskCategory string        `json:"systemDiskCategory"`                    // 系统盘类型
	AutoRenewPeriod    int           `json:"autoRenewPeriod"`                       // 自动续费周期
	PeriodUnit         string        `json:"periodUnit"`                            // Month 月 Year 年
	Period             int           `json:"period"`                                // 购买时长
	AutoRenew          bool          `json:"autoRenew"`                             // 是否自动续费
	InstanceChargeType string        `json:"instanceChargeType"`                    // 付费类型
	SpotStrategy       string        `json:"spotStrategy"`                          // NoSpot 默认值 表示正常按量付费 SpotAsPriceGo 表示自动竞价
	SpotDuration       int           `json:"spotDuration"`                          // 竞价时长
	SystemDiskSize     int           `json:"systemDiskSize"`                        // 系统盘大小
	DataDiskSize       int           `json:"dataDiskSize"`                          // 数据盘大小
	DataDiskCategory   string        `json:"dataDiskCategory"`                      // 数据盘类型
	DryRun             bool          `json:"dryRun"`                                // 是否仅预览而不创建
	Tags               StringList    `json:"tags"`                                  // 资源标签
	OsType             string        `json:"osType"`                                // 操作系统类型,如win,linux
	ImageName          string        `json:"imageName"`                             // 镜像名称
	AuthMode           string        `json:"authMode" binding:"oneof=password key"` // 认证方式,password或key
	Key                string        `json:"key"`                                   // 密钥内容,当authMode为key时使用
	IpAddr             string        `json:"ipAddr"`                                // IP地址
	Port               int           `json:"port"`                                  // SSH端口号
}

// UpdateEcsReq ECS更新请求（新增）
type UpdateEcsReq struct {
	ID               int           `json:"id"`               // 内部ID（从URL参数获取）
	Provider         CloudProvider `json:"provider"`         // 云提供商
	Region           string        `json:"region"`           // 区域
	InstanceId       string        `json:"instanceId"`       // 实例ID
	InstanceName     string        `json:"instanceName"`     // 实例名称
	Description      string        `json:"description"`      // 描述
	Tags             StringList    `json:"tags"`             // 资源标签
	SecurityGroupIds []string      `json:"securityGroupIds"` // 安全组ID
	Hostname         string        `json:"hostname"`         // 主机名
	Password         string        `json:"password"`         // 密码（用于更新密码）
	TreeNodeId       int           `json:"treeNodeId"`       // 服务树节点ID
	Env              string        `json:"environment"`      // 环境标识
	IpAddr           string        `json:"ipAddr"`           // IP地址
	Port             int           `json:"port"`             // SSH端口号
	AuthMode         string        `json:"authMode"`         // 认证方式
	Key              string        `json:"key"`              // 密钥内容
}

// DeleteEcsReq ECS删除请求
type DeleteEcsReq struct {
	ID         int           `json:"id"`         // 内部ID（从URL参数获取）
	Provider   CloudProvider `json:"provider"`   // 云提供商
	Region     string        `json:"region"`     // 区域
	InstanceId string        `json:"instanceId"` // 实例ID
	Force      bool          `json:"force"`      // 是否强制删除
}

// StartEcsReq ECS启动请求
type StartEcsReq struct {
	ID         int           `json:"id"`         // 内部ID（从URL参数获取）
	Provider   CloudProvider `json:"provider"`   // 云提供商
	Region     string        `json:"region"`     // 区域
	InstanceId string        `json:"instanceId"` // 实例ID
}

// StopEcsReq ECS停止请求
type StopEcsReq struct {
	ID         int           `json:"id"`         // 内部ID（从URL参数获取）
	Provider   CloudProvider `json:"provider"`   // 云提供商
	Region     string        `json:"region"`     // 区域
	InstanceId string        `json:"instanceId"` // 实例ID
	ForceStop  bool          `json:"forceStop"`  // 是否强制停止
}

// RestartEcsReq ECS重启请求
type RestartEcsReq struct {
	ID         int           `json:"id"`         // 内部ID（从URL参数获取）
	Provider   CloudProvider `json:"provider"`   // 云提供商
	Region     string        `json:"region"`     // 区域
	InstanceId string        `json:"instanceId"` // 实例ID
	ForceStop  bool          `json:"forceStop"`  // 是否强制重启
}

// ResizeEcsReq ECS调整规格请求（新增）
type ResizeEcsReq struct {
	ID           int           `json:"id"`           // 内部ID（从URL参数获取）
	Provider     CloudProvider `json:"provider"`     // 云提供商
	Region       string        `json:"region"`       // 区域
	InstanceId   string        `json:"instanceId"`   // 实例ID
	InstanceType string        `json:"instanceType"` // 新的实例类型
	SystemDisk   ResizeDisk    `json:"systemDisk"`   // 系统盘调整参数
	DataDisks    []ResizeDisk  `json:"dataDisks"`    // 数据盘调整参数
	DryRun       bool          `json:"dryRun"`       // 是否仅预览
}

// ResetEcsPasswordReq ECS重置密码请求（新增）
type ResetEcsPasswordReq struct {
	ID          int           `json:"id"`          // 内部ID（从URL参数获取）
	Provider    CloudProvider `json:"provider"`    // 云提供商
	Region      string        `json:"region"`      // 区域
	InstanceId  string        `json:"instanceId"`  // 实例ID
	NewPassword string        `json:"newPassword"` // 新密码
	KeyPairName string        `json:"keyPairName"` // 密钥对名称（如果使用密钥认证）
}

// RenewEcsReq ECS续费请求（新增）
type RenewEcsReq struct {
	ID                int           `json:"id"`                // 内部ID（从URL参数获取）
	Provider          CloudProvider `json:"provider"`          // 云提供商
	Region            string        `json:"region"`            // 区域
	InstanceId        string        `json:"instanceId"`        // 实例ID
	Period            int           `json:"period"`            // 续费时长
	PeriodUnit        string        `json:"periodUnit"`        // 时长单位：Month/Year
	AutoRenew         bool          `json:"autoRenew"`         // 是否自动续费
	AutoRenewPeriod   int           `json:"autoRenewPeriod"`   // 自动续费周期
	ExpectedStartTime string        `json:"expectedStartTime"` // 预期生效时间
}

// =============== 响应模型 ===============

// ResourceECSListResp ECS资源列表响应
type ResourceECSListResp struct {
	Total int64          `json:"total"`
	Data  []*ResourceEcs `json:"data"`
}

// ResourceECSDetailResp ECS资源详情响应
type ResourceECSDetailResp struct {
	Data *ResourceEcs `json:"data"`
}

// ListEcsResourceOptionsResp 实例选项列表响应
type ListEcsResourceOptionsResp struct {
	Value              string `json:"value"`
	Label              string `json:"label"`
	DataDiskCategory   string `json:"dataDiskCategory"`
	SystemDiskCategory string `json:"systemDiskCategory"`
	InstanceType       string `json:"instanceType"`
	Region             string `json:"region"`
	Zone               string `json:"zone"`
	PayType            string `json:"payType"`
	Valid              bool   `json:"valid"`
	ImageId            string `json:"imageId"`
	OSName             string `json:"osName"`
	OSType             string `json:"osType"`
	Architecture       string `json:"architecture"`
	Cpu                int    `json:"cpu"`
	Memory             int    `json:"memory"`
}

// =============== 其他辅助模型 ===============

// ListRegionsReq 区域列表请求
type ListRegionsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
}

// ListZonesReq 可用区列表请求
type ListZonesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

// ListInstanceTypesReq 实例类型列表请求
type ListInstanceTypesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

// ListImagesReq 镜像列表请求
type ListImagesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	OsType   string        `json:"osType"` // 操作系统类型过滤
}

// RegionResp 区域信息响应
type RegionResp struct {
	RegionId       string `json:"regionId"`       // 区域ID
	LocalName      string `json:"localName"`      // 区域名称
	RegionEndpoint string `json:"regionEndpoint"` // 区域终端节点
}

// ZoneResp 可用区信息响应
type ZoneResp struct {
	ZoneId    string `json:"zoneId"`
	LocalName string `json:"localName"`
}

// InstanceTypeResp 实例类型响应
type InstanceTypeResp struct {
	InstanceTypeId string `json:"instanceTypeId"`
	CpuCoreCount   int    `json:"cpuCoreCount"`
	MemorySize     int    `json:"memorySize"`
	Description    string `json:"description"`
}

// ImageResp 镜像响应
type ImageResp struct {
	ImageId     string `json:"imageId"`
	ImageName   string `json:"imageName"`
	OSType      string `json:"osType"`
	Description string `json:"description"`
}

// SecurityGroupResp 安全组响应
type SecurityGroupResp struct {
	SecurityGroupId   string `json:"securityGroupId"`
	SecurityGroupName string `json:"securityGroupName"`
	Description       string `json:"description"`
}
