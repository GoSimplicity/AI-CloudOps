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

// 流程状态常量
const (
	ProcessStatusDraft     int8 = 1 // 草稿
	ProcessStatusPublished int8 = 2 // 已发布
	ProcessStatusDisabled  int8 = 3 // 已禁用
)

// 步骤类型常量
const (
	StepTypeStart    = "start"    // 开始节点
	StepTypeApproval = "approval" // 审批节点
	StepTypeEnd      = "end"      // 结束节点
	StepTypeTask     = "task"     // 任务节点
	StepTypeDecision = "decision" // 决策节点
)

// 操作类型常量
const (
	ActionApprove  = "approve"  // 同意
	ActionReject   = "reject"   // 拒绝
	ActionTransfer = "transfer" // 转交
	ActionRevoke   = "revoke"   // 撤回
	ActionCancel   = "cancel"   // 取消
)

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

// Process 流程实体
type Process struct {
	Model
	Name         string         `json:"name" gorm:"column:name;not null;comment:流程名称"`
	Description  string         `json:"description" gorm:"column:description;comment:流程描述"`
	FormDesignID int            `json:"form_design_id" gorm:"column:form_design_id;not null;comment:关联的表单设计ID"`
	Definition   datatypes.JSON `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version      string         `json:"version" gorm:"column:version;not null;comment:版本号"`
	Status       int8           `json:"status" gorm:"column:status;not null;default:1;comment:状态：1-草稿，2-已发布，3-已禁用"`
	CategoryID   *int           `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int            `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName  string         `json:"creator_name" gorm:"-"`
	FormDesign   *FormDesign    `json:"form_design" gorm:"foreignKey:FormDesignID"`
	Category     *Category      `json:"category" gorm:"foreignKey:CategoryID"`
}

// TableName 指定流程表名
func (Process) TableName() string {
	return "workorder_process"
}

// CreateProcessReq 创建流程请求
type CreateProcessReq struct {
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"omitempty,max=500"`
	FormDesignID int               `json:"form_design_id" binding:"required"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id"`
	CreatorID    int               `json:"creator_id" binding:"required"`
	CreatorName  string            `json:"creator_name" binding:"required"`
	Version      string            `json:"version" binding:"required"`
}

// UpdateProcessReq 更新流程请求
type UpdateProcessReq struct {
	ID           int               `json:"id" binding:"required"`
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"omitempty,max=500"`
	FormDesignID int               `json:"form_design_id" binding:"required"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id"`
	Version      string            `json:"version" binding:"required"`
	Status       int8              `json:"status" binding:"required"`
}

// DeleteProcessReq 删除流程请求
type DeleteProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// DetailProcessReq 流程详情请求
type DetailProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// GetProcessWithRelationsReq 获取流程及关联信息请求
type GetProcessWithRelationsReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

// ListProcessReq 流程列表请求
type ListProcessReq struct {
	ListReq
	CategoryID   *int  `json:"category_id" form:"category_id"`
	FormDesignID *int  `json:"form_design_id" form:"form_design_id"`
	Status       *int8 `json:"status" form:"status"`
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

// ValidateProcessReq 验证流程请求
type ValidateProcessReq struct {
	ID int `json:"id" binding:"required"`
}
