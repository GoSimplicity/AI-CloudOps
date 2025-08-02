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
	FormDesignStatusDraft     int8 = 1 // 草稿
	FormDesignStatusPublished int8 = 2 // 已发布
	FormDesignStatusArchived  int8 = 3 // 已归档
)

const (
	FormFieldTypeText     = "text"     // 文本输入
	FormFieldTypeNumber   = "number"   // 数字输入
	FormFieldTypePassword = "password" // 密码输入
	FormFieldTypeTextarea = "textarea" // 多行文本
	FormFieldTypeSelect   = "select"   // 下拉选择
	FormFieldTypeRadio    = "radio"    // 单选框
	FormFieldTypeCheckbox = "checkbox" // 复选框
	FormFieldTypeDate     = "date"     // 日期选择
	FormFieldTypeSwitch   = "switch"   // 开关
)

// WorkorderFormDesign 工单表单设计实体
type WorkorderFormDesign struct {
	Model
	Name         string             `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:表单名称"`
	Description  string             `json:"description" gorm:"column:description;type:varchar(1000);comment:表单描述"`
	Schema       JSONMap            `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Status       int8               `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-草稿，2-已发布，3-已归档"`
	CategoryID   *int               `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	OperatorID   int                `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	OperatorName string             `json:"operator_name" gorm:"column:operator_name;type:varchar(100);not null;index;comment:操作人名称"`
	Tags         StringList         `json:"tags" gorm:"column:tags;comment:标签"`
	IsTemplate   int8               `json:"is_template" gorm:"column:is_template;not null;default:1;comment:是否为模板：1-是，2-否"`
	Category     *WorkorderCategory `json:"category" gorm:"foreignKey:CategoryID;references:ID"`
}

// TableName 指定工单表单设计表名
func (WorkorderFormDesign) TableName() string {
	return "cl_workorder_form_design"
}

// FormField 表单字段定义
type FormField struct {
	ID          string   `json:"id"`                       // 字段唯一标识（系统自动生成）
	Name        string   `json:"name" binding:"required"`  // 字段名称
	Type        string   `json:"type" binding:"required"`  // 字段类型
	Label       string   `json:"label" binding:"required"` // 字段标签
	Required    int8     `json:"required"`                 // 是否必填
	Placeholder string   `json:"placeholder"`              // 占位符
	Default     any      `json:"default"`                  // 默认值
	Options     []string `json:"options,omitempty"`        // 选项（如下拉、单选等）
}

// FormSchema 表单结构定义
type FormSchema struct {
	Fields []FormField `json:"fields" binding:"required"` // 字段列表
}

// CreateWorkorderFormDesignReq 创建工单表单设计请求
type CreateWorkorderFormDesignReq struct {
	Name         string     `json:"name" binding:"required,min=1,max=200"`
	Description  string     `json:"description" binding:"omitempty,max=1000"`
	Schema       FormSchema `json:"schema" binding:"required"`
	Status       int8       `json:"status" binding:"required,oneof=1 2 3"`
	CategoryID   *int       `json:"category_id" binding:"omitempty,min=1"`
	OperatorID   int        `json:"operator_id" binding:"required,min=1"`
	OperatorName string     `json:"operator_name" binding:"required,min=1,max=100"`
	Tags         StringList `json:"tags" binding:"omitempty"`
	IsTemplate   int8       `json:"is_template" binding:"required,oneof=1 2"`
}

// UpdateWorkorderFormDesignReq 更新工单表单设计请求
type UpdateWorkorderFormDesignReq struct {
	ID          int        `json:"id" binding:"required,min=1"`
	Name        string     `json:"name" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"omitempty,max=1000"`
	Schema      FormSchema `json:"schema" binding:"required"`
	Status      int8       `json:"status" binding:"required,oneof=1 2 3"`
	CategoryID  *int       `json:"category_id" binding:"omitempty,min=1"`
	Tags        StringList `json:"tags" binding:"omitempty"`
	IsTemplate  int8       `json:"is_template" binding:"required,oneof=1 2"`
}

// DeleteWorkorderFormDesignReq 删除工单表单设计请求
type DeleteWorkorderFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderFormDesignReq 获取工单表单设计详情请求
type DetailWorkorderFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderFormDesignReq 获取工单表单设计列表请求
type ListWorkorderFormDesignReq struct {
	ListReq
	CategoryID *int  `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	Status     *int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2 3"`
	IsTemplate *int8 `json:"is_template" form:"is_template" binding:"omitempty,oneof=1 2"`
}
