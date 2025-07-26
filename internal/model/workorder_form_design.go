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

import "gorm.io/datatypes"

// FormField 表单字段定义
type FormField struct {
	ID           string                 `json:"id"`                       // 字段唯一标识
	Type         string                 `json:"type" binding:"required"`  // 字段类型
	Label        string                 `json:"label" binding:"required"` // 字段标签
	Name         string                 `json:"name" binding:"required"`  // 字段名称
	Required     bool                   `json:"required"`                 // 是否必填
	Placeholder  string                 `json:"placeholder"`              // 占位符
	DefaultValue interface{}            `json:"default_value"`            // 默认值
	Options      []FormFieldOption      `json:"options,omitempty"`        // 选项列表
	Validation   FormFieldValidation    `json:"validation,omitempty"`     // 验证规则
	Props        map[string]interface{} `json:"props,omitempty"`          // 其他属性
	SortOrder    int                    `json:"sort_order"`               // 排序
	Disabled     bool                   `json:"disabled"`                 // 是否禁用
	Hidden       bool                   `json:"hidden"`                   // 是否隐藏
	Description  string                 `json:"description,omitempty"`    // 字段描述
}

// FormFieldOption 表单字段选项
type FormFieldOption struct {
	Label string      `json:"label"` // 选项标签
	Value interface{} `json:"value"` // 选项值
}

// FormFieldValidation 表单字段验证规则
type FormFieldValidation struct {
	MinLength *int   `json:"min_length"` // 最小长度
	MaxLength *int   `json:"max_length"` // 最大长度
	Min       *int   `json:"min"`        // 最小值
	Max       *int   `json:"max"`        // 最大值
	Pattern   string `json:"pattern"`    // 正则表达式
	Message   string `json:"message"`    // 验证错误信息
}

// FormSchema 表单结构定义
type FormSchema struct {
	Fields []FormField `json:"fields" binding:"required"` // 字段列表
	Layout string      `json:"layout"`                    // 布局类型
	Style  string      `json:"style"`                     // 样式配置
}

// FormDesign 表单设计实体
type FormDesign struct {
	Model
	Name        string         `json:"name" gorm:"column:name;not null;comment:表单名称"`
	Description string         `json:"description" gorm:"column:description;comment:表单描述"`
	Schema      datatypes.JSON `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Version     int            `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status      int8           `json:"status" gorm:"column:status;not null;default:0;comment:状态：1-草稿，2-已发布，3-已禁用"`
	CategoryID  *int           `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID   int            `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string         `json:"creator_name" gorm:"-"`
	Category    *Category      `json:"category" gorm:"foreignKey:CategoryID"`

	CategoryName string `json:"category_name" gorm:"-"`
}

func (FormDesign) TableName() string {
	return "cl_	workorder_form_design"
}

// CreateFormDesignReq 创建表单设计请求
type CreateFormDesignReq struct {
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
	UserID      int        `json:"user_id"`
	UserName    string     `json:"user_name"`
	Status      int8       `json:"status" binding:"omitempty,oneof=1 2 3"`
	Version     int        `json:"version" binding:"omitempty,min=1"`
}

// UpdateFormDesignReq 更新表单设计请求
type UpdateFormDesignReq struct {
	ID          int        `json:"id" binding:"required"`
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
	Status      int8       `json:"status" binding:"omitempty,oneof=1 2 3"`
	Version     int        `json:"version" binding:"omitempty,min=1"`
}

// DeleteFormDesignReq 删除表单设计请求
type DeleteFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// DetailFormDesignReq 获取表单设计详情请求
type DetailFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// ListFormDesignReq 表单设计列表请求
type ListFormDesignReq struct {
	ListReq
	CategoryID *int  `json:"category_id" form:"category_id"` // 按分类筛选
	Status     *int8 `json:"status" form:"status"`           // 按状态筛选
}

// PublishFormDesignReq 发布表单设计请求
type PublishFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// CloneFormDesignReq 克隆表单设计请求
type CloneFormDesignReq struct {
	ID   int    `json:"id" form:"id" binding:"required"`
	Name string `json:"name" form:"name" binding:"required,min=1,max=100"`
}

// PreviewFormDesignReq 预览表单设计请求
type PreviewFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// FormStatistics 表单设计统计信息
type FormStatistics struct {
	Draft     int64 `json:"draft"`     // 草稿表单数
	Published int64 `json:"published"` // 已发布表单数
	Disabled  int64 `json:"disabled"`  // 已禁用表单数
}
