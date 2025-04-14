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

// CloudProvider 云厂商类型枚举
type CloudProvider string

const (
	CloudProviderLocal   CloudProvider = "local"   // 本地环境
	CloudProviderAliyun  CloudProvider = "aliyun"  // 阿里云
	CloudProviderHuawei  CloudProvider = "huawei"  // 华为云
	CloudProviderTencent CloudProvider = "tencent" // 腾讯云
	CloudProviderAWS     CloudProvider = "aws"     // AWS
	CloudProviderAzure   CloudProvider = "azure"   // Azure
	CloudProviderGCP     CloudProvider = "gcp"     // Google Cloud
)

// ResourceStatus 资源状态枚举
type ResourceStatus string

const (
	StatusRunning   ResourceStatus = "running"   // 运行中
	StatusStopped   ResourceStatus = "stopped"   // 已停止
	StatusCreating  ResourceStatus = "creating"  // 创建中
	StatusFailed    ResourceStatus = "failed"    // 创建失败
	StatusDestroyed ResourceStatus = "destroyed" // 已销毁
)

// PaymentType 付费类型枚举
type PaymentType string

const (
	PaymentTypeOnDemand PaymentType = "on_demand" // 按量付费
	PaymentTypeReserved PaymentType = "reserved"  // 包年包月
)

// TreeNode 服务树节点结构
type TreeNode struct {
	Model
	Title       string `json:"title" gorm:"type:varchar(50);comment:节点名称"`       // 节点名称
	Pid         int    `json:"pId" gorm:"index;comment:父节点ID"`                   // 父节点ID
	Level       int    `json:"level" gorm:"comment:节点层级"`                        // 节点层级
	IsLeaf      bool   `json:"isLeaf" gorm:"comment:是否为叶子节点"`                    // 是否为叶子节点
	Desc        string `json:"desc" gorm:"type:text;comment:节点描述"`               // 节点描述
	ServiceCode string `json:"serviceCode" gorm:"type:varchar(50);comment:服务代码"` // 服务代码，唯一标识服务

	// 责任团队信息
	OpsAdmins []*User `json:"ops_admins" gorm:"many2many:tree_node_ops_admins;comment:运维负责人列表"` // 运维负责人
	RdAdmins  []*User `json:"rd_admins" gorm:"many2many:tree_node_rd_admins;comment:研发负责人列表"`   // 研发负责人
	RdMembers []*User `json:"rd_members" gorm:"many2many:tree_node_rd_members;comment:研发工程师列表"` // 研发工程师

	// 前端展示相关，不存储在数据库
	Key           string      `json:"key" gorm:"-"`             // 节点唯一标识
	Label         string      `json:"label" gorm:"-"`           // 节点显示名称
	Value         int         `json:"value" gorm:"-"`           // 节点值
	OpsAdminUsers StringList  `json:"ops_admin_users" gorm:"-"` // 运维负责人姓名列表
	RdAdminUsers  StringList  `json:"rd_admin_users" gorm:"-"`  // 研发负责人姓名列表
	RdMemberUsers StringList  `json:"rd_member_users" gorm:"-"` // 研发工程师姓名列表
	Children      []*TreeNode `json:"children" gorm:"-"`        // 子节点列表
}

// ResourceBase 资源基础信息
type ResourceBase struct {
	Model
	InstanceName     string         `json:"instance_name" gorm:"uniqueIndex;type:varchar(100);comment:资源实例名称"`
	InstanceId       string         `json:"instance_id" gorm:"uniqueIndex;type:varchar(100);comment:资源实例ID"`
	Hash             string         `json:"resource_hash" gorm:"uniqueIndex;type:varchar(200);comment:资源哈希值"`
	Provider         CloudProvider  `json:"cloud_provider" gorm:"type:varchar(50);comment:云厂商"`
	Region           string         `json:"cloud_region" gorm:"type:varchar(50);comment:地区，如cn-hangzhou"`
	ZoneId           string         `json:"zone_id" gorm:"type:varchar(100);comment:可用区ID"`
	VpcId            string         `json:"vpc_id" gorm:"type:varchar(100);comment:VPC ID"`
	Status           ResourceStatus `json:"resource_status" gorm:"type:varchar(50);comment:资源状态"`
	CreationTime     string         `json:"creation_time" gorm:"type:varchar(30);comment:创建时间,ISO8601格式"`
	Env              string         `json:"environment" gorm:"type:varchar(50);comment:环境标识,如dev,prod"`
	PayType          PaymentType    `json:"payment_type" gorm:"type:varchar(50);comment:付费类型"`
	Description      string         `json:"resource_desc" gorm:"type:text;comment:资源描述"`
	Tags             StringList     `json:"resource_tags" gorm:"type:varchar(500);comment:资源标签集合"`
	SecurityGroupIds StringList     `json:"security_group_ids" gorm:"type:varchar(500);comment:安全组ID列表"`
	PrivateIpAddress string         `json:"private_ip" gorm:"type:varchar(500);comment:私有IP地址"`
	PublicIpAddress  string         `json:"public_ip" gorm:"type:varchar(500);comment:公网IP地址"`

	// 资源创建和管理标志
	CreateByOrder bool      `json:"create_by_order" gorm:"comment:是否由工单创建"`
	LastSyncTime  time.Time `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID    uint      `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
}

// ComputeResource 计算资源通用属性
type ComputeResource struct {
	ResourceBase
	Cpu               int    `json:"cpu" gorm:"comment:CPU核数"`
	Memory            int    `json:"memory" gorm:"comment:内存大小,单位GiB"`
	InstanceType      string `json:"instanceType" gorm:"type:varchar(100);comment:实例类型"`
	Image             string `json:"image" gorm:"type:varchar(100);comment:镜像名称"`
	ImageId           string `json:"imageId" gorm:"type:varchar(100);comment:镜像ID"`
	IpAddr            string `json:"ipAddr" gorm:"type:varchar(45);uniqueIndex;comment:主IP地址"`
	Port              int    `json:"port" gorm:"comment:端口号;default:22"`
	Username          string `json:"username" gorm:"comment:用户名;default:root"`
	EncryptedPassword string `json:"encryptedPassword" gorm:"type:varchar(500);comment:加密密码"`
	Key               string `json:"key" gorm:"comment:密钥"`
	AuthMode          string `json:"authMode" gorm:"comment:认证方式;default:password"` // password或key
}

// ResourceEcs 服务器资源
type ResourceEcs struct {
	ComputeResource
	OsType            string     `json:"osType" gorm:"type:varchar(50);comment:操作系统类型,如win,linux"`
	VmType            int        `json:"vmType" gorm:"default:1;comment:设备类型,1=虚拟设备,2=物理设备"`
	OSName            string     `json:"osName" gorm:"type:varchar(100);comment:操作系统名称"`
	Hostname          string     `json:"hostname" gorm:"type:varchar(100);comment:主机名"`
	Disk              int        `json:"disk" gorm:"comment:系统盘大小,单位GiB"`
	NetworkInterfaces StringList `json:"networkInterfaces" gorm:"type:varchar(500);comment:弹性网卡ID集合"`
	DiskIds           StringList `json:"diskIds" gorm:"type:varchar(500);comment:云盘ID集合"`
	StartTime         string     `json:"startTime" gorm:"type:varchar(30);comment:最近启动时间"`
	AutoReleaseTime   string     `json:"autoReleaseTime" gorm:"type:varchar(30);comment:自动释放时间"`
	LastInvokedTime   string     `json:"lastInvokedTime" gorm:"type:varchar(30);comment:最近调用时间"`
	// 多对多关系
	EcsTreeNodes []*TreeNode `json:"ecsTreeNodes" gorm:"many2many:resource_ecs_tree_nodes;comment:关联服务树节点"`
}

// ResourceElb 负载均衡资源
type ResourceElb struct {
	ResourceBase
	LoadBalancerType   string     `json:"loadBalancerType" gorm:"type:varchar(50);comment:负载均衡类型,如nlb,alb,clb"`
	BandwidthCapacity  int        `json:"bandwidthCapacity" gorm:"comment:带宽容量上限,单位Mb"`
	AddressType        string     `json:"addressType" gorm:"type:varchar(50);comment:地址类型,公网或内网"`
	DNSName            string     `json:"dnsName" gorm:"type:varchar(255);comment:DNS解析地址"`
	BandwidthPackageId string     `json:"bandwidthPackageId" gorm:"type:varchar(100);comment:带宽包ID"`
	CrossZoneEnabled   bool       `json:"crossZoneEnabled" gorm:"comment:是否启用跨可用区"`
	ListenerPorts      StringList `json:"listenerPorts" gorm:"type:varchar(500);comment:监听端口列表"`
	BackendServers     StringList `json:"backendServers" gorm:"type:varchar(1000);comment:后端服务器列表"`
	// 多对多关系
	ElbTreeNodes []*TreeNode `json:"elbTreeNodes" gorm:"many2many:resource_elb_tree_nodes;comment:关联服务树节点"`
}

// ResourceRds 数据库资源
type ResourceRds struct {
	ResourceBase
	Engine            string `json:"engine" gorm:"type:varchar(50);comment:数据库引擎类型,如mysql,postgresql"`
	EngineVersion     string `json:"engineVersion" gorm:"type:varchar(50);comment:数据库版本,如8.0,5.7"`
	DBInstanceClass   string `json:"dbInstanceClass" gorm:"type:varchar(100);comment:实例规格"`
	DBInstanceType    string `json:"dbInstanceType" gorm:"type:varchar(50);comment:实例类型,如Primary,Readonly"`
	DBInstanceNetType string `json:"dbInstanceNetType" gorm:"type:varchar(50);comment:实例网络类型"`
	MasterInstanceId  string `json:"masterInstanceId" gorm:"type:varchar(100);comment:主实例ID"`
	ReplicateId       string `json:"replicateId" gorm:"type:varchar(100);comment:复制实例ID"`
	DBStatus          string `json:"dbStatus" gorm:"type:varchar(50);comment:数据库状态"`
	Port              int    `json:"port" gorm:"comment:数据库端口;default:3306"`
	ConnectionString  string `json:"connectionString" gorm:"type:varchar(255);comment:连接字符串"`
	// 多对多关系
	RdsTreeNodes []*TreeNode `json:"rdsTreeNodes" gorm:"many2many:resource_rds_tree_nodes;comment:关联服务树节点"`
}

// ResourceVpc VPC资源
type ResourceVpc struct {
	ResourceBase
	VpcName          string     `json:"vpcName" gorm:"type:varchar(100);comment:VPC名称"`
	CidrBlock        string     `json:"cidrBlock" gorm:"type:varchar(50);comment:IPv4网段"`
	Ipv6CidrBlock    string     `json:"ipv6CidrBlock" gorm:"type:varchar(50);comment:IPv6网段"`
	VSwitchIds       StringList `json:"vSwitchIds" gorm:"type:varchar(500);comment:交换机ID列表"`
	RouteTableIds    StringList `json:"routeTableIds" gorm:"type:varchar(500);comment:路由表ID列表"`
	NatGatewayIds    StringList `json:"natGatewayIds" gorm:"type:varchar(500);comment:NAT网关ID列表"`
	IsDefault        bool       `json:"isDefault" gorm:"comment:是否为默认VPC"`
	NetworkAclIds    StringList `json:"networkAclIds" gorm:"type:varchar(500);comment:网络ACL ID列表"`
	ResourceGroupId  string     `json:"resourceGroupId" gorm:"type:varchar(100);comment:资源组ID"`
	// 多对多关系
	VpcTreeNodes []*TreeNode `json:"vpcTreeNodes" gorm:"many2many:resource_vpc_tree_nodes;comment:关联服务树节点"`
}

// CloudAccount 云账户信息
type CloudAccount struct {
	Model
	Name            string        `json:"name" gorm:"type:varchar(100);comment:账户名称"`
	Provider        CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	AccountId       string        `json:"accountId" gorm:"type:varchar(100);comment:账户ID"`
	AccessKey       string        `json:"-" gorm:"type:varchar(100);comment:访问密钥ID"`
	EncryptedSecret string        `json:"-" gorm:"type:varchar(500);comment:加密的访问密钥"`
	Regions         StringList    `json:"regions" gorm:"type:varchar(500);comment:可用区域列表"`
	IsEnabled       bool          `json:"isEnabled" gorm:"comment:是否启用"`
	LastSyncTime    time.Time     `json:"lastSyncTime" gorm:"comment:最后同步时间"`
	Description     string        `json:"description" gorm:"type:text;comment:账户描述"`
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

// EcsCreationParams ECS创建参数
type EcsCreationParams struct {
	Provider         CloudProvider     `json:"provider" binding:"required"`
	Region           string            `json:"region" binding:"required"`
	ZoneId           string            `json:"zoneId" binding:"required"`
	InstanceType     string            `json:"instanceType" binding:"required"`
	ImageId          string            `json:"imageId" binding:"required"`
	VSwitchId        string            `json:"vSwitchId" binding:"required"`
	SecurityGroupIds []string          `json:"securityGroupIds" binding:"required"`
	Quantity         int               `json:"quantity" binding:"required,min=1,max=100"`
	HostnamePrefix   string            `json:"hostnamePrefix" binding:"required"`
	InstanceName     string            `json:"instanceName"`
	PayType          PaymentType       `json:"payType" binding:"required"`
	TreeNodeId       uint              `json:"treeNodeId" binding:"required"`
	Description      string            `json:"description"`
	SystemDiskCategory string          `json:"systemDiskCategory"`
	DryRun           bool              `json:"dryRun"`
	Tags             map[string]string `json:"tags"`
}

// ElbCreationParams ELB创建参数
type ElbCreationParams struct {
	Provider          CloudProvider     `json:"provider" binding:"required"`
	Region            string            `json:"region" binding:"required"`
	ZoneId            string            `json:"zoneId" binding:"required"`
	LoadBalancerType  string            `json:"loadBalancerType" binding:"required"`
	VpcId             string            `json:"vpcId" binding:"required"`
	AddressType       string            `json:"addressType" binding:"required"`
	BandwidthCapacity int               `json:"bandwidthCapacity"`
	TreeNodeId        uint              `json:"treeNodeId" binding:"required"`
	Description       string            `json:"description"`
	Tags              map[string]string `json:"tags"`
}

// RdsCreationParams RDS创建参数
type RdsCreationParams struct {
	Provider          CloudProvider     `json:"provider" binding:"required"`
	Region            string            `json:"region" binding:"required"`
	ZoneId            string            `json:"zoneId" binding:"required"`
	Engine            string            `json:"engine" binding:"required"`
	EngineVersion     string            `json:"engineVersion" binding:"required"`
	DBInstanceClass   string            `json:"dbInstanceClass" binding:"required"`
	VpcId             string            `json:"vpcId" binding:"required"`
	DBInstanceNetType string            `json:"dbInstanceNetType" binding:"required"`
	PayType           PaymentType       `json:"payType" binding:"required"`
	TreeNodeId        uint              `json:"treeNodeId" binding:"required"`
	Description       string            `json:"description"`
	Tags              map[string]string `json:"tags"`
}

// VpcCreationParams VPC创建参数
type VpcCreationParams struct {
	Provider         CloudProvider     `json:"provider" binding:"required"`
	Region           string            `json:"region" binding:"required"`
	ZoneId           string            `json:"zoneId" binding:"required"`
	VpcName          string            `json:"vpcName" binding:"required"`
	CidrBlock        string            `json:"cidrBlock" binding:"required"`
	Description      string            `json:"description"`
	Tags             map[string]string `json:"tags"`
	TreeNodeId       uint              `json:"treeNodeId" binding:"required"`
	VSwitchName      string            `json:"vSwitchName" binding:"required"`
	VSwitchCidrBlock string            `json:"vSwitchCidrBlock" binding:"required"`
}

// DiskCreationParams 磁盘创建参数
type DiskCreationParams struct {
	Provider         CloudProvider     `json:"provider" binding:"required"`
	Region           string            `json:"region" binding:"required"`
	ZoneId           string            `json:"zoneId" binding:"required"`
	DiskName         string            `json:"diskName" binding:"required"`
	DiskCategory     string            `json:"diskCategory" binding:"required"`
	Size             int               `json:"size" binding:"required,min=20"`
	VpcId            string            `json:"vpcId" binding:"required"`
	InstanceId       string            `json:"instanceId"`
	PayType          PaymentType       `json:"payType" binding:"required"`
	TreeNodeId       uint              `json:"treeNodeId" binding:"required"`
	Description      string            `json:"description"`
	Tags             map[string]string `json:"tags"`
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

// ListResourcesBaseReq 资源列表基础查询参数
type ListResourcesBaseReq struct {
	Page     int                 `form:"page" json:"page"`
	PageSize int                 `form:"pageSize" json:"pageSize"`
	Provider CloudProvider       `form:"provider" json:"provider"`
	Region   string              `form:"region" json:"region"`
	Env      string              `form:"env" json:"env"`
	Status   ResourceStatus      `form:"status" json:"status"`
	TreeNodeId uint              `form:"treeNodeId" json:"treeNodeId"`
	Keyword   string             `form:"keyword" json:"keyword"`
	OrderBy   string             `form:"orderBy" json:"orderBy"`
	OrderDesc bool               `form:"orderDesc" json:"orderDesc"`
}

// ListEcsResourcesReq ECS资源列表查询参数
type ListEcsResourcesReq struct {
	ListResourcesBaseReq
	OsType      string `form:"osType" json:"osType"`
	InstanceType string `form:"instanceType" json:"instanceType"`
	IpAddr      string `form:"ipAddr" json:"ipAddr"`
	VmType      int    `form:"vmType" json:"vmType"`
}

// ListElbResourcesReq ELB资源列表查询参数
type ListElbResourcesReq struct {
	ListResourcesBaseReq
	LoadBalancerType string `form:"loadBalancerType" json:"loadBalancerType"`
	AddressType      string `form:"addressType" json:"addressType"`
}

// ListRdsResourcesReq RDS资源列表查询参数
type ListRdsResourcesReq struct {
	ListResourcesBaseReq
	Engine        string `form:"engine" json:"engine"`
	EngineVersion string `form:"engineVersion" json:"engineVersion"`
	DBInstanceType string `form:"dbInstanceType" json:"dbInstanceType"`
}

// ListVpcResourcesReq VPC资源列表查询参数
type ListVpcResourcesReq struct {
	ListResourcesBaseReq
	CidrBlock     string `form:"cidrBlock" json:"cidrBlock"`
	IsDefault     bool   `form:"isDefault" json:"isDefault"`
	VpcName       string `form:"vpcName" json:"vpcName"`
}

// CreateNodeReq 创建节点请求
type CreateNodeReq struct {
	Title       string   `json:"title" binding:"required"`
	Pid         int      `json:"pId" binding:"required"`
	Desc        string   `json:"desc"`
	ServiceCode string   `json:"serviceCode"`
	OpsAdminIds []uint   `json:"opsAdminIds"`
	RdAdminIds  []uint   `json:"rdAdminIds"`
	RdMemberIds []uint   `json:"rdMemberIds"`
}

// UpdateNodeReq 更新节点请求
type UpdateNodeReq struct {
	ID          uint     `json:"id" binding:"required"`
	Title       string   `json:"title"`
	Desc        string   `json:"desc"`
	ServiceCode string   `json:"serviceCode"`
	OpsAdminIds []uint   `json:"opsAdminIds"`
	RdAdminIds  []uint   `json:"rdAdminIds"`
	RdMemberIds []uint   `json:"rdMemberIds"`
}

// NodeAdminReq 节点管理员请求
type NodeAdminReq struct {
	NodeId  uint   `json:"nodeId" binding:"required"`
	UserId  uint   `json:"userId" binding:"required"`
	AdminType string `json:"adminType" binding:"required,oneof=ops rd"`
}

// NodeMemberReq 节点成员请求
type NodeMemberReq struct {
	NodeId uint `json:"nodeId" binding:"required"`
	UserId uint `json:"userId" binding:"required"`
}

// ResourceECSResp ECS资源响应
type ResourceECSResp struct {
	ResourceEcs
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ResourceELBResp ELB资源响应
type ResourceELBResp struct {
	ResourceElb
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ResourceRDSResp RDS资源响应
type ResourceRDSResp struct {
	ResourceRds
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// PageResp 分页响应
type PageResp struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Data     interface{} `json:"data"`
}

// RegionResp 区域信息响应
type RegionResp struct {
	RegionId  string `json:"regionId"`
	LocalName string `json:"localName"`
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

// VpcResp VPC响应
type VpcResp struct {
	VpcId        string `json:"vpcId"`
	VpcName      string `json:"vpcName"`
	CidrBlock    string `json:"cidrBlock"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CreationTime string `json:"creationTime"`
}

// SecurityGroupResp 安全组响应
type SecurityGroupResp struct {
	SecurityGroupId   string `json:"securityGroupId"`
	SecurityGroupName string `json:"securityGroupName"`
	Description       string `json:"description"`
}

// TreeNodeResp 服务树节点响应
type TreeNodeResp struct {
	TreeNode
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// NodeResourcesResp 节点资源响应
type NodeResourcesResp struct {
	EcsResources []ResourceECSResp `json:"ecsResources"`
	ElbResources []ResourceELBResp `json:"elbResources"`
	RdsResources []ResourceRDSResp `json:"rdsResources"`
}

// NodePathResp 节点路径响应
type NodePathResp struct {
	Path []*TreeNodeResp `json:"path"`
}

// CloudProviderResp 云厂商响应
type CloudProviderResp struct {
	Provider  CloudProvider `json:"provider"`
	LocalName string              `json:"localName"`
}

type TreeNodeDetailResp struct {
	TreeNode
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type TreeNodePathResp struct {
	Path []*TreeNodeResp `json:"path"`
}

type NodeResourceResp struct {
	ResourceEcs
	ResourceElb
	ResourceRds
	ResourceVpc
}
