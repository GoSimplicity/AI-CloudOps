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

// 时间线操作类型常量 - 包含所有操作记录
const (
	TimelineActionCreate   = "create"   // 创建工单
	TimelineActionSubmit   = "submit"   // 提交工单
	TimelineActionApprove  = "approve"  // 审批通过
	TimelineActionReject   = "reject"   // 审批拒绝
	TimelineActionAssign   = "assign"   // 指派处理人
	TimelineActionCancel   = "cancel"   // 取消工单
	TimelineActionComplete = "complete" // 完成工单
	TimelineActionReturn   = "return"   // 退回工单
	TimelineActionComment  = "comment"  // 添加评论
	TimelineActionUpdate   = "update"   // 更新工单信息
	TimelineActionView     = "view"     // 查看工单
	TimelineActionAttach   = "attach"   // 添加附件
	TimelineActionNotify   = "notify"   // 发送通知
	TimelineActionRemind   = "remind"   // 催办提醒
)

// WorkorderInstanceTimeline 工单操作时间线 - 记录所有操作历史和审计日志
type WorkorderInstanceTimeline struct {
	Model
	InstanceID   int    `json:"instance_id" gorm:"column:instance_id;not null;index:idx_instance_id;index:idx_instance_time,priority:1;index:idx_instance_action,priority:1;comment:工单实例ID"`
	Action       string `json:"action" gorm:"column:action;type:varchar(50);not null;index:idx_action;index:idx_instance_action,priority:2;comment:操作类型"`
	OperatorID   int    `json:"operator_id" gorm:"column:operator_id;not null;index:idx_operator_id;index:idx_operator_time,priority:1;comment:操作人ID"`
	OperatorName string `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人名称"`
	ActionDetail string `json:"action_detail" gorm:"column:action_detail;type:text;comment:操作详情（JSON格式）"`
	Comment      string `json:"comment" gorm:"column:comment;type:varchar(2000);comment:操作备注或说明"`
	RelatedID    *int   `json:"related_id" gorm:"column:related_id;index;comment:关联记录ID（如评论ID、附件ID等）"`
}

func (WorkorderInstanceTimeline) TableName() string {
	return "cl_workorder_instance_timeline"
}

// CreateWorkorderInstanceTimelineReq 创建工单操作时间线请求
type CreateWorkorderInstanceTimelineReq struct {
	InstanceID   int    `json:"instance_id" binding:"required,min=1"`
	Action       string `json:"action" binding:"required,oneof=create submit approve reject assign cancel complete return comment update view attach notify remind"`
	OperatorID   int    `json:"operator_id" binding:"required,min=1"`
	OperatorName string `json:"operator_name" binding:"required,min=1,max=100"`
	ActionDetail string `json:"action_detail" binding:"omitempty"`
	Comment      string `json:"comment" binding:"omitempty,max=2000"`
	RelatedID    *int   `json:"related_id" binding:"omitempty,min=1"`
}

// UpdateWorkorderInstanceTimelineReq 更新工单操作时间线请求
type UpdateWorkorderInstanceTimelineReq struct {
	ID           int    `json:"id" binding:"required,min=1"`
	ActionDetail string `json:"action_detail" binding:"omitempty"`
	Comment      string `json:"comment" binding:"omitempty,max=2000"`
}

// DeleteWorkorderInstanceTimelineReq 删除工单实例时间线请求
type DeleteWorkorderInstanceTimelineReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderInstanceTimelineReq 获取工单实例时间线详情请求
type DetailWorkorderInstanceTimelineReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderInstanceTimelineReq 工单操作时间线列表请求
type ListWorkorderInstanceTimelineReq struct {
	ListReq
	InstanceID *int       `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	Action     *string    `json:"action" form:"action" binding:"omitempty,oneof=create submit approve reject assign cancel complete return comment update view attach notify remind"`
	StartDate  *time.Time `json:"start_date" form:"start_date" binding:"omitempty"`
	EndDate    *time.Time `json:"end_date" form:"end_date" binding:"omitempty"`
}
