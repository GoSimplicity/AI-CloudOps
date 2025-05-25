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

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ==================== 表单设计相关 ====================

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

type FormFieldOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

type FormFieldValidation struct {
	MinLength *int   `json:"min_length"`
	MaxLength *int   `json:"max_length"`
	Min       *int   `json:"min"`
	Max       *int   `json:"max"`
	Pattern   string `json:"pattern"`
	Message   string `json:"message"`
}

type FormSchema struct {
	Fields []FormField `json:"fields"`
	Layout string      `json:"layout"` // 布局类型
	Style  string      `json:"style"`  // 样式配置
}

// FormDesign 表单设计实体（DAO层）
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

func (FormDesign) TableName() string {
	return "form_design"
}

// 表单设计请求结构
type CreateFormDesignReq struct {
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
}

type UpdateFormDesignReq struct {
	ID          int        `json:"id" binding:"required"`
	Name        string     `json:"name" binding:"required,min=1,max=100"`
	Description string     `json:"description" binding:"omitempty,max=500"`
	Schema      FormSchema `json:"schema" binding:"required"`
	CategoryID  *int       `json:"category_id"`
}

type DeleteFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type DetailFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListFormDesignReq struct {
	ListReq
	CategoryID *int `json:"category_id" form:"category_id"`
}

type PublishFormDesignReq struct {
	ID int `json:"id" binding:"required"`
}

type CloneFormDesignReq struct {
	ID   int    `json:"id" binding:"required"`
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type PreviewFormDesignReq struct {
	ID     int        `json:"id" binding:"required"`
	Schema FormSchema `json:"schema" binding:"required"`
}

// 表单设计响应结构
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

// 表单设计列表项（用于列表展示）
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

// ==================== 流程定义相关 ====================

// ProcessStep 流程步骤定义
type ProcessStep struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Roles      []string               `json:"roles"`
	Users      []int                  `json:"users"`
	Actions    []string               `json:"actions"`
	Conditions []ProcessCondition     `json:"conditions"`
	TimeLimit  *int                   `json:"time_limit"`
	AutoAssign bool                   `json:"auto_assign"`
	Parallel   bool                   `json:"parallel"`
	Props      map[string]interface{} `json:"props"`
	Position   ProcessPosition        `json:"position"`
}

type ProcessCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type ProcessPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ProcessConnection struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition"`
	Label     string `json:"label"`
}

type ProcessDefinition struct {
	Steps       []ProcessStep       `json:"steps"`
	Connections []ProcessConnection `json:"connections"`
	Variables   []ProcessVariable   `json:"variables"`
}

type ProcessVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	DefaultValue interface{} `json:"default_value"`
	Description  string      `json:"description"`
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

func (Process) TableName() string {
	return "process"
}

// 流程请求结构
type CreateProcessReq struct {
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"omitempty,max=500"`
	FormDesignID int               `json:"form_design_id" binding:"required"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id"`
}

type UpdateProcessReq struct {
	ID           int               `json:"id" binding:"required"`
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"omitempty,max=500"`
	FormDesignID int               `json:"form_design_id" binding:"required"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id"`
}

type DeleteProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type DetailProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListProcessReq struct {
	ListReq
	Name        *string `json:"name" form:"name"`
	CategoryID   *int   `json:"category_id" form:"category_id"`
	FormDesignID *int   `json:"form_design_id" form:"form_design_id"`
}

type PublishProcessReq struct {
	ID int `json:"id" binding:"required"`
}

type CloneProcessReq struct {
	ID   int    `json:"id" binding:"required"`
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// 流程响应结构
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

type ValidateProcessResp struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
}

// 流程列表项（用于列表展示）
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

// ==================== 工单模板相关 ====================

type TemplateDefaultValues struct {
	Fields    map[string]interface{} `json:"fields"`
	Approvers []int                  `json:"approvers"`
	Priority  int8                   `json:"priority"`
	DueHours  *int                   `json:"due_hours"`
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
	CreatorName   string    `json:"creator_name" gorm:"-"`
	Process       *Process  `json:"process" gorm:"foreignKey:ProcessID"`
	Category      *Category `json:"category" gorm:"foreignKey:CategoryID"`
}

func (Template) TableName() string {
	return "template"
}

// 模板请求结构
type CreateTemplateReq struct {
	Name          string                `json:"name" binding:"required,min=1,max=100"`
	Description   string                `json:"description" binding:"omitempty,max=500"`
	ProcessID     int                   `json:"process_id" binding:"required"`
	DefaultValues TemplateDefaultValues `json:"default_values"`
	Icon          string                `json:"icon" binding:"omitempty,url"`
	CategoryID    *int                  `json:"category_id"`
	SortOrder     int                   `json:"sort_order"`
}

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

type DeleteTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type DetailTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListTemplateReq struct {
	ListReq
	CategoryID *int `json:"category_id" form:"category_id"`
	ProcessID  *int `json:"process_id" form:"process_id"`
}

// 模板响应结构
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

// 模板列表项（用于列表展示）
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

// ==================== 工单实例相关 ====================

// 工单状态常量
const (
	InstanceStatusDraft      int8 = 0 // 草稿
	InstanceStatusProcessing int8 = 1 // 处理中
	InstanceStatusCompleted  int8 = 2 // 已完成
	InstanceStatusCancelled  int8 = 3 // 已取消
	InstanceStatusRejected   int8 = 4 // 已拒绝
	InstanceStatusPending    int8 = 5 // 待处理
	InstanceStatusOverdue    int8 = 6 // 已超时
)

// 优先级常量
const (
	PriorityLow      int8 = 0 // 低
	PriorityNormal   int8 = 1 // 普通
	PriorityHigh     int8 = 2 // 高
	PriorityUrgent   int8 = 3 // 紧急
	PriorityCritical int8 = 4 // 严重
)

// JSONMap 自定义JSON类型，用于处理map[string]interface{}
type JSONMap map[string]interface{}

// StringSlice 自定义字符串切片类型，用于处理数组到逗号分隔字符串的转换
type StringSlice []string

// Value 实现driver.Valuer接口，存储到数据库时转为逗号分隔的字符串
func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "", nil
	}
	return strings.Join(s, ","), nil
}

// Scan 实现sql.Scanner接口，从数据库读取时转为字符串切片
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("无法扫描 %T 到 StringSlice", value)
	}

	if str == "" {
		*s = StringSlice{}
		return nil
	}

	*s = strings.Split(str, ",")
	return nil
}

// MarshalJSON 实现json.Marshaler接口
func (s StringSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(s))
}

// UnmarshalJSON 实现json.Unmarshaler接口
func (s *StringSlice) UnmarshalJSON(data []byte) error {
	var slice []string
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	*s = StringSlice(slice)
	return nil
}

// Instance 工单实例
type Instance struct {
	Model
	Title        string      `json:"title" gorm:"column:title;not null;comment:工单标题"`
	TemplateID   *int        `json:"template_id" gorm:"column:template_id;comment:模板ID"`
	ProcessID    int         `json:"process_id" gorm:"column:process_id;not null;comment:流程ID"`
	FormData     JSONMap     `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	CurrentStep  string      `json:"current_step" gorm:"column:current_step;not null;comment:当前步骤"`
	Status       int8        `json:"status" gorm:"column:status;not null;comment:状态"`
	Priority     int8        `json:"priority" gorm:"column:priority;default:1;comment:优先级"`
	CategoryID   *int        `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int         `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	Description  string      `json:"description" gorm:"column:description;comment:描述"`
	CreatorName  string      `json:"creator_name" gorm:"-"`
	AssigneeID   *int        `json:"assignee_id" gorm:"column:assignee_id;comment:当前处理人ID"`
	AssigneeName string      `json:"assignee_name" gorm:"-"`
	CompletedAt  *time.Time  `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`
	DueDate      *time.Time  `json:"due_date" gorm:"column:due_date;comment:截止时间"`
	Tags         StringSlice `json:"tags" gorm:"column:tags;comment:标签"`
	ProcessData  JSONMap     `json:"process_data" gorm:"column:process_data;type:json;comment:流程数据"`

	// 关联数据
	Template *Template `json:"template" gorm:"foreignKey:TemplateID"`
	Process  *Process  `json:"process" gorm:"foreignKey:ProcessID"`
	Category *Category `json:"category" gorm:"foreignKey:CategoryID"`
}

func (Instance) TableName() string {
	return "instance"
}

// 工单实例请求结构
type CreateInstanceReq struct {
	Title       string                 `json:"title" binding:"required,min=1,max=200"`
	TemplateID  *int                   `json:"template_id"`
	ProcessID   int                    `json:"process_id" binding:"required"`
	FormData    map[string]interface{} `json:"form_data" binding:"required"`
	Description string                 `json:"description" binding:"omitempty,max=1000"`
	Priority    int8                   `json:"priority" binding:"omitempty,oneof=0 1 2 3 4"`
	CategoryID  *int                   `json:"category_id"`
	DueDate     *time.Time             `json:"due_date"`
	Tags        []string               `json:"tags"`
	AssigneeID  *int                   `json:"assignee_id"`
}

type UpdateInstanceReq struct {
	ID          int                    `json:"id" binding:"required"`
	Title       string                 `json:"title" binding:"required,min=1,max=200"`
	FormData    map[string]interface{} `json:"form_data" binding:"required"`
	Description string                 `json:"description" binding:"omitempty,max=1000"`
	Priority    int8                   `json:"priority" binding:"omitempty,oneof=0 1 2 3 4"`
	CategoryID  *int                   `json:"category_id"`
	DueDate     *time.Time             `json:"due_date"`
	Tags        []string               `json:"tags"`
}

type DeleteInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type DetailInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListInstanceReq struct {
	ListReq
	Status     *int8      `json:"status" form:"status"`
	Priority   *int8      `json:"priority" form:"priority"`
	CategoryID *int       `json:"category_id" form:"category_id"`
	CreatorID  *int       `json:"creator_id" form:"creator_id"`
	AssigneeID *int       `json:"assignee_id" form:"assignee_id"`
	ProcessID  *int       `json:"process_id" form:"process_id"`
	TemplateID *int       `json:"template_id" form:"template_id"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
	Tags       []string   `json:"tags" form:"tags"`
	Overdue    *bool      `json:"overdue" form:"overdue"`
}

type MyInstanceReq struct {
	ListReq
	Type       string     `json:"type" form:"type" binding:"omitempty,oneof=created assigned"`
	Status     *int8      `json:"status" form:"status"`
	Priority   *int8      `json:"priority" form:"priority"`
	CategoryID *int       `json:"category_id" form:"category_id"`
	ProcessID  *int       `json:"process_id" form:"process_id"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
}

type InstanceActionReq struct {
	InstanceID int                    `json:"instance_id" binding:"required"`
	Action     string                 `json:"action" binding:"required,oneof=approve reject transfer revoke"`
	Comment    string                 `json:"comment" binding:"omitempty,max=1000"`
	FormData   map[string]interface{} `json:"form_data"`
	AssigneeID *int                   `json:"assignee_id"`
	StepID     string                 `json:"step_id"`
}

type InstanceCommentReq struct {
	InstanceID int    `json:"instance_id" binding:"required"`
	Content    string `json:"content" binding:"required,max=1000"`
	ParentID   *int   `json:"parent_id"`
}

// 工单实例响应结构
type InstanceResp struct {
	ID           int                    `json:"id"`
	Title        string                 `json:"title"`
	TemplateID   *int                   `json:"template_id"`
	Template     *Template              `json:"template"`
	ProcessID    int                    `json:"process_id"`
	Process      *Process               `json:"process"`
	FormData     map[string]interface{} `json:"form_data"`
	CurrentStep  string                 `json:"current_step"`
	Status       int8                   `json:"status"`
	Priority     int8                   `json:"priority"`
	CategoryID   *int                   `json:"category_id"`
	Category     *Category              `json:"category"`
	CreatorID    int                    `json:"creator_id"`
	CreatorName  string                 `json:"creator_name"`
	Description  string                 `json:"description"`
	AssigneeID   *int                   `json:"assignee_id"`
	AssigneeName string                 `json:"assignee_name"`
	CompletedAt  *time.Time             `json:"completed_at"`
	DueDate      *time.Time             `json:"due_date"`
	Tags         []string               `json:"tags"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`

	// 扩展信息
	Flows       []InstanceFlowResp       `json:"flows"`
	Comments    []InstanceCommentResp    `json:"comments"`
	Attachments []InstanceAttachmentResp `json:"attachments"`
	NextSteps   []string                 `json:"next_steps"`
	IsOverdue   bool                     `json:"is_overdue"`
}

// 工单实例列表项（用于列表展示）
type InstanceItem struct {
	ID           int        `json:"id"`
	Title        string     `json:"title"`
	TemplateID   *int       `json:"template_id"`
	Template     *Template  `json:"template"`
	ProcessID    int        `json:"process_id"`
	Process      *Process   `json:"process"`
	CurrentStep  string     `json:"current_step"`
	Status       int8       `json:"status"`
	Priority     int8       `json:"priority"`
	CategoryID   *int       `json:"category_id"`
	Category     *Category  `json:"category"`
	CreatorID    int        `json:"creator_id"`
	CreatorName  string     `json:"creator_name"`
	AssigneeID   *int       `json:"assignee_id"`
	AssigneeName string     `json:"assignee_name"`
	CompletedAt  *time.Time `json:"completed_at"`
	DueDate      *time.Time `json:"due_date"`
	Tags         []string   `json:"tags"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	IsOverdue    bool       `json:"is_overdue"`
}

// ==================== 工单流转记录相关 ====================

// InstanceFlow 工单流转记录实体（DAO层）
type InstanceFlow struct {
	Model
	InstanceID   int     `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	StepID       string  `json:"step_id" gorm:"column:step_id;not null;comment:步骤ID"`
	StepName     string  `json:"step_name" gorm:"column:step_name;not null;comment:步骤名称"`
	Action       string  `json:"action" gorm:"column:action;not null;comment:操作"`
	OperatorID   int     `json:"operator_id" gorm:"column:operator_id;not null;comment:操作人ID"`
	OperatorName string  `json:"operator_name" gorm:"-"`
	Comment      string  `json:"comment" gorm:"column:comment;type:text;comment:处理意见"`
	FormData     JSONMap `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	Duration     *int    `json:"duration" gorm:"column:duration;comment:处理时长(分钟)"`
	FromStepID   string  `json:"from_step_id" gorm:"column:from_step_id;comment:来源步骤ID"`
	ToStepID     string  `json:"to_step_id" gorm:"column:to_step_id;comment:目标步骤ID"`
}

func (InstanceFlow) TableName() string {
	return "instance_flow"
}

// 工单流转记录响应结构
type InstanceFlowResp struct {
	ID           int                    `json:"id"`
	InstanceID   int                    `json:"instance_id"`
	StepID       string                 `json:"step_id"`
	StepName     string                 `json:"step_name"`
	Action       string                 `json:"action"`
	OperatorID   int                    `json:"operator_id"`
	OperatorName string                 `json:"operator_name"`
	Comment      string                 `json:"comment"`
	FormData     map[string]interface{} `json:"form_data"`
	Duration     *int                   `json:"duration"`
	FromStepID   string                 `json:"from_step_id"`
	ToStepID     string                 `json:"to_step_id"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ==================== 工单评论相关 ====================

// InstanceComment 工单评论实体（DAO层）
type InstanceComment struct {
	Model
	InstanceID  int    `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	Content     string `json:"content" gorm:"column:content;type:text;not null;comment:评论内容"`
	CreatorID   int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name" gorm:"-"`
	ParentID    *int   `json:"parent_id" gorm:"column:parent_id;comment:父评论ID"`
	IsSystem    bool   `json:"is_system" gorm:"column:is_system;default:false;comment:是否系统评论"`
}

func (InstanceComment) TableName() string {
	return "instance_comment"
}

// 工单评论响应结构
type InstanceCommentResp struct {
	ID          int                   `json:"id"`
	InstanceID  int                   `json:"instance_id"`
	Content     string                `json:"content"`
	CreatorID   int                   `json:"creator_id"`
	CreatorName string                `json:"creator_name"`
	ParentID    *int                  `json:"parent_id"`
	IsSystem    bool                  `json:"is_system"`
	CreatedAt   time.Time             `json:"created_at"`
	Children    []InstanceCommentResp `json:"children"`
}

// ==================== 工单附件相关 ====================

// InstanceAttachment 工单附件实体（DAO层）
type InstanceAttachment struct {
	Model
	InstanceID   int    `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	FileName     string `json:"file_name" gorm:"column:file_name;not null;comment:文件名"`
	FileSize     int64  `json:"file_size" gorm:"column:file_size;not null;comment:文件大小(字节)"`
	FilePath     string `json:"file_path" gorm:"column:file_path;not null;comment:文件路径"`
	FileType     string `json:"file_type" gorm:"column:file_type;not null;comment:文件类型"`
	UploaderID   int    `json:"uploader_id" gorm:"column:uploader_id;not null;comment:上传人ID"`
	UploaderName string `json:"uploader_name" gorm:"-"`
}

func (InstanceAttachment) TableName() string {
	return "instance_attachment"
}

// 工单附件响应结构
type InstanceAttachmentResp struct {
	ID           int       `json:"id"`
	InstanceID   int       `json:"instance_id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	FilePath     string    `json:"file_path"`
	FileType     string    `json:"file_type"`
	UploaderID   int       `json:"uploader_id"`
	UploaderName string    `json:"uploader_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// ==================== 分类相关 ====================

// Category 分类实体（DAO层）
type Category struct {
	ID          int        `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name        string     `json:"name" gorm:"column:name;not null;comment:分类名称"`
	ParentID    *int       `json:"parent_id" gorm:"column:parent_id;comment:父分类ID"`
	Icon        string     `json:"icon" gorm:"column:icon;comment:图标"`
	SortOrder   int        `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	Status      int8       `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	Description string     `json:"description" gorm:"column:description;comment:分类描述"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"column:deleted_at;index;comment:删除时间"`

	Children []Category `json:"children" gorm:"-"`
}

func (Category) TableName() string {
	return "category"
}

// 分类请求结构
type CreateCategoryReq struct {
	Name        string `json:"name" binding:"required"`
	ParentID    *int   `json:"parent_id"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
}

type UpdateCategoryReq struct {
	ID          int    `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	ParentID    *int   `json:"parent_id"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
	Status      int8   `json:"status" binding:"required"`
}

type DeleteCategoryReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListCategoryReq struct {
	Name     string `json:"name" form:"name"`
	Status   *int8  `json:"status" form:"status"`
	Page     int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize int    `json:"page_size" form:"page_size" binding:"required,min=1"`
}

type DetailCategoryReq struct {
	ID int `json:"id" uri:"id" binding:"required"`
}

// 分类响应结构
type CategoryResp struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	ParentID    *int       `json:"parent_id"`
	Icon        string     `json:"icon"`
	SortOrder   int        `json:"sort_order"`
	Status      int8       `json:"status"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Children    []Category `json:"children"`
}

// ==================== 统计相关 ====================

// 统计请求结构
type OverviewStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
}

type TrendStatsReq struct {
	StartDate  time.Time `json:"start_date" form:"start_date" binding:"required"`
	EndDate    time.Time `json:"end_date" form:"end_date" binding:"required"`
	Dimension  string    `json:"dimension" form:"dimension" binding:"required,oneof=day week month"`
	CategoryID *int      `json:"category_id" form:"category_id"`
}

type CategoryStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	Top       int        `json:"top" form:"top" binding:"omitempty,min=5,max=20"`
}

type PerformanceStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	UserID    *int       `json:"user_id" form:"user_id"`
	Top       int        `json:"top" form:"top" binding:"omitempty,min=5,max=50"`
}

type UserStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	UserID    *int       `json:"user_id" form:"user_id"`
}

// 统计响应结构
type OverviewStatsResp struct {
	TotalCount      int64   `json:"total_count"`
	CompletedCount  int64   `json:"completed_count"`
	ProcessingCount int64   `json:"processing_count"`
	PendingCount    int64   `json:"pending_count"`
	OverdueCount    int64   `json:"overdue_count"`
	CompletionRate  float64 `json:"completion_rate"`
	AvgProcessTime  float64 `json:"avg_process_time"`
	TodayCreated    int64   `json:"today_created"`
	TodayCompleted  int64   `json:"today_completed"`
}

type TrendStatsResp struct {
	Dates            []string `json:"dates"`
	CreatedCounts    []int64  `json:"created_counts"`
	CompletedCounts  []int64  `json:"completed_counts"`
	ProcessingCounts []int64  `json:"processing_counts"`
}

type CategoryStatsItem struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Count        int64   `json:"count"`
	Percentage   float64 `json:"percentage"`
}

type CategoryStatsResp struct {
	Items      []CategoryStatsItem `json:"items"`
	TotalCount int64               `json:"total_count"`
	Total      int64               `json:"total"`
	Count      int64               `json:"count"`
	Percentage int64               `json:"percentage"`
}

// PerformanceStatsItem 绩效统计项
type PerformanceStatsItem struct {
	UserID            int     `json:"user_id"`
	UserName          string  `json:"user_name"`
	AssignedCount     int64   `json:"assigned_count"`
	CompletedCount    int64   `json:"completed_count"`
	CompletionRate    float64 `json:"completion_rate"`
	AvgResponseTime   float64 `json:"avg_response_time"`
	AvgProcessingTime float64 `json:"avg_processing_time"`
	OverdueCount      int64   `json:"overdue_count"`
	SatisfactionScore float64 `json:"satisfaction_score"`
}

// PerformanceStatsResp 绩效统计响应
type PerformanceStatsResp struct {
	Items             []PerformanceStatsItem `json:"items"`
	UserID            int                    `json:"user_id"`             // 用户ID
	TotalAssigned     int64                  `json:"total_assigned"`      // 总分配数
	TotalCompleted    int64                  `json:"total_completed"`     // 总完成数
	TotalOverdue      int64                  `json:"total_overdue"`       // 总超时数
	AvgResponseTime   float64                `json:"avg_response_time"`   // 平均响应时间
	AvgProcessingTime float64                `json:"avg_processing_time"` // 平均处理时间
	CompletionRate    float64                `json:"completion_rate"`     // 总完成率
	CompletedCount    int64                  `json:"completed_count"`     // 总完成数（兼容字段）
	OverdueCount      int64                  `json:"overdue_count"`       // 总超时数（兼容字段）
	AssignedCount     int64                  `json:"assigned_count"`      // 总分配数（兼容字段）
}

type UserStatsResp struct {
	UserID            int     `json:"user_id"`
	CreatedCount      int64   `json:"created_count"`
	AssignedCount     int64   `json:"assigned_count"`
	CompletedCount    int64   `json:"completed_count"`
	PendingCount      int64   `json:"pending_count"`
	OverdueCount      int64   `json:"overdue_count"`
	AvgResponseTime   float64 `json:"avg_response_time"`
	AvgProcessingTime float64 `json:"avg_processing_time"`
	SatisfactionScore float64 `json:"satisfaction_score"`
}

// ==================== 实体表定义（用于统计） ====================

// WorkOrderStatistics 工单统计实体（DAO层）
type WorkOrderStatistics struct {
	ID              int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Date            time.Time `json:"date" gorm:"column:date;not null;index;comment:统计日期"`
	TotalCount      int       `json:"total_count" gorm:"column:total_count;not null;default:0;comment:工单总数"`
	CompletedCount  int       `json:"completed_count" gorm:"column:completed_count;not null;default:0;comment:已完成工单数"`
	ProcessingCount int       `json:"processing_count" gorm:"column:processing_count;not null;default:0;comment:处理中工单数"`
	PendingCount    int       `json:"pending_count" gorm:"column:pending_count;not null;default:0;comment:待处理工单数"`
	CanceledCount   int       `json:"canceled_count" gorm:"column:canceled_count;not null;default:0;comment:已取消工单数"`
	RejectedCount   int       `json:"rejected_count" gorm:"column:rejected_count;not null;default:0;comment:已拒绝工单数"`
	OverdueCount    int       `json:"overdue_count" gorm:"column:overdue_count;not null;default:0;comment:超时工单数"`
	AvgProcessTime  float64   `json:"avg_process_time" gorm:"column:avg_process_time;not null;default:0;comment:平均处理时间(小时)"`
	AvgResponseTime float64   `json:"avg_response_time" gorm:"column:avg_response_time;not null;default:0;comment:平均响应时间(小时)"`
	CategoryStats   string    `json:"category_stats" gorm:"column:category_stats;type:json;comment:分类统计JSON"`
	UserStats       string    `json:"user_stats" gorm:"column:user_stats;type:json;comment:用户统计JSON"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
}

func (WorkOrderStatistics) TableName() string {
	return "work_order_statistics"
}

// UserPerformance 用户绩效实体（DAO层）
type UserPerformance struct {
	ID                int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	UserID            int       `json:"user_id" gorm:"column:user_id;not null;index;comment:用户ID"`
	UserName          string    `json:"user_name" gorm:"column:user_name;not null;comment:用户姓名"`
	Date              time.Time `json:"date" gorm:"column:date;not null;index;comment:统计日期"`
	AssignedCount     int       `json:"assigned_count" gorm:"column:assigned_count;not null;default:0;comment:分配工单数"`
	CompletedCount    int       `json:"completed_count" gorm:"column:completed_count;not null;default:0;comment:完成工单数"`
	OverdueCount      int       `json:"overdue_count" gorm:"column:overdue_count;not null;default:0;comment:超时工单数"`
	AvgResponseTime   float64   `json:"avg_response_time" gorm:"column:avg_response_time;not null;default:0;comment:平均响应时间(小时)"`
	AvgProcessingTime float64   `json:"avg_processing_time" gorm:"column:avg_processing_time;not null;default:0;comment:平均处理时间(小时)"`
	SatisfactionScore float64   `json:"satisfaction_score" gorm:"column:satisfaction_score;default:0;comment:满意度评分"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
}

func (UserPerformance) TableName() string {
	return "user_performance"
}
