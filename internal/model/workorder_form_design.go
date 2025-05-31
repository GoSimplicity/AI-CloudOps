package model

import (
	"time"
)

// FormField 表单字段定义
type FormField struct {
	ID           string                 `json:"id"`                       // 字段唯一标识
	Type         string                 `json:"type" binding:"required"`  // 字段类型
	Label        string                 `json:"label" binding:"required"` // 字段标签
	Name         string                 `json:"name" binding:"required"`  // 字段名称
	Required     bool                   `json:"required"`                 // 是否必填
	Placeholder  string                 `json:"placeholder"`              // 占位符
	DefaultValue interface{}            `json:"default_value"`            // 默认值
	Options      []FormFieldOption      `json:"options"`                  // 选项列表
	Validation   FormFieldValidation    `json:"validation"`               // 验证规则
	Props        map[string]interface{} `json:"props"`                    // 其他属性
	SortOrder    int                    `json:"sort_order"`               // 排序
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
	Name        string    `json:"name" gorm:"column:name;not null;comment:表单名称"`
	Description string    `json:"description" gorm:"column:description;comment:表单描述"`
	Schema      string    `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Version     int       `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status      int8      `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID  *int      `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID   int       `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string    `json:"creator_name" gorm:"-"`
	Category    *Category `json:"category" gorm:"foreignKey:CategoryID"`
}

// TableName 表名
func (FormDesign) TableName() string {
	return "workorder_form_design"
}

// CreateFormDesignReq 创建表单设计请求
type CreateFormDesignReq struct {
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
}

// UpdateFormDesignReq 更新表单设计请求
type UpdateFormDesignReq struct {
	ID          int        `json:"id" binding:"required"`
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
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

// PreviewFormDesignResp 预览表单设计响应
type PreviewFormDesignResp struct {
	ID     int        `json:"id"`
	Schema FormSchema `json:"schema"`
}

// FormDesignResp 表单设计响应
type FormDesignResp struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Schema      FormSchema `json:"schema"`
	Version     int        `json:"version"`
	Status      int8       `json:"status"`
	CategoryID  *int       `json:"category_id"`
	Category    *Category  `json:"category"`
	CreatorID   int        `json:"creator_id"`
	CreatorName string     `json:"creator_name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ValidateFormDesignResp 表单验证结果响应
type ValidateFormDesignResp struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
}

// FormDesignItem 表单设计列表项（用于列表展示）
type FormDesignItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	Status      int8      `json:"status"`
	CategoryID  *int      `json:"category_id"`
	Category    *Category `json:"category"`
	CreatorID   int       `json:"creator_id"`
	CreatorName string    `json:"creator_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
