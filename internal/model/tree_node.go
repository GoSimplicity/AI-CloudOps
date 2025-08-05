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

// TreeNode 服务树节点结构
type TreeNode struct {
	Model
	Name        string `json:"name" gorm:"type:varchar(50);not null;comment:节点名称"`                                   // 节点名称
	ParentID    int    `json:"parentId" gorm:"index;comment:父节点ID;default:0"`                                        // 父节点ID
	Level       int    `json:"level" gorm:"comment:节点层级,默认在第1层;default:1"`                                           // 节点层级
	Description string `json:"description" gorm:"type:text;comment:节点描述"`                                            // 节点描述
	CreatorID   int    `json:"creator_id" gorm:"comment:创建者ID;default:0"`                                            // 创建者ID
	Status      string `json:"status" gorm:"type:varchar(20);default:active;comment:节点状态"`                           // 节点状态：active, inactive, deleted
	AdminUsers  []User `json:"admins" gorm:"many2many:cl_tree_node_admin;joinForeignKey:ID;joinReferences:UserID"`   // 管理员多对多关系
	MemberUsers []User `json:"members" gorm:"many2many:cl_tree_node_member;joinForeignKey:ID;joinReferences:UserID"` // 成员多对多关系
	IsLeaf      bool   `json:"isLeaf" gorm:"comment:是否为叶子节点;default:false"`                                          // 是否为叶子节点

	// 非数据库字段
	ChildCount    int         `json:"child_count" gorm:"-"`    // 子节点数量
	ResourceCount int         `json:"resource_count" gorm:"-"` // 关联资源数量
	ParentName    string      `json:"parent_name" gorm:"-"`    // 父节点名称
	CreatorName   string      `json:"creator_name" gorm:"-"`
	Children      []*TreeNode `json:"children" gorm:"-"` // 子节点列表
}

func (t *TreeNode) TableName() string {
	return "cl_tree_nodes"
}

// TreeNodeResource 节点资源关联表
type TreeNodeResource struct {
	Model
	TreeNodeID   int           `json:"tree_node_id" gorm:"index:idx_node_resource;not null;comment:节点ID"`
	ResourceID   string        `json:"resource_id" gorm:"index:idx_node_resource;not null;comment:资源ID"`
	ResourceType CloudProvider `json:"resource_type" gorm:"type:varchar(50);not null;comment:资源类型，可选：ecs, elb, rds, local"`
}

func (t *TreeNodeResource) TableName() string {
	return "cl_tree_node_resources"
}

type ResourceItems struct {
	ResourceID   string        `json:"resource_id"`
	ResourceName string        `json:"resource_name"`
	ResourceType CloudProvider `json:"resource_type"`
	Status       string        `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// GetTreeListReq 获取树节点列表请求
type GetTreeNodeListReq struct {
	Level  int    `json:"level" form:"level" binding:"omitempty,min=1"`
	Status string `json:"status" form:"status" binding:"omitempty,oneof=active inactive deleted"`
}

// GetTreeNodeDetailReq 获取节点详情请求
type GetTreeNodeDetailReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetChildNodesReq 获取子节点列表请求
type GetTreeNodeChildNodesReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// CreateNodeReq 创建节点请求
type CreateTreeNodeReq struct {
	Name        string `json:"name" form:"name" binding:"required,min=1,max=50"`
	ParentID    int    `json:"parent_id" form:"parent_id"` // 父节点ID，0表示根节点
	CreatorID   int    `json:"creator_id" form:"creator_id"`
	Description string `json:"description" form:"description"`
	IsLeaf      bool   `json:"is_leaf" form:"is_leaf"`
	Status      string `json:"status" form:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateNodeReq 更新节点请求
type UpdateTreeNodeReq struct {
	ID          int    `json:"id" form:"id" binding:"required"`
	Name        string `json:"name" form:"name" binding:"required,min=1,max=50"`
	ParentID    int    `json:"parent_id" form:"parent_id"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateNodeStatusReq 更新节点状态请求
type UpdateTreeNodeStatusReq struct {
	ID     int    `json:"id" binding:"required"`
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// DeleteNodeReq 删除节点请求
type DeleteTreeNodeReq struct {
	ID int `json:"id" binding:"required"`
}

// MoveNodeReq 移动节点请求
type MoveTreeNodeReq struct {
	ID          int `json:"id" form:"id" binding:"required"`
	NewParentID int `json:"new_parent_id" form:"new_parent_id" binding:"required"` // 新父节点ID，必填
}

// GetNodeMembersReq 获取节点成员请求
type GetTreeNodeMembersReq struct {
	ID   int    `json:"id" binding:"required"`
	Type string `json:"type" binding:"omitempty,oneof=admin member"`
}

// AddNodeMemberReq 添加节点成员请求
type AddTreeNodeMemberReq struct {
	NodeID     int    `json:"node_id" form:"node_id" binding:"required"`
	UserID     int    `json:"user_id" form:"user_id" binding:"required"`
	MemberType string `json:"member_type" form:"member_type" binding:"required,oneof=admin member"`
}

// RemoveNodeMemberReq 移除节点成员请求
type RemoveTreeNodeMemberReq struct {
	NodeID     int    `json:"node_id" form:"node_id" binding:"required"`
	UserID     int    `json:"user_id" form:"user_id" binding:"required"`
	MemberType string `json:"member_type" form:"member_type" binding:"required,oneof=admin member"`
}

// GetNodeResourcesReq 获取节点资源请求
type GetTreeNodeResourcesReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// BindResourceReq 绑定资源请求
type BindTreeNodeResourceReq struct {
	NodeID       int           `json:"node_id" binding:"required"`
	ResourceType CloudProvider `json:"resource_type" binding:"omitempty,oneof=ecs elb rds local"`
	ResourceIDs  []string      `json:"resource_ids" binding:"required,min=1"`
}

// UnbindResourceReq 解绑资源请求
type UnbindTreeNodeResourceReq struct {
	NodeID       int           `json:"node_id" binding:"required"`
	ResourceID   string        `json:"resource_id" binding:"required"`
	ResourceType CloudProvider `json:"resource_type" binding:"required"`
}

// CheckNodePermissionReq 检查节点权限请求
type CheckTreeNodePermissionReq struct {
	UserID    int    `json:"user_id" binding:"required"`
	NodeID    int    `json:"node_id" binding:"required"`
	Operation string `json:"operation" binding:"required"`
}

// GetUserNodesReq 获取用户相关节点请求
type GetUserTreeNodesReq struct {
	UserID int    `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"omitempty,oneof=admin member"`
}

// TreeStatisticsResp 服务树统计响应
type TreeNodeStatisticsResp struct {
	TotalNodes     int `json:"total_nodes"`     // 节点总数
	TotalResources int `json:"total_resources"` // 资源总数
	TotalAdmins    int `json:"total_admins"`    // 管理员总数
	TotalMembers   int `json:"total_members"`   // 成员总数
	ActiveNodes    int `json:"active_nodes"`    // 活跃节点数
	InactiveNodes  int `json:"inactive_nodes"`  // 非活跃节点数
}
