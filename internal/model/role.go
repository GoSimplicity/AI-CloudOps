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
	ID        int     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键ID"`                                    // 主键ID，自增
	CreatedAt int64   `json:"created_at" gorm:"autoCreateTime;comment:创建时间"`                                      // 创建时间，自动记录
	UpdatedAt int64   `json:"updated_at" gorm:"autoUpdateTime;comment:更新时间"`                                      // 更新时间，自动记录
	DeletedAt int64   `json:"deleted_at" gorm:"index;default:0;comment:删除时间"`                                     // 软删除时间，使用普通索引
	Name      string  `json:"name" gorm:"type:varchar(50);uniqueIndex:idx_name_del;not null;comment:角色名称"`         // 角色名称，唯一且非空
	Desc      string  `json:"desc" gorm:"type:varchar(255);comment:角色描述"`                                         // 角色描述
	RoleType  int8    `json:"role_type" gorm:"type:tinyint(1);not null;comment:角色类型 1系统角色 2自定义角色" binding:"oneof=1 2"` // 角色类型，使用int8节省空间
	IsDefault int8    `json:"is_default" gorm:"type:tinyint(1);default:0;comment:是否为默认角色 0否 1是" binding:"oneof=0 1"`  // 是否默认角色，使用int8节省空间
	Users     []*User `json:"users" gorm:"many2many:user_roles;comment:关联用户"`                                     // 多对多关联用户
	Menus     []*Menu `json:"menus" gorm:"many2many:role_menus;comment:关联菜单"`                                    // 多对多关联菜单
	Apis      []*Api  `json:"apis" gorm:"many2many:role_apis;comment:关联接口"`                                      // 多对多关联接口
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`        // 角色名称
	Description string `json:"description"`                    // 角色描述
	RoleType    int    `json:"role_type" binding:"required"`   // 角色类型
	IsDefault   int    `json:"is_default" binding:"oneof=0 1"` // 是否默认角色
	ApiIds      []int  `json:"api_ids"`                        // API ID列表
}

type GetRoleRequest struct {
	Id int `json:"id" binding:"required,gt=0"` // 角色ID
}

type UpdateRoleRequest struct {
	Id          int    `json:"id" binding:"required,gt=0"`     // 角色ID
	Name        string `json:"name" binding:"required"`        // 角色名称
	Description string `json:"description"`                    // 角色描述
	RoleType    int    `json:"role_type" binding:"required"`   // 角色类型
	IsDefault   int    `json:"is_default" binding:"oneof=0 1"` // 是否默认角色
	ApiIds      []int  `json:"api_ids"`                        // API ID列表
}

type ListRolesRequest struct {
	PageNumber int `json:"page_number" binding:"required,gt=0"` // 页码
	PageSize   int `json:"page_size" binding:"required,gt=0"`   // 每页数量
}

type ListUserRolesRequest struct {
	PageNumber int `json:"page_number" binding:"required,gt=0"` // 页码
	PageSize   int `json:"page_size" binding:"required,gt=0"`   // 每页数量
}

type UpdateUserRoleRequest struct {
	UserId  int   `json:"user_id" binding:"required,gt=0"` // 用户ID
	ApiIds  []int `json:"api_ids"`                         // API ID列表
	RoleIds []int `json:"role_ids"`                        // 角色ID列表
}

type AssignUserRoleRequest struct {
	UserId  int   `json:"user_id" binding:"required,gt=0"` // 用户ID
	RoleIds []int `json:"role_ids"`                        // 角色ID列表
	ApiIds  []int `json:"api_ids"`                         // API ID列表
}

type AssignUsersRoleRequest struct {
	UserIds []int `json:"user_ids" binding:"required,gt=0"` // 用户ID
	RoleIds []int `json:"role_ids"`                         // 角色ID列表
}
