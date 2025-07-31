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

// 流转动作类型常量
const (
	FlowActionCreate    = "create"    // 创建
	FlowActionSubmit    = "submit"    // 提交
	FlowActionApprove   = "approve"   // 同意/批准
	FlowActionReject    = "reject"    // 拒绝
	FlowActionTransfer  = "transfer"  // 转交
	FlowActionAssign    = "assign"    // 分配
	FlowActionRevoke    = "revoke"    // 撤回
	FlowActionCancel    = "cancel"    // 取消
	FlowActionReturn    = "return"    // 退回
	FlowActionComplete  = "complete"  // 完成
	FlowActionSuspend   = "suspend"   // 暂停
	FlowActionResume    = "resume"    // 恢复
	FlowActionReopen    = "reopen"    // 重新打开
	FlowActionUpdate    = "update"    // 更新
	FlowActionComment   = "comment"   // 添加评论
	FlowActionAttach    = "attach"    // 添加附件
	FlowActionNotify    = "notify"    // 发送通知
)

// 流转结果常量
const (
	FlowResultSuccess = "success" // 成功
	FlowResultFailed  = "failed"  // 失败
	FlowResultPending = "pending" // 处理中
)

// WorkorderInstanceFlow 工单实例流转记录实体
type WorkorderInstanceFlow struct {
	Model
	InstanceID       int     `json:"instance_id" gorm:"column:instance_id;not null;index;index:idx_instance_created,priority:1;comment:工单实例ID"`
	StepID           string  `json:"step_id" gorm:"column:step_id;type:varchar(100);not null;index;comment:步骤ID"`
	StepName         string  `json:"step_name" gorm:"column:step_name;type:varchar(200);not null;comment:步骤名称"`
	Action           string  `json:"action" gorm:"column:action;type:varchar(50);not null;index;comment:操作动作"`
	ActionName       string  `json:"action_name" gorm:"column:action_name;type:varchar(100);not null;comment:操作名称"`
	OperatorID       int     `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	OperatorName     string  `json:"operator_name" gorm:"-"`
	AssigneeID       *int    `json:"assignee_id" gorm:"column:assignee_id;index;comment:处理人ID"`
	AssigneeName     string  `json:"assignee_name" gorm:"-"`
	Comment          string  `json:"comment" gorm:"column:comment;type:text;comment:处理意见"`
	FormData         JSONMap `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	AttachmentCount  int     `json:"attachment_count" gorm:"column:attachment_count;not null;default:0;comment:附件数量"`
	Duration         *int    `json:"duration" gorm:"column:duration;comment:处理时长(分钟)"`
	FromStepID       string  `json:"from_step_id" gorm:"column:from_step_id;type:varchar(100);comment:来源步骤ID"`
	FromStepName     string  `json:"from_step_name" gorm:"column:from_step_name;type:varchar(200);comment:来源步骤名称"`
	ToStepID         string  `json:"to_step_id" gorm:"column:to_step_id;type:varchar(100);comment:目标步骤ID"`
	ToStepName       string  `json:"to_step_name" gorm:"column:to_step_name;type:varchar(200);comment:目标步骤名称"`
	FromUserID       *int    `json:"from_user_id" gorm:"column:from_user_id;comment:来源用户ID"`
	FromUserName     string  `json:"from_user_name" gorm:"-"`
	ToUserID         *int    `json:"to_user_id" gorm:"column:to_user_id;comment:目标用户ID"`
	ToUserName       string  `json:"to_user_name" gorm:"-"`
	Result           string  `json:"result" gorm:"column:result;type:varchar(20);not null;default:'success';comment:处理结果"`
	ErrorMessage     string  `json:"error_message" gorm:"column:error_message;type:text;comment:错误信息"`
	ClientIP         string  `json:"client_ip" gorm:"column:client_ip;type:varchar(50);comment:客户端IP"`
	UserAgent        string  `json:"user_agent" gorm:"column:user_agent;type:varchar(500);comment:用户代理"`
	ExtendedData     JSONMap `json:"extended_data" gorm:"column:extended_data;type:json;comment:扩展数据"`

	// 关联信息（不存储到数据库）
	InstanceTitle string `json:"instance_title,omitempty" gorm:"-"`
}

// TableName 指定工单实例流转记录表名
func (WorkorderInstanceFlow) TableName() string {
	return "cl_workorder_instance_flow"
}

// WorkorderInstanceFlowResp 工单实例流转记录响应结构
type WorkorderInstanceFlowResp struct {
	ID               int                    `json:"id"`
	InstanceID       int                    `json:"instance_id"`
	StepID           string                 `json:"step_id"`
	StepName         string                 `json:"step_name"`
	Action           string                 `json:"action"`
	ActionName       string                 `json:"action_name"`
	OperatorID       int                    `json:"operator_id"`
	OperatorName     string                 `json:"operator_name"`
	AssigneeID       *int                   `json:"assignee_id"`
	AssigneeName     string                 `json:"assignee_name"`
	Comment          string                 `json:"comment"`
	FormData         map[string]interface{} `json:"form_data"`
	AttachmentCount  int                    `json:"attachment_count"`
	Duration         *int                   `json:"duration"`
	FromStepID       string                 `json:"from_step_id"`
	FromStepName     string                 `json:"from_step_name"`
	ToStepID         string                 `json:"to_step_id"`
	ToStepName       string                 `json:"to_step_name"`
	FromUserID       *int                   `json:"from_user_id"`
	FromUserName     string                 `json:"from_user_name"`
	ToUserID         *int                   `json:"to_user_id"`
	ToUserName       string                 `json:"to_user_name"`
	Result           string                 `json:"result"`
	ErrorMessage     string                 `json:"error_message"`
	InstanceTitle    string                 `json:"instance_title"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// ToResp 转换为响应结构
func (f *WorkorderInstanceFlow) ToResp() *WorkorderInstanceFlowResp {
	return &WorkorderInstanceFlowResp{
		ID:               f.ID,
		InstanceID:       f.InstanceID,
		StepID:           f.StepID,
		StepName:         f.StepName,
		Action:           f.Action,
		ActionName:       f.ActionName,
		OperatorID:       f.OperatorID,
		OperatorName:     f.OperatorName,
		AssigneeID:       f.AssigneeID,
		AssigneeName:     f.AssigneeName,
		Comment:          f.Comment,
		FormData:         f.FormData,
		AttachmentCount:  f.AttachmentCount,
		Duration:         f.Duration,
		FromStepID:       f.FromStepID,
		FromStepName:     f.FromStepName,
		ToStepID:         f.ToStepID,
		ToStepName:       f.ToStepName,
		FromUserID:       f.FromUserID,
		FromUserName:     f.FromUserName,
		ToUserID:         f.ToUserID,
		ToUserName:       f.ToUserName,
		Result:           f.Result,
		ErrorMessage:     f.ErrorMessage,
		InstanceTitle:    f.InstanceTitle,
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
	}
}

// CreateWorkorderInstanceFlowReq 创建工单实例流转记录请求
type CreateWorkorderInstanceFlowReq struct {
	InstanceID     int                    `json:"instance_id" binding:"required,min=1"`
	StepID         string                 `json:"step_id" binding:"required,min=1,max=100"`
	StepName       string                 `json:"step_name" binding:"required,min=1,max=200"`
	Action         string                 `json:"action" binding:"required,min=1,max=50"`
	ActionName     string                 `json:"action_name" binding:"required,min=1,max=100"`
	AssigneeID     *int                   `json:"assignee_id" binding:"omitempty,min=1"`
	Comment        string                 `json:"comment" binding:"omitempty,max=5000"`
	FormData       map[string]interface{} `json:"form_data"`
	FromStepID     string                 `json:"from_step_id" binding:"omitempty,max=100"`
	FromStepName   string                 `json:"from_step_name" binding:"omitempty,max=200"`
	ToStepID       string                 `json:"to_step_id" binding:"omitempty,max=100"`
	ToStepName     string                 `json:"to_step_name" binding:"omitempty,max=200"`
	FromUserID     *int                   `json:"from_user_id" binding:"omitempty,min=1"`
	ToUserID       *int                   `json:"to_user_id" binding:"omitempty,min=1"`
	ExtendedData   map[string]interface{} `json:"extended_data"`
}

// ListWorkorderInstanceFlowReq 工单实例流转记录列表请求
type ListWorkorderInstanceFlowReq struct {
	ListReq
	InstanceID *int      `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	StepID     *string   `json:"step_id" form:"step_id"`
	Action     *string   `json:"action" form:"action"`
	OperatorID *int      `json:"operator_id" form:"operator_id" binding:"omitempty,min=1"`
	AssigneeID *int      `json:"assignee_id" form:"assignee_id" binding:"omitempty,min=1"`
	Result     *string   `json:"result" form:"result" binding:"omitempty,oneof=success failed pending"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
}

// GetWorkorderInstanceFlowsReq 获取工单实例流转记录请求
type GetWorkorderInstanceFlowsReq struct {
	InstanceID int `json:"instance_id" form:"instance_id" binding:"required,min=1"`
}

// DetailWorkorderInstanceFlowReq 获取工单实例流转记录详情请求
type DetailWorkorderInstanceFlowReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// WorkorderInstanceActionReq 工单实例操作请求
type WorkorderInstanceActionReq struct {
	InstanceID     int                    `json:"instance_id" binding:"required,min=1"`
	Action         string                 `json:"action" binding:"required,oneof=approve reject transfer revoke cancel return complete submit"`
	Comment        string                 `json:"comment" binding:"omitempty,max=5000"`
	FormData       map[string]interface{} `json:"form_data"`
	AssigneeID     *int                   `json:"assignee_id" binding:"omitempty,min=1"`
	StepID         string                 `json:"step_id" binding:"required,min=1,max=100"`
	ToStepID       string                 `json:"to_step_id" binding:"omitempty,max=100"`
	NotifyUsers    []int                  `json:"notify_users" binding:"omitempty"`
	ExtendedData   map[string]interface{} `json:"extended_data"`
}

// BatchActionWorkorderInstanceReq 批量操作工单实例请求
type BatchActionWorkorderInstanceReq struct {
	InstanceIDs  []int                  `json:"instance_ids" binding:"required,min=1,dive,min=1"`
	Action       string                 `json:"action" binding:"required,oneof=approve reject cancel assign"`
	Comment      string                 `json:"comment" binding:"omitempty,max=5000"`
	AssigneeID   *int                   `json:"assignee_id" binding:"omitempty,min=1"`
	ExtendedData map[string]interface{} `json:"extended_data"`
}

// GetProcessDefinitionReq 获取流程定义请求
type GetProcessDefinitionReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// GetInstanceCurrentStepReq 获取工单实例当前步骤请求
type GetInstanceCurrentStepReq struct {
	InstanceID int `json:"instance_id" form:"instance_id" binding:"required,min=1"`
}

// GetInstanceAvailableActionsReq 获取工单实例可用操作请求
type GetInstanceAvailableActionsReq struct {
	InstanceID int `json:"instance_id" form:"instance_id" binding:"required,min=1"`
}

// RollbackWorkorderInstanceReq 回滚工单实例请求
type RollbackWorkorderInstanceReq struct {
	InstanceID int    `json:"instance_id" binding:"required,min=1"`
	FlowID     int    `json:"flow_id" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"required,min=1,max=1000"`
}

// GetFlowStatisticsReq 获取流转统计请求
type GetFlowStatisticsReq struct {
	InstanceID *int       `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
	GroupBy    string     `json:"group_by" form:"group_by" binding:"omitempty,oneof=action step operator day week month"`
}

// WorkorderInstanceFlowStatistics 工单实例流转统计
type WorkorderInstanceFlowStatistics struct {
	TotalCount       int64   `json:"total_count"`        // 总流转次数
	SuccessCount     int64   `json:"success_count"`      // 成功次数
	FailedCount      int64   `json:"failed_count"`       // 失败次数
	PendingCount     int64   `json:"pending_count"`      // 处理中次数
	AvgProcessTime   float64 `json:"avg_process_time"`   // 平均处理时间(分钟)
	MaxProcessTime   int     `json:"max_process_time"`   // 最长处理时间(分钟)
	MinProcessTime   int     `json:"min_process_time"`   // 最短处理时间(分钟)
	ApprovalRate     float64 `json:"approval_rate"`      // 审批通过率
	RejectionRate    float64 `json:"rejection_rate"`     // 拒绝率
	TransferRate     float64 `json:"transfer_rate"`      // 转交率
	AutoProcessCount int64   `json:"auto_process_count"` // 自动处理次数
}

// WorkorderInstanceFlowChart 工单实例流转图表数据
type WorkorderInstanceFlowChart struct {
	Date   string `json:"date"`   // 日期
	Action string `json:"action"` // 操作
	Count  int64  `json:"count"`  // 数量
}

// WorkorderInstanceStepDuration 工单实例步骤耗时
type WorkorderInstanceStepDuration struct {
	StepID       string  `json:"step_id"`       // 步骤ID
	StepName     string  `json:"step_name"`     // 步骤名称
	AvgDuration  float64 `json:"avg_duration"`  // 平均耗时(分钟)
	MaxDuration  int     `json:"max_duration"`  // 最长耗时(分钟)
	MinDuration  int     `json:"min_duration"`  // 最短耗时(分钟)
	Count        int64   `json:"count"`         // 处理次数
	TimeoutCount int64   `json:"timeout_count"` // 超时次数
}

// WorkorderInstanceOperatorStats 工单实例操作人统计
type WorkorderInstanceOperatorStats struct {
	OperatorID    int     `json:"operator_id"`    // 操作人ID
	OperatorName  string  `json:"operator_name"`  // 操作人姓名
	ProcessCount  int64   `json:"process_count"`  // 处理次数
	AvgDuration   float64 `json:"avg_duration"`   // 平均处理时间(分钟)
	ApprovalCount int64   `json:"approval_count"` // 批准次数
	RejectionCount int64  `json:"rejection_count"` // 拒绝次数
	TransferCount int64   `json:"transfer_count"` // 转交次数
}
