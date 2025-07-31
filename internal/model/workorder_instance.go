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
	InstanceStatusOverdue    int8 = 7 // 已超时
	InstanceStatusSuspended  int8 = 8 // 已暂停
)

// 优先级常量
const (
	PriorityLow      int8 = 1 // 低
	PriorityNormal   int8 = 2 // 普通
	PriorityHigh     int8 = 3 // 高
	PriorityUrgent   int8 = 4 // 紧急
	PriorityCritical int8 = 5 // 严重
)

// 紧急程度常量
const (
	UrgencyLow    int8 = 1 // 低
	UrgencyMedium int8 = 2 // 中
	UrgencyHigh   int8 = 3 // 高
)

// 影响范围常量
const (
	ImpactLow    int8 = 1 // 低
	ImpactMedium int8 = 2 // 中
	ImpactHigh   int8 = 3 // 高
)

// WorkorderInstance 工单实例实体
type WorkorderInstance struct {
	Model
	Title           string     `json:"title" gorm:"column:title;type:varchar(500);not null;index;comment:工单标题"`
	SerialNumber    string     `json:"serial_number" gorm:"column:serial_number;type:varchar(50);not null;uniqueIndex;comment:工单编号"`
	TemplateID      *int       `json:"template_id" gorm:"column:template_id;index;comment:模板ID"`
	ProcessID       int        `json:"process_id" gorm:"column:process_id;not null;index;comment:流程ID"`
	FormDesignID    int        `json:"form_design_id" gorm:"column:form_design_id;not null;index;comment:表单设计ID"`
	FormData        JSONMap    `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	CurrentStepID   string     `json:"current_step_id" gorm:"column:current_step_id;type:varchar(100);not null;index;comment:当前步骤ID"`
	CurrentStepName string     `json:"current_step_name" gorm:"column:current_step_name;type:varchar(200);not null;comment:当前步骤名称"`
	Status          int8       `json:"status" gorm:"column:status;not null;default:1;index;index:idx_assignee_status,priority:2;comment:状态"`
	Priority        int8       `json:"priority" gorm:"column:priority;not null;default:2;index;comment:优先级"`
	Urgency         int8       `json:"urgency" gorm:"column:urgency;not null;default:2;comment:紧急程度"`
	Impact          int8       `json:"impact" gorm:"column:impact;not null;default:2;comment:影响范围"`
	CategoryID      *int       `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	CreatorID       int        `json:"creator_id" gorm:"column:creator_id;not null;index;comment:创建人ID"`
	CreatorName     string     `json:"creator_name" gorm:"-"`
	AssigneeID      *int       `json:"assignee_id" gorm:"column:assignee_id;index;index:idx_assignee_status,priority:1;comment:当前处理人ID"`
	AssigneeName    string     `json:"assignee_name" gorm:"-"`
	Description     string     `json:"description" gorm:"column:description;type:text;comment:详细描述"`
	Solution        string     `json:"solution" gorm:"column:solution;type:text;comment:解决方案"`
	Tags            StringList `json:"tags" gorm:"column:tags;comment:标签"`
	DueDate         *time.Time `json:"due_date" gorm:"column:due_date;index;comment:截止时间"`
	StartedAt       *time.Time `json:"started_at" gorm:"column:started_at;comment:开始处理时间"`
	CompletedAt     *time.Time `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`
	EstimatedHours  *float64   `json:"estimated_hours" gorm:"column:estimated_hours;comment:预估工时"`
	ActualHours     *float64   `json:"actual_hours" gorm:"column:actual_hours;comment:实际工时"`
	ProcessData     JSONMap    `json:"process_data" gorm:"column:process_data;type:json;comment:流程数据"`
	ExtendedFields  JSONMap    `json:"extended_fields" gorm:"column:extended_fields;type:json;comment:扩展字段"`
	Source          string     `json:"source" gorm:"column:source;type:varchar(50);not null;default:'web';comment:来源"`
	SourceID        string     `json:"source_id" gorm:"column:source_id;type:varchar(100);comment:来源ID"`

	// 关联信息（不存储到数据库）
	TemplateName   string `json:"template_name,omitempty" gorm:"-"`
	ProcessName    string `json:"process_name,omitempty" gorm:"-"`
	FormDesignName string `json:"form_design_name,omitempty" gorm:"-"`
	CategoryName   string `json:"category_name,omitempty" gorm:"-"`
	Duration       int64  `json:"duration,omitempty" gorm:"-"` // 处理时长(秒)
}

// TableName 指定工单实例表名
func (WorkorderInstance) TableName() string {
	return "cl_workorder_instance"
}

// CreateWorkorderInstanceReq 创建工单实例请求
type CreateWorkorderInstanceReq struct {
	Title          string                 `json:"title" binding:"required,min=1,max=500"`
	TemplateID     *int                   `json:"template_id" binding:"omitempty,min=1"`
	ProcessID      int                    `json:"process_id" binding:"required,min=1"`
	FormDesignID   int                    `json:"form_design_id" binding:"required,min=1"`
	FormData       map[string]interface{} `json:"form_data" binding:"required"`
	Description    string                 `json:"description" binding:"omitempty,max=5000"`
	Priority       int8                   `json:"priority" binding:"omitempty,oneof=1 2 3 4 5"`
	Urgency        int8                   `json:"urgency" binding:"omitempty,oneof=1 2 3"`
	Impact         int8                   `json:"impact" binding:"omitempty,oneof=1 2 3"`
	CategoryID     *int                   `json:"category_id" binding:"omitempty,min=1"`
	AssigneeID     *int                   `json:"assignee_id" binding:"omitempty,min=1"`
	Tags           []string               `json:"tags" binding:"omitempty"`
	DueDate        *time.Time             `json:"due_date"`
	EstimatedHours *float64               `json:"estimated_hours" binding:"omitempty,min=0"`
	ExtendedFields map[string]interface{} `json:"extended_fields"`
	Source         string                 `json:"source" binding:"omitempty,max=50"`
	SourceID       string                 `json:"source_id" binding:"omitempty,max=100"`
}

// UpdateWorkorderInstanceReq 更新工单实例请求
type UpdateWorkorderInstanceReq struct {
	ID             int                    `json:"id" binding:"required,min=1"`
	Title          string                 `json:"title" binding:"required,min=1,max=500"`
	Description    string                 `json:"description" binding:"omitempty,max=5000"`
	Priority       int8                   `json:"priority" binding:"omitempty,oneof=1 2 3 4 5"`
	Urgency        int8                   `json:"urgency" binding:"omitempty,oneof=1 2 3"`
	Impact         int8                   `json:"impact" binding:"omitempty,oneof=1 2 3"`
	CategoryID     *int                   `json:"category_id" binding:"omitempty,min=1"`
	Tags           []string               `json:"tags" binding:"omitempty"`
	DueDate        *time.Time             `json:"due_date"`
	EstimatedHours *float64               `json:"estimated_hours" binding:"omitempty,min=0"`
	ActualHours    *float64               `json:"actual_hours" binding:"omitempty,min=0"`
	Solution       string                 `json:"solution" binding:"omitempty,max=5000"`
	ExtendedFields map[string]interface{} `json:"extended_fields"`
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
	Status       *int8      `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7 8"`
	Priority     *int8      `json:"priority" form:"priority" binding:"omitempty,oneof=1 2 3 4 5"`
	Urgency      *int8      `json:"urgency" form:"urgency" binding:"omitempty,oneof=1 2 3"`
	Impact       *int8      `json:"impact" form:"impact" binding:"omitempty,oneof=1 2 3"`
	CategoryID   *int       `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	CreatorID    *int       `json:"creator_id" form:"creator_id" binding:"omitempty,min=1"`
	AssigneeID   *int       `json:"assignee_id" form:"assignee_id" binding:"omitempty,min=1"`
	ProcessID    *int       `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
	TemplateID   *int       `json:"template_id" form:"template_id" binding:"omitempty,min=1"`
	StartDate    *time.Time `json:"start_date" form:"start_date"`
	EndDate      *time.Time `json:"end_date" form:"end_date"`
	Tags         []string   `json:"tags" form:"tags"`
	Overdue      *bool      `json:"overdue" form:"overdue"`
	Source       *string    `json:"source" form:"source"`
	CurrentStep  *string    `json:"current_step" form:"current_step"`
}

// MyWorkorderInstanceReq 我的工单实例请求
type MyWorkorderInstanceReq struct {
	ListReq
	Type       string     `json:"type" form:"type" binding:"omitempty,oneof=created assigned participated all"`
	Status     *int8      `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7 8"`
	Priority   *int8      `json:"priority" form:"priority" binding:"omitempty,oneof=1 2 3 4 5"`
	CategoryID *int       `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	ProcessID  *int       `json:"process_id" form:"process_id" binding:"omitempty,min=1"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
}

// TransferWorkorderInstanceReq 转交工单实例请求
type TransferWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	AssigneeID int    `json:"assignee_id" binding:"required,min=1"`
	Comment    string `json:"comment" binding:"omitempty,max=1000"`
	Reason     string `json:"reason" binding:"omitempty,max=500"`
}

// AssignWorkorderInstanceReq 分配工单实例请求
type AssignWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	AssigneeID int    `json:"assignee_id" binding:"required,min=1"`
	Comment    string `json:"comment" binding:"omitempty,max=1000"`
}

// ReopenWorkorderInstanceReq 重新打开工单实例请求
type ReopenWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"required,min=1,max=1000"`
}

// SuspendWorkorderInstanceReq 暂停工单实例请求
type SuspendWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"required,min=1,max=1000"`
	Duration   *int   `json:"duration" binding:"omitempty,min=1"` // 暂停时长(小时)
}

// ResumeWorkorderInstanceReq 恢复工单实例请求
type ResumeWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	Comment    string `json:"comment" binding:"omitempty,max=1000"`
}

// CancelWorkorderInstanceReq 取消工单实例请求
type CancelWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"required,min=1,max=1000"`
}

// BatchUpdateInstanceStatusReq 批量更新工单实例状态请求
type BatchUpdateInstanceStatusReq struct {
	IDs    []int  `json:"ids" binding:"required,min=1,dive,min=1"`
	Status int8   `json:"status" binding:"required,oneof=1 2 3 4 5 6 7 8"`
	Reason string `json:"reason" binding:"omitempty,max=1000"`
}

// BatchAssignInstanceReq 批量分配工单实例请求
type BatchAssignInstanceReq struct {
	IDs        []int  `json:"ids" binding:"required,min=1,dive,min=1"`
	AssigneeID int    `json:"assignee_id" binding:"required,min=1"`
	Comment    string `json:"comment" binding:"omitempty,max=1000"`
}

// UpdateInstanceProgressReq 更新工单实例进度请求
type UpdateInstanceProgressReq struct {
	InstanceID   int      `json:"instance_id" binding:"required,min=1"`
	ActualHours  *float64 `json:"actual_hours" binding:"omitempty,min=0"`
	Progress     int      `json:"progress" binding:"omitempty,min=0,max=100"`
	ProgressNote string   `json:"progress_note" binding:"omitempty,max=1000"`
}

// GetInstanceTimelineReq 获取工单实例时间线请求
type GetInstanceTimelineReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ExportInstanceReq 导出工单实例请求
type ExportInstanceReq struct {
	IDs        []int  `json:"ids" binding:"omitempty,dive,min=1"`
	Status     *int8  `json:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7 8"`
	CategoryID *int   `json:"category_id" binding:"omitempty,min=1"`
	StartDate  *int64 `json:"start_date" binding:"omitempty,min=0"`
	EndDate    *int64 `json:"end_date" binding:"omitempty,min=0"`
	Format     string `json:"format" binding:"required,oneof=excel csv pdf"`
}

// WorkorderInstanceStatistics 工单实例统计
type WorkorderInstanceStatistics struct {
	TotalCount      int64 `json:"total_count"`      // 总数量
	DraftCount      int64 `json:"draft_count"`      // 草稿数量
	PendingCount    int64 `json:"pending_count"`    // 待处理数量
	ProcessingCount int64 `json:"processing_count"` // 处理中数量
	CompletedCount  int64 `json:"completed_count"`  // 已完成数量
	RejectedCount   int64 `json:"rejected_count"`   // 已拒绝数量
	CancelledCount  int64 `json:"cancelled_count"`  // 已取消数量
	OverdueCount    int64 `json:"overdue_count"`    // 已超时数量
	SuspendedCount  int64 `json:"suspended_count"`  // 已暂停数量
	AvgProcessTime  int64 `json:"avg_process_time"` // 平均处理时间(秒)
	AvgResponseTime int64 `json:"avg_response_time"` // 平均响应时间(秒)
}

// WorkorderInstanceTimeline 工单实例时间线
type WorkorderInstanceTimeline struct {
	ID          int                    `json:"id"`
	InstanceID  int                    `json:"instance_id"`
	Action      string                 `json:"action"`
	ActionName  string                 `json:"action_name"`
	OperatorID  int                    `json:"operator_id"`
	OperatorName string                `json:"operator_name"`
	Comment     string                 `json:"comment"`
	FormData    map[string]interface{} `json:"form_data"`
	StepID      string                 `json:"step_id"`
	StepName    string                 `json:"step_name"`
	FromStepID  string                 `json:"from_step_id"`
	ToStepID    string                 `json:"to_step_id"`
	CreatedAt   time.Time              `json:"created_at"`
}
