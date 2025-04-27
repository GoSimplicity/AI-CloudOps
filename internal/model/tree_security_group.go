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

// ResourceSecurityGroup 安全组资源
type ResourceSecurityGroup struct {
	ResourceBase
	SecurityGroupName  string               `json:"securityGroupName" gorm:"type:varchar(100);comment:安全组名称"`
	Description        string               `json:"description" gorm:"type:varchar(255);comment:安全组描述"`
	VpcId              string               `json:"vpcId" gorm:"type:varchar(100);comment:VPC ID"`
	SecurityGroupType  string               `json:"securityGroupType" gorm:"type:varchar(50);comment:安全组类型"`
	ResourceGroupId    string               `json:"resourceGroupId" gorm:"type:varchar(100);comment:资源组ID"`
	SecurityGroupRules []*SecurityGroupRule `json:"securityGroupRules,omitempty" gorm:"foreignKey:SecurityGroupID;references:ID;comment:安全组规则"`
	// 多对多关系
	SecurityGroupTreeNodes []*TreeNode `json:"securityGroupTreeNodes" gorm:"many2many:resource_security_group_tree_nodes;comment:关联服务树节点"`
}

// SecurityGroupRule 安全组规则
type SecurityGroupRule struct {
	ID              uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	SecurityGroupID uint   `json:"securityGroupId" gorm:"comment:安全组ID"`
	IpProtocol      string `json:"ipProtocol" gorm:"type:varchar(20);comment:IP协议"`
	PortRange       string `json:"portRange" gorm:"type:varchar(50);comment:端口范围"`
	Direction       string `json:"direction" gorm:"type:varchar(20);comment:方向:ingress(入)、egress(出)"`
	Policy          string `json:"policy" gorm:"type:varchar(20);comment:授权策略:accept(接受)、drop(拒绝)"`
	Priority        int    `json:"priority" gorm:"comment:优先级:1-100,默认1"`
	SourceCidrIp    string `json:"sourceCidrIp" gorm:"type:varchar(50);comment:源IP地址段(入方向)"`
	DestCidrIp      string `json:"destCidrIp" gorm:"type:varchar(50);comment:目标IP地址段(出方向)"`
	SourceGroupId   string `json:"sourceGroupId" gorm:"type:varchar(100);comment:源安全组ID(入方向)"`
	DestGroupId     string `json:"destGroupId" gorm:"type:varchar(100);comment:目标安全组ID(出方向)"`
	Description     string `json:"description" gorm:"type:varchar(255);comment:规则描述"`
}

// CreateSecurityGroupReq 创建安全组请求
type CreateSecurityGroupReq struct {
	Provider           CloudProvider        `json:"provider" binding:"required"`
	Region             string               `json:"region" binding:"required"`
	SecurityGroupName  string               `json:"securityGroupName" binding:"required"`
	Description        string               `json:"description"`
	VpcId              string               `json:"vpcId" binding:"required"`
	SecurityGroupType  string               `json:"securityGroupType"`
	ResourceGroupId    string               `json:"resourceGroupId"`
	TreeNodeId         uint                 `json:"treeNodeId" binding:"required"`
	SecurityGroupRules []*SecurityGroupRule `json:"securityGroupRules"`
	Tags               map[string]string    `json:"tags"`
}

// ListSecurityGroupsReq 安全组列表查询参数
type ListSecurityGroupsReq struct {
	PageNumber int           `form:"pageNumber" json:"pageNumber"`
	PageSize   int           `form:"pageSize" json:"pageSize"`
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
}

// ResourceSecurityGroupListResp 安全组列表响应
type ResourceSecurityGroupListResp struct {
	Total int64                    `json:"total"`
	Data  []*ResourceSecurityGroup `json:"data"`
}

// ResourceSecurityGroupDetailResp 安全组详情响应
type ResourceSecurityGroupDetailResp struct {
	Data *ResourceSecurityGroup `json:"data"`
}

// GetSecurityGroupDetailReq 获取安全组详情请求
type GetSecurityGroupDetailReq struct {
	Provider        CloudProvider `json:"provider" binding:"required"`
	Region          string        `json:"region" binding:"required"`
	SecurityGroupId string        `json:"securityGroupId" binding:"required"`
}

// DeleteSecurityGroupReq 删除安全组请求
type DeleteSecurityGroupReq struct {
	Provider        CloudProvider `json:"provider" binding:"required"`
	Region          string        `json:"region" binding:"required"`
	SecurityGroupId string        `json:"securityGroupId" binding:"required"`
}

// AddSecurityGroupRuleReq 添加安全组规则请求
type AddSecurityGroupRuleReq struct {
	Provider        CloudProvider      `json:"provider" binding:"required"`
	Region          string             `json:"region" binding:"required"`
	SecurityGroupId string             `json:"securityGroupId" binding:"required"`
	Rule            *SecurityGroupRule `json:"rule" binding:"required"`
}

// RemoveSecurityGroupRuleReq 删除安全组规则请求
type RemoveSecurityGroupRuleReq struct {
	Provider        CloudProvider `json:"provider" binding:"required"`
	Region          string        `json:"region" binding:"required"`
	SecurityGroupId string        `json:"securityGroupId" binding:"required"`
	RuleId          uint          `json:"ruleId" binding:"required"`
}

// ListSecurityGroupRulesReq 获取安全组规则列表请求
type ListSecurityGroupRulesReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
}

// ResourceSecurityGroupRuleListResp 安全组规则列表响应
type ResourceSecurityGroupRuleListResp struct {
	Total int64                    `json:"total"`
	Data  []*ResourceSecurityGroup `json:"data"`
}

