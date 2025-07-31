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

import "gorm.io/datatypes"

// 流程状态常量
const (
	ProcessStatusDraft     int8 = 1 // 草稿
	ProcessStatusPublished int8 = 2 // 已发布
	ProcessStatusArchived  int8 = 3 // 已归档
)

// 步骤类型常量
const (
	StepTypeStart    = "start"    // 开始节点
	StepTypeApproval = "approval" // 审批节点
	StepTypeTask     = "task"     // 任务节点
	StepTypeDecision = "decision" // 决策节点
	StepTypeEnd      = "end"      // 结束节点
	StepTypeScript   = "script"   // 脚本节点
	StepTypeSubflow  = "subflow"  // 子流程节点
)

// 操作类型常量
const (
	ActionApprove  = "approve"  // 同意
	ActionReject   = "reject"   // 拒绝
	ActionTransfer = "transfer" // 转交
	ActionRevoke   = "revoke"   // 撤回
	ActionCancel   = "cancel"   // 取消
	ActionSubmit   = "submit"   // 提交
	ActionReturn   = "return"   // 退回
)

// 分配策略常量
const (
	AssignStrategyManual   = "manual"   // 手动分配
	AssignStrategyAuto     = "auto"     // 自动分配
	AssignStrategyRole     = "role"     // 角色分配
	AssignStrategyDept     = "dept"     // 部门分配
	AssignStrategyRuleBased = "rule"    // 规则分配
)

// ProcessStep 流程步骤定义
type ProcessStep struct {
	ID              string                 `json:"id"`                          // 步骤ID
	Name            string                 `json:"name"`                        // 步骤名称
	Type            string                 `json:"type"`                        // 步骤类型
	Description     string                 `json:"description,omitempty"`       // 步骤描述
	AssignStrategy  string                 `json:"assign_strategy"`             // 分配策略
	AssigneeType    string                 `json:"assignee_type"`               // 受理人类型：user, role, dept
	AssigneeIDs     []int                  `json:"assignee_ids"`                // 受理人ID列表
	Actions         []string               `json:"actions"`                     // 可执行的动作
	FormFields      []string               `json:"form_fields,omitempty"`       // 表单字段权限
	Conditions      []ProcessCondition     `json:"conditions,omitempty"`        // 执行条件
	TimeLimit       *int                   `json:"time_limit,omitempty"`        // 时间限制(分钟)
	AutoComplete    bool                   `json:"auto_complete"`               // 是否自动完成
	AllowParallel   bool                   `json:"allow_parallel"`              // 是否允许并行
	RequireComment  bool                   `json:"require_comment"`             // 是否必须填写意见
	NotifyUsers     []int                  `json:"notify_users,omitempty"`      // 通知用户列表
	Props           map[string]interface{} `json:"props,omitempty"`             // 步骤属性
	Position        ProcessPosition        `json:"position"`                    // 步骤位置
	SortOrder       int                    `json:"sort_order"`                  // 排序顺序
}

// ProcessCondition 流程条件
type ProcessCondition struct {
	ID       string      `json:"id"`       // 条件ID
	Field    string      `json:"field"`    // 字段名
	Operator string      `json:"operator"` // 操作符：eq, ne, gt, lt, gte, lte, in, not_in, contains, not_contains
	Value    interface{} `json:"value"`    // 条件值
	Logic    string      `json:"logic"`    // 逻辑关系：and, or
}

// ProcessPosition 流程步骤位置
type ProcessPosition struct {
	X int `json:"x"` // X坐标
	Y int `json:"y"` // Y坐标
}

// ProcessConnection 流程连接
type ProcessConnection struct {
	ID        string             `json:"id"`                   // 连接ID
	From      string             `json:"from"`                 // 来源步骤ID
	To        string             `json:"to"`                   // 目标步骤ID
	Label     string             `json:"label,omitempty"`      // 连接标签
	Condition *ProcessCondition  `json:"condition,omitempty"`  // 条件表达式
	Props     map[string]interface{} `json:"props,omitempty"` // 连接属性
}

// ProcessDefinition 流程定义
type ProcessDefinition struct {
	Version     string              `json:"version"`             // 流程版本
	Steps       []ProcessStep       `json:"steps"`               // 步骤列表
	Connections []ProcessConnection `json:"connections"`         // 连接列表
	Variables   []ProcessVariable   `json:"variables,omitempty"` // 变量列表
	Settings    ProcessSettings     `json:"settings,omitempty"`  // 流程设置
}

// ProcessVariable 流程变量
type ProcessVariable struct {
	Name         string      `json:"name"`                    // 变量名
	Type         string      `json:"type"`                    // 变量类型：string, number, boolean, object, array
	DefaultValue interface{} `json:"default_value,omitempty"` // 默认值
	Description  string      `json:"description,omitempty"`   // 变量描述
	Required     bool        `json:"required"`                // 是否必填
	Scope        string      `json:"scope"`                   // 作用域：global, local
}

// ProcessSettings 流程设置
type ProcessSettings struct {
	AllowWithdraw     bool                   `json:"allow_withdraw"`              // 允许撤回
	AllowReassign     bool                   `json:"allow_reassign"`              // 允许重新分配
	AllowParallel     bool                   `json:"allow_parallel"`              // 允许并行审批
	AutoArchive       bool                   `json:"auto_archive"`                // 自动归档
	MaxRetries        int                    `json:"max_retries"`                 // 最大重试次数
	RetryInterval     int                    `json:"retry_interval"`              // 重试间隔(分钟)
	NotificationRules []NotificationRule     `json:"notification_rules"`          // 通知规则
	EscalationRules   []EscalationRule       `json:"escalation_rules,omitempty"`  // 升级规则
	Props             map[string]interface{} `json:"props,omitempty"`             // 其他设置
}

// NotificationRule 通知规则
type NotificationRule struct {
	Event       string   `json:"event"`       // 事件类型：created, approved, rejected, transferred, etc.
	Recipients  []string `json:"recipients"`  // 接收人类型：creator, assignee, manager, custom
	UserIDs     []int    `json:"user_ids"`    // 自定义用户ID列表
	Channels    []string `json:"channels"`    // 通知渠道：email, sms, webhook
	Template    string   `json:"template"`    // 消息模板
	Enabled     bool     `json:"enabled"`     // 是否启用
}

// EscalationRule 升级规则
type EscalationRule struct {
	StepID      string `json:"step_id"`      // 步骤ID
	TimeLimit   int    `json:"time_limit"`   // 超时时间(分钟)
	Action      string `json:"action"`       // 升级动作：notify, reassign, auto_approve
	TargetUsers []int  `json:"target_users"` // 目标用户
	Enabled     bool   `json:"enabled"`      // 是否启用
}

// WorkorderProcess 工单流程实体
type WorkorderProcess struct {
	Model
	Name            string         `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:流程名称"`
	Description     string         `json:"description" gorm:"column:description;type:varchar(1000);comment:流程描述"`
	FormDesignID    int            `json:"form_design_id" gorm:"column:form_design_id;not null;index;comment:关联的表单设计ID"`
	Definition      datatypes.JSON `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version         string         `json:"version" gorm:"column:version;type:varchar(20);not null;default:'1.0.0';comment:版本号"`
	Status          int8           `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-草稿，2-已发布，3-已归档"`
	CategoryID      *int           `json:"category_id" gorm:"column:category_id;index;comment:分类ID"`
	CreatorID       int            `json:"creator_id" gorm:"column:creator_id;not null;index;comment:创建人ID"`
	CreatorName     string         `json:"creator_name" gorm:"-"`
	Tags            StringList     `json:"tags" gorm:"column:tags;comment:标签"`
	UseCount        int            `json:"use_count" gorm:"column:use_count;not null;default:0;comment:使用次数"`
	IsDefault       bool           `json:"is_default" gorm:"column:is_default;not null;default:false;comment:是否为默认流程"`

	// 关联信息（不存储到数据库）
	FormDesignName string `json:"form_design_name,omitempty" gorm:"-"`
	CategoryName   string `json:"category_name,omitempty" gorm:"-"`
}

// TableName 指定工单流程表名
func (WorkorderProcess) TableName() string {
	return "cl_workorder_process"
}

// CreateWorkorderProcessReq 创建工单流程请求
type CreateWorkorderProcessReq struct {
	Name         string            `json:"name" binding:"required,min=1,max=200"`
	Description  string            `json:"description" binding:"omitempty,max=1000"`
	FormDesignID int               `json:"form_design_id" binding:"required,min=1"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id" binding:"omitempty,min=1"`
	Tags         []string          `json:"tags" binding:"omitempty"`
	IsDefault    bool              `json:"is_default"`
}

// UpdateWorkorderProcessReq 更新工单流程请求
type UpdateWorkorderProcessReq struct {
	ID           int               `json:"id" binding:"required,min=1"`
	Name         string            `json:"name" binding:"required,min=1,max=200"`
	Description  string            `json:"description" binding:"omitempty,max=1000"`
	FormDesignID int               `json:"form_design_id" binding:"required,min=1"`
	Definition   ProcessDefinition `json:"definition" binding:"required"`
	CategoryID   *int              `json:"category_id" binding:"omitempty,min=1"`
	Tags         []string          `json:"tags" binding:"omitempty"`
	IsDefault    bool              `json:"is_default"`
}

// DeleteWorkorderProcessReq 删除工单流程请求
type DeleteWorkorderProcessReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderProcessReq 获取工单流程详情请求
type DetailWorkorderProcessReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderProcessReq 工单流程列表请求
type ListWorkorderProcessReq struct {
	ListReq
	CategoryID   *int  `json:"category_id" form:"category_id" binding:"omitempty,min=1"`
	FormDesignID *int  `json:"form_design_id" form:"form_design_id" binding:"omitempty,min=1"`
	Status       *int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2 3"`
	IsDefault    *bool `json:"is_default" form:"is_default"`
	Tags         []string `json:"tags" form:"tags"`
}

// PublishWorkorderProcessReq 发布工单流程请求
type PublishWorkorderProcessReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// ArchiveWorkorderProcessReq 归档工单流程请求
type ArchiveWorkorderProcessReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// CloneWorkorderProcessReq 克隆工单流程请求
type CloneWorkorderProcessReq struct {
	ID   int    `json:"id" binding:"required,min=1"`
	Name string `json:"name" binding:"required,min=1,max=200"`
}

// ValidateWorkorderProcessReq 验证工单流程请求
type ValidateWorkorderProcessReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// SetDefaultProcessReq 设置默认流程请求
type SetDefaultProcessReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// TestWorkorderProcessReq 测试工单流程请求
type TestWorkorderProcessReq struct {
	ID       int                    `json:"id" binding:"required,min=1"`
	FormData map[string]interface{} `json:"form_data" binding:"required"`
}

// GetProcessStepsReq 获取流程步骤请求
type GetProcessStepsReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// BatchUpdateProcessStatusReq 批量更新流程状态请求
type BatchUpdateProcessStatusReq struct {
	IDs    []int `json:"ids" binding:"required,min=1,dive,min=1"`
	Status int8  `json:"status" binding:"required,oneof=1 2 3"`
}

// ProcessStatistics 流程统计
type ProcessStatistics struct {
	DraftCount     int64 `json:"draft_count"`     // 草稿数量
	PublishedCount int64 `json:"published_count"` // 已发布数量
	ArchivedCount  int64 `json:"archived_count"`  // 已归档数量
	DefaultCount   int64 `json:"default_count"`   // 默认流程数量
	TotalUseCount  int64 `json:"total_use_count"` // 总使用次数
}

// ProcessStepInfo 流程步骤信息
type ProcessStepInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Description    string   `json:"description"`
	AssignStrategy string   `json:"assign_strategy"`
	AssigneeType   string   `json:"assignee_type"`
	AssigneeIDs    []int    `json:"assignee_ids"`
	Actions        []string `json:"actions"`
	TimeLimit      *int     `json:"time_limit"`
	RequireComment bool     `json:"require_comment"`
	SortOrder      int      `json:"sort_order"`
}
