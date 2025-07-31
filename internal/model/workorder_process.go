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
	ProcessStatusArchived  int8 = 3 // 已归档
)

// 流程步骤类型常量
const (
	ProcessStepTypeStart    = "start"    // 开始
	ProcessStepTypeApproval = "approval" // 审批
	ProcessStepTypeTask     = "task"     // 任务
	ProcessStepTypeEnd      = "end"      // 结束
)

// WorkorderProcess 工单流程实体
type WorkorderProcess struct {
	Model
	Name           string         `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:流程名称"`
	Description    string         `json:"description" gorm:"column:description;type:varchar(1000);comment:流程描述"`
	FormDesignID   int            `json:"form_design_id" gorm:"column:form_design_id;not null;index;comment:关联表单设计ID"`
	Definition     datatypes.JSON `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Status         int8           `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-草稿，2-已发布，3-已归档"`
	CategoryID     *int           `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	CreateUserID   int            `json:"create_user_id" gorm:"column:create_user_id;not null;index;comment:创建人ID"`
	CreateUserName string         `json:"create_user_name" gorm:"column:create_user_name;type:varchar(100);not null;index;comment:创建人名称"`
	Tags           StringList     `json:"tags" gorm:"column:tags;comment:标签"`
	IsDefault      int8           `json:"is_default" gorm:"column:is_default;not null;default:2;comment:是否为默认流程：1-是，2-否"`
}

// TableName 指定工单流程表名
func (WorkorderProcess) TableName() string {
	return "cl_workorder_process"
}

// ProcessStep 流程步骤定义
type ProcessStep struct {
	ID           string   `json:"id"`                      // 步骤唯一标识
	Type         string   `json:"type" binding:"required"` // 步骤类型
	Name         string   `json:"name" binding:"required"` // 步骤名称
	AssigneeType string   `json:"assignee_type"`           // 受理人类型
	AssigneeIDs  []int    `json:"assignee_ids,omitempty"`  // 受理人ID列表
	Actions      []string `json:"actions,omitempty"`       // 可执行动作
	SortOrder    int      `json:"sort_order"`              // 排序
}

// ProcessConnection 流程连接定义
type ProcessConnection struct {
	From string `json:"from"` // 来源步骤ID
	To   string `json:"to"`   // 目标步骤ID
}

// ProcessDefinition 流程定义
type ProcessDefinition struct {
	Steps       []ProcessStep       `json:"steps" binding:"required"`       // 步骤列表
	Connections []ProcessConnection `json:"connections" binding:"required"` // 连接列表
}

// CreateWorkorderProcessReq 创建工单流程请求
type CreateWorkorderProcessReq struct {
	Name           string            `json:"name" binding:"required,min=1,max=200"`
	Description    string            `json:"description" binding:"omitempty,max=1000"`
	FormDesignID   int               `json:"form_design_id" binding:"required,min=1"`
	Definition     ProcessDefinition `json:"definition" binding:"required"`
	Status         int8              `json:"status" binding:"required,oneof=1 2 3"`
	CategoryID     *int              `json:"category_id" binding:"omitempty,min=1"`
	CreateUserID   int               `json:"create_user_id" binding:"required,min=1"`
	CreateUserName string            `json:"create_user_name" binding:"required,min=1,max=100"`
	Tags           StringList        `json:"tags" binding:"omitempty"`
	IsDefault      int8              `json:"is_default" binding:"required,oneof=1 2"`
}

// UpdateWorkorderProcessReq 更新工单流程请求
type UpdateWorkorderProcessReq struct {
	ID           int               `json:"id" binding:"required,min=1"`
	Name         string            `json:"name" binding:"required,min=1,max=200"`
	Description  string            `json:"description" binding:"omitempty,max=1000"`
	FormDesignID int               `json:"form_design_id" binding:"required,min=1"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	Status       int8              `json:"status" binding:"required,oneof=1 2 3"`
	CategoryID   *int              `json:"category_id" binding:"omitempty,min=1"`
	Tags         StringList        `json:"tags" binding:"omitempty"`
	IsDefault    int8              `json:"is_default" binding:"required,oneof=1 2"`
}

// DeleteWorkorderProcessReq 删除工单流程请求
type DeleteWorkorderProcessReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderProcessReq 获取工单流程详情请求
type DetailWorkorderProcessReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderProcessReq 工单流程列表请求
type ListWorkorderProcessReq struct {
	ListReq
	CategoryID   *int  `json:"category_id" binding:"omitempty,min=1"`
	FormDesignID *int  `json:"form_design_id" binding:"omitempty,min=1"`
	Status       *int8 `json:"status" binding:"omitempty,oneof=1 2 3"`
	IsDefault    *int8 `json:"is_default" binding:"omitempty,oneof=1 2"`
}
