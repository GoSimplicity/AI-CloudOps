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

// ResourceEcs 服务器资源
type ResourceEcs struct {
	ResourceBase

	Cpu               int        `json:"cpu" gorm:"comment:CPU核数"`
	Memory            int        `json:"memory" gorm:"comment:内存大小,单位GiB"`
	InstanceType      string     `json:"instanceType" gorm:"type:varchar(100);comment:实例类型"`
	ImageId           string     `json:"imageId" gorm:"type:varchar(100);comment:镜像ID"`
	IpAddr            string     `json:"ipAddr" gorm:"type:varchar(45);comment:主IP地址"`
	Port              int        `json:"port" gorm:"comment:端口号;default:22"`
	HostName          string     `json:"hostname" gorm:"comment:主机名"`
	Password          string     `json:"password" gorm:"type:varchar(500);comment:密码"`
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

// ListEcsResourcesReq ECS资源列表查询参数
type ListEcsResourcesReq struct {
	ListReq
	Provider CloudProvider `form:"provider" json:"provider"`
	Region   string        `form:"region" json:"region"`
}

// ResourceECSListResp ECS资源列表响应
type ResourceECSListResp struct {
	Total int64          `json:"total"`
	Data  []*ResourceEcs `json:"data"`
}

// ResourceECSDetailResp ECS资源详情响应
type ResourceECSDetailResp struct {
	Data *ResourceEcs `json:"data"`
}

// StartEcsReq ECS启动请求
type StartEcsReq struct {
	Provider   CloudProvider `json:"provider" binding:"required"`
	Region     string        `json:"region" binding:"required"`
	InstanceId string        `json:"instanceId" binding:"required"`
}

// StopEcsReq ECS停止请求
type StopEcsReq struct {
	Provider   CloudProvider `json:"provider" binding:"required"`
	Region     string        `json:"region" binding:"required"`
	InstanceId string        `json:"instanceId" binding:"required"`
}

// RestartEcsReq ECS重启请求
type RestartEcsReq struct {
	Provider   CloudProvider `json:"provider" binding:"required"`
	Region     string        `json:"region" binding:"required"`
	InstanceId string        `json:"instanceId" binding:"required"`
}

// DeleteEcsReq ECS删除请求
type DeleteEcsReq struct {
	Provider   CloudProvider `json:"provider" binding:"required"`
	Region     string        `json:"region" binding:"required"`
	InstanceId string        `json:"instanceId" binding:"required"`
}

// GetEcsDetailReq 获取ECS详情请求
type GetEcsDetailReq struct {
	Provider   CloudProvider `json:"provider" binding:"required"`
	Region     string        `json:"region" binding:"required"`
	InstanceId string        `json:"instanceId" binding:"required"`
}

// ListInstanceOptionsReq 实例选项列表请求
type ListInstanceOptionsReq struct {
	Provider           CloudProvider `json:"provider" binding:"required"`
	PayType            string        `json:"payType"`
	Region             string        `json:"region"`
	Zone               string        `json:"zone"`
	InstanceType       string        `json:"instanceType"`
	ImageId            string        `json:"imageId"`
	SystemDiskCategory string        `json:"systemDiskCategory"`
	DataDiskCategory   string        `json:"dataDiskCategory"`
	PageSize           int           `json:"pageSize"`
	PageNumber         int           `json:"pageNumber"`
}

type ListInstanceOptionsResp struct {
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

type ListRegionsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
}

type ListZonesReq struct {
}

type ListInstanceTypesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

type ListImagesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
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
