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
	Name   string `json:"name" binding:"required"`   // 角色名称
	Domain string `json:"domain" binding:"required"` // 域ID
	Path   string `json:"path" binding:"required"`   // 路径
	Method string `json:"method" binding:"required"` // 方法
}

type CreateRoleRequest struct {
	Role
}

type UpdateRoleRequest struct {
	NewRole Role `json:"new_role" binding:"required"`
	OldRole Role `json:"old_role" binding:"required"`
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

type DeleteRoleRequest struct {
	Role
}

type GenerateRoleResp struct {
	Total int     `json:"total"`
	Items []*Role `json:"items"`
}

// CasbinRule 对应 casbin_rule 表结构
type CasbinRule struct {
	ID    int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Ptype string `json:"ptype" gorm:"type:varchar(100)"`
	V0    string `json:"v0" gorm:"type:varchar(100)"`
	V1    string `json:"v1" gorm:"type:varchar(100)"`
	V2    string `json:"v2" gorm:"type:varchar(100)"`
	V3    string `json:"v3" gorm:"type:varchar(100)"`
	V4    string `json:"v4" gorm:"type:varchar(100)"`
	V5    string `json:"v5" gorm:"type:varchar(100)"`
}

// TableName 指定表名
func (c *CasbinRule) TableName() string {
	return "casbin_rule"
}
