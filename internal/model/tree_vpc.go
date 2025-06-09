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

// ResourceVpc VPC资源
type ResourceVpc struct {
	Model

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
	CreateByOrder   bool       `json:"create_by_order" gorm:"comment:是否由工单创建"`
	LastSyncTime    time.Time  `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID      int        `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
	VpcName         string     `json:"vpc_name" gorm:"type:varchar(100);comment:VPC名称"`
	CidrBlock       string     `json:"cidr_block" gorm:"type:varchar(50);comment:IPv4网段"`
	Ipv6CidrBlock   string     `json:"ipv6_cidr_block" gorm:"type:varchar(50);comment:IPv6网段"`
	VSwitchIds      StringList `json:"vswitch_ids" gorm:"type:varchar(500);comment:交换机ID列表"`
	RouteTableIds   StringList `json:"route_table_ids" gorm:"type:varchar(500);comment:路由表ID列表"`
	NatGatewayIds   StringList `json:"nat_gateway_ids" gorm:"type:varchar(500);comment:NAT网关ID列表"`
	IsDefault       bool       `json:"is_default" gorm:"comment:是否为默认VPC"`
	ResourceGroupId string     `json:"resource_group_id" gorm:"type:varchar(100);comment:资源组ID"`

	// 多对多关系
	VpcTreeNodes []*TreeNode `json:"vpc_tree_nodes" gorm:"many2many:resource_vpc_tree_nodes;comment:关联服务树节点"`
}

// ResourceSubnet 子网资源
type ResourceSubnet struct {
	Model

	SubnetId            string        `json:"subnet_id" gorm:"type:varchar(100);comment:子网ID"`
	SubnetName          string        `json:"subnet_name" gorm:"type:varchar(100);comment:子网名称"`
	VpcId               string        `json:"vpc_id" gorm:"type:varchar(100);comment:所属VPC ID"`
	Provider            CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	RegionId            string        `json:"region_id" gorm:"type:varchar(50);comment:地区"`
	ZoneId              string        `json:"zone_id" gorm:"type:varchar(100);comment:可用区ID"`
	CidrBlock           string        `json:"cidr_block" gorm:"type:varchar(50);comment:IPv4网段"`
	Ipv6CidrBlock       string        `json:"ipv6_cidr_block" gorm:"type:varchar(50);comment:IPv6网段"`
	Status              string        `json:"status" gorm:"type:varchar(50);comment:状态"`
	CreationTime        string        `json:"creation_time" gorm:"type:varchar(30);comment:创建时间"`
	Description         string        `json:"description" gorm:"type:text;comment:描述"`
	AvailableIpCount    int           `json:"available_ip_count" gorm:"comment:可用IP数量"`
	TotalIpCount        int           `json:"total_ip_count" gorm:"comment:总IP数量"`
	Tags                StringList    `json:"tags" gorm:"type:varchar(500);comment:标签"`
	RouteTableId        string        `json:"route_table_id" gorm:"type:varchar(100);comment:关联路由表ID"`
	NetworkAclId        string        `json:"network_acl_id" gorm:"type:varchar(100);comment:网络ACL ID"`
	IsDefault           bool          `json:"is_default" gorm:"comment:是否为默认子网"`
	MapPublicIpOnLaunch bool          `json:"map_public_ip_on_launch" gorm:"comment:是否在启动时分配公网IP"`

	// 关联关系
	TreeNodeID int         `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
	TreeNodes  []*TreeNode `json:"tree_nodes" gorm:"many2many:resource_subnet_tree_nodes;comment:关联服务树节点"`
}

// ResourceVpcPeering VPC对等连接资源
type ResourceVpcPeering struct {
	Model

	PeeringId          string        `json:"peering_id" gorm:"type:varchar(100);comment:对等连接ID"`
	PeeringName        string        `json:"peering_name" gorm:"type:varchar(100);comment:对等连接名称"`
	Provider           CloudProvider `json:"provider" gorm:"type:varchar(50);comment:云厂商"`
	RegionId           string        `json:"region_id" gorm:"type:varchar(50);comment:地区"`
	LocalVpcId         string        `json:"local_vpc_id" gorm:"type:varchar(100);comment:本端VPC ID"`
	PeerVpcId          string        `json:"peer_vpc_id" gorm:"type:varchar(100);comment:对端VPC ID"`
	PeerRegionId       string        `json:"peer_region_id" gorm:"type:varchar(50);comment:对端地区"`
	PeerAccountId      string        `json:"peer_account_id" gorm:"type:varchar(100);comment:对端账号ID"`
	Status             string        `json:"status" gorm:"type:varchar(50);comment:状态"`
	CreationTime       string        `json:"creation_time" gorm:"type:varchar(30);comment:创建时间"`
	AcceptanceRequired bool          `json:"acceptance_required" gorm:"comment:是否需要接受"`
	Description        string        `json:"description" gorm:"type:text;comment:描述"`
	Tags               StringList    `json:"tags" gorm:"type:varchar(500);comment:标签"`

	// 关联关系
	TreeNodeID int         `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
	TreeNodes  []*TreeNode `json:"tree_nodes" gorm:"many2many:resource_vpc_peering_tree_nodes;comment:关联服务树节点"`
}

// =========================== VPC相关请求结构体 ===========================

// GetVpcDetailReq 获取VPC详情请求
type GetVpcDetailReq struct {
	ID       int           `json:"id" uri:"id" binding:"required"`
	Provider CloudProvider `json:"provider"`
	Region   string        `json:"region"`
	VpcId    string        `json:"vpc_id"`
}

// CreateVpcResourceReq 创建VPC资源请求
type CreateVpcResourceReq struct {
	Provider         CloudProvider     `json:"provider" binding:"required"`
	Region           string            `json:"region" binding:"required"`
	ZoneId           string            `json:"zone_id" binding:"required"`
	VpcName          string            `json:"vpc_name" binding:"required"`
	Description      string            `json:"description"`
	CidrBlock        string            `json:"cidr_block" binding:"required"`
	VSwitchName      string            `json:"vswitch_name" binding:"required"`
	VSwitchCidrBlock string            `json:"vswitch_cidr_block" binding:"required"`
	DryRun           bool              `json:"dry_run"`
	Tags             map[string]string `json:"tags"`
	TreeNodeID       int               `json:"tree_node_id"`
	Env              string            `json:"environment"`
}

// DeleteVpcReq 删除VPC请求
type DeleteVpcReq struct {
	ID       int           `json:"id" uri:"id" binding:"required"`
	Provider CloudProvider `json:"provider"`
	Region   string        `json:"region"`
	VpcId    string        `json:"vpc_id"`
	Force    bool          `json:"force"` // 是否强制删除
}

// ListVpcResourcesReq 获取VPC列表请求
type ListVpcResourcesReq struct {
	PageNumber int           `json:"page_number" form:"page_number"`
	PageSize   int           `json:"page_size" form:"page_size"`
	Provider   CloudProvider `json:"provider" form:"provider"`
	Region     string        `json:"region" form:"region"`
	Status     string        `json:"status" form:"status"`
	VpcName    string        `json:"vpc_name" form:"vpc_name"`
	TreeNodeID int           `json:"tree_node_id" form:"tree_node_id"`
	Env        string        `json:"environment" form:"environment"`
}

// UpdateVpcReq 更新VPC请求
type UpdateVpcReq struct {
	ID          int               `json:"id" uri:"id" binding:"required"`
	VpcName     string            `json:"vpc_name"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	TreeNodeID  int               `json:"tree_node_id"`
	Env         string            `json:"environment"`
}

// =========================== 子网相关请求结构体 ===========================

// CreateSubnetReq 创建子网请求
type CreateSubnetReq struct {
	VpcId               string            `json:"vpc_id" binding:"required"`
	Provider            CloudProvider     `json:"provider" binding:"required"`
	Region              string            `json:"region" binding:"required"`
	ZoneId              string            `json:"zone_id" binding:"required"`
	SubnetName          string            `json:"subnet_name" binding:"required"`
	CidrBlock           string            `json:"cidr_block" binding:"required"`
	Description         string            `json:"description"`
	MapPublicIpOnLaunch bool              `json:"map_public_ip_on_launch"`
	Tags                map[string]string `json:"tags"`
	TreeNodeID          int               `json:"tree_node_id"`
}

// DeleteSubnetReq 删除子网请求
type DeleteSubnetReq struct {
	ID       int           `json:"id" uri:"id" binding:"required"`
	Provider CloudProvider `json:"provider"`
	Region   string        `json:"region"`
	SubnetId string        `json:"subnet_id"`
	Force    bool          `json:"force"`
}

// ListSubnetsReq 获取子网列表请求
type ListSubnetsReq struct {
	PageNumber int           `json:"page_number" form:"page_number"`
	PageSize   int           `json:"page_size" form:"page_size"`
	Provider   CloudProvider `json:"provider" form:"provider"`
	Region     string        `json:"region" form:"region"`
	VpcId      string        `json:"vpc_id" form:"vpc_id"`
	ZoneId     string        `json:"zone_id" form:"zone_id"`
	Status     string        `json:"status" form:"status"`
	SubnetName string        `json:"subnet_name" form:"subnet_name"`
	TreeNodeID int           `json:"tree_node_id" form:"tree_node_id"`
}

// GetSubnetDetailReq 获取子网详情请求
type GetSubnetDetailReq struct {
	ID       int           `json:"id" uri:"id" binding:"required"`
	Provider CloudProvider `json:"provider"`
	Region   string        `json:"region"`
	SubnetId string        `json:"subnet_id"`
}

// UpdateSubnetReq 更新子网请求
type UpdateSubnetReq struct {
	ID                  int               `json:"id" uri:"id" binding:"required"`
	SubnetName          string            `json:"subnet_name"`
	Description         string            `json:"description"`
	MapPublicIpOnLaunch bool              `json:"map_public_ip_on_launch"`
	Tags                map[string]string `json:"tags"`
	TreeNodeID          int               `json:"tree_node_id"`
}

// =========================== VPC对等连接相关请求结构体 ===========================

// CreateVpcPeeringReq 创建VPC对等连接请求
type CreateVpcPeeringReq struct {
	ID            int               `json:"id" uri:"id" binding:"required"` // 从路径参数获取的VPC ID
	Provider      CloudProvider     `json:"provider" binding:"required"`
	Region        string            `json:"region" binding:"required"`
	LocalVpcId    string            `json:"local_vpc_id" binding:"required"`
	PeerVpcId     string            `json:"peer_vpc_id" binding:"required"`
	PeerRegionId  string            `json:"peer_region_id"`
	PeerAccountId string            `json:"peer_account_id"`
	PeeringName   string            `json:"peering_name" binding:"required"`
	Description   string            `json:"description"`
	Tags          map[string]string `json:"tags"`
	TreeNodeID    int               `json:"tree_node_id"`
}

// DeleteVpcPeeringReq 删除VPC对等连接请求
type DeleteVpcPeeringReq struct {
	ID        int           `json:"id" uri:"id" binding:"required"`
	Provider  CloudProvider `json:"provider"`
	Region    string        `json:"region"`
	PeeringId string        `json:"peering_id"`
	Force     bool          `json:"force"`
}

// ListVpcPeeringsReq 获取VPC对等连接列表请求
type ListVpcPeeringsReq struct {
	PageNumber  int           `json:"page_number" form:"page_number"`
	PageSize    int           `json:"page_size" form:"page_size"`
	Provider    CloudProvider `json:"provider" form:"provider"`
	Region      string        `json:"region" form:"region"`
	LocalVpcId  string        `json:"local_vpc_id" form:"local_vpc_id"`
	PeerVpcId   string        `json:"peer_vpc_id" form:"peer_vpc_id"`
	Status      string        `json:"status" form:"status"`
	PeeringName string        `json:"peering_name" form:"peering_name"`
	TreeNodeID  int           `json:"tree_node_id" form:"tree_node_id"`
}
