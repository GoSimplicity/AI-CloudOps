package model

import "time"

// ==================== 工单模板相关 ====================

// TemplateDefaultValues 模板默认值结构
type TemplateDefaultValues struct {
	Fields    map[string]interface{} `json:"fields"`    // 表单字段默认值
	Approvers []int                  `json:"approvers"` // 默认审批人
	Priority  int8                   `json:"priority"`  // 默认优先级
	DueHours  *int                   `json:"due_hours"` // 默认处理时限(小时)
}

// Template 模板实体（DAO层）
type Template struct {
	Model
	Name          string    `json:"name" gorm:"column:name;not null;comment:模板名称"`
	Description   string    `json:"description" gorm:"column:description;comment:模板描述"`
	ProcessID     int       `json:"process_id" gorm:"column:process_id;not null;comment:关联的流程ID"`
	DefaultValues string    `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Icon          string    `json:"icon" gorm:"column:icon;comment:图标URL"`
	Status        int8      `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	SortOrder     int       `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	CategoryID    *int      `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID     int       `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName   string    `json:"creator_name" gorm:"-"` // 不存储到数据库中
	Process       *Process  `json:"process" gorm:"foreignKey:ProcessID"`
	Category      *Category `json:"category" gorm:"foreignKey:CategoryID"`
}

// TableName 指定模板表名
func (Template) TableName() string {
	return "template"
}

// 模板请求结构
// CreateTemplateReq 创建模板请求
type CreateTemplateReq struct {
	Name          string                `json:"name" binding:"required,min=1,max=100"`
	Description   string                `json:"description" binding:"omitempty,max=500"`
	ProcessID     int                   `json:"process_id" binding:"required"`
	DefaultValues TemplateDefaultValues `json:"default_values"`
	Icon          string                `json:"icon" binding:"omitempty,url"`
	CategoryID    *int                  `json:"category_id"`
	SortOrder     int                   `json:"sort_order"`
}

// UpdateTemplateReq 更新模板请求
type UpdateTemplateReq struct {
	ID            int                   `json:"id" binding:"required"`
	Name          string                `json:"name" binding:"required,min=1,max=100"`
	Description   string                `json:"description" binding:"omitempty,max=500"`
	ProcessID     int                   `json:"process_id" binding:"required"`
	DefaultValues TemplateDefaultValues `json:"default_values"`
	Icon          string                `json:"icon" binding:"omitempty,url"`
	CategoryID    *int                  `json:"category_id"`
	SortOrder     int                   `json:"sort_order"`
	Status        int8                  `json:"status" binding:"omitempty,oneof=0 1"`
}

// DeleteTemplateReq 删除模板请求
type DeleteTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// DetailTemplateReq 模板详情请求
type DetailTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// ListTemplateReq 模板列表请求
type ListTemplateReq struct {
	ListReq
	Name       *string `json:"name" form:"name"`           // 模板名称
	CategoryID *int    `json:"category_id" form:"category_id"`
	ProcessID  *int    `json:"process_id" form:"process_id"`
	Status     *int8   `json:"status" form:"status"`       // 状态过滤
}

// 模板响应结构
// TemplateResp 模板详情响应
type TemplateResp struct {
	ID            int                   `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	ProcessID     int                   `json:"process_id"`
	Process       *Process              `json:"process"`
	DefaultValues TemplateDefaultValues `json:"default_values"`
	Icon          string                `json:"icon"`
	Status        int8                  `json:"status"`
	SortOrder     int                   `json:"sort_order"`
	CategoryID    *int                  `json:"category_id"`
	Category      *Category             `json:"category"`
	CreatorID     int                   `json:"creator_id"`
	CreatorName   string                `json:"creator_name"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// TemplateItem 模板列表项（用于列表展示）
type TemplateItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ProcessID   int       `json:"process_id"`
	Process     *Process  `json:"process"`
	Icon        string    `json:"icon"`
	Status      int8      `json:"status"`
	SortOrder   int       `json:"sort_order"`
	CategoryID  *int      `json:"category_id"`
	Category    *Category `json:"category"`
	CreatorID   int       `json:"creator_id"`
	CreatorName string    `json:"creator_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
