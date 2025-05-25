package model

import "time"

// ==================== 统计相关 ====================

// 统计请求结构
// OverviewStatsReq 概览统计请求
type OverviewStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
}

// TrendStatsReq 趋势统计请求
type TrendStatsReq struct {
	StartDate  time.Time `json:"start_date" form:"start_date" binding:"required"`
	EndDate    time.Time `json:"end_date" form:"end_date" binding:"required"`
	Dimension  string    `json:"dimension" form:"dimension" binding:"required,oneof=day week month"`
	CategoryID *int      `json:"category_id" form:"category_id"`
}

// CategoryStatsReq 分类统计请求
type CategoryStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	Top       int        `json:"top" form:"top" binding:"omitempty,min=5,max=20"`
}

// PerformanceStatsReq 绩效统计请求
type PerformanceStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	UserID    *int       `json:"user_id" form:"user_id"`
	Top       int        `json:"top" form:"top" binding:"omitempty,min=5,max=50"`
}

// UserStatsReq 用户统计请求
type UserStatsReq struct {
	StartDate *time.Time `json:"start_date" form:"start_date"`
	EndDate   *time.Time `json:"end_date" form:"end_date"`
	UserID    *int       `json:"user_id" form:"user_id"`
}

// 统计响应结构
// OverviewStatsResp 概览统计响应
type OverviewStatsResp struct {
	TotalCount      int64   `json:"total_count"`       // 总工单数
	CompletedCount  int64   `json:"completed_count"`   // 已完成工单数
	ProcessingCount int64   `json:"processing_count"`  // 处理中工单数
	PendingCount    int64   `json:"pending_count"`     // 待处理工单数
	OverdueCount    int64   `json:"overdue_count"`     // 超时工单数
	CompletionRate  float64 `json:"completion_rate"`   // 完成率
	AvgProcessTime  float64 `json:"avg_process_time"`  // 平均处理时间
	TodayCreated    int64   `json:"today_created"`     // 今日创建数
	TodayCompleted  int64   `json:"today_completed"`   // 今日完成数
}

// TrendStatsResp 趋势统计响应
type TrendStatsResp struct {
	Dates            []string `json:"dates"`             // 日期列表
	CreatedCounts    []int64  `json:"created_counts"`    // 创建数量列表
	CompletedCounts  []int64  `json:"completed_counts"`  // 完成数量列表
	ProcessingCounts []int64  `json:"processing_counts"` // 处理中数量列表
}

// CategoryStatsItem 分类统计项
type CategoryStatsItem struct {
	CategoryID   int     `json:"category_id"`   // 分类ID
	CategoryName string  `json:"category_name"` // 分类名称
	Count        int64   `json:"count"`         // 数量
	Percentage   float64 `json:"percentage"`    // 百分比
}

// CategoryStatsResp 分类统计响应
type CategoryStatsResp struct {
	Items      []CategoryStatsItem `json:"items"`      // 分类统计项列表
	TotalCount int64               `json:"total_count"` // 总数量
	Total      int64               `json:"total"`      // 总数（兼容字段）
	Count      int64               `json:"count"`      // 数量（兼容字段）
	Percentage float64             `json:"percentage"` // 百分比（修正为float64类型）
}

// PerformanceStatsItem 绩效统计项
type PerformanceStatsItem struct {
	UserID            int     `json:"user_id"`              // 用户ID
	UserName          string  `json:"user_name"`            // 用户名称
	AssignedCount     int64   `json:"assigned_count"`       // 分配数量
	CompletedCount    int64   `json:"completed_count"`      // 完成数量
	CompletionRate    float64 `json:"completion_rate"`      // 完成率
	AvgResponseTime   float64 `json:"avg_response_time"`    // 平均响应时间
	AvgProcessingTime float64 `json:"avg_processing_time"`  // 平均处理时间
	OverdueCount      int64   `json:"overdue_count"`        // 超时数量
	SatisfactionScore float64 `json:"satisfaction_score"`   // 满意度评分
}

// PerformanceStatsResp 绩效统计响应
type PerformanceStatsResp struct {
	Items             []PerformanceStatsItem `json:"items"`              // 绩效统计项列表
	UserID            int                    `json:"user_id"`             // 用户ID
	TotalAssigned     int64                  `json:"total_assigned"`      // 总分配数
	TotalCompleted    int64                  `json:"total_completed"`     // 总完成数
	TotalOverdue      int64                  `json:"total_overdue"`       // 总超时数
	AvgResponseTime   float64                `json:"avg_response_time"`   // 平均响应时间
	AvgProcessingTime float64                `json:"avg_processing_time"` // 平均处理时间
	CompletionRate    float64                `json:"completion_rate"`     // 总完成率
	CompletedCount    int64                  `json:"completed_count"`     // 总完成数（兼容字段）
	OverdueCount      int64                  `json:"overdue_count"`       // 总超时数（兼容字段）
	AssignedCount     int64                  `json:"assigned_count"`      // 总分配数（兼容字段）
}

// UserStatsResp 用户统计响应
type UserStatsResp struct {
	UserID            int     `json:"user_id"`              // 用户ID
	CreatedCount      int64   `json:"created_count"`        // 创建数量
	AssignedCount     int64   `json:"assigned_count"`       // 分配数量
	CompletedCount    int64   `json:"completed_count"`      // 完成数量
	PendingCount      int64   `json:"pending_count"`        // 待处理数量
	OverdueCount      int64   `json:"overdue_count"`        // 超时数量
	AvgResponseTime   float64 `json:"avg_response_time"`    // 平均响应时间
	AvgProcessingTime float64 `json:"avg_processing_time"`  // 平均处理时间
	SatisfactionScore float64 `json:"satisfaction_score"`   // 满意度评分
}

// ==================== 实体表定义（用于统计） ====================

// WorkOrderStatistics 工单统计实体（DAO层）
type WorkOrderStatistics struct {
	ID              int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	Date            time.Time `json:"date" gorm:"column:date;not null;index;comment:统计日期"`
	TotalCount      int       `json:"total_count" gorm:"column:total_count;not null;default:0;comment:工单总数"`
	CompletedCount  int       `json:"completed_count" gorm:"column:completed_count;not null;default:0;comment:已完成工单数"`
	ProcessingCount int       `json:"processing_count" gorm:"column:processing_count;not null;default:0;comment:处理中工单数"`
	PendingCount    int       `json:"pending_count" gorm:"column:pending_count;not null;default:0;comment:待处理工单数"`
	CanceledCount   int       `json:"canceled_count" gorm:"column:canceled_count;not null;default:0;comment:已取消工单数"`
	RejectedCount   int       `json:"rejected_count" gorm:"column:rejected_count;not null;default:0;comment:已拒绝工单数"`
	OverdueCount    int       `json:"overdue_count" gorm:"column:overdue_count;not null;default:0;comment:超时工单数"`
	AvgProcessTime  float64   `json:"avg_process_time" gorm:"column:avg_process_time;not null;default:0;comment:平均处理时间(小时)"`
	AvgResponseTime float64   `json:"avg_response_time" gorm:"column:avg_response_time;not null;default:0;comment:平均响应时间(小时)"`
	CategoryStats   string    `json:"category_stats" gorm:"column:category_stats;type:json;comment:分类统计JSON"`
	UserStats       string    `json:"user_stats" gorm:"column:user_stats;type:json;comment:用户统计JSON"`
	CreatedAt       time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
}

// TableName 指定工单统计表名
func (WorkOrderStatistics) TableName() string {
	return "work_order_statistics"
}

// UserPerformance 用户绩效实体（DAO层）
type UserPerformance struct {
	ID                int       `json:"id" gorm:"primaryKey;column:id;comment:主键ID"`
	UserID            int       `json:"user_id" gorm:"column:user_id;not null;index;comment:用户ID"`
	UserName          string    `json:"user_name" gorm:"column:user_name;not null;comment:用户姓名"`
	Date              time.Time `json:"date" gorm:"column:date;not null;index;comment:统计日期"`
	AssignedCount     int       `json:"assigned_count" gorm:"column:assigned_count;not null;default:0;comment:分配工单数"`
	CompletedCount    int       `json:"completed_count" gorm:"column:completed_count;not null;default:0;comment:完成工单数"`
	OverdueCount      int       `json:"overdue_count" gorm:"column:overdue_count;not null;default:0;comment:超时工单数"`
	AvgResponseTime   float64   `json:"avg_response_time" gorm:"column:avg_response_time;not null;default:0;comment:平均响应时间(小时)"`
	AvgProcessingTime float64   `json:"avg_processing_time" gorm:"column:avg_processing_time;not null;default:0;comment:平均处理时间(小时)"`
	SatisfactionScore float64   `json:"satisfaction_score" gorm:"column:satisfaction_score;default:0;comment:满意度评分"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at;not null;comment:更新时间"`
}

// TableName 指定用户绩效表名
func (UserPerformance) TableName() string {
	return "user_performance"
}
