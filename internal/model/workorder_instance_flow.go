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

// 工单状态流转动作类型常量 - 仅包含状态变更相关操作
const (
	FlowActionSubmit   = "submit"   // 提交工单
	FlowActionApprove  = "approve"  // 审批通过
	FlowActionReject   = "reject"   // 审批拒绝
	FlowActionAssign   = "assign"   // 指派处理人
	FlowActionCancel   = "cancel"   // 取消工单
	FlowActionComplete = "complete" // 完成工单
	FlowActionReturn   = "return"   // 退回工单
)

// WorkorderInstanceFlow 工单状态流转记录 - 专注于状态变更的业务流程
type WorkorderInstanceFlow struct {
	Model
	InstanceID     int     `json:"instance_id" gorm:"column:instance_id;not null;index;comment:工单实例ID"`
	Action         string  `json:"action" gorm:"column:action;type:varchar(32);not null;comment:流转动作"`
	OperatorID     int     `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	OperatorName   string  `json:"operator_name" gorm:"column:operator_name;type:varchar(100);not null;comment:操作人名称"`
	FromStatus     int8    `json:"from_status" gorm:"column:from_status;not null;comment:变更前状态"`
	ToStatus       int8    `json:"to_status" gorm:"column:to_status;not null;comment:变更后状态"`
	Comment        string  `json:"comment" gorm:"column:comment;type:varchar(1000);comment:审批意见或处理说明"`
	IsSystemAction int8    `json:"is_system_action" gorm:"column:is_system_action;not null;default:2;comment:是否为系统自动操作：1-是，2-否"`
}

func (WorkorderInstanceFlow) TableName() string {
	return "cl_workorder_instance_flow"
}

// CreateWorkorderInstanceFlowReq 创建工单流转记录请求
type CreateWorkorderInstanceFlowReq struct {
	InstanceID     int    `json:"instance_id" binding:"required,min=1"`
	Action         string `json:"action" binding:"required,oneof=submit approve reject assign cancel complete return"`
	OperatorID     int    `json:"operator_id" binding:"required,min=1"`
	OperatorName   string `json:"operator_name" binding:"required,min=1,max=100"`
	FromStatus     int8   `json:"from_status" binding:"required,min=1,max=6"`
	ToStatus       int8   `json:"to_status" binding:"required,min=1,max=6"`
	Comment        string `json:"comment" binding:"omitempty,max=1000"`
	IsSystemAction int8   `json:"is_system_action" binding:"omitempty,oneof=1 2"`
}

// ListWorkorderInstanceFlowReq 工单流转记录列表请求
type ListWorkorderInstanceFlowReq struct {
	ListReq
	InstanceID     *int    `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	Action         *string `json:"action" form:"action" binding:"omitempty,oneof=submit approve reject assign cancel complete return"`
	OperatorID     *int    `json:"operator_id" form:"operator_id" binding:"omitempty,min=1"`
	IsSystemAction *int8   `json:"is_system_action" form:"is_system_action" binding:"omitempty,oneof=1 2"`
}

// DetailWorkorderInstanceFlowReq 获取工单流转记录详情请求
type DetailWorkorderInstanceFlowReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}
