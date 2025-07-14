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

// InstanceFlow 工单流转记录实体
type InstanceFlow struct {
	Model
	InstanceID   int     `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	StepID       string  `json:"step_id" gorm:"column:step_id;not null;comment:步骤ID"`
	StepName     string  `json:"step_name" gorm:"column:step_name;not null;comment:步骤名称"`
	Action       string  `json:"action" gorm:"column:action;not null;comment:操作"`
	OperatorID   int     `json:"operator_id" gorm:"column:operator_id;not null;comment:操作人ID"`
	OperatorName string  `json:"operator_name" gorm:"-"`
	Comment      string  `json:"comment" gorm:"column:comment;type:text;comment:处理意见"`
	FormData     JSONMap `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据"`
	Duration     *int    `json:"duration" gorm:"column:duration;comment:处理时长(分钟)"`
	FromStepID   string  `json:"from_step_id" gorm:"column:from_step_id;comment:来源步骤ID"`
	ToStepID     string  `json:"to_step_id" gorm:"column:to_step_id;comment:目标步骤ID"`
	FromUserID   int     `json:"from_user_id" gorm:"column:from_user_id;comment:来源用户ID"`
	ToUserID     int     `json:"to_user_id" gorm:"column:to_user_id;comment:目标用户ID"`
	ToUserName   string  `json:"to_user_name" gorm:"-"`
}

func (InstanceFlow) TableName() string {
	return "workorder_instance_flow"
}

type InstanceFlowResp struct {
	ID           int                    `json:"id"`
	InstanceID   int                    `json:"instance_id"`
	StepID       string                 `json:"step_id"`
	StepName     string                 `json:"step_name"`
	Action       string                 `json:"action"`
	OperatorID   int                    `json:"operator_id"`
	OperatorName string                 `json:"operator_name"`
	Comment      string                 `json:"comment"`
	FormData     map[string]interface{} `json:"form_data"`
	Duration     *int                   `json:"duration"`
	FromStepID   string                 `json:"from_step_id"`
	ToStepID     string                 `json:"to_step_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

func (i *InstanceFlow) ToResp() *InstanceFlowResp {
	return &InstanceFlowResp{
		ID:           i.ID,
		InstanceID:   i.InstanceID,
		StepID:       i.StepID,
		StepName:     i.StepName,
		Action:       i.Action,
		OperatorID:   i.OperatorID,
		OperatorName: i.OperatorName,
		Comment:      i.Comment,
		FormData:     i.FormData,
		Duration:     i.Duration,
		FromStepID:   i.FromStepID,
		ToStepID:     i.ToStepID,
		CreatedAt:    i.CreatedAt,
		UpdatedAt:    i.UpdatedAt,
	}
}

type GetInstanceFlowsReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type GetProcessDefinitionReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type InstanceActionReq struct {
	InstanceID int                    `json:"instance_id" binding:"required"`
	Action     string                 `json:"action" binding:"required,oneof=approve reject transfer revoke cancel"`
	Comment    string                 `json:"comment" binding:"omitempty,max=1000"`
	FormData   map[string]interface{} `json:"form_data"`
	AssigneeID *int                   `json:"assignee_id" binding:"omitempty,min=1"`
	StepID     string                 `json:"step_id" binding:"required"`
}
