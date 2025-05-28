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
	GetOverviewStats(ctx context.Context, startDate *time.Time, endDate *time.Time) (*model.OverviewStatsResp, error)
	GetInstanceTrendStats(ctx context.Context, startDate time.Time, endDate time.Time, dimension string, categoryID *int) (*model.TrendStatsResp, error)
	GetWorkloadByCategory(ctx context.Context, startDate *time.Time, endDate *time.Time, top *int) ([]model.CategoryStatsResp, error)
	GetOperatorPerformance(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int, top *int) ([]model.PerformanceStatsResp, error)
	GetStatsByUser(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int) (*model.UserStatsResp, error)
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

// SQLDialect 数据库方言类型
type SQLDialect string

const (
	SQLiteDialect   SQLDialect = "sqlite"
	MySQLDialect    SQLDialect = "mysql"
	PostgresDialect SQLDialect = "postgres"
)

// getSQLDialect 获取当前数据库方言
func (dao *statisticsDAO) getSQLDialect() SQLDialect {
	switch dao.db.Dialector.Name() {
	case "mysql":
		return MySQLDialect
	case "postgres":
		return PostgresDialect
	default:
		return SQLiteDialect
	}
}

// buildDateSelectSQL 构建日期选择SQL
func (dao *statisticsDAO) buildDateSelectSQL(dimension string) (string, error) {
	dialect := dao.getSQLDialect()

	switch dialect {
	case SQLiteDialect:
		return dao.buildSQLiteDateSQL(dimension)
	case MySQLDialect:
		return dao.buildMySQLDateSQL(dimension)
	case PostgresDialect:
		return dao.buildPostgresDateSQL(dimension)
	default:
		return "", fmt.Errorf("不支持的数据库方言: %s", dialect)
	}
}

// buildSQLiteDateSQL SQLite日期SQL
func (dao *statisticsDAO) buildSQLiteDateSQL(dimension string) (string, error) {
	switch dimension {
	case "day":
		return "DATE(created_at) as date_str", nil
	case "week":
		return "strftime('%Y-%W', created_at) as date_str", nil
	case "month":
		return "strftime('%Y-%m', created_at) as date_str", nil
	default:
		return "", fmt.Errorf("无效的统计维度: %s", dimension)
	}
}

// buildMySQLDateSQL MySQL日期SQL
func (dao *statisticsDAO) buildMySQLDateSQL(dimension string) (string, error) {
	switch dimension {
	case "day":
		return "DATE_FORMAT(created_at, '%Y-%m-%d') as date_str", nil
	case "week":
		return "DATE_FORMAT(created_at, '%x-%v') as date_str", nil
	case "month":
		return "DATE_FORMAT(created_at, '%Y-%m') as date_str", nil
	default:
		return "", fmt.Errorf("无效的统计维度: %s", dimension)
	}
}

// buildPostgresDateSQL PostgreSQL日期SQL
func (dao *statisticsDAO) buildPostgresDateSQL(dimension string) (string, error) {
	switch dimension {
	case "day":
		return "TO_CHAR(created_at, 'YYYY-MM-DD') as date_str", nil
	case "week":
		return "TO_CHAR(created_at, 'IYYY-IW') as date_str", nil
	case "month":
		return "TO_CHAR(created_at, 'YYYY-MM') as date_str", nil
	default:
		return "", fmt.Errorf("无效的统计维度: %s", dimension)
	}
}

// buildTimeDifferenceSQL 构建时间差SQL
func (dao *statisticsDAO) buildTimeDifferenceSQL() string {
	dialect := dao.getSQLDialect()

	switch dialect {
	case MySQLDialect:
		return "AVG(TIMESTAMPDIFF(SECOND, created_at, completed_at))"
	case PostgresDialect:
		return "AVG(EXTRACT(EPOCH FROM (completed_at - created_at)))"
	default: // SQLite
		return "AVG(CAST(strftime('%s', completed_at) - strftime('%s', created_at) AS REAL))"
	}
}

// dateRangeScope 日期范围查询作用域
func dateRangeScope(startDate *time.Time, endDate *time.Time, tableName ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tablePrefix := ""
		if len(tableName) > 0 && tableName[0] != "" {
			tablePrefix = tableName[0] + "."
		}
		columnName := tablePrefix + "created_at"

		if startDate != nil && endDate != nil {
			endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
			return db.Where(columnName+" BETWEEN ? AND ?", *startDate, endOfDay)
		}
		if startDate != nil {
			return db.Where(columnName+" >= ?", *startDate)
		}
		if endDate != nil {
			endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
			return db.Where(columnName+" <= ?", endOfDay)
		}
		return db
	}
}

// todayRange 获取今日时间范围
func todayRange() (time.Time, time.Time) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)
	return start, end
}

// getStatusCounts 获取各状态工单数量的通用方法
func (dao *statisticsDAO) getStatusCounts(ctx context.Context, startDate *time.Time, endDate *time.Time) (map[string]int64, error) {
	var results []struct {
		Status int8  `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}

	err := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select("status, COUNT(*) as count").
		Scopes(dateRangeScope(startDate, endDate)).
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		switch result.Status {
		case model.InstanceStatusCompleted:
			counts["completed"] = result.Count
		case model.InstanceStatusProcessing:
			counts["processing"] = result.Count
		case model.InstanceStatusPending:
			counts["pending"] = result.Count
		}
	}

	return counts, nil
}

// GetOverviewStats 获取总览统计数据
func (dao *statisticsDAO) GetOverviewStats(ctx context.Context, startDate *time.Time, endDate *time.Time) (*model.OverviewStatsResp, error) {
	dao.logger.Debug("开始获取总览统计数据",
		zap.Timep("startDate", startDate),
		zap.Timep("endDate", endDate))

	overview := &model.OverviewStatsResp{}
	now := time.Now()

	// 使用事务确保数据一致性
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取总工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Count(&overview.TotalCount).Error; err != nil {
			return fmt.Errorf("获取总工单数失败: %w", err)
		}

		// 获取各状态统计
		statusCounts, err := dao.getStatusCountsInTx(ctx, tx, startDate, endDate)
		if err != nil {
			return fmt.Errorf("获取状态统计失败: %w", err)
		}

		overview.CompletedCount = statusCounts["completed"]
		overview.ProcessingCount = statusCounts["processing"]
		overview.PendingCount = statusCounts["pending"]

		// 获取超时工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("status NOT IN ? AND due_date < ?",
				[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}, now).
			Count(&overview.OverdueCount).Error; err != nil {
			return fmt.Errorf("获取超时工单数失败: %w", err)
		}

		// 计算完成率
		var relevantTotal int64
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("status NOT IN ?", []int8{model.InstanceStatusDraft, model.InstanceStatusCancelled}).
			Count(&relevantTotal).Error; err != nil {
			return fmt.Errorf("获取相关总数失败: %w", err)
		}

		if relevantTotal > 0 {
			overview.CompletionRate = (float64(overview.CompletedCount) / float64(relevantTotal)) * 100
		}

		// 计算平均处理时长
		avgTime, err := dao.getAvgProcessTimeInTx(ctx, tx, startDate, endDate)
		if err != nil {
			dao.logger.Warn("获取平均处理时长失败", zap.Error(err))
		} else {
			overview.AvgProcessTime = avgTime
		}

		// 获取今日统计
		todayStart, todayEnd := todayRange()
		if err := tx.Model(&model.Instance{}).
			Where("created_at >= ? AND created_at < ?", todayStart, todayEnd).
			Count(&overview.TodayCreated).Error; err != nil {
			return fmt.Errorf("获取今日创建工单数失败: %w", err)
		}

		if err := tx.Model(&model.Instance{}).
			Where("status = ? AND completed_at >= ? AND completed_at < ?",
				model.InstanceStatusCompleted, todayStart, todayEnd).
			Count(&overview.TodayCompleted).Error; err != nil {
			return fmt.Errorf("获取今日完成工单数失败: %w", err)
		}

		return nil
	})

	if err != nil {
		dao.logger.Error("获取总览统计数据失败", zap.Error(err))
		return nil, err
	}

	dao.logger.Debug("总览统计数据获取成功")
	return overview, nil
}

// getStatusCountsInTx 在事务中获取状态统计
func (dao *statisticsDAO) getStatusCountsInTx(ctx context.Context, tx *gorm.DB, startDate *time.Time, endDate *time.Time) (map[string]int64, error) {
	var results []struct {
		Status int8  `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}

	err := tx.WithContext(ctx).Model(&model.Instance{}).
		Select("status, COUNT(*) as count").
		Scopes(dateRangeScope(startDate, endDate)).
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, result := range results {
		switch result.Status {
		case model.InstanceStatusCompleted:
			counts["completed"] = result.Count
		case model.InstanceStatusProcessing:
			counts["processing"] = result.Count
		case model.InstanceStatusPending:
			counts["pending"] = result.Count
		}
	}

	return counts, nil
}

// getAvgProcessTimeInTx 在事务中获取平均处理时长
func (dao *statisticsDAO) getAvgProcessTimeInTx(ctx context.Context, tx *gorm.DB, startDate *time.Time, endDate *time.Time) (float64, error) {
	type DurationResult struct {
		AvgDuration float64 `gorm:"column:avg_duration"`
	}

	var result DurationResult
	timeDiffSQL := dao.buildTimeDifferenceSQL()

	err := tx.WithContext(ctx).Model(&model.Instance{}).
		Select(timeDiffSQL+" as avg_duration").
		Where("status = ? AND completed_at IS NOT NULL AND created_at IS NOT NULL",
			model.InstanceStatusCompleted).
		Scopes(dateRangeScope(startDate, endDate)).
		Scan(&result).Error

	if err != nil {
		return 0, err
	}

	return result.AvgDuration / 3600, nil // 转换为小时
}

// GetInstanceTrendStats 获取工单趋势统计
func (dao *statisticsDAO) GetInstanceTrendStats(ctx context.Context, startDate time.Time, endDate time.Time, dimension string, categoryID *int) (*model.TrendStatsResp, error) {
	dao.logger.Debug("开始获取工单趋势统计",
		zap.Time("startDate", startDate),
		zap.Time("endDate", endDate),
		zap.String("dimension", dimension),
		zap.Intp("categoryID", categoryID))

	// 构建日期选择SQL
	dateSelectSQL, err := dao.buildDateSelectSQL(dimension)
	if err != nil {
		return nil, err
	}

	var results []struct {
		DateStr         string `gorm:"column:date_str"`
		CreatedCount    int64  `gorm:"column:created_count"`
		CompletedCount  int64  `gorm:"column:completed_count"`
		ProcessingCount int64  `gorm:"column:processing_count"`
	}

	endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	query := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select(dateSelectSQL+
			", COUNT(*) as created_count"+
			", SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as completed_count"+
			", SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as processing_count",
			model.InstanceStatusCompleted, model.InstanceStatusProcessing).
		Where("created_at BETWEEN ? AND ?", startDate, endOfDay)

	if categoryID != nil && *categoryID > 0 {
		query = query.Where("category_id = ?", *categoryID)
	}

	err = query.Group("date_str").Order("date_str ASC").Scan(&results).Error
	if err != nil {
		dao.logger.Error("获取工单趋势统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单趋势统计失败: %w", err)
	}

	// 构建响应
	resp := &model.TrendStatsResp{
		Dates:            make([]string, len(results)),
		CreatedCounts:    make([]int64, len(results)),
		CompletedCounts:  make([]int64, len(results)),
		ProcessingCounts: make([]int64, len(results)),
	}

	for i, res := range results {
		resp.Dates[i] = res.DateStr
		resp.CreatedCounts[i] = res.CreatedCount
		resp.CompletedCounts[i] = res.CompletedCount
		resp.ProcessingCounts[i] = res.ProcessingCount
	}

	dao.logger.Debug("工单趋势统计获取成功")
	return resp, nil
}

// GetWorkloadByCategory 按分类获取工单负载统计
func (dao *statisticsDAO) GetWorkloadByCategory(ctx context.Context, startDate *time.Time, endDate *time.Time, top *int) ([]model.CategoryStatsResp, error) {
	dao.logger.Debug("开始按分类获取工单负载统计",
		zap.Timep("startDate", startDate),
		zap.Timep("endDate", endDate),
		zap.Intp("top", top))

	var results []model.CategoryStatsResp

	// 获取总工单数用于计算百分比
	var totalCount int64
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Scopes(dateRangeScope(startDate, endDate)).
		Count(&totalCount).Error; err != nil {
		dao.logger.Error("获取总工单数失败", zap.Error(err))
		return nil, fmt.Errorf("获取总工单数失败: %w", err)
	}

	// 构建查询
	query := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select("instance.category_id, COALESCE(category.name, '未分类') as category_name, COUNT(instance.id) as count").
		Joins("LEFT JOIN category ON category.id = instance.category_id").
		Scopes(dateRangeScope(startDate, endDate, "instance")).
		Group("instance.category_id, category.name").
		Order("count DESC")

	if top != nil && *top > 0 {
		query = query.Limit(*top)
	}

	if err := query.Scan(&results).Error; err != nil {
		dao.logger.Error("按分类获取工单负载统计失败", zap.Error(err))
		return nil, fmt.Errorf("按分类获取工单负载统计失败: %w", err)
	}



	dao.logger.Debug("按分类获取工单负载统计成功")
	return results, nil
}

// GetOperatorPerformance 获取操作员绩效统计
func (dao *statisticsDAO) GetOperatorPerformance(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int, top *int) ([]model.PerformanceStatsResp, error) {
	dao.logger.Debug("开始获取操作员绩效统计",
		zap.Timep("startDate", startDate),
		zap.Timep("endDate", endDate),
		zap.Intp("userID", userID),
		zap.Intp("top", top))

	var results []model.PerformanceStatsResp
	now := time.Now()

	query := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select(`assignee_id as user_id,
				 COUNT(*) as assigned_count,
				 SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as completed_count,
				 SUM(CASE WHEN status NOT IN (?, ?, ?) AND due_date < ? THEN 1 ELSE 0 END) as overdue_count`,
			model.InstanceStatusCompleted,
			model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected, now).
		Where("assignee_id IS NOT NULL").
		Scopes(dateRangeScope(startDate, endDate))

	if userID != nil && *userID > 0 {
		query = query.Where("assignee_id = ?", *userID)
	}

	query = query.Group("assignee_id").Order("completed_count DESC")

	if top != nil && *top > 0 {
		query = query.Limit(*top)
	}

	if err := query.Scan(&results).Error; err != nil {
		dao.logger.Error("获取操作员绩效统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取操作员绩效统计失败: %w", err)
	}

	// 计算完成率
	for i := range results {
		if results[i].AssignedCount > 0 {
			results[i].CompletionRate = (float64(results[i].CompletedCount) / float64(results[i].AssignedCount)) * 100
		}
		// TODO: 实现平均响应时间和处理时间的计算
		results[i].AvgResponseTime = 0
		results[i].AvgProcessingTime = 0
	}

	dao.logger.Debug("操作员绩效统计获取成功")
	return results, nil
}

// GetStatsByUser 获取用户个人相关的统计数据
func (dao *statisticsDAO) GetStatsByUser(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int) (*model.UserStatsResp, error) {
	if userID == nil || *userID == 0 {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	dao.logger.Debug("开始获取用户个人统计",
		zap.Timep("startDate", startDate),
		zap.Timep("endDate", endDate),
		zap.Int("userID", *userID))

	stats := &model.UserStatsResp{
		UserID: *userID,
	}
	now := time.Now()

	// 使用事务确保数据一致性
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建的工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("creator_id = ?", *userID).
			Count(&stats.CreatedCount).Error; err != nil {
			return fmt.Errorf("获取用户创建工单数失败: %w", err)
		}

		// 分配的工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("assignee_id = ?", *userID).
			Count(&stats.AssignedCount).Error; err != nil {
			return fmt.Errorf("获取用户分配工单数失败: %w", err)
		}

		// 完成的工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("assignee_id = ? AND status = ?", *userID, model.InstanceStatusCompleted).
			Count(&stats.CompletedCount).Error; err != nil {
			return fmt.Errorf("获取用户完成工单数失败: %w", err)
		}

		// 待处理工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("assignee_id = ? AND status IN ?", *userID,
				[]int8{model.InstanceStatusPending, model.InstanceStatusProcessing}).
			Count(&stats.PendingCount).Error; err != nil {
			return fmt.Errorf("获取用户待处理工单数失败: %w", err)
		}

		// 超时工单数
		if err := tx.Model(&model.Instance{}).
			Scopes(dateRangeScope(startDate, endDate)).
			Where("assignee_id = ? AND status NOT IN ? AND due_date < ?",
				*userID, []int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}, now).
			Count(&stats.OverdueCount).Error; err != nil {
			return fmt.Errorf("获取用户超时工单数失败: %w", err)
		}

		return nil
	})

	if err != nil {
		dao.logger.Error("获取用户个人统计失败", zap.Error(err), zap.Int("userID", *userID))
		return nil, err
	}

	// TODO: 实现平均响应时间、处理时间和满意度评分的计算
	stats.AvgResponseTime = 0
	stats.AvgProcessingTime = 0
	stats.SatisfactionScore = 0

	dao.logger.Debug("用户个人统计获取成功")
	return stats, nil
}
