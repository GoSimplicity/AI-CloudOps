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

import "time"

// 工单状态
const (
	InstanceStatusDraft      int8 = 1 // 草稿
	InstanceStatusPending    int8 = 2 // 待处理
	InstanceStatusProcessing int8 = 3 // 处理中
	InstanceStatusCompleted  int8 = 4 // 已完成
	InstanceStatusRejected   int8 = 5 // 已拒绝
	InstanceStatusCancelled  int8 = 6 // 已取消
)

// 工单优先级
const (
	PriorityLow    int8 = 1 // 低
	PriorityNormal int8 = 2 // 普通
	PriorityHigh   int8 = 3 // 高
)

// WorkorderInstance 工单实例
type WorkorderInstance struct {
	Model
	Title          string     `json:"title" gorm:"column:title;type:varchar(200);not null;index;comment:工单标题"`
	SerialNumber   string     `json:"serial_number" gorm:"column:serial_number;type:varchar(50);not null;uniqueIndex;comment:工单编号"`
	ProcessID      int        `json:"process_id" gorm:"column:process_id;not null;index;comment:流程ID"`
	FormData       JSONMap    `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	Status         int8       `json:"status" gorm:"column:status;not null;default:1;index;comment:状态"`
	Priority       int8       `json:"priority" gorm:"column:priority;not null;default:2;index;comment:优先级"`  
	OperatorID     int        `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	OperatorName   string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);not null;comment:操作人名称"`
	AssigneeID     *int       `json:"assignee_id" gorm:"column:assignee_id;index;comment:当前处理人ID"`
	Description    string     `json:"description" gorm:"column:description;type:text;comment:详细描述"`
	Tags           StringList `json:"tags" gorm:"column:tags;comment:标签"`
	DueDate        *time.Time `json:"due_date" gorm:"column:due_date;index;comment:截止时间"`
	CompletedAt    *time.Time `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`

	// 关联查询字段
	Comments []WorkorderInstanceComment  `json:"comments,omitempty" gorm:"foreignKey:InstanceID;references:ID"`
	FlowLogs []WorkorderInstanceFlow     `json:"flow_logs,omitempty" gorm:"foreignKey:InstanceID;references:ID"`
	Timeline []WorkorderInstanceTimeline `json:"timeline,omitempty" gorm:"foreignKey:InstanceID;references:ID"`
}

func (WorkorderInstance) TableName() string {
	return "cl_workorder_instance"
}

// 创建工单实例请求
type CreateWorkorderInstanceReq struct {
	Title          string     `json:"title" binding:"required,min=1,max=200"`
	SerialNumber   string     `json:"serial_number" binding:"required,min=1,max=50"`
	ProcessID      int        `json:"process_id" binding:"required,min=1"`
	FormData       JSONMap    `json:"form_data" binding:"required"`
	Status         int8       `json:"status" binding:"required,oneof=1 2 3 4 5 6"`
	Priority       int8       `json:"priority" binding:"required,oneof=1 2 3"`
	OperatorID     int        `json:"operator_id" binding:"required,min=1"`
	OperatorName   string     `json:"operator_name" binding:"required,min=1,max=100"`
	AssigneeID     *int       `json:"assignee_id" binding:"omitempty,min=1"`
	Description    string     `json:"description" binding:"omitempty,max=2000"`
	Tags           StringList `json:"tags" binding:"omitempty"`
	DueDate        *time.Time `json:"due_date" binding:"omitempty"`
}

// 更新工单实例请求
type UpdateWorkorderInstanceReq struct {
	ID          int        `json:"id" binding:"required,min=1"`
	Title       string     `json:"title" binding:"omitempty,min=1,max=200"`
	Description string     `json:"description" binding:"omitempty,max=2000"`
	Priority    int8       `json:"priority" binding:"omitempty,oneof=1 2 3"`
	Tags        StringList `json:"tags" binding:"omitempty"`
	DueDate     *time.Time `json:"due_date" binding:"omitempty"`
	Status      int8       `json:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	AssigneeID  *int       `json:"assignee_id" binding:"omitempty,min=1"`
	FormData    JSONMap    `json:"form_data" binding:"omitempty"`
	CompletedAt *time.Time `json:"completed_at" binding:"omitempty"`
}

// 删除工单实例请求
type DeleteWorkorderInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// 工单实例详情请求
type DetailWorkorderInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// 工单实例列表请求
type ListWorkorderInstanceReq struct {
	ListReq
	Status    *int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Priority  *int8 `json:"priority" form:"priority" binding:"omitempty,oneof=1 2 3"`
	ProcessID *int  `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
}
