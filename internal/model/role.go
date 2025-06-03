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

type Role struct {
	Model
	Name        string  `json:"name" gorm:"type:varchar(50);not null;comment:角色名称"`                             // 角色名称，唯一且非空
	Code        string  `json:"code" gorm:"type:varchar(50);not null;comment:角色编码"`                             // 角色编码，唯一且非空
	Description string  `json:"description" gorm:"type:varchar(500);comment:角色描述"`                              // 角色描述
	Status      int8    `json:"status" gorm:"type:tinyint(1);default:1;comment:状态 0禁用 1启用" binding:"oneof=0 1"` // 角色状态
	IsSystem    int8    `json:"is_system" gorm:"type:tinyint(1);default:0;comment:是否系统角色 0否 1是"`                // 是否系统角色，系统角色不可删除
	Apis        []*Api  `json:"apis" gorm:"many2many:role_apis;comment:关联API"`                                  // 多对多关联API
	Users       []*User `json:"users" gorm:"many2many:user_roles;comment:关联用户"`                                 // 多对多关联用户
}

// RoleApi 角色API权限关联表
type RoleApi struct {
	ID     int `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	RoleID int `json:"role_id" gorm:"not null;index;comment:角色ID"`
	ApiID  int `json:"api_id" gorm:"not null;index;comment:API ID"`
}

// UserRole 用户角色关联表
type UserRole struct {
	ID     int `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`
	UserID int `json:"user_id" gorm:"not null;index;comment:用户ID"`
	RoleID int `json:"role_id" gorm:"not null;index;comment:角色ID"`
}

// CreateRoleRequest 创建角色请求结构体
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required,max=50"` // 角色名称
	Code        string `json:"code" binding:"required,max=50"` // 角色编码
	Description string `json:"description" binding:"max=500"`  // 角色描述
	Status      int    `json:"status" binding:"oneof=0 1"`     // 状态
	ApiIds      []int  `json:"api_ids"`                        // 关联的API ID列表
}

// UpdateRoleRequest 更新角色请求结构体
type UpdateRoleRequest struct {
	ID          int    `json:"id" binding:"required,gt=0"`     // 角色ID
	Name        string `json:"name" binding:"required,max=50"` // 角色名称
	Code        string `json:"code" binding:"required,max=50"` // 角色编码
	Description string `json:"description" binding:"max=500"`  // 角色描述
	Status      int    `json:"status" binding:"oneof=0 1"`     // 状态
	ApiIds      []int  `json:"api_ids"`                        // 关联的API ID列表
}

// GetRoleRequest 获取角色请求结构体
type GetRoleRequest struct {
	ID int `json:"id"` // 角色ID
}

type GetRoleApiRequest struct {
	ID int `json:"id"` // 角色ID
}

type AssignRolesToUserRequest struct {
	UserID  int   `json:"user_id" binding:"required,gt=0"`       // 用户ID
	RoleIds []int `json:"role_ids" binding:"required,dive,gt=0"` // 角色ID列表
}

type RevokeRolesFromUserRequest struct {
	UserID  int   `json:"user_id" binding:"required,gt=0"`       // 用户ID
	RoleIds []int `json:"role_ids" binding:"required,dive,gt=0"` // 角色ID列表
}

type GetRoleUsersRequest struct {
	ID int `json:"id"` // 角色ID
}

type GetUserRolesRequest struct {
	ID int `json:"id"` // 用户ID
}

type GetUserPermissionsRequest struct {
	ID int `json:"id"` // 用户ID
}

type CheckUserPermissionRequest struct {
	UserID int    `json:"user_id" binding:"required,gt=0"`
	Method string `json:"method" binding:"required"`
	Path   string `json:"path" binding:"required"`
}

type ListRolesRequest struct {
	ListReq
	Status *int `json:"status" binding:"omitempty,oneof=0 1"` // 状态筛选，可选
}

// AssignRoleRequest 分配角色请求结构体
type AssignRoleRequest struct {
	UserID  int   `json:"user_id" binding:"required,gt=0"`       // 用户ID
	RoleIds []int `json:"role_ids" binding:"required,dive,gt=0"` // 角色ID列表
}

// RevokeRoleRequest 撤销角色请求结构体
type RevokeRoleRequest struct {
	UserID  int   `json:"user_id" binding:"required,gt=0"`       // 用户ID
	RoleIds []int `json:"role_ids" binding:"required,dive,gt=0"` // 角色ID列表
}

type AssignRoleApiRequest struct {
	RoleID int   `json:"role_id" binding:"required,gt=0"`
	ApiIds []int `json:"api_ids" binding:"required,dive,gt=0"`
}

type RevokeRoleApiRequest struct {
	RoleID int   `json:"role_id" binding:"required,gt=0"`
	ApiIds []int `json:"api_ids" binding:"required,dive,gt=0"`
}

// DeleteRoleRequest 删除角色请求结构体
type DeleteRoleRequest struct {
	ID int `json:"id" ` // 角色ID
}
