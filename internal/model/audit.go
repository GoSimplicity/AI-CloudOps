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

import (
	"gorm.io/datatypes"
)

// AuditLog 审计日志模型 - 优化存储结构
type AuditLog struct {
	Model
	UserID        int            `json:"user_id" gorm:"index:idx_user_time;not null;comment:操作用户ID"`
	TraceID       string         `json:"trace_id" gorm:"size:32;index;comment:链路追踪ID"`
	IPAddress     string         `json:"ip_address" gorm:"size:45;not null;comment:操作IP地址"`
	UserAgent     string         `json:"user_agent" gorm:"size:500;comment:用户代理"`
	HttpMethod    string         `json:"http_method" gorm:"size:10;not null;index:idx_method_status;comment:HTTP请求方法"`
	Endpoint      string         `json:"endpoint" gorm:"size:255;not null;index;comment:请求端点"`
	OperationType string         `json:"operation_type" gorm:"type:VARCHAR(20);index:idx_operation_time;not null;comment:操作类型"`
	TargetType    string         `json:"target_type" gorm:"size:64;not null;index;comment:目标资源类型"`
	TargetID      string         `json:"target_id" gorm:"size:255;index;comment:目标资源ID"`
	StatusCode    int            `json:"status_code" gorm:"not null;index:idx_method_status;comment:HTTP状态码"`
	RequestBody   datatypes.JSON `json:"request_body" gorm:"type:json;comment:请求体"`
	ResponseBody  datatypes.JSON `json:"response_body" gorm:"type:json;comment:响应体"`
	Duration      int64          `json:"duration" gorm:"not null;comment:请求耗时(微秒)"`
	ErrorMsg      string         `json:"error_msg" gorm:"size:1000;comment:错误信息"`
}

// TableName 指定表名，支持分表
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLogBatch 批量写入的审计日志
type AuditLogBatch struct {
	Logs []AuditLog `json:"logs"`
}

// CreateAuditLogRequest 创建审计日志请求 - middleware使用
type CreateAuditLogRequest struct {
	UserID        int            `json:"user_id" binding:"required"`
	TraceID       string         `json:"trace_id"`
	IPAddress     string         `json:"ip_address" binding:"required"`
	UserAgent     string         `json:"user_agent"`
	HttpMethod    string         `json:"http_method" binding:"required"`
	Endpoint      string         `json:"endpoint" binding:"required"`
	OperationType string         `json:"operation_type" binding:"required"`
	TargetType    string         `json:"target_type"`
	TargetID      string         `json:"target_id"`
	StatusCode    int            `json:"status_code" binding:"required"`
	RequestBody   datatypes.JSON `json:"request_body"`
	ResponseBody  datatypes.JSON `json:"response_body"`
	Duration      int64          `json:"duration" binding:"required"`
	ErrorMsg      string         `json:"error_msg"`
}

// ListAuditLogsRequest 审计日志列表查询参数
type ListAuditLogsRequest struct {
	ListReq
	OperationType string `json:"operation_type" form:"operation_type"`
	UserID        int    `json:"user_id" form:"user_id"`
	TargetType    string `json:"target_type" form:"target_type"`
	StatusCode    int    `json:"status_code" form:"status_code"`
	StartTime     int64  `json:"start_time" form:"start_time"`
	EndTime       int64  `json:"end_time" form:"end_time"`
	TraceID       string `json:"trace_id" form:"trace_id"`
}

type GetAuditLogDetailRequest struct {
	ID int `json:"id" binding:"required"`
}

// SearchAuditLogsRequest 审计日志搜索请求
type SearchAuditLogsRequest struct {
	ListAuditLogsRequest
	Advanced *AdvancedSearchOptions `json:"advanced"`
}

// AdvancedSearchOptions 高级搜索选项
type AdvancedSearchOptions struct {
	IPAddressList   []string `json:"ip_address_list"`
	StatusCodeList  []int    `json:"status_code_list"`
	DurationMin     int64    `json:"duration_min"`
	DurationMax     int64    `json:"duration_max"`
	HasError        *bool    `json:"has_error"`
	EndpointPattern string   `json:"endpoint_pattern"`
}

// AuditStatistics 审计统计信息
type AuditStatistics struct {
	TotalCount         int64                    `json:"total_count"`
	TodayCount         int64                    `json:"today_count"`
	ErrorCount         int64                    `json:"error_count"`
	AvgDuration        float64                  `json:"avg_duration"`
	TypeDistribution   []TypeDistributionItem   `json:"type_distribution"`
	StatusDistribution []StatusDistributionItem `json:"status_distribution"`
	RecentActivity     []RecentActivityItem     `json:"recent_activity"`
	HourlyTrend        []HourlyTrendItem        `json:"hourly_trend"`
}

// TypeDistributionItem 操作类型分布项
type TypeDistributionItem struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// StatusDistributionItem 状态码分布项
type StatusDistributionItem struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}

// RecentActivityItem 最近活动项
type RecentActivityItem struct {
	Time          int64  `json:"time"`
	OperationType string `json:"operation_type"`
	UserID        int    `json:"user_id"`
	Username      string `json:"username"`
	TargetType    string `json:"target_type"`
	StatusCode    int    `json:"status_code"`
	Duration      int64  `json:"duration"`
}

// HourlyTrendItem 小时趋势项
type HourlyTrendItem struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

// ExportAuditLogsRequest 导出审计日志请求
type ExportAuditLogsRequest struct {
	ListAuditLogsRequest
	Format  string   `json:"format" binding:"oneof=csv json excel" form:"format"`
	Fields  []string `json:"fields" form:"fields"`
	MaxRows int      `json:"max_rows" binding:"max=10000" form:"max_rows"`
}

type DeleteAuditLogRequest struct {
	ID int `json:"id" binding:"required"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1,max=100"`
}

// ArchiveAuditLogsRequest 归档审计日志请求
type ArchiveAuditLogsRequest struct {
	StartTime int64 `json:"start_time" binding:"required"`
	EndTime   int64 `json:"end_time" binding:"required"`
}

// AuditTypeInfo 审计类型信息
type AuditTypeInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Category    string `json:"category"`
}
