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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuditDAO interface {
	CreateAuditLog(ctx context.Context, req *model.AuditLog) error
	ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) ([]*model.AuditLog, int64, error)
	GetAuditLogDetail(ctx context.Context, id uint) (*model.AuditLogDetail, error)
	GetAuditTypes(ctx context.Context) ([]string, error)
	GetAuditStatistics(ctx context.Context) (interface{}, error)
	SearchAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) ([]*model.AuditLog, int64, error)
	ExportAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (interface{}, error)
	DeleteAuditLog(ctx context.Context, id uint) error
	BatchDeleteLogs(ctx context.Context, ids []uint) error
	ArchiveAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) error
}

type auditDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAuditDAO(db *gorm.DB, l *zap.Logger) AuditDAO {
	return &auditDAO{
		db: db,
		l:  l,
	}
}

// ListAuditLogs 获取审计日志列表
func (d *auditDAO) ListAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) ([]*model.AuditLog, int64, error) {
	var (
		logs  []*model.AuditLog
		total int64
		err   error
	)

	query := d.db.WithContext(ctx).Model(&model.AuditLog{})

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.OperationType != "" {
		query = query.Where("operation_type = ?", req.OperationType)
	}
	query = query.Where("created_at BETWEEN ? AND ?", req.StartTime, req.EndTime)

	if err = query.Count(&total).Error; err != nil {
		d.l.Error("统计审计日志总数失败", zap.Error(err))
		return nil, 0, err
	}

	offset := (req.PageNumber - 1) * req.PageSize
	if err = query.Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&logs).Error; err != nil {
		d.l.Error("查询审计日志列表失败", zap.Error(err))
		return nil, 0, err
	}

	return logs, total, nil
}

// GetAuditLogDetail 获取审计日志详情
func (d *auditDAO) GetAuditLogDetail(ctx context.Context, id uint) (*model.AuditLogDetail, error) {
	var detail model.AuditLogDetail
	err := d.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Select("id, username, operation_type, target_type, target_id, created_at").
		Where("id = ?", id).
		First(&detail).Error
	if err != nil {
		d.l.Error("获取审计日志详情失败", zap.Error(err), zap.Uint("id", id))
		return nil, err
	}
	return &detail, nil
}

// GetAuditTypes 获取审计类型列表
func (d *auditDAO) GetAuditTypes(ctx context.Context) ([]string, error) {
	var types []string
	err := d.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Distinct("operation_type").
		Order("operation_type").
		Pluck("operation_type", &types).Error
	if err != nil {
		d.l.Error("获取审计类型列表失败", zap.Error(err))
	}
	return types, err
}

// GetAuditStatistics 获取审计统计信息
func (d *auditDAO) GetAuditStatistics(ctx context.Context) (interface{}, error) {
	var stats struct {
		Total       int64 `json:"total"`
		CreateCount int64 `json:"create_count"`
		UpdateCount int64 `json:"update_count"`
		DeleteCount int64 `json:"delete_count"`
		OtherCount  int64 `json:"other_count"`
	}

	db := d.db.WithContext(ctx).Model(&model.AuditLog{})

	if err := db.Count(&stats.Total).Error; err != nil {
		d.l.Error("获取审计日志总数失败", zap.Error(err))
		return nil, err
	}

	operationTypes := []string{"CREATE", "UPDATE", "DELETE", "OTHER"}
	countMap := map[string]*int64{
		"CREATE": &stats.CreateCount,
		"UPDATE": &stats.UpdateCount,
		"DELETE": &stats.DeleteCount,
		"OTHER":  &stats.OtherCount,
	}

	for _, opType := range operationTypes {
		if err := db.Where("operation_type = ?", opType).Count(countMap[opType]).Error; err != nil {
			d.l.Error("获取操作类型统计数据失败",
				zap.String("operation_type", opType),
				zap.Error(err))
			return nil, err
		}
	}

	return stats, nil
}

// SearchAuditLogs 搜索审计日志
func (d *auditDAO) SearchAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) ([]*model.AuditLog, int64, error) {
	var (
		logs  []*model.AuditLog
		total int64
		err   error
	)

	query := d.db.WithContext(ctx).Model(&model.AuditLog{})

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.OperationType != "" {
		query = query.Where("operation_type = ?", req.OperationType)
	}
	query = query.Where("created_at BETWEEN ? AND ?", req.StartTime, req.EndTime)

	if err = query.Count(&total).Error; err != nil {
		d.l.Error("统计搜索结果总数失败", zap.Error(err))
		return nil, 0, err
	}

	offset := (req.PageNumber - 1) * req.PageSize
	if err = query.Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&logs).Error; err != nil {
		d.l.Error("搜索审计日志失败", zap.Error(err))
		return nil, 0, err
	}

	return logs, total, nil
}

// ExportAuditLogs 导出审计日志
func (d *auditDAO) ExportAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) (interface{}, error) {
	var logs []*model.AuditLog
	err := d.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Where("created_at BETWEEN ? AND ?", req.StartTime, req.EndTime).
		Order("created_at DESC").
		Find(&logs).Error
	if err != nil {
		d.l.Error("导出审计日志失败", zap.Error(err))
		return nil, err
	}
	return logs, nil
}

// DeleteAuditLog 删除单条审计日志
func (d *auditDAO) DeleteAuditLog(ctx context.Context, id uint) error {
	result := d.db.WithContext(ctx).Delete(&model.AuditLog{}, id)
	if result.Error != nil {
		d.l.Error("删除审计日志失败", zap.Error(result.Error), zap.Uint("id", id))
		return result.Error
	}
	if result.RowsAffected == 0 {
		d.l.Warn("要删除的审计日志不存在", zap.Uint("id", id))
	}
	return nil
}

// BatchDeleteLogs 批量删除审计日志
func (d *auditDAO) BatchDeleteLogs(ctx context.Context, ids []uint) error {
	result := d.db.WithContext(ctx).Delete(&model.AuditLog{}, ids)
	if result.Error != nil {
		d.l.Error("批量删除审计日志失败", zap.Error(result.Error), zap.Uints("ids", ids))
		return result.Error
	}
	if int64(len(ids)) != result.RowsAffected {
		d.l.Warn("部分审计日志不存在",
			zap.Int64("deleted", result.RowsAffected),
			zap.Int("expected", len(ids)))
	}
	return nil
}

// ArchiveAuditLogs 归档审计日志
func (d *auditDAO) ArchiveAuditLogs(ctx context.Context, req *model.ListAuditLogsRequest) error {
	result := d.db.WithContext(ctx).
		Where("created_at BETWEEN ? AND ?", req.StartTime, req.EndTime).
		Delete(&model.AuditLog{})
	if result.Error != nil {
		d.l.Error("归档审计日志失败", zap.Error(result.Error))
		return result.Error
	}
	d.l.Info("归档审计日志成功", zap.Int64("archived_count", result.RowsAffected))
	return nil
}

// CreateAuditLog 创建审计日志
func (d *auditDAO) CreateAuditLog(ctx context.Context, req *model.AuditLog) error {
	//if err := d.db.WithContext(ctx).Create(req).Error; err != nil {
	//	d.l.Error("创建审计日志失败", zap.Error(err))
	//	return err
	//}

	return nil
}
