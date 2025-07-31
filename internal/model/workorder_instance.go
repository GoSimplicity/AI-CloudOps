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
	"time"
)

// 工单状态常量
const (
	InstanceStatusDraft      int8 = 1 // 草稿
	InstanceStatusPending    int8 = 2 // 待处理
	InstanceStatusProcessing int8 = 3 // 处理中
	InstanceStatusCompleted  int8 = 4 // 已完成
	InstanceStatusRejected   int8 = 5 // 已拒绝
	InstanceStatusCancelled  int8 = 6 // 已取消
)

// 优先级常量
const (
	PriorityLow    int8 = 1 // 低
	PriorityNormal int8 = 2 // 普通
	PriorityHigh   int8 = 3 // 高
)

// WorkorderInstance 工单实例实体
type WorkorderInstance struct {
	Model
	Title        string     `json:"title" gorm:"type:varchar(200);not null;index;comment:工单标题"`
	SerialNumber string     `json:"serial_number" gorm:"type:varchar(50);not null;uniqueIndex;comment:工单编号"`
	ProcessID    int        `json:"process_id" gorm:"not null;index;comment:流程ID"`
	FormData     JSONMap    `json:"form_data" gorm:"type:json;comment:表单数据"`
	Status       int8       `json:"status" gorm:"not null;default:1;index;comment:状态"`
	Priority     int8       `json:"priority" gorm:"not null;default:2;index;comment:优先级"`
	CreatorID    int        `json:"creator_id" gorm:"not null;index;comment:创建人ID"`
	AssigneeID   *int       `json:"assignee_id" gorm:"index;comment:当前处理人ID"`
	Description  string     `json:"description" gorm:"type:text;comment:详细描述"`
	Tags         StringList `json:"tags" gorm:"column:tags;comment:标签"`
	DueDate      *time.Time `json:"due_date" gorm:"index;comment:截止时间"`
	CompletedAt  *time.Time `json:"completed_at" gorm:"comment:完成时间"`
}

// TableName 指定工单实例表名
func (WorkorderInstance) TableName() string {
	return "cl_workorder_instance"
}

// CreateWorkorderInstanceReq 创建工单实例请求
type CreateWorkorderInstanceReq struct {
	Title       string     `json:"title" binding:"required,min=1,max=200"`
	ProcessID   int        `json:"process_id" binding:"required,min=1"`
	FormData    JSONMap    `json:"form_data" binding:"required"`
	Description string     `json:"description" binding:"omitempty,max=2000"`
	Priority    int8       `json:"priority" binding:"omitempty,oneof=1 2 3"`
	AssigneeID  *int       `json:"assignee_id" binding:"omitempty,min=1"`
	Tags        StringList `json:"tags" binding:"omitempty"`
	DueDate     *time.Time `json:"due_date" binding:"omitempty"`
}

// UpdateWorkorderInstanceReq 更新工单实例请求
type UpdateWorkorderInstanceReq struct {
	ID          int        `json:"id" binding:"required,min=1"`
	Title       string     `json:"title" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"omitempty,max=2000"`
	Priority    int8       `json:"priority" binding:"omitempty,oneof=1 2 3"`
	Tags        StringList `json:"tags" binding:"omitempty"`
	DueDate     *time.Time `json:"due_date" binding:"omitempty"`
}

// DeleteWorkorderInstanceReq 删除工单实例请求
type DeleteWorkorderInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderInstanceReq 获取工单实例详情请求
type DetailWorkorderInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderInstanceReq 工单实例列表请求
type ListWorkorderInstanceReq struct {
	ListReq
	Status     *int8      `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	Priority   *int8      `json:"priority" form:"priority" binding:"omitempty,oneof=1 2 3"`
	CreatorID  *int       `json:"creator_id" form:"creator_id" binding:"omitempty,min=1"`
	AssigneeID *int       `json:"assignee_id" form:"assignee_id" binding:"omitempty,min=1"`
	ProcessID  *int       `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
	StartDate  *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate    *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
}
