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
	"fmt"
	"time"
)

// TreeNode 服务树节点结构
type TreeNode struct {
	Model
	Name        string `json:"name" gorm:"type:varchar(50);not null;comment:节点名称"`         // 节点名称
	ParentID    int    `json:"parentId" gorm:"index;comment:父节点ID"`                        // 父节点ID
	Path        string `json:"path" gorm:"type:varchar(255);comment:节点路径"`                 // 节点完整路径
	Level       int    `json:"level" gorm:"comment:节点层级"`                                  // 节点层级
	Description string `json:"description" gorm:"type:text;comment:节点描述"`                  // 节点描述
	ServiceCode string `json:"serviceCode" gorm:"type:varchar(50);comment:服务代码"`           // 服务代码，唯一标识服务
	Creator     string `json:"creator" gorm:"type:varchar(50);comment:创建者"`                // 创建者
	Status      string `json:"status" gorm:"type:varchar(20);default:active;comment:节点状态"` // 节点状态：active, inactive, deleted

	// 关联关系
	Admins  []*User `json:"admins" gorm:"many2many:tree_node_admins;comment:管理员列表"`  // 管理员列表
	Members []*User `json:"members" gorm:"many2many:tree_node_members;comment:成员列表"` // 普通成员列表

	// 资源统计
	ChildCount    int `json:"childCount" gorm:"-"`    // 子节点数量
	ResourceCount int `json:"resourceCount" gorm:"-"` // 关联资源数量

	// 前端展示相关，不存储在数据库
	Key         string      `json:"key" gorm:"-"`         // 节点唯一标识
	Title       string      `json:"title" gorm:"-"`       // 节点显示名称
	ParentName  string      `json:"parentName" gorm:"-"`  // 父节点名称
	AdminUsers  StringList  `json:"adminUsers" gorm:"-"`  // 管理员用户名列表
	MemberUsers StringList  `json:"memberUsers" gorm:"-"` // 成员用户名列表
	Children    []*TreeNode `json:"children" gorm:"-"`    // 子节点列表
	IsLeaf      bool        `json:"isLeaf" gorm:"-"`      // 是否为叶子节点
}

// TableName 设置表名
func (TreeNode) TableName() string {
	return "tree_nodes"
}

// BeforeSave 保存前的钩子
func (t *TreeNode) BeforeSave() error {
	// 基本验证
	if t.Name == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	return nil
}

// AfterFind 查询后的钩子
func (t *TreeNode) AfterFind() error {
	// 设置前端展示字段
	t.Key = fmt.Sprintf("node-%d", t.ID)
	t.Title = t.Name
	// 如果子节点数量为-1，则表示该节点为叶子节点
	t.IsLeaf = t.ChildCount == -1

	// 如果管理员列表不为空，设置管理员用户名列表
	if len(t.Admins) > 0 {
		t.AdminUsers = make(StringList, 0, len(t.Admins))
		for _, admin := range t.Admins {
			t.AdminUsers = append(t.AdminUsers, admin.Username)
		}
	}

	// 如果成员列表不为空，设置成员用户名列表
	if len(t.Members) > 0 {
		t.MemberUsers = make(StringList, 0, len(t.Members))
		for _, member := range t.Members {
			t.MemberUsers = append(t.MemberUsers, member.Username)
		}
	}

	return nil
}

// TreeNodeCreateReq 创建节点请求
type TreeNodeCreateReq struct {
	Name        string `json:"name" binding:"required"`
	ParentID    int    `json:"parentId"`
	Description string `json:"description"`
	ServiceCode string `json:"serviceCode"`
	AdminIDs    []int  `json:"adminIds"`
	MemberIDs   []int  `json:"memberIds"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// TreeNodeUpdateReq 更新节点请求
type TreeNodeUpdateReq struct {
	ID          int    `json:"id" binding:"required"`
	Name        string `json:"name" binding:"omitempty,min=1,max=50"`
	Description string `json:"description"`
	ServiceCode string `json:"serviceCode"`
	AdminIDs    []int  `json:"adminIds"`
	MemberIDs   []int  `json:"memberIds"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive deleted"`
}

// TreeNodeMemberReq 节点成员请求
type TreeNodeMemberReq struct {
	NodeID int    `json:"nodeId" binding:"required"`
	UserID int    `json:"userId" binding:"required"`
	Type   string `json:"type" binding:"required,oneof=admin member"` // admin 或 member
}

// TreeNodeResourceBindReq 资源绑定请求
type TreeNodeResourceBindReq struct {
	NodeID       int      `json:"nodeId" binding:"required"`
	ResourceType string   `json:"resourceType" binding:"required"`
	ResourceIDs  []string `json:"resourceIds" binding:"required"`
}

// TreeNodeResourceUnbindReq 资源解绑请求
type TreeNodeResourceUnbindReq struct {
	NodeID       int    `json:"nodeId" binding:"required"`
	ResourceID   string `json:"resourceId" binding:"required"`
	ResourceType string `json:"resourceType" binding:"required"`
}

// TreeNodeResp 服务树节点响应
type TreeNodeResp struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	ParentID    int       `json:"parentId"`
	Path        string    `json:"path"`
	Level       int       `json:"level"`
	Description string    `json:"description"`
	ServiceCode string    `json:"serviceCode"`
	Creator     string    `json:"creator"`
	Status      string    `json:"status"`
	Key         string    `json:"key"`
	Title       string    `json:"title"`
	ParentName  string    `json:"parentName"`
	ChildCount  int       `json:"childCount"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

// TreeNodeDetailReq 服务树节点详情请求
type TreeNodeDetailReq struct {
}

// TreeNodeDetailResp 服务树节点详情响应
type TreeNodeDetailResp struct {
	TreeNodeResp
	AdminUsers    StringList `json:"adminUsers"`
	MemberUsers   StringList `json:"memberUsers"`
	ResourceCount int        `json:"resourceCount"`
}

// TreeNodePathReq 节点路径请求
type TreeNodePathReq struct {
}

// TreeNodePathResp 节点路径响应
type TreeNodePathResp struct {
	Path  []*TreeNodeResp `json:"path"`
	Total int             `json:"total"`
}

// TreeStatisticsResp 服务树统计响应
type TreeStatisticsResp struct {
	TotalNodes     int `json:"totalNodes"`     // 节点总数
	TotalResources int `json:"totalResources"` // 资源总数
	TotalMembers   int `json:"totalMembers"`   // 成员总数
	ActiveNodes    int `json:"activeNodes"`    // 活跃节点数
	InactiveNodes  int `json:"inactiveNodes"`  // 非活跃节点数
}

type TreeNodeResourceResp struct {
	ID                 int    `json:"id"`                 // 资源ID
	ResourceID         string `json:"resourceId"`         // 资源ID
	ResourceType       string `json:"resourceType"`       // 资源类型
	ResourceName       string `json:"resourceName"`       // 资源名称
	ResourceStatus     string `json:"resourceStatus"`     // 资源状态
	ResourceCreateTime string `json:"resourceCreateTime"` // 资源创建时间
	ResourceUpdateTime string `json:"resourceUpdateTime"` // 资源更新时间
	ResourceDeleteTime string `json:"resourceDeleteTime"` // 资源删除时间
}

// TreeNodeListReq 服务树节点列表请求
type TreeNodeListReq struct {
	ParentID int `json:"parentId"`
	Level    int `json:"level"`
}

// TreeNodeDeleteReq 服务树节点删除请求
type TreeNodeDeleteReq struct {
}

// TreeNodeResourceReq 服务树节点资源请求
type TreeNodeResourceReq struct {
}
