package model

import (
	"gorm.io/datatypes"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID            uint           `json:"id" gorm:"primarykey;comment:主键ID"`
	UserID        uint           `json:"user_id" gorm:"index;not null;comment:操作用户ID"`
	IPAddress     string         `json:"ip_address" gorm:"size:45;not null;comment:操作IP地址"`
	UserAgent     string         `json:"user_agent" gorm:"size:255;not null;comment:用户代理"`
	HttpMethod    string         `json:"http_method" gorm:"size:10;not null;comment:HTTP请求方法"`
	Endpoint      string         `json:"endpoint" gorm:"size:255;not null;comment:请求端点"`
	OperationType string         `json:"operation_type" gorm:"type:ENUM('CREATE','UPDATE','DELETE','OTHER');index;not null;comment:操作类型"`
	TargetType    string         `json:"target_type" gorm:"size:64;not null;comment:目标资源类型"`
	TargetID      string         `json:"target_id" gorm:"size:255;index;comment:目标资源ID"`
	StatusCode    int            `json:"status_code" gorm:"not null;comment:HTTP状态码"`
	RequestBody   datatypes.JSON `json:"request_body" gorm:"type:json;comment:请求体"`
	ResponseBody  datatypes.JSON `json:"response_body" gorm:"type:json;comment:响应体"`
	Duration      int64          `json:"duration" gorm:"not null;comment:请求耗时"`
	CreatedAt     int64          `json:"created_at" gorm:"index;comment:创建时间"`
	UpdatedAt     int64          `json:"-" gorm:"comment:更新时间"`
	DeletedAt     int64          `json:"-" gorm:"index;comment:删除时间"`
}

// AuditLogDetail 审计日志详情视图模型(脱敏后)
type AuditLogDetail struct {
	ID            uint   `json:"id" gorm:"comment:日志ID"`
	UserID        uint   `json:"user_id" gorm:"comment:操作用户ID"`
	OperationType string `json:"operation_type" gorm:"comment:操作类型"`
	TargetType    string `json:"target_type" gorm:"comment:目标资源类型"`
	TargetID      string `json:"target_id" gorm:"comment:目标资源ID"`
	CreatedAt     int64  `json:"created_at" gorm:"comment:操作时间"`
	DetailInfo    string `json:"detail_info" gorm:"comment:格式化后的可读信息"`
}

// ListAuditLogsRequest 审计日志列表查询参数
type ListAuditLogsRequest struct {
	PageNumber    int    `json:"page_number" validate:"required,min=1" gorm:"comment:页码"`
	PageSize      int    `json:"page_size" validate:"required,min=1,max=100" gorm:"comment:每页大小"`
	OperationType string `json:"operation_type" validate:"omitempty,oneof=CREATE UPDATE DELETE OTHER" gorm:"comment:操作类型过滤"`
	UserID        uint   `json:"user_id" validate:"omitempty,min=1" gorm:"comment:操作人ID过滤"`
	StartTime     int64  `json:"start_time" validate:"required" gorm:"comment:开始时间"`
	EndTime       int64  `json:"end_time" validate:"required,gtfield=StartTime" gorm:"comment:结束时间"`
}
