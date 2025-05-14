package model

import (
	"time"
)

// ListReq 列表请求
type ListFormDesignReq struct {
	Page     int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize int    `json:"page_size" form:"size" binding:"required,min=10,max=100"`
	Status   int    `json:"status" form:"status" binding:"omitempty"`
	Search   string `json:"search" form:"search" binding:"omitempty"`
}

type DetailFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type PublishFormDesignReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type CloneFormDesignReq struct {
	ID   int    `json:"id" form:"id" binding:"required"`
	Name string `json:"name" form:"name" binding:"required"`
}

type Field struct {
	Type     string `json:"type"`
	Label    string `json:"label"`
	Field    string `json:"field"`
	Required bool   `json:"required"`
}

type Schema struct {
	Fields []Field `json:"fields"`
}

// FormDesign 表单设计表
type FormDesignReq struct {
	ID          int    `json:"id"`
	Name        string `json:"name" gorm:"column:name;not null;comment:表单名称"`
	Description string `json:"description" gorm:"column:description;comment:表单描述"`
	Schema      string `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Version     int    `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status      int8   `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID  int    `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID   int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name"`
}

// FormDesign 表单设计表
type FormDesign struct {
	Model
	Name        string `json:"name" gorm:"column:name;not null;comment:表单名称"`
	Description string `json:"description" gorm:"column:description;comment:表单描述"`
	Schema      string `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Version     int    `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status      int8   `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID  int    `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID   int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name" gorm:"-"`
}

func (FormDesign) TableName() string {
	return "form_design"
}

// Process 流程定义表
type ProcessReq struct {
	ID           int    `json:"id"`
	Name         string `json:"name" gorm:"column:name;not null;comment:流程名称"`
	Description  string `json:"description" gorm:"column:description;comment:流程描述"`
	FormDesignID int    `json:"form_design_id" gorm:"column:form_design_id;not null;comment:关联的表单设计ID"`
	Definition   string `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version      int    `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status       int8   `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID   int    `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
}

// Process 流程定义表
type CreateProcessReq struct {
	Name         string     `json:"name" gorm:"column:name;not null;comment:流程名称"`
	Description  string     `json:"description" gorm:"column:description;comment:流程描述"`
	FormDesignID int        `json:"form_design_id" gorm:"column:form_design_id;not null;comment:关联的表单设计ID"`
	Definition   Definition `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version      int        `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status       int8       `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID   int        `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int        `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
}

type UpdateProcessReq struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	FormDesignID int        `json:"form_design_id"`
	Definition   Definition `json:"definition"`
	Version      int        `json:"version"`
	Status       int8       `json:"status"`
	CategoryID   int        `json:"category_id"`
	CreatorID    int        `json:"creator_id"`
}

// Definition 流程定义
type Definition struct {
	Steps []Step `json:"steps"`
}

// Step 流程步骤
type Step struct {
	Step   string `json:"step"`
	Role   string `json:"role"`
	Action string `json:"action"`
}

type DeleteProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type DetailProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListProcessReq struct {
	Page     int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize int    `json:"page_size" form:"size" binding:"required,min=10,max=100"`
	Status   int    `json:"status" form:"status" binding:"omitempty"`
	Search   string `json:"search" form:"search" binding:"omitempty"`
}

type PublishProcessReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type CloneProcessReq struct {
	ID   int    `json:"id" form:"id" binding:"required"`
	Name string `json:"name" form:"name" binding:"required"`
}

// Process 流程定义表
type Process struct {
	Model
	Name         string `json:"name" gorm:"column:name;not null;comment:流程名称"`
	Description  string `json:"description" gorm:"column:description;comment:流程描述"`
	FormDesignID int    `json:"form_design_id" gorm:"column:form_design_id;not null;comment:关联的表单设计ID"`
	Definition   string `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version      int    `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status       int8   `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID   int    `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName  string `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
}

func (Process) TableName() string {
	return "process"
}

type DefaultValues struct {
	Approver string `json:"approver"`
	Deadline string `json:"deadline"`
}

type DeleteTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type DetailTemplateReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ListTemplateReq struct {
	Page     int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize int    `json:"page_size" form:"size" binding:"required,min=10,max=100"`
	Status   int    `json:"status" form:"status" binding:"omitempty"`
	Search   string `json:"search" form:"search" binding:"omitempty"`
}

// Template 工单模板表
type TemplateReq struct {
	ID            int    `json:"id"`
	Name          string `json:"name" gorm:"column:name;not null;comment:模板名称"`
	Description   string `json:"description" gorm:"column:description;comment:模板描述"`
	ProcessID     int    `json:"process_id" gorm:"column:process_id;not null;comment:关联的流程ID"`
	DefaultValues string `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Icon          string `json:"icon" gorm:"column:icon;comment:图标URL"`
	Status        int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	SortOrder     int    `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	CategoryID    int    `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID     int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName   string `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
}

// Template 工单模板表
type Template struct {
	Model
	Name          string `json:"name" gorm:"column:name;not null;comment:模板名称"`
	Description   string `json:"description" gorm:"column:description;comment:模板描述"`
	ProcessID     int    `json:"process_id" gorm:"column:process_id;not null;comment:关联的流程ID"`
	DefaultValues string `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Icon          string `json:"icon" gorm:"column:icon;comment:图标URL"`
	Status        int8   `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	SortOrder     int    `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	CategoryID    int    `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID     int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName   string `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
}

func (Template) TableName() string {
	return "template"
}

type DeleteInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
type DetailInstanceReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
type ListInstanceReq struct {
	Page       int      `json:"page"`        // 页码
	PageSize   int      `json:"page_size"`   // 每页大小
	Status     int      `json:"status"`      // 状态
	Keyword    string   `json:"keyword"`     // 关键字搜索
	DateRange  []string `json:"date_range"`  // 日期范围 ["开始日期", "结束日期"]
	CreatorID  int      `json:"creator_id"`  // 创建人ID
	AssigneeID int      `json:"assignee_id"` // 处理人ID
}

type FormData struct {
	Reason      string   `json:"reason"`        // 请假原因
	DateRange   []string `json:"date_range"`    // 请假日期范围
	Type        string   `json:"type"`          // 请假类型（个人、病假等）
	ApproveDays int      `json:"approved_days"` // 可请假天数
}

type InstanceReq struct {
	ID             int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Title          string    `json:"title" gorm:"column:title;not null;comment:工单标题"`
	ProcessID      int       `json:"process_id" gorm:"column:process_id;not null;comment:流程ID"`
	ProcessVersion int       `json:"process_version" gorm:"column:process_version;not null;comment:流程版本"`
	FormData       FormData  `json:"form_data" gorm:"column:form_data;type:json;not null;comment:表单数据"`
	CurrentNode    string    `json:"current_node" gorm:"column:current_node;not null;comment:当前节点ID"`
	Status         int8      `json:"status" gorm:"column:status;not null;comment:状态：0-草稿，1-处理中，2-已完成，3-已取消，4-已拒绝"`
	Priority       int8      `json:"priority" gorm:"column:priority;default:0;comment:优先级：0-普通，1-紧急，2-非常紧急"`
	CategoryID     int       `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID      int       `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName    string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	AssigneeID     int       `json:"assignee_id" gorm:"column:assignee_id;comment:当前处理人ID"`
	AssigneeName   string    `json:"assignee_name" gorm:"column:assignee_name;comment:当前处理人姓名"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	CompletedAt    time.Time `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`
	DueDate        time.Time `json:"due_date" gorm:"column:due_date;comment:截止时间"`
}

// Instance 工单实例表
type Instance struct {
	Model
	Title          string    `json:"title" gorm:"column:title;not null;comment:工单标题"`
	ProcessID      int       `json:"process_id" gorm:"column:process_id;not null;comment:流程ID"`
	ProcessVersion int       `json:"process_version" gorm:"column:process_version;not null;comment:流程版本"`
	FormData       string    `json:"form_data" gorm:"column:form_data;type:json;not null;comment:表单数据"`
	CurrentNode    string    `json:"current_node" gorm:"column:current_node;not null;comment:当前节点ID"`
	Status         int8      `json:"status" gorm:"column:status;not null;comment:状态：0-草稿，1-处理中，2-已完成，3-已取消，4-已拒绝"`
	Priority       int8      `json:"priority" gorm:"column:priority;default:0;comment:优先级：0-普通，1-紧急，2-非常紧急"`
	CategoryID     int       `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID      int       `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName    string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	AssigneeID     int       `json:"assignee_id" gorm:"column:assignee_id;comment:当前处理人ID"`
	AssigneeName   string    `json:"assignee_name" gorm:"column:assignee_name;comment:当前处理人姓名"`
	CompletedAt    time.Time `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`
	DueDate        time.Time `json:"due_date" gorm:"column:due_date;comment:截止时间"`
}

func (Instance) TableName() string {
	return "instance"
}

// InstanceFlow 工单流转记录表
type InstanceFlowReq struct {
	ID           int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	InstanceID   int       `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	NodeID       string    `json:"node_id" gorm:"column:node_id;not null;comment:节点ID"`
	NodeName     string    `json:"node_name" gorm:"column:node_name;not null;comment:节点名称"`
	Action       string    `json:"action" gorm:"column:action;not null;comment:操作：approve-同意，reject-拒绝，transfer-转交，comment-评论"`
	TargetUserID int       `json:"target_user_id,omitempty" gorm:"column:target_user_id;comment:转交用户ID"` // 允许为空
	OperatorID   int       `json:"operator_id" gorm:"column:operator_id;not null;comment:操作人ID"`
	OperatorName string    `json:"operator_name" gorm:"column:operator_name;not null;comment:操作人姓名"`
	Comment      string    `json:"comment" gorm:"column:comment;type:text;comment:处理意见"`
	FormData     FormData  `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据（如有修改）"`
	Attachments  string    `json:"attachments" gorm:"column:attachments;type:json;comment:附件列表"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
}
type InstanceFlow struct {
	Model
	InstanceID   int    `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	NodeID       string `json:"node_id" gorm:"column:node_id;not null;comment:节点ID"`
	NodeName     string `json:"node_name" gorm:"column:node_name;not null;comment:节点名称"`
	Action       string `json:"action" gorm:"column:action;not null;comment:操作：approve-同意，reject-拒绝，transfer-转交，comment-评论"`
	TargetUserID int    `json:"target_user_id,omitempty" gorm:"column:target_user_id;comment:转交用户ID"` // 允许为空
	OperatorID   int    `json:"operator_id" gorm:"column:operator_id;not null;comment:操作人ID"`
	OperatorName string `json:"operator_name" gorm:"column:operator_name;not null;comment:操作人姓名"`
	Comment      string `json:"comment" gorm:"column:comment;type:text;comment:处理意见"`
	FormData     string `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据（如有修改）"`
	Attachments  string `json:"attachments" gorm:"column:attachments;type:json;comment:附件列表"`
}

func (InstanceFlow) TableName() string {
	return "instance_flow"
}

// InstanceComment 工单评论表
type InstanceCommentReq struct {
	ID          int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	InstanceID  int       `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	Content     string    `json:"content" gorm:"column:content;type:text;not null;comment:评论内容"`
	Attachments string    `json:"attachments" gorm:"column:attachments;type:json;comment:附件列表"`
	CreatorID   int       `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	ParentID    int       `json:"parent_id" gorm:"column:parent_id;default:0;comment:父评论ID，用于回复功能"`
}

type InstanceComment struct {
	Model
	InstanceID  int    `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	Content     string `json:"content" gorm:"column:content;type:text;not null;comment:评论内容"`
	Attachments string `json:"attachments" gorm:"column:attachments;type:json;comment:附件列表"`
	CreatorID   int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	ParentID    int    `json:"parent_id" gorm:"column:parent_id;default:0;comment:父评论ID，用于回复功能"`
}

func (InstanceComment) TableName() string {
	return "instance_comment"
}

// Category 工单分类表
type Category struct {
	ID          int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name        string    `json:"name" gorm:"column:name;not null;comment:分类名称"`
	ParentID    int       `json:"parent_id" gorm:"column:parent_id;default:0;comment:父分类ID，0表示顶级分类"`
	Icon        string    `json:"icon" gorm:"column:icon;comment:图标URL"`
	SortOrder   int       `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	Status      int8      `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	Description string    `json:"description" gorm:"column:description;comment:分类描述"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	DeletedAt   time.Time `json:"deleted_at" gorm:"column:deleted_at;index;comment:删除时间"`
}

func (Category) TableName() string {
	return "category"
}

// WorkOrderStatistics 工单统计表
type WorkOrderStatistics struct {
	ID              int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Date            time.Time `json:"date" gorm:"column:date;not null;index;comment:统计日期"`
	TotalCount      int       `json:"total_count" gorm:"column:total_count;not null;default:0;comment:工单总数"`
	CompletedCount  int       `json:"completed_count" gorm:"column:completed_count;not null;default:0;comment:已完成工单数"`
	ProcessingCount int       `json:"processing_count" gorm:"column:processing_count;not null;default:0;comment:处理中工单数"`
	CanceledCount   int       `json:"canceled_count" gorm:"column:canceled_count;not null;default:0;comment:已取消工单数"`
	RejectedCount   int       `json:"rejected_count" gorm:"column:rejected_count;not null;default:0;comment:已拒绝工单数"`
	AvgProcessTime  float64   `json:"avg_process_time" gorm:"column:avg_process_time;not null;default:0;comment:平均处理时间(小时)"`
	CategoryStats   string    `json:"category_stats" gorm:"column:category_stats;type:json;comment:分类统计JSON"`
	UserStats       string    `json:"user_stats" gorm:"column:user_stats;type:json;comment:用户统计JSON"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
}

func (WorkOrderStatistics) TableName() string {
	return "work_order_statistics"
}

// UserPerformance 用户工单处理绩效表
type UserPerformance struct {
	ID                int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	UserID            int       `json:"user_id" gorm:"column:user_id;not null;index;comment:用户ID"`
	UserName          string    `json:"user_name" gorm:"column:user_name;not null;comment:用户姓名"`
	Date              time.Time `json:"date" gorm:"column:date;not null;index;comment:统计日期"`
	AssignedCount     int       `json:"assigned_count" gorm:"column:assigned_count;not null;default:0;comment:分配工单数"`
	CompletedCount    int       `json:"completed_count" gorm:"column:completed_count;not null;default:0;comment:完成工单数"`
	AvgResponseTime   float64   `json:"avg_response_time" gorm:"column:avg_response_time;not null;default:0;comment:平均响应时间(小时)"`
	AvgProcessingTime float64   `json:"avg_processing_time" gorm:"column:avg_processing_time;not null;default:0;comment:平均处理时间(小时)"`
	SatisfactionScore float64   `json:"satisfaction_score" gorm:"column:satisfaction_score;default:0;comment:满意度评分"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
}

func (UserPerformance) TableName() string {
	return "user_performance"
}
