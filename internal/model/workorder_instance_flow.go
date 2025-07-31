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

// 工单流转动作类型常量
const (
	FlowActionCreate   = "create"   // 创建
	FlowActionSubmit   = "submit"   // 提交
	FlowActionApprove  = "approve"  // 审批通过
	FlowActionReject   = "reject"   // 审批拒绝
	FlowActionTransfer = "transfer" // 转交
	FlowActionAssign   = "assign"   // 指派
	FlowActionRevoke   = "revoke"   // 撤回
	FlowActionCancel   = "cancel"   // 取消
	FlowActionReturn   = "return"   // 退回
	FlowActionComplete = "complete" // 完成
)

// 工单流转结果常量
const (
	FlowResultSuccess = "success" // 成功
	FlowResultFailed  = "failed"  // 失败
	FlowResultPending = "pending" // 待处理
)

// WorkorderInstanceFlow 工单流转记录
type WorkorderInstanceFlow struct {
	Model
	InstanceID   int     `json:"instance_id" gorm:"not null;index;comment:工单实例ID"`
	StepID       string  `json:"step_id" gorm:"type:varchar(64);not null;comment:步骤ID"`
	StepName     string  `json:"step_name" gorm:"type:varchar(128);not null;comment:步骤名称"`
	Action       string  `json:"action" gorm:"type:varchar(32);not null;comment:操作动作"`
	OperatorID   int     `json:"operator_id" gorm:"not null;index;comment:操作人ID"`
	OperatorName string  `json:"operator_name" gorm:"type:varchar(128);not null;comment:操作人名称"`
	AssigneeID   *int    `json:"assignee_id" gorm:"index;comment:处理人ID"`
	Comment      string  `json:"comment" gorm:"type:varchar(1000);comment:处理意见"`
	Result       string  `json:"result" gorm:"type:varchar(16);not null;default:'success';comment:处理结果"`
	FormData     JSONMap `json:"form_data" gorm:"type:json;comment:表单数据"`
}

func (WorkorderInstanceFlow) TableName() string {
	return "cl_workorder_instance_flow"
}

// CreateWorkorderInstanceFlowReq 创建工单流转记录请求
type CreateWorkorderInstanceFlowReq struct {
	InstanceID   int     `json:"instance_id" binding:"required,min=1"`
	StepID       string  `json:"step_id" binding:"required,min=1,max=64"`
	StepName     string  `json:"step_name" binding:"required,min=1,max=128"`
	Action       string  `json:"action" binding:"required,oneof=create submit approve reject transfer assign revoke cancel return complete"`
	OperatorID   int     `json:"operator_id" binding:"required,min=1"`
	OperatorName string  `json:"operator_name" binding:"required,min=1,max=128"`
	AssigneeID   *int    `json:"assignee_id" binding:"omitempty,min=1"`
	Comment      string  `json:"comment" binding:"omitempty,max=1000"`
	Result       string  `json:"result" binding:"omitempty,oneof=success failed pending"`
	FormData     JSONMap `json:"form_data" binding:"omitempty"`
}

// ListWorkorderInstanceFlowReq 工单流转记录列表请求
type ListWorkorderInstanceFlowReq struct {
	ListReq
	InstanceID *int    `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	StepID     *string `json:"step_id" form:"step_id" binding:"omitempty,min=1,max=64"`
	Action     *string `json:"action" form:"action" binding:"omitempty,oneof=create submit approve reject transfer assign revoke cancel return complete"`
	OperatorID *int    `json:"operator_id" form:"operator_id" binding:"omitempty,min=1"`
	Result     *string `json:"result" form:"result" binding:"omitempty,oneof=success failed pending"`
}

// DetailWorkorderInstanceFlowReq 获取工单流转记录详情请求
type DetailWorkorderInstanceFlowReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}
