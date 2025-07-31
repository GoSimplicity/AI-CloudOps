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

const (
	CategoryStatusEnabled  int8 = 1 // 启用
	CategoryStatusDisabled int8 = 2 // 禁用
)

// WorkorderCategory 工单分类实体
type WorkorderCategory struct {
	Model
	Name           string `json:"name" gorm:"column:name;type:varchar(100);not null;index;comment:分类名称"`
	Status         int8   `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-启用，2-禁用"`
	Description    string `json:"description" gorm:"column:description;type:varchar(500);comment:分类描述"`
	CreateUserID   int    `json:"create_user_id" gorm:"column:create_user_id;not null;index;comment:创建人ID"`
	CreateUserName string `json:"create_user_name" gorm:"column:create_user_name;type:varchar(100);not null;index;comment:创建人名称"`
}

// TableName 指定工单分类表名
func (WorkorderCategory) TableName() string {
	return "cl_workorder_category"
}

// CreateWorkorderCategoryReq 创建工单分类请求
type CreateWorkorderCategoryReq struct {
	Name           string `json:"name" binding:"required,min=1,max=100"`
	Status         int8   `json:"status" binding:"required,oneof=1 2"`
	Description    string `json:"description" binding:"omitempty,max=500"`
	CreateUserID   int    `json:"create_user_id" binding:"required,min=1"`
	CreateUserName string `json:"create_user_name" binding:"required,min=1,max=100"`
}

// UpdateWorkorderCategoryReq 更新工单分类请求
type UpdateWorkorderCategoryReq struct {
	ID          int    `json:"id" binding:"required,min=1"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Status      int8   `json:"status" binding:"required,oneof=1 2"`
}

// DeleteWorkorderCategoryReq 删除工单分类请求
type DeleteWorkorderCategoryReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderCategoryReq 获取工单分类详情请求
type DetailWorkorderCategoryReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderCategoryReq 工单分类列表请求
type ListWorkorderCategoryReq struct {
	ListReq
	Status *int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}
