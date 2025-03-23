package model

import (
	"time"
)

// ListReq 列表请求
type ListFormDesignReq struct {
	Page     int    `json:"page" form:"page" binding:"required,min=1"`
	PageSize int    `json:"size" form:"size" binding:"required,min=10,max=100"`
	Status   int    `json:"status" form:"status" binding:"omitempty"`
	Search   string `json:"search" form:"search" binding:"omitempty"`
}

type DetailFormDesignReq struct {
	ID int64 `json:"id" form:"id" binding:"required"`
}

type PublishFormDesignReq struct {
	ID int64 `json:"id" form:"id" binding:"required"`
}

type CloneFormDesignReq struct {
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
type FormDesign struct {
	Model
	ID          int64  `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name        string `json:"name" gorm:"column:name;not null;comment:表单名称"`
	Description string `json:"description" gorm:"column:description;comment:表单描述"`
	Schema      Schema `json:"schema" gorm:"column:schema;type:json;not null;comment:表单JSON结构"`
	Version     int    `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status      int8   `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID  int64  `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID   int64  `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
}

func (FormDesign) TableName() string {
	return "form_design"
}

// Process 流程定义表
type Process struct {
	ID           int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name         string    `json:"name" gorm:"column:name;not null;comment:流程名称"`
	Description  string    `json:"description" gorm:"column:description;comment:流程描述"`
	FormDesignID int64     `json:"form_design_id" gorm:"column:form_design_id;not null;comment:关联的表单设计ID"`
	Definition   string    `json:"definition" gorm:"column:definition;type:json;not null;comment:流程定义JSON"`
	Version      int       `json:"version" gorm:"column:version;not null;default:1;comment:版本号"`
	Status       int8      `json:"status" gorm:"column:status;not null;default:0;comment:状态：0-草稿，1-已发布，2-已禁用"`
	CategoryID   int64     `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID    int64     `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName  string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	DeletedAt    time.Time `json:"deleted_at" gorm:"column:deleted_at;index;comment:删除时间"`
}

func (Process) TableName() string {
	return "process"
}

// Template 工单模板表
type Template struct {
	ID            int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name          string    `json:"name" gorm:"column:name;not null;comment:模板名称"`
	Description   string    `json:"description" gorm:"column:description;comment:模板描述"`
	ProcessID     int64     `json:"process_id" gorm:"column:process_id;not null;comment:关联的流程ID"`
	DefaultValues string    `json:"default_values" gorm:"column:default_values;type:json;comment:默认值JSON"`
	Icon          string    `json:"icon" gorm:"column:icon;comment:图标URL"`
	Status        int8      `json:"status" gorm:"column:status;not null;default:1;comment:状态：0-禁用，1-启用"`
	SortOrder     int       `json:"sort_order" gorm:"column:sort_order;default:0;comment:排序顺序"`
	CategoryID    int64     `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID     int64     `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName   string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	DeletedAt     time.Time `json:"deleted_at" gorm:"column:deleted_at;index;comment:删除时间"`
}

func (Template) TableName() string {
	return "template"
}

// Instance 工单实例表
type Instance struct {
	ID             int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Title          string    `json:"title" gorm:"column:title;not null;comment:工单标题"`
	ProcessID      int64     `json:"process_id" gorm:"column:process_id;not null;comment:流程ID"`
	ProcessVersion int       `json:"process_version" gorm:"column:process_version;not null;comment:流程版本"`
	FormData       string    `json:"form_data" gorm:"column:form_data;type:json;not null;comment:表单数据"`
	CurrentNode    string    `json:"current_node" gorm:"column:current_node;not null;comment:当前节点ID"`
	Status         int8      `json:"status" gorm:"column:status;not null;comment:状态：0-草稿，1-处理中，2-已完成，3-已取消，4-已拒绝"`
	Priority       int8      `json:"priority" gorm:"column:priority;default:0;comment:优先级：0-普通，1-紧急，2-非常紧急"`
	CategoryID     int64     `json:"category_id" gorm:"column:category_id;comment:分类ID"`
	CreatorID      int64     `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName    string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	AssigneeID     int64     `json:"assignee_id" gorm:"column:assignee_id;comment:当前处理人ID"`
	AssigneeName   string    `json:"assignee_name" gorm:"column:assignee_name;comment:当前处理人姓名"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
	CompletedAt    time.Time `json:"completed_at" gorm:"column:completed_at;comment:完成时间"`
	DueDate        time.Time `json:"due_date" gorm:"column:due_date;comment:截止时间"`
}

func (Instance) TableName() string {
	return "instance"
}

// InstanceFlow 工单流转记录表
type InstanceFlow struct {
	ID           int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	InstanceID   int64     `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	NodeID       string    `json:"node_id" gorm:"column:node_id;not null;comment:节点ID"`
	NodeName     string    `json:"node_name" gorm:"column:node_name;not null;comment:节点名称"`
	Action       string    `json:"action" gorm:"column:action;not null;comment:操作：approve-同意，reject-拒绝，transfer-转交，comment-评论"`
	OperatorID   int64     `json:"operator_id" gorm:"column:operator_id;not null;comment:操作人ID"`
	OperatorName string    `json:"operator_name" gorm:"column:operator_name;not null;comment:操作人姓名"`
	Comment      string    `json:"comment" gorm:"column:comment;type:text;comment:处理意见"`
	FormData     string    `json:"form_data" gorm:"column:form_data;type:json;comment:表单数据（如有修改）"`
	Attachments  string    `json:"attachments" gorm:"column:attachments;type:json;comment:附件列表"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
}

func (InstanceFlow) TableName() string {
	return "instance_flow"
}

// InstanceComment 工单评论表
type InstanceComment struct {
	ID          int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	InstanceID  int64     `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	Content     string    `json:"content" gorm:"column:content;type:text;not null;comment:评论内容"`
	Attachments string    `json:"attachments" gorm:"column:attachments;type:json;comment:附件列表"`
	CreatorID   int64     `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string    `json:"creator_name" gorm:"column:creator_name;not null;comment:创建人姓名"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	ParentID    int64     `json:"parent_id" gorm:"column:parent_id;default:0;comment:父评论ID，用于回复功能"`
}

func (InstanceComment) TableName() string {
	return "instance_comment"
}

// Category 工单分类表
type Category struct {
	ID          int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Name        string    `json:"name" gorm:"column:name;not null;comment:分类名称"`
	ParentID    int64     `json:"parent_id" gorm:"column:parent_id;default:0;comment:父分类ID，0表示顶级分类"`
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
	ID              int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
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
	ID                int64     `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	UserID            int64     `json:"user_id" gorm:"column:user_id;not null;index;comment:用户ID"`
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
