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

// Category 分类实体
type Category struct {
	Model
	Name        string `json:"name" gorm:"column:name;not null;comment:分类名称"`
	ParentID    *int   `json:"parent_id" gorm:"column:parent_id;comment:父分类ID"`
	Icon        string `json:"icon" gorm:"column:icon;comment:图标"`
	SortOrder   int    `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	Status      int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态：1-启用，2-禁用"`
	Description string `json:"description" gorm:"column:description;comment:分类描述"`
	CreatorID   int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name" gorm:"-"`
}

func (Category) TableName() string {
	return "workorder_category"
}

// 分类请求结构
type CreateCategoryReq struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	ParentID    *int   `json:"parent_id"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description" binding:"omitempty,max=500"`
	UserID      int    `json:"user_id" binding:"required"`
	UserName    string `json:"user_name" binding:"required"`
	Status      int8   `json:"status" binding:"required,oneof=1 2"` // 状态，必填，0-禁用，1-启用
}

// UpdateCategoryReq 更新分类请求结构
type UpdateCategoryReq struct {
	ID          int    `json:"id" form:"id" binding:"required"`         // 分类ID，必填
	Name        string `json:"name" binding:"required,min=1,max=100"`   // 分类名称，必填，长度1-100
	ParentID    *int   `json:"parent_id"`                               // 父分类ID，可选
	Icon        string `json:"icon"`                                    // 图标，可选
	SortOrder   int    `json:"sort_order"`                              // 排序顺序，可选
	Description string `json:"description" binding:"omitempty,max=500"` // 分类描述，最大500字符
	Status      *int8  `json:"status" binding:"required,oneof=1 2"`     // 状态，必填，0-禁用，1-启用
}

type DeleteCategoryReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListCategoryReq struct {
	ListReq
	Status *int8 `json:"status" form:"status"`
}

type DetailCategoryReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// TreeCategoryReq 获取分类树请求
type TreeCategoryReq struct {
	Status *int8 `json:"status" form:"status"`
}

type CategoryStatistics struct {
	EnabledCount  int64 `json:"enabled_count"`
	DisabledCount int64 `json:"disabled_count"`
}
