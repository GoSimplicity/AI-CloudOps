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

// ResourceVpc VPC资源
type ResourceVpc struct {
	ResourceBase
	VpcName         string     `json:"vpcName" gorm:"type:varchar(100);comment:VPC名称"`
	CidrBlock       string     `json:"cidrBlock" gorm:"type:varchar(50);comment:IPv4网段"`
	Ipv6CidrBlock   string     `json:"ipv6CidrBlock" gorm:"type:varchar(50);comment:IPv6网段"`
	VSwitchIds      StringList `json:"vSwitchIds" gorm:"type:varchar(500);comment:交换机ID列表"`
	RouteTableIds   StringList `json:"routeTableIds" gorm:"type:varchar(500);comment:路由表ID列表"`
	NatGatewayIds   StringList `json:"natGatewayIds" gorm:"type:varchar(500);comment:NAT网关ID列表"`
	IsDefault       bool       `json:"isDefault" gorm:"comment:是否为默认VPC"`
	ResourceGroupId string     `json:"resourceGroupId" gorm:"type:varchar(100);comment:资源组ID"`
	// 多对多关系
	VpcTreeNodes []*TreeNode `json:"vpcTreeNodes" gorm:"many2many:resource_vpc_tree_nodes;comment:关联服务树节点"`
}

// CreateVpcResourceReq VPC创建参数
type CreateVpcResourceReq struct {
	Provider         CloudProvider     `json:"provider" binding:"required"`
	Region           string            `json:"region" binding:"required"`
	ZoneId           string            `json:"zoneId" binding:"required"`
	VpcName          string            `json:"vpcName" binding:"required"`
	Description      string            `json:"description"`
	CidrBlock        string            `json:"cidrBlock" binding:"required"`        // cidr网段
	VSwitchName      string            `json:"vSwitchName" binding:"required"`      // 交换机名称
	VSwitchCidrBlock string            `json:"vSwitchCidrBlock" binding:"required"` // 交换机网段
	DryRun           bool              `json:"dryRun"`                              // 是否仅预览而不创建
	Tags             map[string]string `json:"tags"`
}

// ListVpcResourcesReq VPC资源列表查询参数
type ListVpcResourcesReq struct {
	PageNumber int           `form:"pageNumber" json:"pageNumber"`
	PageSize   int           `form:"pageSize" json:"pageSize"`
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
}

// ResourceVPCListResp VPC资源列表响应
type ResourceVPCListResp struct {
	Total int64          `json:"total"`
	Data  []*ResourceVpc `json:"data"`
}

// ResourceVPCDetailResp VPC资源详情响应
type ResourceVPCDetailResp struct {
	Data *ResourceVpc `json:"data"`
}

// GetVpcDetailReq 获取VPC详情请求
type GetVpcDetailReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	VpcId    string        `json:"vpcId" binding:"required"`
}

// DeleteVpcReq VPC删除请求
type DeleteVpcReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	VpcId    string        `json:"vpcId" binding:"required"`
}
