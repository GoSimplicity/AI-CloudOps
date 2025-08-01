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


// 模板状态常量
const (
	TemplateStatusEnabled  int8 = 1 // 启用
	TemplateStatusDisabled int8 = 2 // 禁用
)

// 模板可见性常量
const (
	TemplateVisibilityPrivate = "private" // 私有
	TemplateVisibilityPublic  = "public"  // 公开
	TemplateVisibilityShared  = "shared"  // 共享
)

// WorkorderTemplate 工单模板实体
type WorkorderTemplate struct {
	Model
	Name           string         `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:模板名称"`
	Description    string         `json:"description" gorm:"column:description;type:varchar(1000);comment:模板描述"`
	ProcessID      int            `json:"process_id" gorm:"column:process_id;not null;index;comment:关联的流程ID"`
	FormDesignID   int            `json:"form_design_id" gorm:"column:form_design_id;not null;index;comment:关联的表单设计ID"`
	DefaultValues  JSONMap `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Status         int8           `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-启用，2-禁用"`
	CategoryID     *int           `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	OperatorID   int            `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	OperatorName string         `json:"operator_name" gorm:"column:operator_name;type:varchar(100);not null;comment:操作人名称"`
	Tags           StringList     `json:"tags" gorm:"column:tags;comment:标签"`
}

func (WorkorderTemplate) TableName() string {
	return "cl_workorder_template"
}

// CreateWorkorderTemplateReq 创建工单模板请求
type CreateWorkorderTemplateReq struct {
	Name           string         `json:"name" binding:"required,min=1,max=200"`
	Description    string         `json:"description" binding:"omitempty,max=1000"`
	ProcessID      int            `json:"process_id" binding:"required,min=1"`
	FormDesignID   int            `json:"form_design_id" binding:"required,min=1"`
	DefaultValues  JSONMap `json:"default_values" binding:"omitempty"`
	Status         int8           `json:"status" binding:"required,oneof=1 2"`
	CategoryID     *int           `json:"category_id" binding:"omitempty,min=1"`
	OperatorID   int            `json:"operator_id" binding:"required,min=1"`
	OperatorName string         `json:"operator_name" binding:"required,min=1,max=100"`
	Tags           StringList     `json:"tags" binding:"omitempty"`
}

// UpdateWorkorderTemplateReq 更新工单模板请求
type UpdateWorkorderTemplateReq struct {
	ID            int            `json:"id" binding:"required,min=1"`
	Name          string         `json:"name" binding:"required,min=1,max=200"`
	Description   string         `json:"description" binding:"omitempty,max=1000"`
	ProcessID     int            `json:"process_id" binding:"required,min=1"`
	FormDesignID  int            `json:"form_design_id" binding:"required,min=1"`
	DefaultValues JSONMap `json:"default_values" binding:"omitempty"`
	Status        int8           `json:"status" binding:"required,oneof=1 2"`
	CategoryID    *int           `json:"category_id" binding:"omitempty,min=1"`
	Tags          StringList     `json:"tags" binding:"omitempty"`
}

// DeleteWorkorderTemplateReq 删除工单模板请求
type DeleteWorkorderTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderTemplateReq 获取工单模板详情请求
type DetailWorkorderTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderTemplateReq 工单模板列表请求
type ListWorkorderTemplateReq struct {
	ListReq
	CategoryID   *int  `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	ProcessID    *int  `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
	FormDesignID *int  `json:"form_design_id" form:"form_design_id" binding:"omitempty,min=1"`
	Status       *int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}
