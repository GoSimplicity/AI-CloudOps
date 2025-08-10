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

type AuthMode int8

const (
	AuthModePassword AuthMode = iota + 1
	AuthModeKey
)

type ResourceStatus int8

const (
	RUNNING ResourceStatus = iota + 1
	STOPPED
	STARTING
	STOPPING
	RESTARTING
	DELETING
	ERROR
)

type TreeLocalResource struct {
	Model
	Name           string         `json:"name" gorm:"type:varchar(100);comment:资源名称"`
	Status         ResourceStatus `json:"status" gorm:"type:tinyint(1);comment:资源状态;default:1"`
	Environment    string         `json:"environment" gorm:"type:varchar(50);comment:环境标识,如dev,prod"`
	Description    string         `json:"description" gorm:"type:text;comment:资源描述"`
	Tags           StringList     `json:"tags" gorm:"type:varchar(500);comment:资源标签集合"`
	Cpu            int            `json:"cpu" gorm:"comment:CPU核数"`
	Memory         int            `json:"memory" gorm:"comment:内存大小,单位GiB"`
	Disk           int            `json:"disk" gorm:"comment:系统盘大小,单位GiB"`
	IpAddr         string         `json:"ip_addr" gorm:"type:varchar(45);comment:主IP地址"`
	Port           int            `json:"port" gorm:"comment:端口号;default:22"`
	Username       string         `json:"username" gorm:"type:varchar(100);comment:用户名;default:root"`
	Password       string         `json:"-" gorm:"type:varchar(500);comment:密码,加密存储"`
	CreateUserID   int            `json:"create_user_id" gorm:"comment:创建者ID;default:0"`
	CreateUserName string         `json:"create_user_name" gorm:"type:varchar(100);comment:创建者姓名"`
	Key            string         `json:"key" gorm:"type:text;comment:密钥"`
	AuthMode       AuthMode       `json:"auth_mode" gorm:"type:tinyint(1);comment:认证方式,1:密码,2:密钥;default:1"`
	OsType         string         `json:"os_type" gorm:"type:varchar(50);comment:操作系统类型,如win,linux"`
	OSName         string         `json:"os_name" gorm:"type:varchar(100);comment:操作系统名称"`
	ImageName      string         `json:"image_name" gorm:"type:varchar(100);comment:镜像名称"`
	TreeNodes      []*TreeNode    `json:"tree_nodes" gorm:"many2many:cl_tree_node_local"`
}

func (t *TreeLocalResource) TableName() string {
	return "cl_tree_local_resource"
}

// GetTreeLocalListReq 获取本地树资源列表请求
type GetTreeLocalResourceListReq struct {
	ListReq
	Status ResourceStatus `json:"status" form:"status"`
}

// GetTreeLocalDetailReq 获取本地树资源详情请求
type GetTreeLocalResourceDetailReq struct {
	ID int `json:"id" form:"id"`
}

// CreateTreeLocalReq 创建本地树资源请求
type CreateTreeLocalResourceReq struct {
	Name           string     `json:"name" binding:"required"`
	Environment    string     `json:"environment"`
	Description    string     `json:"description"`
	Tags           StringList `json:"tags"`
	IpAddr         string     `json:"ip_addr" binding:"required"`
	Port           int        `json:"port"`
	Username       string     `json:"username"`
	Password       string     `json:"password"`
	CreateUserID   int        `json:"create_user_id"`
	CreateUserName string     `json:"create_user_name"`
	OsType         string     `json:"os_type"`
	OSName         string     `json:"os_name"`
	ImageName      string     `json:"image_name"`
	Key            string     `json:"key"`
	AuthMode       AuthMode   `json:"auth_mode"`
}

// UpdateTreeLocalReq 更新本地树资源请求
type UpdateTreeLocalResourceReq struct {
	ID          int        `json:"id" form:"id"`
	Name        string     `json:"name"`
	Environment string     `json:"environment"`
	Description string     `json:"description"`
	Tags        StringList `json:"tags"`
	IpAddr      string     `json:"ip_addr"`
	Port        int        `json:"port"`
	OsType      string     `json:"os_type"`
	OSName      string     `json:"os_name"`
	ImageName   string     `json:"image_name"`
	Username    string     `json:"username"`
	Password    string     `json:"password"`
	Key         string     `json:"key"`
	AuthMode    AuthMode   `json:"auth_mode"`
}

// DeleteTreeLocalReq 删除本地树资源请求
type DeleteTreeLocalResourceReq struct {
	ID int `json:"id" form:"id"`
}

// ConnectTerminalReq 连接终端请求
type ConnectTerminalResourceReq struct {
	ID     int `json:"id" form:"id"`
	UserID int `json:"user_id"`
}

type BindTreeLocalResourceReq struct {
	ID          int   `json:"id" form:"id"`
	TreeNodeIDs []int `json:"tree_node_ids" form:"tree_node_ids"`
}

type UnBindTreeLocalResourceReq struct {
	ID          int   `json:"id" form:"id"`
	TreeNodeIDs []int `json:"tree_node_ids" form:"tree_node_ids"`
}
