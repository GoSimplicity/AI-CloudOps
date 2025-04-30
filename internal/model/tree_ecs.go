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
	ComputeResource
	OsType            string     `json:"osType" gorm:"type:varchar(50);comment:操作系统类型,如win,linux"`
	VmType            int        `json:"vmType" gorm:"default:1;comment:设备类型,1=虚拟设备,2=物理设备"`
	OSName            string     `json:"osName" gorm:"type:varchar(100);comment:操作系统名称"`
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
	Provider           CloudProvider     `json:"provider" binding:"required"`
	Region             string            `json:"region" binding:"required"`
	ZoneId             string            `json:"zoneId" binding:"required"`
	InstanceType       string            `json:"instanceType" binding:"required"`
	ImageId            string            `json:"imageId" binding:"required"`
	VSwitchId          string            `json:"vSwitchId" binding:"required"`
	SecurityGroupIds   []string          `json:"securityGroupIds" binding:"required"`
	Amount             int               `json:"amount" binding:"required,min=1,max=100"` // 创建数量
	Hostname           string            `json:"hostname" binding:"required"`             // 主机名
	Password           string            `json:"password" binding:"required"`             // 密码
	InstanceName       string            `json:"instanceName"`                            // 实例名称
	TreeNodeId         int               `json:"treeNodeId"`
	Description        string            `json:"description"`
	SystemDiskCategory string            `json:"systemDiskCategory"`
	AutoRenewPeriod    int               `json:"autoRenewPeriod"`
	PeriodUnit         string            `json:"periodUnit"`         // Month 月 Year 年
	Period             int               `json:"period"`             // 购买时长
	AutoRenew          bool              `json:"autoRenew"`          // 是否自动续费
	InstanceChargeType string            `json:"instanceChargeType"` // 付费类型
	SpotStrategy       string            `json:"spotStrategy"`       // NoSpot 默认值 表示正常按量付费 SpotAsPriceGo 表示自动竞价
	SpotDuration       int               `json:"spotDuration"`       // 竞价时长
	SystemDiskSize     int               `json:"systemDiskSize"`     // 系统盘大小
	DataDiskSize       int               `json:"dataDiskSize"`       // 数据盘大小
	DataDiskCategory   string            `json:"dataDiskCategory"`   // 数据盘类型
	DryRun             bool              `json:"dryRun"`             // 是否仅预览而不创建
	Tags               map[string]string `json:"tags"`
}

// ListEcsResourcesReq ECS资源列表查询参数
type ListEcsResourcesReq struct {
	PageNumber int           `form:"pageNumber" json:"pageNumber"`
	PageSize   int           `form:"pageSize" json:"pageSize"`
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
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
