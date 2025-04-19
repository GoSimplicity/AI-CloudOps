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

// ListElbResourcesReq ELB资源列表查询参数
type ListElbResourcesReq struct {
	PageNumber int           `form:"pageNumber" json:"pageNumber"`
	PageSize   int           `form:"pageSize" json:"pageSize"`
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
}

// ResourceELBResp ELB资源响应
type ResourceELBResp struct {
	ResourceElb
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type GetElbDetailReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	ElbId    string        `json:"elbId" binding:"required"`
}

type DeleteElbReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	ElbId    string        `json:"elbId" binding:"required"`
}
