package model

import "time"

// ==================== 流程定义相关 ====================

// ProcessStep 流程步骤定义
type ProcessStep struct {
	ID         string                 `json:"id"`          // 步骤ID
	Name       string                 `json:"name"`        // 步骤名称
	Type       string                 `json:"type"`        // 步骤类型
	Roles      []string               `json:"roles"`       // 角色列表
	Users      []int                  `json:"users"`       // 用户ID列表
	Actions    []string               `json:"actions"`     // 可执行的动作
	Conditions []ProcessCondition     `json:"conditions"`  // 条件列表
	TimeLimit  *int                   `json:"time_limit"`  // 时间限制(分钟)
	AutoAssign bool                   `json:"auto_assign"` // 是否自动分配
	Parallel   bool                   `json:"parallel"`    // 是否并行处理
	Props      map[string]interface{} `json:"props"`       // 步骤属性
	Position   ProcessPosition        `json:"position"`    // 步骤位置
}

// ProcessCondition 流程条件
type ProcessCondition struct {
	Field    string      `json:"field"`    // 字段名
	Operator string      `json:"operator"` // 操作符
	Value    interface{} `json:"value"`    // 条件值
}

// ProcessPosition 流程步骤位置
type ProcessPosition struct {
	X int `json:"x"` // X坐标
	Y int `json:"y"` // Y坐标
}

// ProcessConnection 流程连接
type ProcessConnection struct {
	From      string `json:"from"`      // 来源步骤ID
	To        string `json:"to"`        // 目标步骤ID
	Condition string `json:"condition"` // 条件表达式
	Label     string `json:"label"`     // 连接标签
}

// ProcessDefinition 流程定义
type ProcessDefinition struct {
	Steps       []ProcessStep       `json:"steps"`       // 步骤列表
	Connections []ProcessConnection `json:"connections"` // 连接列表
	Variables   []ProcessVariable   `json:"variables"`   // 变量列表
}

// ProcessVariable 流程变量
type ProcessVariable struct {
	Name         string      `json:"name"`          // 变量名
	Type         string      `json:"type"`          // 变量类型
	DefaultValue interface{} `json:"default_value"` // 默认值
	Description  string      `json:"description"`   // 变量描述
}

// Process 流程实体（DAO层）
type Process struct {
	Model
	Name         string      `json:"name" gorm:"column:name;not null;comment:流程名称"`
	Description  string      `json:"description" gorm:"column:description;comment:流程描述"`
	FormDesignID int         `json:"form_design_id" gorm:"column:form_design_id;not null;comment:关联的表单设计ID"`
	Definition   string      `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version      int         `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status       int8        `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID   *int        `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int         `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName  string      `json:"creator_name" gorm:"-"`
	FormDesign   *FormDesign `json:"form_design" gorm:"foreignKey:FormDesignID"`
	Category     *Category   `json:"category" gorm:"foreignKey:CategoryID"`
}

// TableName 指定流程表名
func (Process) TableName() string {
	return "process"
}

// 流程请求结构
// CreateProcessReq 创建流程请求
type CreateProcessReq struct {
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"omitempty,max=500"`
	FormDesignID int               `json:"form_design_id" binding:"required"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id"`
}

// UpdateProcessReq 更新流程请求
type UpdateProcessReq struct {
	ID           int               `json:"id" binding:"required"`
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"omitempty,max=500"`
	FormDesignID int               `json:"form_design_id" binding:"required"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id"`
}

// DeleteProcessReq 删除流程请求
type DeleteProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// DetailProcessReq 流程详情请求
type DetailProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// ListProcessReq 流程列表请求
type ListProcessReq struct {
	ListReq
	Name         *string `json:"name" form:"name"`
	CategoryID   *int    `json:"category_id" form:"category_id"`
	FormDesignID *int    `json:"form_design_id" form:"form_design_id"`
	Status       *int8   `json:"status" form:"status"`
}

// PublishProcessReq 发布流程请求
type PublishProcessReq struct {
	ID int `json:"id" binding:"required"`
}

// CloneProcessReq 克隆流程请求
type CloneProcessReq struct {
	ID   int    `json:"id" binding:"required"`
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// 流程响应结构
// ProcessResp 流程详情响应
type ProcessResp struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	FormDesignID int               `json:"form_design_id"`
	FormDesign   *FormDesign       `json:"form_design"`
	Definition   ProcessDefinition `json:"definition"`
	Version      int               `json:"version"`
	Status       int8              `json:"status"`
	CategoryID   *int              `json:"category_id"`
	Category     *Category         `json:"category"`
	CreatorID    int               `json:"creator_id"`
	CreatorName  string            `json:"creator_name"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ValidateProcessResp 流程验证响应
type ValidateProcessResp struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
}

// ProcessItem 流程列表项（用于列表展示）
type ProcessItem struct {
	ID           int         `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	FormDesignID int         `json:"form_design_id"`
	FormDesign   *FormDesign `json:"form_design"`
	Version      int         `json:"version"`
	Status       int8        `json:"status"`
	CategoryID   *int        `json:"category_id"`
	Category     *Category   `json:"category"`
	CreatorID    int         `json:"creator_id"`
	CreatorName  string      `json:"creator_name"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
