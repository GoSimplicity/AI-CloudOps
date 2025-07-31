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

// 表单设计状态常量
const (
	FormDesignStatusDraft     int8 = 1 // 草稿
	FormDesignStatusPublished int8 = 2 // 已发布
	FormDesignStatusArchived  int8 = 3 // 已归档
)

// 表单字段类型常量
const (
	FormFieldTypeText     = "text"     // 文本输入
	FormFieldTypeNumber   = "number"   // 数字输入
	FormFieldTypeEmail    = "email"    // 邮箱输入
	FormFieldTypePassword = "password" // 密码输入
	FormFieldTypeTextarea = "textarea" // 多行文本
	FormFieldTypeSelect   = "select"   // 下拉选择
	FormFieldTypeRadio    = "radio"    // 单选框
	FormFieldTypeCheckbox = "checkbox" // 复选框
	FormFieldTypeDate     = "date"     // 日期选择
	FormFieldTypeTime     = "time"     // 时间选择
	FormFieldTypeDatetime = "datetime" // 日期时间
	FormFieldTypeFile     = "file"     // 文件上传
	FormFieldTypeImage    = "image"    // 图片上传
	FormFieldTypeSwitch   = "switch"   // 开关
	FormFieldTypeSlider   = "slider"   // 滑块
	FormFieldTypeRate     = "rate"     // 评分
	FormFieldTypeColor    = "color"    // 颜色选择
)

// FormField 表单字段定义
type FormField struct {
	ID           string                 `json:"id"`                        // 字段唯一标识
	Type         string                 `json:"type" binding:"required"`   // 字段类型
	Label        string                 `json:"label" binding:"required"`  // 字段标签
	Name         string                 `json:"name" binding:"required"`   // 字段名称
	Required     bool                   `json:"required"`                  // 是否必填
	Placeholder  string                 `json:"placeholder"`               // 占位符
	DefaultValue interface{}            `json:"default_value"`             // 默认值
	Options      []FormFieldOption      `json:"options,omitempty"`         // 选项列表
	Validation   FormFieldValidation    `json:"validation,omitempty"`      // 验证规则
	Props        map[string]interface{} `json:"props,omitempty"`           // 其他属性
	SortOrder    int                    `json:"sort_order"`                // 排序顺序
	Disabled     bool                   `json:"disabled"`                  // 是否禁用
	Hidden       bool                   `json:"hidden"`                    // 是否隐藏
	Description  string                 `json:"description,omitempty"`     // 字段描述
	Width        string                 `json:"width,omitempty"`           // 字段宽度
	ColSpan      int                    `json:"col_span,omitempty"`        // 列跨度
	Group        string                 `json:"group,omitempty"`           // 字段分组
}

// FormFieldOption 表单字段选项
type FormFieldOption struct {
	Label    string      `json:"label"`     // 选项标签
	Value    interface{} `json:"value"`     // 选项值
	Disabled bool        `json:"disabled"`  // 是否禁用
	Color    string      `json:"color"`     // 选项颜色
}

// FormFieldValidation 表单字段验证规则
type FormFieldValidation struct {
	MinLength *int     `json:"min_length,omitempty"` // 最小长度
	MaxLength *int     `json:"max_length,omitempty"` // 最大长度
	Min       *float64 `json:"min,omitempty"`        // 最小值
	Max       *float64 `json:"max,omitempty"`        // 最大值
	Pattern   string   `json:"pattern,omitempty"`    // 正则表达式
	Message   string   `json:"message,omitempty"`    // 验证错误信息
	Custom    string   `json:"custom,omitempty"`     // 自定义验证规则
}

// FormLayout 表单布局配置
type FormLayout struct {
	Type        string                 `json:"type"`         // 布局类型：grid, flex, tabs
	Columns     int                    `json:"columns"`      // 列数
	LabelWidth  string                 `json:"label_width"`  // 标签宽度
	LabelAlign  string                 `json:"label_align"`  // 标签对齐方式
	Size        string                 `json:"size"`         // 表单尺寸
	Spacing     int                    `json:"spacing"`      // 间距
	Background  string                 `json:"background"`   // 背景色
	BorderStyle string                 `json:"border_style"` // 边框样式
	Props       map[string]interface{} `json:"props"`        // 其他布局属性
}

// FormSchema 表单结构定义
type FormSchema struct {
	Version string      `json:"version" binding:"required"`      // 模式版本
	Fields  []FormField `json:"fields" binding:"required"`       // 字段列表
	Layout  FormLayout  `json:"layout"`                          // 布局配置
	Groups  []FormGroup `json:"groups,omitempty"`                // 字段分组
	Rules   []FormRule  `json:"rules,omitempty"`                 // 表单规则
	Events  []FormEvent `json:"events,omitempty"`                // 表单事件
}

// FormGroup 表单分组
type FormGroup struct {
	Name        string `json:"name"`         // 分组名称
	Title       string `json:"title"`        // 分组标题
	Description string `json:"description"`  // 分组描述
	Collapsible bool   `json:"collapsible"`  // 是否可折叠
	Collapsed   bool   `json:"collapsed"`    // 是否默认折叠
	SortOrder   int    `json:"sort_order"`   // 排序顺序
}

// FormRule 表单规则
type FormRule struct {
	ID         string                 `json:"id"`         // 规则ID
	Name       string                 `json:"name"`       // 规则名称
	Type       string                 `json:"type"`       // 规则类型：show, hide, required, readonly
	Condition  FormCondition          `json:"condition"`  // 触发条件
	Actions    []FormAction           `json:"actions"`    // 执行动作
	Props      map[string]interface{} `json:"props"`      // 规则属性
}

// FormCondition 表单条件
type FormCondition struct {
	Field    string      `json:"field"`    // 字段名
	Operator string      `json:"operator"` // 操作符：eq, ne, gt, lt, in, contains
	Value    interface{} `json:"value"`    // 条件值
	Logic    string      `json:"logic"`    // 逻辑关系：and, or
}

// FormAction 表单动作
type FormAction struct {
	Type   string                 `json:"type"`   // 动作类型
	Target string                 `json:"target"` // 目标字段
	Value  interface{}            `json:"value"`  // 动作值
	Props  map[string]interface{} `json:"props"`  // 动作属性
}

// FormEvent 表单事件
type FormEvent struct {
	Name    string                 `json:"name"`    // 事件名称
	Trigger string                 `json:"trigger"` // 触发器
	Actions []FormAction           `json:"actions"` // 执行动作
	Props   map[string]interface{} `json:"props"`   // 事件属性
}

// WorkorderFormDesign 工单表单设计实体
type WorkorderFormDesign struct {
	Model
	Name        string         `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:表单名称"`
	Description string         `json:"description" gorm:"column:description;type:varchar(1000);comment:表单描述"`
	Schema      datatypes.JSON `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Version     string         `json:"version" gorm:"column:version;type:varchar(20);not null;default:'1.0.0';comment:版本号"`
	Status      int8           `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-草稿，2-已发布，3-已归档"`
	CategoryID  *int           `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	CreatorID   int            `json:"creator_id" gorm:"column:creator_id;not null;index;comment:创建人ID"`
	CreatorName string         `json:"creator_name" gorm:"-"`
	Tags        StringList     `json:"tags" gorm:"column:tags;comment:标签"`
	IsTemplate  bool           `json:"is_template" gorm:"column:is_template;not null;default:false;comment:是否为模板"`
	UseCount    int            `json:"use_count" gorm:"column:use_count;not null;default:0;comment:使用次数"`

	// 关联信息（不存储到数据库）
	CategoryName string `json:"category_name,omitempty" gorm:"-"`
}

// TableName 指定工单表单设计表名
func (WorkorderFormDesign) TableName() string {
	return "cl_workorder_form_design"
}

// CreateWorkorderFormDesignReq 创建工单表单设计请求
type CreateWorkorderFormDesignReq struct {
	Name        string     `json:"name" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"omitempty,max=1000"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id" binding:"omitempty,min=1"`
	Tags        []string   `json:"tags" binding:"omitempty"`
	IsTemplate  bool       `json:"is_template"`
}

// UpdateWorkorderFormDesignReq 更新工单表单设计请求
type UpdateWorkorderFormDesignReq struct {
	ID          int        `json:"id" binding:"required,min=1"`
	Name        string     `json:"name" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"omitempty,max=1000"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id" binding:"omitempty,min=1"`
	Tags        []string   `json:"tags" binding:"omitempty"`
	IsTemplate  bool       `json:"is_template"`
}

// DeleteWorkorderFormDesignReq 删除工单表单设计请求
type DeleteWorkorderFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderFormDesignReq 获取工单表单设计详情请求
type DetailWorkorderFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderFormDesignReq 工单表单设计列表请求
type ListWorkorderFormDesignReq struct {
	ListReq
	CategoryID *int  `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	Status     *int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2 3"`
	IsTemplate *bool `json:"is_template" form:"is_template"`
	Tags       []string `json:"tags" form:"tags"`
}

// PublishWorkorderFormDesignReq 发布工单表单设计请求
type PublishWorkorderFormDesignReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// ArchiveWorkorderFormDesignReq 归档工单表单设计请求
type ArchiveWorkorderFormDesignReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// CloneWorkorderFormDesignReq 克隆工单表单设计请求
type CloneWorkorderFormDesignReq struct {
	ID   int    `json:"id" binding:"required,min=1"`
	Name string `json:"name" binding:"required,min=1,max=200"`
}

// PreviewWorkorderFormDesignReq 预览工单表单设计请求
type PreviewWorkorderFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ValidateWorkorderFormDesignReq 验证工单表单设计请求
type ValidateWorkorderFormDesignReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// ImportWorkorderFormDesignReq 导入工单表单设计请求
type ImportWorkorderFormDesignReq struct {
	Data []FormImportData `json:"data" binding:"required,min=1"`
}

// FormImportData 表单导入数据
type FormImportData struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
	Tags        []string   `json:"tags"`
}

// ExportWorkorderFormDesignReq 导出工单表单设计请求
type ExportWorkorderFormDesignReq struct {
	IDs []int `json:"ids" binding:"required,min=1,dive,min=1"`
}

// BatchUpdateFormDesignStatusReq 批量更新表单设计状态请求
type BatchUpdateFormDesignStatusReq struct {
	IDs    []int `json:"ids" binding:"required,min=1,dive,min=1"`
	Status int8  `json:"status" binding:"required,oneof=1 2 3"`
}

// FormDesignStatistics 表单设计统计
type FormDesignStatistics struct {
	DraftCount     int64 `json:"draft_count"`     // 草稿数量
	PublishedCount int64 `json:"published_count"` // 已发布数量
	ArchivedCount  int64 `json:"archived_count"`  // 已归档数量
	TemplateCount  int64 `json:"template_count"`  // 模板数量
	TotalUseCount  int64 `json:"total_use_count"` // 总使用次数
}
