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

type TreeNodeStatus int8

const (
	ACTIVE TreeNodeStatus = iota + 1
	INACTIVE
)

type TreeNodeMemberType int8

const (
	AdminRole TreeNodeMemberType = iota + 1
	MemberRole
)

// 叶子节点标识常量，提升可读性，避免魔法数字
const (
	IsLeafYes int8 = 1 // 是叶子节点
	IsLeafNo  int8 = 2 // 不是叶子节点
)

// TreeNode 服务树节点结构
type TreeNode struct {
	Model
	Name               string               `json:"name" gorm:"type:varchar(50);not null;comment:节点名称"`      // 节点名称
	ParentID           int                  `json:"parent_id" gorm:"index;comment:父节点ID;default:0"`          // 父节点ID
	Level              int                  `json:"level" gorm:"comment:节点层级,默认在第1层;default:1"`              // 节点层级
	Description        string               `json:"description" gorm:"type:text;comment:节点描述"`               // 节点描述
	CreateUserID       int                  `json:"create_user_id" gorm:"comment:创建者ID;default:0"`           // 创建者ID
	CreateUserName     string               `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"` // 创建者姓名
	Status             TreeNodeStatus       `json:"status" gorm:"default:1;comment:节点状态, 1:活跃 2:非活跃"`        // 节点状态
	AdminUsers         []User               `json:"admins" gorm:"many2many:cl_tree_node_admin;"`             // 管理员多对多关系
	MemberUsers        []User               `json:"members" gorm:"many2many:cl_tree_node_member;"`           // 成员多对多关系
	IsLeaf             int8                 `json:"is_leaf" gorm:"comment:是否为叶子节点1:是 2:不是;default:2"`        // 是否为叶子节点
	Children           []*TreeNode          `json:"children" gorm:"-"`                                       // 子节点列表
	TreeLocalResources []*TreeLocalResource `json:"tree_local_resources" gorm:"many2many:cl_tree_node_local;"`
}

func (t *TreeNode) TableName() string {
	return "cl_tree_node"
}

// GetTreeNodeListReq 获取树节点列表请求
type GetTreeNodeListReq struct {
	Level  int            `json:"level" form:"level" binding:"omitempty,min=1"`
	Status TreeNodeStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
	Search string         `json:"search" form:"search" binding:"omitempty"`
}

// GetTreeNodeDetailReq 获取节点详情请求
type GetTreeNodeDetailReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetTreeNodeChildNodesReq 获取子节点列表请求
type GetTreeNodeChildNodesReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// CreateTreeNodeReq 创建节点请求
type CreateTreeNodeReq struct {
	Name           string         `json:"name" form:"name" binding:"required,min=1,max=50"`
	ParentID       int            `json:"parent_id" form:"parent_id"` // 父节点ID，0表示根节点
	CreateUserID   int            `json:"creator_id"`                 // 创建者ID
	CreateUserName string         `json:"creator_name"`               // 创建者姓名
	Description    string         `json:"description" form:"description"`
	IsLeaf         int8           `json:"is_leaf" form:"is_leaf" binding:"omitempty,oneof=1 2"`
	Status         TreeNodeStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}

// UpdateTreeNodeReq 更新节点请求
type UpdateTreeNodeReq struct {
	ID          int            `json:"id" form:"id" binding:"required"`
	Name        string         `json:"name" form:"name" binding:"required,min=1,max=50"`
	ParentID    int            `json:"parent_id" form:"parent_id"`
	Description string         `json:"description"`
	Status      TreeNodeStatus `json:"status" binding:"omitempty,oneof=1 2"`
	IsLeaf      int8           `json:"is_leaf" form:"is_leaf" binding:"omitempty,oneof=1 2"`
}

// UpdateTreeNodeStatusReq 更新节点状态请求
type UpdateTreeNodeStatusReq struct {
	ID     int            `json:"id" binding:"required"`
	Status TreeNodeStatus `json:"status" binding:"required,oneof=1 2"`
}

// DeleteTreeNodeReq 删除节点请求
type DeleteTreeNodeReq struct {
	ID int `json:"id" binding:"required"`
}

// MoveTreeNodeReq 移动节点请求
type MoveTreeNodeReq struct {
	ID          int `json:"id" form:"id" binding:"required"`
	NewParentID int `json:"new_parent_id" form:"new_parent_id" binding:"required"` // 新父节点ID，必填
}

// GetTreeNodeMembersReq 获取节点成员请求
type GetTreeNodeMembersReq struct {
	ID   int                `json:"id" binding:"required"`
	Type TreeNodeMemberType `json:"type" form:"type" binding:"omitempty,oneof=1 2"`
}

// AddTreeNodeMemberReq 添加节点成员请求
type AddTreeNodeMemberReq struct {
	NodeID     int                `json:"node_id" form:"node_id" binding:"required"`
	UserID     int                `json:"user_id" form:"user_id" binding:"required"`
	MemberType TreeNodeMemberType `json:"member_type" form:"member_type" binding:"required,oneof=1 2"`
}

// RemoveTreeNodeMemberReq 移除节点成员请求
type RemoveTreeNodeMemberReq struct {
	NodeID     int                `json:"node_id" form:"node_id" binding:"required"`
	UserID     int                `json:"user_id" form:"user_id" binding:"required"`
	MemberType TreeNodeMemberType `json:"member_type" form:"member_type" binding:"required,oneof=1 2"`
}

// BindTreeNodeResourceReq 绑定资源请求
type BindTreeNodeResourceReq struct {
	NodeID      int   `json:"node_id" binding:"required"`
	ResourceIDs []int `json:"resource_ids" binding:"required,min=1"`
}

// UnbindTreeNodeResourceReq 解绑资源请求
type UnbindTreeNodeResourceReq struct {
	NodeID     int `json:"node_id" binding:"required"`
	ResourceID int `json:"resource_id" binding:"required"`
}

// CheckTreeNodePermissionReq 检查节点权限请求
type CheckTreeNodePermissionReq struct {
	UserID    int    `json:"user_id" binding:"required"`
	NodeID    int    `json:"node_id" binding:"required"`
	Operation string `json:"operation" binding:"required"`
}

// GetUserTreeNodesReq 获取用户相关节点请求
type GetUserTreeNodesReq struct {
	UserID int                `json:"user_id" binding:"required"`
	Role   TreeNodeMemberType `json:"role" binding:"omitempty,oneof=1 2"`
}

// TreeNodeStatisticsResp 服务树统计响应
type TreeNodeStatisticsResp struct {
	TotalNodes     int `json:"total_nodes"`     // 节点总数
	TotalResources int `json:"total_resources"` // 资源总数
	TotalAdmins    int `json:"total_admins"`    // 管理员总数
	TotalMembers   int `json:"total_members"`   // 成员总数
	ActiveNodes    int `json:"active_nodes"`    // 活跃节点数
	InactiveNodes  int `json:"inactive_nodes"`  // 非活跃节点数
}
