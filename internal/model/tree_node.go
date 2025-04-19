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

// TreeNode 服务树节点结构
type TreeNode struct {
	Model
	Title       string `json:"title" gorm:"type:varchar(50);comment:节点名称"`       // 节点名称
	Pid         int    `json:"pId" gorm:"index;comment:父节点ID"`                   // 父节点ID
	Level       int    `json:"level" gorm:"comment:节点层级"`                        // 节点层级
	IsLeaf      bool   `json:"isLeaf" gorm:"comment:是否为叶子节点"`                    // 是否为叶子节点
	Desc        string `json:"desc" gorm:"type:text;comment:节点描述"`               // 节点描述
	ServiceCode string `json:"serviceCode" gorm:"type:varchar(50);comment:服务代码"` // 服务代码，唯一标识服务

	// 责任团队信息
	OpsAdmins []*User `json:"ops_admins" gorm:"many2many:tree_node_ops_admins;comment:运维负责人列表"` // 运维负责人
	RdAdmins  []*User `json:"rd_admins" gorm:"many2many:tree_node_rd_admins;comment:研发负责人列表"`   // 研发负责人
	RdMembers []*User `json:"rd_members" gorm:"many2many:tree_node_rd_members;comment:研发工程师列表"` // 研发工程师

	// 前端展示相关，不存储在数据库
	Key           string      `json:"key" gorm:"-"`             // 节点唯一标识
	Label         string      `json:"label" gorm:"-"`           // 节点显示名称
	Value         int         `json:"value" gorm:"-"`           // 节点值
	OpsAdminUsers StringList  `json:"ops_admin_users" gorm:"-"` // 运维负责人姓名列表
	RdAdminUsers  StringList  `json:"rd_admin_users" gorm:"-"`  // 研发负责人姓名列表
	RdMemberUsers StringList  `json:"rd_member_users" gorm:"-"` // 研发工程师姓名列表
	Children      []*TreeNode `json:"children" gorm:"-"`        // 子节点列表
}

// CreateNodeReq 创建节点请求
type CreateNodeReq struct {
	Title       string `json:"title" binding:"required"`
	Pid         int    `json:"pId" binding:"required"`
	Desc        string `json:"desc"`
	ServiceCode string `json:"serviceCode"`
	OpsAdminIds []uint `json:"opsAdminIds"`
	RdAdminIds  []uint `json:"rdAdminIds"`
	RdMemberIds []uint `json:"rdMemberIds"`
}

// UpdateNodeReq 更新节点请求
type UpdateNodeReq struct {
	ID          uint   `json:"id" binding:"required"`
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	ServiceCode string `json:"serviceCode"`
	OpsAdminIds []uint `json:"opsAdminIds"`
	RdAdminIds  []uint `json:"rdAdminIds"`
	RdMemberIds []uint `json:"rdMemberIds"`
}

// NodeAdminReq 节点管理员请求
type NodeAdminReq struct {
	NodeId    uint   `json:"nodeId" binding:"required"`
	UserId    uint   `json:"userId" binding:"required"`
	AdminType string `json:"adminType" binding:"required,oneof=ops rd"`
}

// NodeMemberReq 节点成员请求
type NodeMemberReq struct {
	NodeId uint `json:"nodeId" binding:"required"`
	UserId uint `json:"userId" binding:"required"`
}

// TreeNodeResp 服务树节点响应
type TreeNodeResp struct {
	TreeNode
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// TreeNodeDetailResp 服务树节点详情响应
type TreeNodeDetailResp struct {
	TreeNode
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// NodePathResp 节点路径响应
type NodePathResp struct {
	Path []*TreeNodeResp `json:"path"`
}

// TreeNodePathResp 服务树节点路径响应
type TreeNodePathResp struct {
	Path []*TreeNodeResp `json:"path"`
}
