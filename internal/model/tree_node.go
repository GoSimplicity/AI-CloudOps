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
	Name        string           `json:"name" gorm:"type:varchar(50);not null;comment:节点名称"`                                // 节点名称
	ParentID    int              `json:"parentId" gorm:"index;comment:父节点ID;default:0"`                                     // 父节点ID
	Level       int              `json:"level" gorm:"comment:节点层级,默认在第1层;default:1"`                                        // 节点层级
	Description string           `json:"description" gorm:"type:text;comment:节点描述"`                                         // 节点描述
	CreatorID   int              `json:"creatorId" gorm:"comment:创建者ID;default:0"`                                          // 创建者ID
	Status      string           `json:"status" gorm:"type:varchar(20);default:active;comment:节点状态"`                        // 节点状态：active, inactive, deleted
	Admins      []TreeNodeAdmin  `json:"admins" gorm:"many2many:tree_node_admin;joinForeignKey:ID;joinReferences:UserID"`   // 管理员多对多关系
	Members     []TreeNodeMember `json:"members" gorm:"many2many:tree_node_member;joinForeignKey:ID;joinReferences:UserID"` // 成员多对多关系
	IsLeaf      bool             `json:"isLeaf" gorm:"comment:是否为叶子节点;default:false"`                                       // 是否为叶子节点

	// 非数据库字段
	ChildCount    int         `json:"childCount" gorm:"-"`    // 子节点数量
	ResourceCount int         `json:"resourceCount" gorm:"-"` // 关联资源数量
	ParentName    string      `json:"parentName" gorm:"-"`    // 父节点名称
	CreatorName   string      `json:"creatorName" gorm:"-"`
	AdminUsers    StringList  `json:"adminUsers" gorm:"-"`  // 管理员用户名列表
	MemberUsers   StringList  `json:"memberUsers" gorm:"-"` // 成员用户名列表
	Children      []*TreeNode `json:"children" gorm:"-"`    // 子节点列表
}

// TreeNodeAdmin 节点管理员关联表
type TreeNodeAdmin struct {
	ID         int `gorm:"primaryKey;autoIncrement"`
	TreeNodeID int `gorm:"index:idx_node_admin,unique;not null;comment:节点ID"`
	UserID     int `gorm:"index:idx_node_admin,unique;not null;comment:用户ID"`
}

// TreeNodeMember 节点成员关联表
type TreeNodeMember struct {
	ID         int `gorm:"primaryKey;autoIncrement"`
	TreeNodeID int `gorm:"index:idx_node_member,unique;not null;comment:节点ID"`
	UserID     int `gorm:"index:idx_node_member,unique;not null;comment:用户ID"`
}

// TreeNodeResource 节点资源关联表
type TreeNodeResource struct {
	ID           int    `gorm:"primaryKey;autoIncrement"`
	TreeNodeID   int    `gorm:"index:idx_node_resource;not null;comment:节点ID"`
	ResourceID   string `gorm:"index:idx_node_resource;not null;comment:资源ID"`
	ResourceType string `gorm:"type:varchar(50);not null;comment:资源类型"`
}

// ==================== 请求结构体 ====================

// GetTreeListReq 获取树节点列表请求
type GetTreeListReq struct {
	Level  int    `json:"level" form:"level" binding:"omitempty,min=1"`
	Status string `json:"status" form:"status" binding:"omitempty,oneof=active inactive deleted"`
}

// GetNodeDetailReq 获取节点详情请求
type GetNodeDetailReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetChildNodesReq 获取子节点列表请求
type GetChildNodesReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// CreateNodeReq 创建节点请求
type CreateNodeReq struct {
	Name        string `json:"name" form:"name" binding:"required,min=1,max=50"`
	ParentID    int    `json:"parentId" form:"parentId"` // 父节点ID，0表示根节点
	CreatorID   int    `json:"creatorId" form:"creatorId"`
	Description string `json:"description" form:"description"`
	IsLeaf      bool   `json:"isLeaf" form:"isLeaf"`
	Status      string `json:"status" form:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateNodeReq 更新节点请求
type UpdateNodeReq struct {
	ID          int    `json:"id" form:"id" binding:"required"`
	Name        string `json:"name" form:"name" binding:"required,min=1,max=50"`
	ParentID    int    `json:"parentId" form:"parentId"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// UpdateNodeStatusReq 更新节点状态请求
type UpdateNodeStatusReq struct {
	ID     int    `json:"id" binding:"required"`
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// DeleteNodeReq 删除节点请求
type DeleteNodeReq struct {
	ID int `json:"id" binding:"required"`
}

// MoveNodeReq 移动节点请求
type MoveNodeReq struct {
	ID          int `json:"id" form:"id" binding:"required"`
	NewParentID int `json:"newParentId" form:"newParentId"`
}

// GetNodeMembersReq 获取节点成员请求
type GetNodeMembersReq struct {
	ID   int    `json:"id" binding:"required"`
	Type string `json:"type" binding:"omitempty,oneof=admin member"`
}

// AddNodeMemberReq 添加节点成员请求
type AddNodeMemberReq struct {
	NodeID     int    `json:"nodeId" form:"nodeId" binding:"required"`
	UserID     int    `json:"userId" form:"userId" binding:"required"`
	MemberType string `json:"memberType" form:"memberType" binding:"required,oneof=admin member"`
}

// RemoveNodeMemberReq 移除节点成员请求
type RemoveNodeMemberReq struct {
	NodeID     int    `json:"nodeId" form:"nodeId" binding:"required"`
	UserID     int    `json:"userId" form:"userId" binding:"required"`
	MemberType string `json:"memberType" form:"memberType" binding:"required,oneof=admin member"`
}

// BatchAddNodeMembersReq 批量添加节点成员请求
type BatchAddNodeMembersReq struct {
	NodeID     int    `json:"nodeId" binding:"required"`
	UserIDs    []int  `json:"userIds" binding:"required,min=1"`
	MemberType string `json:"memberType" binding:"required,oneof=admin member"`
}

// BatchRemoveNodeMembersReq 批量移除节点成员请求
type BatchRemoveNodeMembersReq struct {
	NodeID     int    `json:"nodeId" binding:"required"`
	UserIDs    []int  `json:"userIds" binding:"required,min=1"`
	MemberType string `json:"memberType" binding:"required,oneof=admin member"`
}

// GetNodeResourcesReq 获取节点资源请求
type GetNodeResourcesReq struct {
	ID int `json:"id" binding:"required"`
}

// BindResourceReq 绑定资源请求
type BindResourceReq struct {
	NodeID       int      `json:"nodeId" binding:"required"`
	ResourceType string   `json:"resourceType" binding:"required"`
	ResourceIDs  []string `json:"resourceIds" binding:"required,min=1"`
}

// UnbindResourceReq 解绑资源请求
type UnbindResourceReq struct {
	NodeID       int    `json:"nodeId" binding:"required"`
	ResourceID   string `json:"resourceId" binding:"required"`
	ResourceType string `json:"resourceType" binding:"required"`
}

// CheckNodePermissionReq 检查节点权限请求
type CheckNodePermissionReq struct {
	UserID    int    `json:"userId" binding:"required"`
	NodeID    int    `json:"nodeId" binding:"required"`
	Operation string `json:"operation" binding:"required"`
}

// GetUserNodesReq 获取用户相关节点请求
type GetUserNodesReq struct {
	UserID int    `json:"userId" binding:"required"`
	Role   string `json:"role" binding:"omitempty,oneof=admin member"`
}

// BatchCreateNodesReq 批量创建节点请求
type BatchCreateNodesReq struct {
	CreatorID int             `json:"creatorId"`
	Nodes     []CreateNodeReq `json:"nodes" binding:"required,min=1"`
}

// BatchDeleteNodesReq 批量删除节点请求
type BatchDeleteNodesReq struct {
	NodeIDs []int `json:"nodeIds" binding:"required,min=1"`
}

// BatchUpdateNodeStatusReq 批量更新节点状态请求
type BatchUpdateNodeStatusReq struct {
	NodeIDs []int  `json:"nodeIds" binding:"required,min=1"`
	Status  string `json:"status" binding:"required,oneof=active inactive"`
}

// ==================== 响应结构体 ====================

// TreeNodeResp 服务树节点响应
type TreeNodeResp struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	ParentID    int       `json:"parentId"`
	Level       int       `json:"level"`
	Description string    `json:"description"`
	CreatorID   int       `json:"creatorId"`
	Status      string    `json:"status"`
	CreatorName string    `json:"creatorName"`
	ParentName  string    `json:"parentName"`
	ChildCount  int       `json:"childCount"`
	IsLeaf      bool      `json:"isLeaf"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TreeNodeDetailResp 服务树节点详情响应
type TreeNodeDetailResp struct {
	TreeNodeResp
	AdminUsers    StringList `json:"adminUsers"`
	MemberUsers   StringList `json:"memberUsers"`
	ResourceCount int        `json:"resourceCount"`
}

// TreeNodeListResp 服务树节点列表响应
type TreeNodeListResp struct {
	ID        int                 `json:"id"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Name      string              `json:"name"`
	ParentID  int                 `json:"parentId"`
	Level     int                 `json:"level"`
	CreatorID int                 `json:"creatorId"`
	Status    string              `json:"status"`
	Children  []*TreeNodeListResp `json:"children"`
	IsLeaf    bool                `json:"isLeaf"`
}

// TreeStatisticsResp 服务树统计响应
type TreeStatisticsResp struct {
	TotalNodes     int `json:"totalNodes"`     // 节点总数
	TotalResources int `json:"totalResources"` // 资源总数
	TotalAdmins    int `json:"totalAdmins"`    // 管理员总数
	TotalMembers   int `json:"totalMembers"`   // 成员总数
	ActiveNodes    int `json:"activeNodes"`    // 活跃节点数
	InactiveNodes  int `json:"inactiveNodes"`  // 非活跃节点数
}

// TreeNodeResourceResp 节点资源响应
type TreeNodeResourceResp struct {
	ID                 int    `json:"id"`                 // 关联ID
	ResourceID         string `json:"resourceId"`         // 资源ID
	ResourceType       string `json:"resourceType"`       // 资源类型
	ResourceName       string `json:"resourceName"`       // 资源名称
	ResourceStatus     string `json:"resourceStatus"`     // 资源状态
	ResourceCreateTime string `json:"resourceCreateTime"` // 资源创建时间
	ResourceUpdateTime string `json:"resourceUpdateTime"` // 资源更新时间
	ResourceDeleteTime string `json:"resourceDeleteTime"` // 资源删除时间
}
