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

// ==================== 统计请求结构 ====================

// StatsReq 统一的统计请求
type StatsReq struct {
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
	Dimension  string     `json:"dimension" form:"dimension" binding:"omitempty,oneof=day week month"`                     // 趋势统计用
	CategoryID *int       `json:"category_id" form:"category_id"`                                                          // 分类筛选
	UserID     *int       `json:"user_id" form:"user_id"`                                                                  // 用户筛选
	Status     *string    `json:"status" form:"status"`                                                                    // 状态筛选
	Priority   *string    `json:"priority" form:"priority"`                                                                // 优先级筛选
	Top        int        `json:"top" form:"top" binding:"omitempty,min=5,max=50" default:"10"`                            // 排行榜数量
	SortBy     string     `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=count completion_rate avg_process_time"` // 排序字段
}

// ==================== 统计响应结构 ====================

// OverviewStats 概览统计
type OverviewStats struct {
	TotalCount      int64   `json:"total_count"`       // 总工单数
	CompletedCount  int64   `json:"completed_count"`   // 已完成
	ProcessingCount int64   `json:"processing_count"`  // 处理中
	PendingCount    int64   `json:"pending_count"`     // 待处理
	OverdueCount    int64   `json:"overdue_count"`     // 超时
	CompletionRate  float64 `json:"completion_rate"`   // 完成率
	AvgProcessTime  float64 `json:"avg_process_time"`  // 平均处理时间(小时)
	AvgResponseTime float64 `json:"avg_response_time"` // 平均响应时间(小时)
	TodayCreated    int64   `json:"today_created"`     // 今日创建
	TodayCompleted  int64   `json:"today_completed"`   // 今日完成
}

// TrendStats 趋势统计
type TrendStats struct {
	Dates           []string  `json:"dates"`             // 日期列表
	CreatedCounts   []int64   `json:"created_counts"`    // 创建数量
	CompletedCounts []int64   `json:"completed_counts"`  // 完成数量
	CompletionRates []float64 `json:"completion_rates"`  // 完成率
	AvgProcessTimes []float64 `json:"avg_process_times"` // 平均处理时间
}

// CategoryStats 分类统计
type CategoryStats struct {
	CategoryID     int     `json:"category_id"`      // 分类ID
	CategoryName   string  `json:"category_name"`    // 分类名称
	Count          int64   `json:"count"`            // 数量
	Percentage     float64 `json:"percentage"`       // 百分比
	CompletionRate float64 `json:"completion_rate"`  // 完成率
	AvgProcessTime float64 `json:"avg_process_time"` // 平均处理时间
}

// UserStats 用户统计
type UserStats struct {
	UserID            int     `json:"user_id"`             // 用户ID
	UserName          string  `json:"user_name"`           // 用户名
	AssignedCount     int64   `json:"assigned_count"`      // 分配数量
	CompletedCount    int64   `json:"completed_count"`     // 完成数量
	PendingCount      int64   `json:"pending_count"`       // 待处理数量
	CompletionRate    float64 `json:"completion_rate"`     // 完成率
	AvgResponseTime   float64 `json:"avg_response_time"`   // 平均响应时间
	AvgProcessingTime float64 `json:"avg_processing_time"` // 平均处理时间
	OverdueCount      int64   `json:"overdue_count"`       // 超时数量
}

// TemplateStats 模板统计
type TemplateStats struct {
	TemplateID        int     `json:"template_id"`         // 模板ID
	TemplateName      string  `json:"template_name"`       // 模板名称
	CategoryName      string  `json:"category_name"`       // 分类名称
	Count             int64   `json:"count"`               // 使用数量
	Percentage        float64 `json:"percentage"`          // 百分比
	CompletionRate    float64 `json:"completion_rate"`     // 完成率
	AvgProcessingTime float64 `json:"avg_processing_time"` // 平均处理时间
}

// StatusDistribution 状态分布
type StatusDistribution struct {
	Status     string  `json:"status"`     // 状态
	Count      int64   `json:"count"`      // 数量
	Percentage float64 `json:"percentage"` // 百分比
}

// PriorityDistribution 优先级分布
type PriorityDistribution struct {
	Priority   string  `json:"priority"`   // 优先级
	Count      int64   `json:"count"`      // 数量
	Percentage float64 `json:"percentage"` // 百分比
}

// ==================== 数据库实体 ====================

// WorkOrderStatistics 工单统计表
type WorkOrderStatistics struct {
	ID              int       `json:"id" gorm:"primaryKey"`
	Date            time.Time `json:"date" gorm:"index;not null"`
	TotalCount      int       `json:"total_count" gorm:"default:0"`
	CompletedCount  int       `json:"completed_count" gorm:"default:0"`
	ProcessingCount int       `json:"processing_count" gorm:"default:0"`
	PendingCount    int       `json:"pending_count" gorm:"default:0"`
	OverdueCount    int       `json:"overdue_count" gorm:"default:0"`
	AvgProcessTime  float64   `json:"avg_process_time" gorm:"default:0"`
	AvgResponseTime float64   `json:"avg_response_time" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WorkOrderStatistics) TableName() string {
	return "workorder_statistics"
}

// UserPerformance 用户绩效表
type UserPerformance struct {
	ID                int       `json:"id" gorm:"primaryKey"`
	UserID            int       `json:"user_id" gorm:"index;not null"`
	UserName          string    `json:"user_name" gorm:"not null"`
	Date              time.Time `json:"date" gorm:"index;not null"`
	AssignedCount     int       `json:"assigned_count" gorm:"default:0"`
	CompletedCount    int       `json:"completed_count" gorm:"default:0"`
	PendingCount      int       `json:"pending_count" gorm:"default:0"`
	OverdueCount      int       `json:"overdue_count" gorm:"default:0"`
	AvgResponseTime   float64   `json:"avg_response_time" gorm:"default:0"`
	AvgProcessingTime float64   `json:"avg_processing_time" gorm:"default:0"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (UserPerformance) TableName() string {
	return "workorder_user_performance"
}

// CategoryPerformance 分类绩效表
type CategoryPerformance struct {
	ID                int       `json:"id" gorm:"primaryKey"`
	CategoryID        int       `json:"category_id" gorm:"index;not null"`
	CategoryName      string    `json:"category_name" gorm:"not null"`
	Date              time.Time `json:"date" gorm:"index;not null"`
	TotalCount        int       `json:"total_count" gorm:"default:0"`
	CompletedCount    int       `json:"completed_count" gorm:"default:0"`
	OverdueCount      int       `json:"overdue_count" gorm:"default:0"`
	AvgProcessingTime float64   `json:"avg_processing_time" gorm:"default:0"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (CategoryPerformance) TableName() string {
	return "workorder_category_performance"
}

// TemplatePerformance 模板绩效表
type TemplatePerformance struct {
	ID                int       `json:"id" gorm:"primaryKey"`
	TemplateID        int       `json:"template_id" gorm:"index;not null"`
	TemplateName      string    `json:"template_name" gorm:"not null"`
	CategoryID        *int      `json:"category_id" gorm:"index"`
	Date              time.Time `json:"date" gorm:"index;not null"`
	UsageCount        int       `json:"usage_count" gorm:"default:0"`
	CompletedCount    int       `json:"completed_count" gorm:"default:0"`
	AvgProcessingTime float64   `json:"avg_processing_time" gorm:"default:0"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (TemplatePerformance) TableName() string {
	return "workorder_template_performance"
}
