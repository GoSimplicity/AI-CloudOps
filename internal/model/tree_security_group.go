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

// ResourceSecurityGroup 安全组资源
type ResourceSecurityGroup struct {
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
	CreateByOrder      bool                 `json:"create_by_order" gorm:"comment:是否由工单创建"`
	LastSyncTime       time.Time            `json:"last_sync_time" gorm:"comment:最后同步时间"`
	TreeNodeID         int                  `json:"tree_node_id" gorm:"comment:关联的服务树节点ID"`
	SecurityGroupName  string               `json:"security_group_name" gorm:"type:varchar(100);comment:安全组名称"`
	SecurityGroupType  string               `json:"security_group_type" gorm:"type:varchar(50);comment:安全组类型"`
	ResourceGroupId    string               `json:"resource_group_id" gorm:"type:varchar(100);comment:资源组ID"`
	SecurityGroupRules []*SecurityGroupRule `json:"security_group_rules,omitempty" gorm:"foreignKey:SecurityGroupID;references:ID;comment:安全组规则"`

	// 多对多关系
	SecurityGroupTreeNodes []*TreeNode `json:"security_group_tree_nodes" gorm:"many2many:resource_security_group_tree_nodes;comment:关联服务树节点"`
}

// SecurityGroupRule 安全组规则
type SecurityGroupRule struct {
	ID              int   `json:"id" gorm:"primaryKey;autoIncrement"`
	SecurityGroupID int   `json:"security_group_id" gorm:"comment:安全组ID"`
	IpProtocol      string `json:"ip_protocol" gorm:"type:varchar(20);comment:IP协议"`
	PortRange       string `json:"port_range" gorm:"type:varchar(50);comment:端口范围"`
	Direction       string `json:"direction" gorm:"type:varchar(20);comment:方向:ingress(入)、egress(出)"`
	Policy          string `json:"policy" gorm:"type:varchar(20);comment:授权策略:accept(接受)、drop(拒绝)"`
	Priority        int    `json:"priority" gorm:"comment:优先级:1-100,默认1"`
	SourceCidrIp    string `json:"source_cidr_ip" gorm:"type:varchar(50);comment:源IP地址段(入方向)"`
	DestCidrIp      string `json:"dest_cidr_ip" gorm:"type:varchar(50);comment:目标IP地址段(出方向)"`
	SourceGroupId   string `json:"source_group_id" gorm:"type:varchar(100);comment:源安全组ID(入方向)"`
	DestGroupId     string `json:"dest_group_id" gorm:"type:varchar(100);comment:目标安全组ID(出方向)"`
	Description     string `json:"description" gorm:"type:varchar(255);comment:规则描述"`
}

// CreateSecurityGroupReq 创建安全组请求
type CreateSecurityGroupReq struct {
	Provider           CloudProvider        `json:"provider" binding:"required"`
	Region             string               `json:"region" binding:"required"`
	SecurityGroupName  string               `json:"security_group_name" binding:"required"`
	Description        string               `json:"description"`
	VpcId              string               `json:"vpc_id" binding:"required"`
	SecurityGroupType  string               `json:"security_group_type"`
	ResourceGroupId    string               `json:"resource_group_id"`
	TreeNodeId         int                 `json:"tree_node_id"`
	SecurityGroupRules []*SecurityGroupRule `json:"security_group_rules"`
	Tags               map[string]string    `json:"tags"`
}

// DeleteSecurityGroupReq 删除安全组请求
type DeleteSecurityGroupReq struct {
	ID int `json:"id"`
}

// ListSecurityGroupsReq 安全组列表查询参数
type ListSecurityGroupsReq struct {
	ListReq
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
	VpcId      string        `form:"vpc_id" json:"vpc_id"`
	TreeNodeId int          `form:"tree_node_id" json:"tree_node_id"`
	Status     string        `form:"status" json:"status"`
	Env        string        `form:"env" json:"env"`
}

// GetSecurityGroupDetailReq 获取安全组详情请求
type GetSecurityGroupDetailReq struct {
	ID int `json:"id"`
}

// UpdateSecurityGroupReq 更新安全组请求
type UpdateSecurityGroupReq struct {
	ID                int              `json:"id"`
	SecurityGroupName string            `json:"security_group_name"`
	Description       string            `json:"description"`
	Tags              map[string]string `json:"tags"`
}

// AddSecurityGroupRuleReq 添加安全组规则请求
type AddSecurityGroupRuleReq struct {
	ID    int                 `json:"id"`
	Rules []*SecurityGroupRule `json:"rules" binding:"required"`
}

// RemoveSecurityGroupRuleReq 删除安全组规则请求
type RemoveSecurityGroupRuleReq struct {
	ID      int   `json:"id"`
	RuleIds []int `json:"rule_ids" binding:"required"`
}

// BindInstanceToSecurityGroupReq 绑定实例到安全组请求
type BindInstanceToSecurityGroupReq struct {
	ID          int     `json:"id"`
	InstanceIds []string `json:"instance_ids" binding:"required"`
}

// UnbindInstanceFromSecurityGroupReq 解绑实例从安全组请求
type UnbindInstanceFromSecurityGroupReq struct {
	ID          int     `json:"id"`
	InstanceIds []string `json:"instance_ids" binding:"required"`
}
