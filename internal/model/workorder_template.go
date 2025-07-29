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

// TemplateDefaultValues 模板默认值结构
type TemplateDefaultValues struct {
	Fields    map[string]any `json:"fields"`    // 表单字段默认值
	Approvers []int          `json:"approvers"` // 默认审批人
	Priority  int8           `json:"priority"`  // 默认优先级
	DueHours  *int           `json:"due_hours"` // 默认处理时限(小时)
}

// Template 模板实体
type Template struct {
	Model
	Name          string    `json:"name" gorm:"column:name;type:varchar(255);not null;uniqueIndex:idx_workorder_template_name,length:255;comment:模板名称"`
	Description   string    `json:"description" gorm:"column:description;type:text;comment:模板描述"`
	ProcessID     int       `json:"process_id" gorm:"column:process_id;not null;index;comment:关联的流程ID"`
	DefaultValues string    `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Icon          string    `json:"icon" gorm:"column:icon;type:varchar(500);comment:图标URL"`
	Status        int8      `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：0-禁用，1-启用"`
	SortOrder     int       `json:"sort_order" gorm:"column:sort_order;default:0;index;comment:排序顺序"`
	CategoryID    *int      `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	CreatorID     int       `json:"creator_id" gorm:"column:creator_id;not null;index;comment:创建人ID"`
	CreatorName   string    `json:"creator_name" gorm:"column:creator_name;type:varchar(100);comment:创建人名称"`
	Process       *Process  `json:"process,omitempty" gorm:"foreignKey:ProcessID"`
	Category      *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}

// TableName 指定模板表名
func (Template) TableName() string {
	return "cl_workorder_templates"
}

// CreateTemplateReq 创建模板请求
type CreateTemplateReq struct {
	Name          string                `json:"name" binding:"required,min=1,max=100"`
	Description   string                `json:"description" binding:"max=500"`
	ProcessID     int                   `json:"process_id" binding:"required,min=1"`
	DefaultValues TemplateDefaultValues `json:"default_values"`
	Icon          string                `json:"icon" binding:"omitempty,url"`
	CategoryID    *int                  `json:"category_id" binding:"omitempty,min=1"`
	SortOrder     int                   `json:"sort_order" binding:"min=0"`
}

type CloneTemplateReq struct {
	ID   int    `json:"id" binding:"required,min=1"`
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// UpdateTemplateReq 更新模板请求
type UpdateTemplateReq struct {
	ID            int                   `json:"id" binding:"required,min=1"`
	Name          string                `json:"name" binding:"required,min=1,max=100"`
	Description   string                `json:"description" binding:"max=500"`
	ProcessID     int                   `json:"process_id" binding:"required,min=1"`
	DefaultValues TemplateDefaultValues `json:"default_values"`
	Icon          string                `json:"icon" binding:"omitempty,url"`
	CategoryID    *int                  `json:"category_id" binding:"omitempty,min=1"`
	SortOrder     int                   `json:"sort_order" binding:"min=0"`
	Status        int8                  `json:"status" binding:"omitempty,oneof=0 1"`
}

// ListTemplateReq 模板列表请求
type ListTemplateReq struct {
	ListReq
	CategoryID *int  `json:"category_id" form:"category_id"`
	ProcessID  *int  `json:"process_id" form:"process_id"`
	Status     *int8 `json:"status" form:"status" binding:"omitempty,oneof=0 1"`
}
