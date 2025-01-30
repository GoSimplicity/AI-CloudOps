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
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScrapePoolDAO interface {
	GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolList(ctx context.Context, offset, limit int) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error)
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, poolId int) error
	SearchMonitorScrapePoolsByName(ctx context.Context, name string) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error)
	CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error)
	GetMonitorScrapePoolTotal(ctx context.Context) (int, error)
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

// getTime 获取当前时间戳
func getTime() int64 {
	return time.Now().Unix()
}

// GetAllMonitorScrapePool 获取所有监控采集池
func (s *scrapePoolDAO) GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).Where("deleted_at = ?", 0).Find(&pools).Error; err != nil {
		s.l.Error("获取所有 MonitorScrapePool 记录失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetMonitorScrapePoolList 获取监控采集池列表
func (s *scrapePoolDAO) GetMonitorScrapePoolList(ctx context.Context, offset, limit int) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).Where("deleted_at = ?", 0).Offset(offset).Limit(limit).Find(&pools).Error; err != nil {
		s.l.Error("获取所有 MonitorScrapePool 记录失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// CreateMonitorScrapePool 创建监控采集池
func (s *scrapePoolDAO) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 检查是否已存在相同名称的pool
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.MonitorScrapePool{}).
		Where("name = ? AND deleted_at = ?", monitorScrapePool.Name, 0).
		Count(&count).Error; err != nil {
		s.l.Error("检查 MonitorScrapePool 是否存在失败", zap.Error(err))
		return err
	}

	if count > 0 {
		return fmt.Errorf("pool已存在,请勿重复创建")
	}

	monitorScrapePool.CreatedAt = getTime()
	monitorScrapePool.UpdatedAt = getTime()

	if err := s.db.WithContext(ctx).Create(monitorScrapePool).Error; err != nil {
		s.l.Error("创建 MonitorScrapePool 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorScrapePoolById 根据 ID 获取监控采集池
func (s *scrapePoolDAO) GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	if id <= 0 {
		s.l.Error("GetMonitorScrapePoolById 失败：无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var pool model.MonitorScrapePool

	if err := s.db.WithContext(ctx).Where("deleted_at = ?", 0).First(&pool, id).Error; err != nil {
		s.l.Error("根据 ID 获取 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &pool, nil
}

// UpdateMonitorScrapePool 更新监控采集池
func (s *scrapePoolDAO) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	monitorScrapePool.UpdatedAt = getTime()

	if monitorScrapePool.ID <= 0 {
		s.l.Error("UpdateMonitorScrapePool 失败: ID 为 0", zap.Any("pool", monitorScrapePool))
		return fmt.Errorf("monitorScrapePool 的 ID 必须设置且非零")
	}

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ? AND deleted_at = ?", monitorScrapePool.ID, 0).
		Updates(map[string]interface{}{
			"name":                    monitorScrapePool.Name,
			"prometheus_instances":    monitorScrapePool.PrometheusInstances,
			"alert_manager_instances": monitorScrapePool.AlertManagerInstances,
			"scrape_interval":         monitorScrapePool.ScrapeInterval,
			"scrape_timeout":          monitorScrapePool.ScrapeTimeout,
			"remote_timeout_seconds":  monitorScrapePool.RemoteTimeoutSeconds,
			"support_alert":           monitorScrapePool.SupportAlert,
			"support_record":          monitorScrapePool.SupportRecord,
			"external_labels":         monitorScrapePool.ExternalLabels,
			"remote_write_url":        monitorScrapePool.RemoteWriteUrl,
			"remote_read_url":         monitorScrapePool.RemoteReadUrl,
			"alert_manager_url":       monitorScrapePool.AlertManagerUrl,
			"rule_file_path":          monitorScrapePool.RuleFilePath,
			"record_file_path":        monitorScrapePool.RecordFilePath,
			"updated_at":              monitorScrapePool.UpdatedAt,
		}).Error; err != nil {
		s.l.Error("更新 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", monitorScrapePool.ID))
		return err
	}

	return nil
}

// DeleteMonitorScrapePool 删除监控采集池
func (s *scrapePoolDAO) DeleteMonitorScrapePool(ctx context.Context, poolId int) error {
	if poolId <= 0 {
		s.l.Error("DeleteMonitorScrapePool 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return fmt.Errorf("无效的 poolId: %d", poolId)
	}

	result := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ? AND deleted_at = ?", poolId, 0).
		Update("deleted_at", getTime())

	if result.Error != nil {
		s.l.Error("删除 MonitorScrapePool 失败", zap.Error(result.Error), zap.Int("poolId", poolId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapePool 失败: %w", poolId, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorScrapePool 或已被删除", poolId)
	}

	return nil
}

// SearchMonitorScrapePoolsByName 根据名称搜索监控采集池
func (s *scrapePoolDAO) SearchMonitorScrapePoolsByName(ctx context.Context, name string) ([]*model.MonitorScrapePool, error) {
	if name == "" {
		return nil, fmt.Errorf("搜索名称不能为空")
	}

	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).
		Where("LOWER(name) LIKE ? AND deleted_at = ?", "%"+strings.ToLower(name)+"%", 0).
		Find(&pools).Error; err != nil {
		s.l.Error("通过名称搜索 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetMonitorScrapePoolSupportedAlert 获取支持警报的监控采集池
func (s *scrapePoolDAO) GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).
		Where("support_alert = ? AND deleted_at = ?", true, 0).
		Find(&pools).Error; err != nil {
		s.l.Error("获取支持警报的 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetMonitorScrapePoolSupportedRecord 获取支持记录规则的监控采集池
func (s *scrapePoolDAO) GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).
		Where("support_record = ? AND deleted_at = ?", true, 0).
		Find(&pools).Error; err != nil {
		s.l.Error("获取支持记录规则的 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// CheckMonitorScrapePoolExists 检查监控采集池是否存在
func (s *scrapePoolDAO) CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error) {
	if scrapePool.Name == "" {
		return false, fmt.Errorf("scrapePool 或 name 不能为空")
	}

	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("name = ? AND deleted_at = ?", scrapePool.Name, 0).
		Count(&count).Error; err != nil {
		s.l.Error("检查 MonitorScrapePool 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetMonitorScrapePoolTotal 获取监控采集池总数
func (s *scrapePoolDAO) GetMonitorScrapePoolTotal(ctx context.Context) (int, error) {
	var count int64

	if err := s.db.WithContext(ctx).Model(&model.MonitorScrapePool{}).Where("deleted_at = ?", 0).Count(&count).Error; err != nil {
		s.l.Error("获取监控采集池总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
