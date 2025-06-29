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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type AuditDAO interface {
	CreateAuditLog(ctx context.Context, log *model.AuditLog) error
	BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error
	GetAuditLogByID(ctx context.Context, id int) (*model.AuditLog, error)
	ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (int64, []model.AuditLog, error)
	SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (int64, []model.AuditLog, error)
	GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error)
	DeleteAuditLog(ctx context.Context, id int) error
	BatchDeleteAuditLogs(ctx context.Context, ids []int) error
	ArchiveAuditLogs(ctx context.Context, startTime, endTime int64) error
}

type auditDAO struct {
	db *gorm.DB
}

func NewAuditDAO(db *gorm.DB) AuditDAO {
	return &auditDAO{db: db}
}

// CreateAuditLog 创建单条审计日志
func (d *auditDAO) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

// BatchCreateAuditLogs 批量创建审计日志 - 优化批次大小
func (d *auditDAO) BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	// 动态调整批次大小，大批量数据使用更大的批次
	batchSize := 100
	if len(logs) > 1000 {
		batchSize = 500
	}

	return d.db.WithContext(ctx).CreateInBatches(logs, batchSize).Error
}

// GetAuditLogByID 根据ID获取审计日志
func (d *auditDAO) GetAuditLogByID(ctx context.Context, id int) (*model.AuditLog, error) {
	var log model.AuditLog
	err := d.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// ListAuditLogs 获取审计日志列表 - 修复分页问题
func (d *auditDAO) ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (int64, []model.AuditLog, error) {
	var total int64
	var logs []model.AuditLog

	// 始终执行精确计数，确保分页数据准确
	countQuery := d.buildListQuery(ctx, req)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if total == 0 {
		return 0, logs, nil
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.Size

	// 验证偏移量是否超出范围
	if offset >= int(total) {
		return total, logs, nil // 返回空结果但保留总数
	}

	// 构建数据查询
	dataQuery := d.buildListQuery(ctx, req)

	// 优化大偏移量查询
	if offset > 5000 {
		// 使用游标分页优化
		err := d.efficientPaginationQuery(ctx, req, dataQuery, &logs)
		return total, logs, err
	}

	// 常规分页查询
	err := dataQuery.
		Select("id, user_id, trace_id, ip_address, http_method, endpoint, operation_type, target_type, target_id, status_code, duration, error_msg, created_at").
		Offset(offset).
		Limit(req.Size).
		Order("created_at DESC, id DESC"). // 添加id作为第二排序字段，确保结果稳定
		Find(&logs).Error

	return total, logs, err
}

// efficientPaginationQuery 优化的大偏移量分页查询
func (d *auditDAO) efficientPaginationQuery(ctx context.Context, req *model.ListAuditLogsRequest, baseQuery *gorm.DB, logs *[]model.AuditLog) error {
	offset := (req.Page - 1) * req.Size

	// 使用子查询获取ID，然后JOIN查询完整数据
	subQuery := baseQuery.
		Select("id").
		Offset(offset).
		Limit(req.Size).
		Order("created_at DESC, id DESC")

	return d.db.WithContext(ctx).
		Table("audit_logs").
		Select("audit_logs.*").
		Joins("JOIN (?) AS sub ON audit_logs.id = sub.id", subQuery).
		Order("audit_logs.created_at DESC, audit_logs.id DESC").
		Find(logs).Error
}

// SearchAuditLogs 搜索审计日志 - 修复搜索分页问题
func (d *auditDAO) SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (int64, []model.AuditLog, error) {
	var total int64
	var logs []model.AuditLog

	query := d.buildSearchQuery(ctx, req)

	// 计数查询
	countQuery := d.buildSearchQuery(ctx, req)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if total == 0 {
		return 0, logs, nil
	}

	// 验证分页参数
	offset := (req.Page - 1) * req.Size
	if offset >= int(total) {
		return total, logs, nil
	}

	// 分页查询
	err := query.
		Offset(offset).
		Limit(req.Size).
		Order("created_at DESC, id DESC").
		Find(&logs).Error

	return total, logs, err
}

// GetAuditStatistics 获取审计统计信息 - 优化统计查询性能
func (d *auditDAO) GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error) {
	stats := &model.AuditStatistics{}

	// 使用一个查询获取基础统计信息
	var basicStats struct {
		TotalCount  int64   `json:"total_count"`
		TodayCount  int64   `json:"today_count"`
		ErrorCount  int64   `json:"error_count"`
		AvgDuration float64 `json:"avg_duration"`
	}

	today := time.Now().Truncate(24 * time.Hour)

	// 优化：使用子查询一次性获取多个统计值
	err := d.db.WithContext(ctx).Raw(`
		  SELECT 
			  COUNT(*) as total_count,
			  COUNT(CASE WHEN created_at >= ? THEN 1 END) as today_count,
			  COUNT(CASE WHEN status_code >= 400 OR error_msg != '' THEN 1 END) as error_count,
			  AVG(duration) as avg_duration
		  FROM audit_logs
	  `, today).Scan(&basicStats).Error

	if err != nil {
		return nil, err
	}

	stats.TotalCount = basicStats.TotalCount
	stats.TodayCount = basicStats.TodayCount
	stats.ErrorCount = basicStats.ErrorCount
	stats.AvgDuration = basicStats.AvgDuration

	// 并发获取其他统计信息
	errChan := make(chan error, 4)

	// 操作类型分布
	go func() {
		var typeDistribution []model.TypeDistributionItem
		err := d.db.WithContext(ctx).Model(&model.AuditLog{}).
			Select("operation_type as type, COUNT(*) as count").
			Group("operation_type").
			Order("count DESC").
			Limit(10).
			Find(&typeDistribution).Error
		stats.TypeDistribution = typeDistribution
		errChan <- err
	}()

	// 状态码分布
	go func() {
		var statusDistribution []model.StatusDistributionItem
		err := d.db.WithContext(ctx).Model(&model.AuditLog{}).
			Select("status_code as status, COUNT(*) as count").
			Group("status_code").
			Order("count DESC").
			Limit(10).
			Find(&statusDistribution).Error
		stats.StatusDistribution = statusDistribution
		errChan <- err
	}()

	// 最近活动
	go func() {
		var recentActivity []model.RecentActivityItem
		err := d.db.WithContext(ctx).Model(&model.AuditLog{}).
			Select("CAST(UNIX_TIMESTAMP(created_at) AS SIGNED) as time, operation_type, user_id, target_type, status_code, duration").
			Order("created_at DESC").
			Limit(20).
			Find(&recentActivity).Error
		stats.RecentActivity = recentActivity
		errChan <- err
	}()

	// 24小时趋势
	go func() {
		var hourlyTrend []model.HourlyTrendItem
		err := d.db.WithContext(ctx).Model(&model.AuditLog{}).
			Select("HOUR(created_at) as hour, COUNT(*) as count").
			Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
			Group("HOUR(created_at)").
			Order("hour").
			Find(&hourlyTrend).Error
		stats.HourlyTrend = hourlyTrend
		errChan <- err
	}()

	// 等待所有goroutine完成
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}

	return stats, nil
}

// DeleteAuditLog 删除单条审计日志
func (d *auditDAO) DeleteAuditLog(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.AuditLog{}, id).Error
}

// BatchDeleteAuditLogs 批量删除审计日志 - 优化大批量删除
func (d *auditDAO) BatchDeleteAuditLogs(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	// 分批删除，避免锁表时间过长
	batchSize := 1000
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		if err := d.db.WithContext(ctx).Delete(&model.AuditLog{}, ids[i:end]).Error; err != nil {
			return err
		}
	}

	return nil
}

// ArchiveAuditLogs 归档审计日志 - 优化归档性能
func (d *auditDAO) ArchiveAuditLogs(ctx context.Context, startTime, endTime int64) error {
	start := time.Unix(startTime, 0)
	end := time.Unix(endTime, 0)

	// 分批处理，避免长时间锁表
	batchSize := 10000
	for {
		var count int64
		// 检查还有多少数据需要处理
		err := d.db.WithContext(ctx).Model(&model.AuditLog{}).
			Where("created_at BETWEEN ? AND ?", start, end).
			Count(&count).Error
		if err != nil {
			return err
		}

		if count == 0 {
			break
		}

		// 分批删除
		err = d.db.WithContext(ctx).
			Where("created_at BETWEEN ? AND ?", start, end).
			Limit(batchSize).
			Delete(&model.AuditLog{}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// buildSearchQuery 修复搜索查询构建
func (d *auditDAO) buildSearchQuery(ctx context.Context, req *model.SearchAuditLogsRequest) *gorm.DB {
	query := d.buildListQuery(ctx, &req.ListAuditLogsRequest)

	if req.Advanced != nil {
		adv := req.Advanced

		// 使用IN查询优化多值搜索
		if len(adv.IPAddressList) > 0 {
			query = query.Where("ip_address IN ?", adv.IPAddressList)
		}

		if len(adv.StatusCodeList) > 0 {
			query = query.Where("status_code IN ?", adv.StatusCodeList)
		}

		// 范围查询优化
		if adv.DurationMin > 0 {
			query = query.Where("duration >= ?", adv.DurationMin)
		}

		if adv.DurationMax > 0 {
			query = query.Where("duration <= ?", adv.DurationMax)
		}

		// 布尔查询优化
		if adv.HasError != nil {
			if *adv.HasError {
				query = query.Where("status_code >= 400 OR error_msg != ''")
			} else {
				query = query.Where("status_code < 400 AND (error_msg = '' OR error_msg IS NULL)")
			}
		}

		// LIKE查询放在最后，减少索引失效影响
		if adv.EndpointPattern != "" {
			query = query.Where("endpoint LIKE ?", "%"+adv.EndpointPattern+"%")
		}
	}

	return query
}

// buildListQuery 修复上下文问题和查询逻辑
func (d *auditDAO) buildListQuery(ctx context.Context, req *model.ListAuditLogsRequest) *gorm.DB {
	query := d.db.WithContext(ctx).Model(&model.AuditLog{}) // 使用传入的ctx

	// 时间范围查询
	if req.StartTime > 0 && req.EndTime > 0 {
		start := time.Unix(req.StartTime, 0)
		end := time.Unix(req.EndTime, 0)
		query = query.Where("created_at BETWEEN ? AND ?", start, end)
	} else if req.StartTime > 0 {
		query = query.Where("created_at >= ?", time.Unix(req.StartTime, 0))
	} else if req.EndTime > 0 {
		query = query.Where("created_at <= ?", time.Unix(req.EndTime, 0))
	}

	if req.OperationType != "" {
		query = query.Where("operation_type = ?", req.OperationType)
	}

	if req.StatusCode > 0 {
		query = query.Where("status_code = ?", req.StatusCode)
	}

	if req.TargetType != "" {
		query = query.Where("target_type = ?", req.TargetType)
	}

	// 模糊搜索（放在最后）
	if req.Search != "" {
		search := "%" + req.Search + "%"
		query = query.Where("trace_id LIKE ? OR endpoint LIKE ? OR error_msg LIKE ?", search, search, search)
	}

	return query
}
