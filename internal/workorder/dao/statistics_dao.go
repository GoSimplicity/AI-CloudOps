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

package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type StatisticsDAO interface {
	GetOverviewStats(ctx context.Context, req *model.StatsReq) (*model.OverviewStats, error)
	GetTrendStats(ctx context.Context, req *model.StatsReq) (*model.TrendStats, error)
	GetCategoryStats(ctx context.Context, req *model.StatsReq) ([]model.CategoryStats, error)
	GetUserStats(ctx context.Context, req *model.StatsReq) ([]model.UserStats, error)
	GetTemplateStats(ctx context.Context, req *model.StatsReq) ([]model.TemplateStats, error)
	GetStatusDistribution(ctx context.Context, req *model.StatsReq) ([]model.StatusDistribution, error)
	GetPriorityDistribution(ctx context.Context, req *model.StatsReq) ([]model.PriorityDistribution, error)
}

type statisticsDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewStatisticsDAO(db *gorm.DB, logger *zap.Logger) StatisticsDAO {
	return &statisticsDAO{
		db:     db,
		logger: logger,
	}
}

// GetOverviewStats 获取总览统计 - 基于统计表
func (d *statisticsDAO) GetOverviewStats(ctx context.Context, req *model.StatsReq) (*model.OverviewStats, error) {
	var result model.OverviewStats

	query := d.db.WithContext(ctx).Model(&model.WorkOrderStatistics{})
	query = d.applyStatisticsDateFilter(query, req.StartDate, req.EndDate)

	// 从统计表聚合数据
	err := query.Select(`
		 SUM(total_count) as total_count,
		 SUM(completed_count) as completed_count,
		 SUM(processing_count) as processing_count,
		 SUM(pending_count) as pending_count,
		 SUM(overdue_count) as overdue_count,
		 ROUND(AVG(avg_process_time), 2) as avg_process_time,
		 ROUND(AVG(avg_response_time), 2) as avg_response_time
	 `).Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("获取总览统计失败: %w", err)
	}

	// 计算完成率
	if result.TotalCount > 0 {
		result.CompletionRate = float64(result.CompletedCount) / float64(result.TotalCount) * 100
	}

	// 今日统计
	today := time.Now().Format("2006-01-02")
	var todayStats struct {
		TodayCreated   int64 `json:"today_created"`
		TodayCompleted int64 `json:"today_completed"`
	}

	err = d.db.WithContext(ctx).Model(&model.WorkOrderStatistics{}).
		Where("DATE(date) = ?", today).
		Select(`
			 COALESCE(total_count, 0) as today_created,
			 COALESCE(completed_count, 0) as today_completed
		 `).
		Scan(&todayStats).Error

	if err != nil {
		d.logger.Warn("获取今日统计失败", zap.Error(err))
	} else {
		result.TodayCreated = todayStats.TodayCreated
		result.TodayCompleted = todayStats.TodayCompleted
	}

	return &result, nil
}

// GetTrendStats 获取趋势统计 - 基于统计表
func (d *statisticsDAO) GetTrendStats(ctx context.Context, req *model.StatsReq) (*model.TrendStats, error) {
	if req.StartDate == nil || req.EndDate == nil {
		return nil, fmt.Errorf("趋势统计需要指定时间范围")
	}

	dateFormat := d.getStatisticsDateFormat(req.Dimension)

	query := d.db.WithContext(ctx).Model(&model.WorkOrderStatistics{})
	query = d.applyStatisticsDateFilter(query, req.StartDate, req.EndDate)

	var trendData []struct {
		Date           string  `json:"date"`
		CreatedCount   int64   `json:"created_count"`
		CompletedCount int64   `json:"completed_count"`
		CompletionRate float64 `json:"completion_rate"`
		AvgProcessTime float64 `json:"avg_process_time"`
	}

	// 修复GROUP BY问题 - 确保SELECT的列都在GROUP BY中
	err := query.Select(fmt.Sprintf(`
		%s as date,
		SUM(total_count) as created_count,
		SUM(completed_count) as completed_count,
		ROUND(SUM(completed_count) * 100.0 / NULLIF(SUM(total_count), 0), 2) as completion_rate,
		ROUND(AVG(avg_process_time), 2) as avg_process_time
	`, dateFormat)).
		Group(dateFormat). // 直接使用dateFormat作为GROUP BY
		Order("date").
		Scan(&trendData).Error

	if err != nil {
		return nil, fmt.Errorf("获取趋势统计失败: %w", err)
	}

	// 转换为返回格式
	result := &model.TrendStats{
		Dates:           make([]string, len(trendData)),
		CreatedCounts:   make([]int64, len(trendData)),
		CompletedCounts: make([]int64, len(trendData)),
		CompletionRates: make([]float64, len(trendData)),
		AvgProcessTimes: make([]float64, len(trendData)),
	}

	for i, data := range trendData {
		result.Dates[i] = data.Date
		result.CreatedCounts[i] = data.CreatedCount
		result.CompletedCounts[i] = data.CompletedCount
		result.CompletionRates[i] = data.CompletionRate
		result.AvgProcessTimes[i] = data.AvgProcessTime
	}

	return result, nil
}

// GetCategoryStats 获取分类统计 - 基于分类绩效表
func (d *statisticsDAO) GetCategoryStats(ctx context.Context, req *model.StatsReq) ([]model.CategoryStats, error) {
	query := d.db.WithContext(ctx).Model(&model.CategoryPerformance{})
	query = d.applyCategoryDateFilter(query, req.StartDate, req.EndDate)

	if req.CategoryID != nil {
		query = query.Where("category_id = ?", *req.CategoryID)
	}

	var categories []model.CategoryStats
	err := query.Select(`
		 category_id,
		 category_name,
		 SUM(total_count) as count,
		 ROUND(SUM(completed_count) * 100.0 / NULLIF(SUM(total_count), 0), 2) as completion_rate,
		 ROUND(AVG(avg_processing_time), 2) as avg_process_time
	 `).
		Group("category_id, category_name").
		Order("count DESC").
		Limit(req.Top).
		Scan(&categories).Error

	if err != nil {
		return nil, fmt.Errorf("获取分类统计失败: %w", err)
	}

	// 计算百分比
	var total int64
	for _, cat := range categories {
		total += cat.Count
	}

	for i := range categories {
		if total > 0 {
			categories[i].Percentage = float64(categories[i].Count) / float64(total) * 100
		}
	}

	return categories, nil
}

// GetUserStats 获取用户统计 - 基于用户绩效表
func (d *statisticsDAO) GetUserStats(ctx context.Context, req *model.StatsReq) ([]model.UserStats, error) {
	query := d.db.WithContext(ctx).Model(&model.UserPerformance{})
	query = d.applyUserDateFilter(query, req.StartDate, req.EndDate)

	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}

	var users []model.UserStats
	err := query.Select(`
		 user_id,
		 user_name,
		 SUM(assigned_count) as assigned_count,
		 SUM(completed_count) as completed_count,
		 SUM(pending_count) as pending_count,
		 SUM(overdue_count) as overdue_count,
		 ROUND(SUM(completed_count) * 100.0 / NULLIF(SUM(assigned_count), 0), 2) as completion_rate,
		 ROUND(AVG(avg_response_time), 2) as avg_response_time,
		 ROUND(AVG(avg_processing_time), 2) as avg_processing_time
	 `).
		Group("user_id, user_name").
		Order(d.getUserSortField(req.SortBy)).
		Limit(req.Top).
		Scan(&users).Error

	if err != nil {
		return nil, fmt.Errorf("获取用户统计失败: %w", err)
	}

	return users, nil
}

// GetTemplateStats 获取模板统计 - 基于模板绩效表
func (d *statisticsDAO) GetTemplateStats(ctx context.Context, req *model.StatsReq) ([]model.TemplateStats, error) {
	query := d.db.WithContext(ctx).Model(&model.TemplatePerformance{})
	query = d.applyTemplateeDateFilter(query, req.StartDate, req.EndDate)

	if req.CategoryID != nil {
		query = query.Where("category_id = ?", *req.CategoryID)
	}

	var templates []model.TemplateStats
	err := query.Select(`
		 template_id,
		 template_name,
		 '' as category_name,
		 SUM(usage_count) as count,
		 ROUND(SUM(completed_count) * 100.0 / NULLIF(SUM(usage_count), 0), 2) as completion_rate,
		 ROUND(AVG(avg_processing_time), 2) as avg_processing_time
	 `).
		Group("template_id, template_name").
		Order("count DESC").
		Limit(req.Top).
		Scan(&templates).Error

	if err != nil {
		return nil, fmt.Errorf("获取模板统计失败: %w", err)
	}

	// 获取分类名称
	for i := range templates {
		var categoryName string
		d.db.WithContext(ctx).Model(&model.TemplatePerformance{}).
			Where("template_id = ?", templates[i].TemplateID).
			Joins("LEFT JOIN workorder_categories c ON workorder_template_performance.category_id = c.id").
			Select("COALESCE(c.name, '未分类')").
			Limit(1).
			Scan(&categoryName)
		templates[i].CategoryName = categoryName
	}

	// 计算百分比
	var total int64
	for _, tmpl := range templates {
		total += tmpl.Count
	}

	for i := range templates {
		if total > 0 {
			templates[i].Percentage = float64(templates[i].Count) / float64(total) * 100
		}
	}

	return templates, nil
}

// GetStatusDistribution 获取状态分布
func (d *statisticsDAO) GetStatusDistribution(ctx context.Context, req *model.StatsReq) ([]model.StatusDistribution, error) {
	query := d.db.WithContext(ctx).Model(&model.Instance{})
	query = d.applyWorkOrderDateFilter(query, req.StartDate, req.EndDate)
	query = d.applyWorkOrderCommonFilters(query, req)

	var distribution []model.StatusDistribution
	err := query.Select(`
		 status,
		 COUNT(*) as count
	 `).
		Group("status").
		Order("count DESC").
		Scan(&distribution).Error

	if err != nil {
		return nil, fmt.Errorf("获取状态分布失败: %w", err)
	}

	// 计算百分比
	var total int64
	for _, item := range distribution {
		total += item.Count
	}

	for i := range distribution {
		if total > 0 {
			distribution[i].Percentage = float64(distribution[i].Count) / float64(total) * 100
		}
	}

	return distribution, nil
}

// GetPriorityDistribution 获取优先级分布
func (d *statisticsDAO) GetPriorityDistribution(ctx context.Context, req *model.StatsReq) ([]model.PriorityDistribution, error) {
	query := d.db.WithContext(ctx).Model(&model.Instance{})
	query = d.applyWorkOrderDateFilter(query, req.StartDate, req.EndDate)
	query = d.applyWorkOrderCommonFilters(query, req)

	var distribution []model.PriorityDistribution
	err := query.Select(`
		 priority,
		 COUNT(*) as count
	 `).
		Group("priority").
		Order("count DESC").
		Scan(&distribution).Error

	if err != nil {
		return nil, fmt.Errorf("获取优先级分布失败: %w", err)
	}

	// 计算百分比
	var total int64
	for _, item := range distribution {
		total += item.Count
	}

	for i := range distribution {
		if total > 0 {
			distribution[i].Percentage = float64(distribution[i].Count) / float64(total) * 100
		}
	}

	return distribution, nil
}

// 统计表日期过滤
func (d *statisticsDAO) applyStatisticsDateFilter(query *gorm.DB, startDate, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		query = query.Where("date >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("date <= ?", endDate.Format("2006-01-02"))
	}
	return query
}

// 分类绩效表日期过滤
func (d *statisticsDAO) applyCategoryDateFilter(query *gorm.DB, startDate, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		query = query.Where("date >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("date <= ?", endDate.Format("2006-01-02"))
	}
	return query
}

// 用户绩效表日期过滤
func (d *statisticsDAO) applyUserDateFilter(query *gorm.DB, startDate, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		query = query.Where("date >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("date <= ?", endDate.Format("2006-01-02"))
	}
	return query
}

// 模板绩效表日期过滤
func (d *statisticsDAO) applyTemplateeDateFilter(query *gorm.DB, startDate, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		query = query.Where("date >= ?", startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query = query.Where("date <= ?", endDate.Format("2006-01-02"))
	}
	return query
}

// 工单表日期过滤（用于状态和优先级分布）
func (d *statisticsDAO) applyWorkOrderDateFilter(query *gorm.DB, startDate, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", endDate)
	}
	return query
}

// 工单表通用过滤条件
func (d *statisticsDAO) applyWorkOrderCommonFilters(query *gorm.DB, req *model.StatsReq) *gorm.DB {
	if req.CategoryID != nil {
		query = query.Where("category_id = ?", *req.CategoryID)
	}
	if req.UserID != nil {
		query = query.Where("assigned_to = ?", *req.UserID)
	}
	if req.Status != nil && *req.Status != "" {
		query = query.Where("status = ?", *req.Status)
	}
	if req.Priority != nil && *req.Priority != "" {
		query = query.Where("priority = ?", *req.Priority)
	}
	return query
}

func (d *statisticsDAO) getStatisticsDateFormat(dimension string) string {
	switch dimension {
	case "day":
		return "DATE(date)"
	case "week":
		return "DATE_FORMAT(date, '%Y-%u')"
	case "month":
		return "DATE_FORMAT(date, '%Y-%m')"
	default:
		return "DATE(date)"
	}
}

// 统计表分组字段
func (d *statisticsDAO) getStatisticsGroupBy(dimension string) string {
	switch dimension {
	case "day":
		return "DATE(date)"
	case "week":
		return "YEAR(date), WEEK(date)"
	case "month":
		return "YEAR(date), MONTH(date)"
	default:
		return "DATE(date)"
	}
}

// 获取用户排序字段
func (d *statisticsDAO) getUserSortField(sortBy string) string {
	switch sortBy {
	case "completion_rate":
		return "completion_rate DESC"
	case "avg_process_time":
		return "avg_processing_time ASC"
	default:
		return "assigned_count DESC"
	}
}
