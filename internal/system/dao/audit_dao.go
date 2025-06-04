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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
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
	ExportAuditLogs(ctx context.Context, req *model.ExportAuditLogsRequest) ([]byte, error)
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

func (d *auditDAO) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

func (d *auditDAO) BatchCreateAuditLogs(ctx context.Context, logs []model.AuditLog) error {
	// 使用批量插入优化性能
	return d.db.WithContext(ctx).CreateInBatches(logs, 500).Error
}

func (d *auditDAO) GetAuditLogByID(ctx context.Context, id int) (*model.AuditLog, error) {
	var log model.AuditLog
	err := d.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (d *auditDAO) ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (int64, []model.AuditLog, error) {
	var total int64
	var logs []model.AuditLog

	query := d.buildListQuery(req)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).
		Order("created_at DESC").Find(&logs).Error; err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}

func (d *auditDAO) SearchAuditLogs(ctx context.Context, req *model.SearchAuditLogsRequest) (int64, []model.AuditLog, error) {
	var total int64
	var logs []model.AuditLog

	query := d.buildSearchQuery(req)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Offset(offset).Limit(req.Size).
		Order("created_at DESC").Find(&logs).Error; err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}

func (d *auditDAO) GetAuditStatistics(ctx context.Context) (*model.AuditStatistics, error) {
	stats := &model.AuditStatistics{}

	// 获取总数
	d.db.WithContext(ctx).Model(&model.AuditLog{}).Count(&stats.TotalCount)

	// 获取今日数量
	today := time.Now().Truncate(24 * time.Hour)
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Where("created_at >= ?", today).Count(&stats.TodayCount)

	// 获取错误数量
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Where("status_code >= 400 OR error_msg != ''").Count(&stats.ErrorCount)

	// 获取平均耗时
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("AVG(duration)").Row().Scan(&stats.AvgDuration)

	// 获取操作类型分布
	var typeDistribution []model.TypeDistributionItem
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("operation_type as type, COUNT(*) as count").
		Group("operation_type").
		Order("count DESC").
		Limit(10).
		Find(&typeDistribution)
	stats.TypeDistribution = typeDistribution

	// 获取状态码分布
	var statusDistribution []model.StatusDistributionItem
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("status_code as status, COUNT(*) as count").
		Group("status_code").
		Order("count DESC").
		Limit(10).
		Find(&statusDistribution)
	stats.StatusDistribution = statusDistribution

	// 获取最近活动
	var recentActivity []model.RecentActivityItem
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("CAST(UNIX_TIMESTAMP(created_at) AS SIGNED) as time, operation_type, user_id, target_type, status_code, duration").
		Order("created_at DESC").
		Limit(20).
		Find(&recentActivity)
	stats.RecentActivity = recentActivity

	// 获取24小时趋势
	var hourlyTrend []model.HourlyTrendItem
	d.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("HOUR(created_at) as hour, COUNT(*) as count").
		Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
		Group("HOUR(created_at)").
		Order("hour").
		Find(&hourlyTrend)
	stats.HourlyTrend = hourlyTrend

	return stats, nil
}

func (d *auditDAO) ExportAuditLogs(ctx context.Context, req *model.ExportAuditLogsRequest) ([]byte, error) {
	var logs []model.AuditLog

	query := d.buildListQuery(&req.ListAuditLogsRequest)
	if req.MaxRows > 0 {
		query = query.Limit(req.MaxRows)
	}

	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	switch req.Format {
	case "json":
		return json.Marshal(logs)
	case "csv":
		return d.exportAsCSV(logs, req.Fields)
	default:
		return nil, fmt.Errorf("不支持的导出格式: %s", req.Format)
	}
}

func (d *auditDAO) DeleteAuditLog(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.AuditLog{}, id).Error
}

func (d *auditDAO) BatchDeleteAuditLogs(ctx context.Context, ids []int) error {
	return d.db.WithContext(ctx).Delete(&model.AuditLog{}, ids).Error
}

func (d *auditDAO) ArchiveAuditLogs(ctx context.Context, startTime, endTime int64) error {
	// 这里可以实现数据归档逻辑，比如移动到历史表或者备份存储
	start := time.Unix(startTime, 0)
	end := time.Unix(endTime, 0)

	// 示例：删除指定时间范围的数据（实际场景可能需要移动到归档表）
	return d.db.WithContext(ctx).
		Where("created_at BETWEEN ? AND ?", start, end).
		Delete(&model.AuditLog{}).Error
}

// buildListQuery 构建列表查询
func (d *auditDAO) buildListQuery(req *model.ListAuditLogsRequest) *gorm.DB {
	query := d.db.Model(&model.AuditLog{})

	if req.OperationType != "" {
		query = query.Where("operation_type = ?", req.OperationType)
	}

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}

	if req.TargetType != "" {
		query = query.Where("target_type = ?", req.TargetType)
	}

	if req.StatusCode > 0 {
		query = query.Where("status_code = ?", req.StatusCode)
	}

	if req.TraceID != "" {
		query = query.Where("trace_id = ?", req.TraceID)
	}

	if req.StartTime > 0 {
		query = query.Where("created_at >= ?", time.Unix(req.StartTime, 0))
	}

	if req.EndTime > 0 {
		query = query.Where("created_at <= ?", time.Unix(req.EndTime, 0))
	}

	return query
}

// buildSearchQuery 构建搜索查询
func (d *auditDAO) buildSearchQuery(req *model.SearchAuditLogsRequest) *gorm.DB {
	query := d.buildListQuery(&req.ListAuditLogsRequest)

	if req.Advanced != nil {
		adv := req.Advanced

		if len(adv.IPAddressList) > 0 {
			query = query.Where("ip_address IN ?", adv.IPAddressList)
		}

		if len(adv.StatusCodeList) > 0 {
			query = query.Where("status_code IN ?", adv.StatusCodeList)
		}

		if adv.DurationMin > 0 {
			query = query.Where("duration >= ?", adv.DurationMin)
		}

		if adv.DurationMax > 0 {
			query = query.Where("duration <= ?", adv.DurationMax)
		}

		if adv.HasError != nil {
			if *adv.HasError {
				query = query.Where("status_code >= 400 OR error_msg != ''")
			} else {
				query = query.Where("status_code < 400 AND error_msg = ''")
			}
		}

		if adv.EndpointPattern != "" {
			query = query.Where("endpoint LIKE ?", "%"+adv.EndpointPattern+"%")
		}
	}

	return query
}

// exportAsCSV 导出为CSV格式
func (d *auditDAO) exportAsCSV(logs []model.AuditLog, fields []string) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// 写入表头
	if len(fields) == 0 {
		fields = []string{"id", "user_id", "trace_id", "ip_address", "http_method",
			"endpoint", "operation_type", "target_type", "target_id", "status_code",
			"duration", "error_msg", "created_at"}
	}
	writer.Write(fields)

	// 写入数据
	for _, log := range logs {
		record := make([]string, len(fields))
		for i, field := range fields {
			switch field {
			case "id":
				record[i] = fmt.Sprintf("%d", log.ID)
			case "user_id":
				record[i] = fmt.Sprintf("%d", log.UserID)
			case "trace_id":
				record[i] = log.TraceID
			case "ip_address":
				record[i] = log.IPAddress
			case "http_method":
				record[i] = log.HttpMethod
			case "endpoint":
				record[i] = log.Endpoint
			case "operation_type":
				record[i] = log.OperationType
			case "target_type":
				record[i] = log.TargetType
			case "target_id":
				record[i] = log.TargetID
			case "status_code":
				record[i] = fmt.Sprintf("%d", log.StatusCode)
			case "duration":
				record[i] = fmt.Sprintf("%d", log.Duration)
			case "error_msg":
				record[i] = log.ErrorMsg
			case "created_at":
				record[i] = log.CreatedAt.Format("2006-01-02 15:04:05")
			}
		}
		writer.Write(record)
	}

	writer.Flush()
	return []byte(buf.String()), writer.Error()
}
