package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	// "gorm.io/gorm/clause" // Not used in this version, but might be useful
)

// StatisticsDAO 定义了统计相关的数据访问对象接口
type StatisticsDAO interface {
	GetOverviewStats(ctx context.Context, startDate *time.Time, endDate *time.Time) (*model.OverviewStatsDAO, error)
	GetInstanceTrendStats(ctx context.Context, startDate time.Time, endDate time.Time, dimension string, categoryID *int) (*model.TrendStatsDAO, error)
	GetWorkloadByCategory(ctx context.Context, startDate *time.Time, endDate *time.Time, top *int) ([]model.CategoryStatsItemDAO, error)
	GetOperatorPerformance(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int, top *int) ([]model.PerformanceStatsItemDAO, error)
	GetStatsByUser(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int) (*model.UserStatsDAO, error)
}

type statisticsDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewStatisticsDAO 创建一个新的 StatisticsDAO 实例
func NewStatisticsDAO(db *gorm.DB, logger *zap.Logger) StatisticsDAO {
	return &statisticsDAO{
		db:     db,
		logger: logger,
	}
}

// dateRangeScope is a helper GORM scope for applying date filters
func dateRangeScope(startDate *time.Time, endDate *time.Time, tableName ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tablePrefix := ""
		if len(tableName) > 0 && tableName[0] != "" {
			tablePrefix = tableName[0] + "."
		}
		columnName := tablePrefix + "created_at"

		if startDate != nil && endDate != nil {
			// Ensure endDate includes the whole day
			endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
			return db.Where(columnName+" BETWEEN ? AND ?", startDate, endOfDay)
		}
		if startDate != nil {
			return db.Where(columnName+" >= ?", startDate)
		}
		if endDate != nil {
			endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
			return db.Where(columnName+" <= ?", endOfDay)
		}
		return db
	}
}

// GetOverviewStats 获取总览统计数据
func (dao *statisticsDAO) GetOverviewStats(ctx context.Context, startDate *time.Time, endDate *time.Time) (*model.OverviewStatsDAO, error) {
	dao.logger.Debug("开始获取总览统计数据 (DAO)", zap.Timep("startDate", startDate), zap.Timep("endDate", endDate))
	var overview model.OverviewStatsDAO

	// TotalCount
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Count(&overview.TotalCount).Error; err != nil {
		dao.logger.Error("获取总工单数失败", zap.Error(err))
		return nil, err
	}

	// CompletedCount
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("status = ?", model.InstanceStatusCompleted).Count(&overview.CompletedCount).Error; err != nil {
		dao.logger.Error("获取已完成工单数失败", zap.Error(err))
		return nil, err
	}

	// ProcessingCount
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("status = ?", model.InstanceStatusProcessing).Count(&overview.ProcessingCount).Error; err != nil {
		dao.logger.Error("获取处理中工单数失败", zap.Error(err))
		return nil, err
	}
	
	// PendingCount
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("status = ?", model.InstanceStatusPending).Count(&overview.PendingCount).Error; err != nil {
		dao.logger.Error("获取待处理工单数失败", zap.Error(err))
		return nil, err
	}

	// OverdueCount
	now := time.Now()
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("status NOT IN (?) AND due_date < ?",
		[]int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}, now).Count(&overview.OverdueCount).Error; err != nil {
		dao.logger.Error("获取已超时工单数失败", zap.Error(err))
		return nil, err
	}
	
	// CompletionRate calculation
	var relevantTotalForRate int64
	err := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Scopes(dateRangeScope(startDate, endDate)).
		Where("status NOT IN (?)", []int8{model.InstanceStatusDraft, model.InstanceStatusCancelled}). // Assuming rejected are still part of "attempted"
		Count(&relevantTotalForRate).Error
	if err != nil {
        dao.logger.Error("获取相关总数以计算完成率失败", zap.Error(err))
        return nil, err
    }

	if relevantTotalForRate > 0 {
		overview.CompletionRate = (float64(overview.CompletedCount) / float64(relevantTotalForRate)) * 100
	} else {
		overview.CompletionRate = 0
	}

	// AvgProcessTime (in hours) for completed instances
	type DurationResult struct {
		AvgDuration float64 // in seconds
	}
	var durationRes DurationResult
	// Assuming DB is SQLite. For other DBs, time difference function will change.
	// MySQL: AVG(TIMESTAMPDIFF(SECOND, created_at, completed_at))
	// PostgreSQL: AVG(EXTRACT(EPOCH FROM (completed_at - created_at)))
	timeDiffSQL := "AVG(CAST(strftime('%s', completed_at) - strftime('%s', created_at) AS REAL))"
	if dao.db.Dialector.Name() == "mysql" {
		timeDiffSQL = "AVG(TIMESTAMPDIFF(SECOND, created_at, completed_at))"
	} else if dao.db.Dialector.Name() == "postgres" {
		timeDiffSQL = "AVG(EXTRACT(EPOCH FROM (completed_at - created_at)))"
	}

	err = dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select(timeDiffSQL+" as avg_duration").
		Where("status = ?", model.InstanceStatusCompleted).
		Where("completed_at IS NOT NULL AND created_at IS NOT NULL").
		Scopes(dateRangeScope(startDate, endDate, "instance")). // Apply date range to created_at
		Scan(&durationRes).Error
	if err != nil {
		dao.logger.Error("获取平均处理时长失败", zap.Error(err))
		overview.AvgProcessTime = 0
	} else {
		overview.AvgProcessTime = durationRes.AvgDuration / 3600 // Convert seconds to hours
	}

	// TodayCreated & TodayCompleted
	todayLocation := time.Local // Or specific location if needed
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, todayLocation)
	todayEnd := todayStart.Add(24 * time.Hour)

	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Where("created_at >= ? AND created_at < ?", todayStart, todayEnd).Count(&overview.TodayCreated).Error; err != nil {
		dao.logger.Error("获取今日创建工单数失败", zap.Error(err))
		return nil, err
	}
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Where("status = ? AND completed_at >= ? AND completed_at < ?", model.InstanceStatusCompleted, todayStart, todayEnd).Count(&overview.TodayCompleted).Error; err != nil {
		dao.logger.Error("获取今日完成工单数失败", zap.Error(err))
		return nil, err
	}

	dao.logger.Debug("总览统计数据获取成功 (DAO)")
	return &overview, nil
}


// GetInstanceTrendStats 获取工单趋势统计
func (dao *statisticsDAO) GetInstanceTrendStats(ctx context.Context, startDate time.Time, endDate time.Time, dimension string, categoryID *int) (*model.TrendStatsDAO, error) {
	dao.logger.Debug("开始获取工单趋势统计 (DAO)", zap.Time("startDate", startDate), zap.Time("endDate", endDate), zap.String("dimension", dimension), zap.Intp("categoryID", categoryID))
	
	var results []struct {
		DateStr         string `gorm:"column:date_str"`
		CreatedCount    int    `gorm:"column:created_count"`
		CompletedCount  int    `gorm:"column:completed_count"`
		ProcessingCount int    `gorm:"column:processing_count"`
	}

	var dateSelectSQL string
	// Date functions vary by DB. These are examples.
	// GORM's gorm.Expr can be used for DB-specific functions.
	switch dao.db.Dialector.Name() {
	case "sqlite":
		switch dimension {
		case "day": dateSelectSQL = "DATE(created_at) as date_str"
		case "week": dateSelectSQL = "strftime('%Y-%W', created_at) as date_str" // ISO week might need more complex handling
		case "month": dateSelectSQL = "strftime('%Y-%m', created_at) as date_str"
		default: return nil, fmt.Errorf("无效的统计维度: %s", dimension)
		}
	case "mysql":
		switch dimension {
		case "day": dateSelectSQL = "DATE_FORMAT(created_at, '%Y-%m-%d') as date_str"
		case "week": dateSelectSQL = "DATE_FORMAT(created_at, '%x-%v') as date_str" // ISO week
		case "month": dateSelectSQL = "DATE_FORMAT(created_at, '%Y-%m') as date_str"
		default: return nil, fmt.Errorf("无效的统计维度: %s", dimension)
		}
	case "postgres":
		switch dimension {
		case "day": dateSelectSQL = "TO_CHAR(created_at, 'YYYY-MM-DD') as date_str"
		case "week": dateSelectSQL = "TO_CHAR(created_at, 'IYYY-IW') as date_str" // ISO week
		case "month": dateSelectSQL = "TO_CHAR(created_at, 'YYYY-MM') as date_str"
		default: return nil, fmt.Errorf("无效的统计维度: %s", dimension)
		}
	default:
		return nil, fmt.Errorf("不支持的数据库方言: %s", dao.db.Dialector.Name())
	}
	
	endOfDayForEndDate := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	query := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select(dateSelectSQL +
			", COUNT(*) as created_count" +
			", SUM(CASE WHEN status = " + fmt.Sprintf("%d", model.InstanceStatusCompleted) + " THEN 1 ELSE 0 END) as completed_count" +
			", SUM(CASE WHEN status = " + fmt.Sprintf("%d", model.InstanceStatusProcessing) + " THEN 1 ELSE 0 END) as processing_count").
		Where("created_at BETWEEN ? AND ?", startDate, endOfDayForEndDate)

	if categoryID != nil && *categoryID > 0 {
		query = query.Where("category_id = ?", *categoryID)
	}

	err := query.Group("date_str").Order("date_str ASC").Scan(&results).Error
	if err != nil {
		dao.logger.Error("获取工单趋势统计失败 (DAO)", zap.Error(err))
		return nil, err
	}

	resp := &model.TrendStatsDAO{
		Dates:            make([]string, len(results)),
		CreatedCounts:    make([]int, len(results)),
		CompletedCounts:  make([]int, len(results)),
		ProcessingCounts: make([]int, len(results)),
	}

	for i, res := range results {
		resp.Dates[i] = res.DateStr
		resp.CreatedCounts[i] = res.CreatedCount
		resp.CompletedCounts[i] = res.CompletedCount
		resp.ProcessingCounts[i] = res.ProcessingCount
	}

	dao.logger.Debug("工单趋势统计获取成功 (DAO)")
	return resp, nil
}

// GetWorkloadByCategory 按分类获取工单负载统计
func (dao *statisticsDAO) GetWorkloadByCategory(ctx context.Context, startDate *time.Time, endDate *time.Time, top *int) ([]model.CategoryStatsItemDAO, error) {
	dao.logger.Debug("开始按分类获取工单负载统计 (DAO)", zap.Timep("startDate", startDate), zap.Timep("endDate", endDate), zap.Intp("top", top))
	var results []model.CategoryStatsItemDAO

	query := dao.db.WithContext(ctx).Model(&model.Instance{}).
		Select("instance.category_id, category.name as category_name, COUNT(instance.id) as count").
		Joins("LEFT JOIN category ON category.id = instance.category_id").
		Scopes(dateRangeScope(startDate, endDate, "instance")).
		Group("instance.category_id, category.name")

	var totalInstancesForPercentage int64
	countQueryBase := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate, "instance"))
	if err := countQueryBase.Count(&totalInstancesForPercentage).Error; err != nil {
		dao.logger.Error("计算分类统计百分比的总工单数失败", zap.Error(err))
		return nil, err
	}

	query = query.Order("count DESC")
	if top != nil && *top > 0 {
		query = query.Limit(*top)
	}

	if err := query.Scan(&results).Error; err != nil {
		dao.logger.Error("按分类获取工单负载统计失败 (DAO)", zap.Error(err))
		return nil, err
	}
	
	for i := range results {
        if totalInstancesForPercentage > 0 {
            results[i].Percentage = (float64(results[i].Count) / float64(totalInstancesForPercentage)) * 100
        } else {
            results[i].Percentage = 0
        }
		// If category_id is nil (not joined properly or instance has no category)
		if results[i].CategoryID == 0 && results[i].CategoryName == "" {
			results[i].CategoryName = "未分类"
		}
    }

	dao.logger.Debug("按分类获取工单负载统计成功 (DAO)")
	return results, nil
}

// GetOperatorPerformance 获取操作员绩效统计
func (dao *statisticsDAO) GetOperatorPerformance(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int, top *int) ([]model.PerformanceStatsItemDAO, error) {
	dao.logger.Debug("开始获取操作员绩效统计 (DAO)", zap.Timep("startDate", startDate), zap.Timep("endDate", endDate), zap.Intp("userID", userID), zap.Intp("top", top))
    var results []model.PerformanceStatsItemDAO
	
	now := time.Now()
    query := dao.db.WithContext(ctx).Model(&model.Instance{}).
        Select("assignee_id as user_id" +
            ", COUNT(*) as assigned_count" +
            ", SUM(CASE WHEN status = " + fmt.Sprintf("%d", model.InstanceStatusCompleted) + " THEN 1 ELSE 0 END) as completed_count" +
            ", SUM(CASE WHEN status NOT IN (" + fmt.Sprintf("%d,%d,%d", model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected) + ") AND due_date < ? THEN 1 ELSE 0 END) as overdue_count", now).
        Where("assignee_id IS NOT NULL").
		Scopes(dateRangeScope(startDate, endDate, "instance")) // Filter by instance creation date for relevance

    if userID != nil && *userID > 0 {
        query = query.Where("assignee_id = ?", *userID)
    }

    query = query.Group("assignee_id").Order("completed_count DESC")

    if top != nil && *top > 0 {
        query = query.Limit(*top)
    }

    if err := query.Scan(&results).Error; err != nil {
        dao.logger.Error("获取操作员绩效统计失败 (DAO)", zap.Error(err))
        return nil, err
    }

    for i := range results {
        if results[i].AssignedCount > 0 {
            results[i].CompletionRate = (float64(results[i].CompletedCount) / float64(results[i].AssignedCount)) * 100
        }
        // AvgResponseTime and AvgProcessingTime need data from instance_flow
        // These are complex and would typically involve subqueries or more advanced calculations
        // For now, they will be returned as 0 from this DAO method and service can decide to populate further
        results[i].AvgResponseTime = 0   // Placeholder: Requires instance_flow analysis
        results[i].AvgProcessingTime = 0 // Placeholder: Requires instance_flow analysis
    }

    dao.logger.Debug("操作员绩效统计获取成功 (DAO)")
    return results, nil
}


// GetStatsByUser 获取用户个人相关的统计数据
func (dao *statisticsDAO) GetStatsByUser(ctx context.Context, startDate *time.Time, endDate *time.Time, userID *int) (*model.UserStatsDAO, error) {
	if userID == nil || *userID == 0 {
        return nil, fmt.Errorf("用户ID不能为空")
    }
	dao.logger.Debug("开始获取用户个人统计 (DAO)", zap.Timep("startDate", startDate), zap.Timep("endDate", endDate), zap.Intp("userID", userID))
	var stats model.UserStatsDAO
	stats.UserID = *userID
	
	now := time.Now()

	// CreatedCount
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("creator_id = ?", *userID).Count(&stats.CreatedCount).Error; err != nil {
		dao.logger.Error("获取用户创建工单数失败", zap.Error(err), zap.Int("userID", *userID))
		return nil, err
	}
	
	// AssignedCount
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("assignee_id = ?", *userID).Count(&stats.AssignedCount).Error; err != nil {
		dao.logger.Error("获取用户分配工单数失败", zap.Error(err), zap.Int("userID", *userID))
		return nil, err
	}

	// CompletedCount (completed by this user as assignee)
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("assignee_id = ? AND status = ?", *userID, model.InstanceStatusCompleted).Count(&stats.CompletedCount).Error; err != nil {
		dao.logger.Error("获取用户完成工单数失败", zap.Error(err), zap.Int("userID", *userID))
		return nil, err
	}

	// PendingCount (assigned to this user and in pending/processing status)
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("assignee_id = ? AND status IN (?)", *userID, []int8{model.InstanceStatusPending, model.InstanceStatusProcessing}).Count(&stats.PendingCount).Error; err != nil {
		dao.logger.Error("获取用户待处理工单数失败", zap.Error(err), zap.Int("userID", *userID))
		return nil, err
	}
	
	// OverdueCount (assigned to this user and overdue)
	if err := dao.db.WithContext(ctx).Model(&model.Instance{}).Scopes(dateRangeScope(startDate, endDate)).Where("assignee_id = ? AND status NOT IN (?) AND due_date < ?", 
		*userID, []int8{model.InstanceStatusCompleted, model.InstanceStatusCancelled, model.InstanceStatusRejected}, now).Count(&stats.OverdueCount).Error; err != nil {
		dao.logger.Error("获取用户超时工单数失败", zap.Error(err), zap.Int("userID", *userID))
		return nil, err
	}

	// AvgResponseTime, AvgProcessingTime, SatisfactionScore are placeholders
	stats.AvgResponseTime = 0
	stats.AvgProcessingTime = 0
	stats.SatisfactionScore = 0

	dao.logger.Debug("用户个人统计获取成功 (DAO)")
	return &stats, nil
}
