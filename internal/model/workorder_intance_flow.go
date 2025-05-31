package model

import "time"

// ==================== 工单流转记录相关 ====================

// InstanceFlow 工单流转记录实体（DAO层）
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

// TableName 指定工单流转记录表名
func (InstanceFlow) TableName() string {
	return "workorder_instance_flow"
}

// InstanceFlowResp 工单流转记录响应结构
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

// ToResp 将DAO实体转换为响应结构
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
