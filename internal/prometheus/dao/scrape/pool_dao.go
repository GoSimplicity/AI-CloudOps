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

package scrape

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm/clause"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScrapePoolDAO interface {
	GetMonitorScrapePoolLifecycleList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error)
	GetMonitorScrapePoolTombstone(ctx context.Context, id int) (*model.MonitorScrapePool, error)
	GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error)
	CreateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error
	GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error)
	UpdateMonitorScrapePool(ctx context.Context, req *model.UpdateMonitorScrapePoolReq) error
	DeleteMonitorScrapePool(ctx context.Context, poolId int) error
	GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, int64, error)
	GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, int64, error)
	CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error)
}

type scrapePoolDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewScrapePoolDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) ScrapePoolDAO {
	return &scrapePoolDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

func (d *scrapePoolDAO) GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	var pools []*model.MonitorScrapePool
	var count int64

	query := d.db.WithContext(ctx).Model(&model.MonitorScrapePool{})

	// 添加搜索条件
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.SupportAlert != nil {
		query = query.Where("support_alert = ?", *req.SupportAlert)
	}

	if req.SupportRecord != nil {
		query = query.Where("support_record = ?", *req.SupportRecord)
	}

	if err := query.Count(&count).Error; err != nil {
		d.l.Error("获取 MonitorScrapePool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := query.Offset((req.Page - 1) * req.Size).Limit(req.Size).Find(&pools).Error; err != nil {
		d.l.Error("获取 MonitorScrapePool 记录失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

func (d *scrapePoolDAO) CreateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error {
	// 检查是否已存在相同名称的pool
	var count int64
	if err := d.db.WithContext(ctx).Model(&model.MonitorScrapePool{}).
		Where("name = ?", pool.Name).
		Count(&count).Error; err != nil {
		d.l.Error("检查 MonitorScrapePool 是否存在失败", zap.Error(err))
		return err
	}

	if count > 0 {
		return fmt.Errorf("pool已存在,请勿重复创建")
	}

	if err := d.db.WithContext(ctx).Create(pool).Error; err != nil {
		d.l.Error("创建 MonitorScrapePool 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorScrapePoolById 根据 ID 获取监控采集池
func (d *scrapePoolDAO) GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	if id <= 0 {
		d.l.Error("GetMonitorScrapePoolById 失败：无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var pool model.MonitorScrapePool

	if err := d.db.WithContext(ctx).First(&pool, id).Error; err != nil {
		d.l.Error("根据 ID 获取 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &pool, nil
}

func (d *scrapePoolDAO) UpdateMonitorScrapePool(ctx context.Context, req *model.UpdateMonitorScrapePoolReq) error {
	if req.ID <= 0 {
		d.l.Error("UpdateMonitorScrapePool 失败: ID 为 0", zap.Any("pool", req))
		return fmt.Errorf("monitorScrapePool 的 ID 必须设置且非零")
	}

	if err := d.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"name":                   req.Name,
			"prometheus_instances":   req.PrometheusInstances,
			"scrape_interval":        req.ScrapeInterval,
			"scrape_timeout":         req.ScrapeTimeout,
			"remote_timeout_seconds": req.RemoteTimeoutSeconds,
			"support_alert":          req.SupportAlert,
			"support_record":         req.SupportRecord,
			"tags":                   req.Tags,
			"remote_write_url":       req.RemoteWriteUrl,
			"remote_read_url":        req.RemoteReadUrl,
			"alert_manager_url":      req.AlertManagerUrl,
			"rule_file_path":         req.RuleFilePath,
			"record_file_path":       req.RecordFilePath,
			"user_id":                req.UserID,
		}).Error; err != nil {
		d.l.Error("更新 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", req.ID))
		return err
	}

	return nil
}

func (d *scrapePoolDAO) DeleteMonitorScrapePool(ctx context.Context, poolId int) error {
	if poolId <= 0 {
		d.l.Error("DeleteMonitorScrapePool 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return fmt.Errorf("无效的 poolId: %d", poolId)
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockLiveScrapePool(tx, poolId); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// A repeated DELETE may resume retirement only from a real DB tombstone.
			var tombstone model.MonitorScrapePool
			if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at > 0", poolId).First(&tombstone).Error; err != nil {
				return err
			}
			return nil
		}
		// Disabled definitions are still user definitions. Default scopes exclude
		// only soft-deleted rows; generated configs are retired separately.
		for _, definition := range []interface{}{&model.MonitorScrapeJob{}, &model.MonitorAlertRule{}, &model.MonitorRecordRule{}} {
			var count int64
			if err := tx.Model(definition).Where("pool_id = ?", poolId).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				return fmt.Errorf("采集池仍有关联的采集任务、告警规则或记录规则，请先删除或迁移定义")
			}
		}
		result := tx.Where("id = ?", poolId).Delete(&model.MonitorScrapePool{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("采集池已删除或不存在")
		}
		return nil
	})
}

func (d *scrapePoolDAO) GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, int64, error) {
	var pools []*model.MonitorScrapePool
	var count int64

	if err := d.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("support_alert = ?", 1).
		Count(&count).Error; err != nil {
		d.l.Error("获取支持警报的 MonitorScrapePool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := d.db.WithContext(ctx).
		Where("support_alert = ?", 1).
		Find(&pools).Error; err != nil {
		d.l.Error("获取支持警报的 MonitorScrapePool 失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// GetMonitorScrapePoolSupportedRecord 获取支持记录规则的监控采集池
func (d *scrapePoolDAO) GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, int64, error) {
	var pools []*model.MonitorScrapePool
	var count int64

	if err := d.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("support_record = ?", 1).
		Count(&count).Error; err != nil {
		d.l.Error("获取支持记录规则的 MonitorScrapePool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := d.db.WithContext(ctx).
		Where("support_record = ?", 1).
		Find(&pools).Error; err != nil {
		d.l.Error("获取支持记录规则的 MonitorScrapePool 失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

func (d *scrapePoolDAO) CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error) {
	if scrapePool.Name == "" {
		return false, fmt.Errorf("scrapePool 或 name 不能为空")
	}

	var count int64

	if err := d.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("name = ?", scrapePool.Name).
		Count(&count).Error; err != nil {
		d.l.Error("检查 MonitorScrapePool 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// Lifecycle inventory is private to controllers, never exposed by list HTTP.
// Ordinary absence is not deletion authorization; callers inspect DeletedAt.
func (d *scrapePoolDAO) GetMonitorScrapePoolLifecycleList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req == nil || req.Page <= 0 || req.Size <= 0 {
		return nil, 0, fmt.Errorf("无效的生命周期分页")
	}
	var rows []*model.MonitorScrapePool
	var total int64
	query := d.db.WithContext(ctx).Unscoped().Model(&model.MonitorScrapePool{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id ASC").Offset((req.Page - 1) * req.Size).Limit(req.Size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
func (d *scrapePoolDAO) GetMonitorScrapePoolTombstone(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的采集池ID")
	}
	var row model.MonitorScrapePool
	if err := d.db.WithContext(ctx).Unscoped().Where("id = ? AND deleted_at > 0", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
