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
	"time"
)

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

// Value 实现driver.Valuer接口，将JSONMap转为JSON字符串存储到数据库
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 实现sql.Scanner接口，从数据库读取JSON字符串并转为JSONMap
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("无法扫描 %T 到 JSONMap", value)
	}

	return json.Unmarshal(data, m)
}

// Instance 工单实例
type Instance struct {
	Model
	Title        string     `json:"title" gorm:"column:title;not null;comment:工单标题"`
	TemplateID   *int       `json:"template_id" gorm:"column:template_id;comment:模板ID"`
	ProcessID    int        `json:"process_id" gorm:"column:process_id;not null;comment:流程ID"`
	FormData     JSONMap    `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	CurrentStep  string     `json:"current_step" gorm:"column:current_step;not null;comment:当前步骤"`
	Status       int8       `json:"status" gorm:"column:status;not null;comment:状态"`
	Priority     int8       `json:"priority" gorm:"column:priority;default:1;comment:优先级"`
	CategoryID   *int       `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int        `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	Description  string     `json:"description" gorm:"column:description;comment:描述"`
	CreatorName  string     `json:"creator_name" gorm:"-"`
	AssigneeID   *int       `json:"assignee_id" gorm:"column:assignee_id;comment:当前处理人ID"`
	AssigneeName string     `json:"assignee_name" gorm:"-"`
	CompletedAt  *time.Time `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`
	DueDate      *time.Time `json:"due_date" gorm:"column:due_date;comment:截止时间"`
	Tags         StringList `json:"tags" gorm:"column:tags;comment:标签"`
	ProcessData  JSONMap    `json:"process_data" gorm:"column:process_data;type:json;comment:流程数据"`

	// 关联数据
	Template *Template `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Process  *Process  `json:"process,omitempty" gorm:"foreignKey:ProcessID"`
	Category *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}

func (Instance) TableName() string {
	return "cl_workorder_instances"
}

// 工单实例请求结构
type CreateInstanceReq struct {
	Title       string     `json:"title" binding:"required,min=1,max=200"`       // 工单标题
	TemplateID  *int       `json:"template_id"`                                  // 模板ID
	ProcessID   int        `json:"process_id" binding:"required"`                // 流程ID
	Description string     `json:"description" binding:"omitempty,max=1000"`     // 描述
	Priority    int8       `json:"priority" binding:"omitempty,oneof=0 1 2 3 4"` // 优先级
	CategoryID  *int       `json:"category_id"`                                  // 分类ID
	DueDate     *time.Time `json:"due_date"`                                     // 截止时间
	Tags        []string   `json:"tags"`                                         // 标签
	AssigneeID  *int       `json:"assignee_id"`                                  // 处理人ID
}

type UpdateInstanceReq struct {
	ID          int        `json:"id" form:"id" binding:"required"`
	Title       string     `json:"title" form:"title" binding:"required,min=1,max=200"`
	Description string     `json:"description" form:"description" binding:"omitempty,max=1000"`
	Priority    int8       `json:"priority" form:"priority" binding:"omitempty,oneof=0 1 2 3 4"`
	CategoryID  *int       `json:"category_id" form:"category_id"`
	DueDate     *time.Time `json:"due_date" form:"due_date"`
	Tags        []string   `json:"tags" form:"tags"`
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
	Type       string     `json:"type" form:"type" binding:"omitempty,oneof=created assigned all"`
	Status     *int8      `json:"status" form:"status"`
	Priority   *int8      `json:"priority" form:"priority"`
	CategoryID *int       `json:"category_id" form:"category_id"`
	ProcessID  *int       `json:"process_id" form:"process_id"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
}

type TransferInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required"`
	AssigneeID int    `json:"assignee_id" binding:"required"`
	Comment    string `json:"comment"`
}
