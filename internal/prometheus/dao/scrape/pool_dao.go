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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScrapePoolDAO interface {
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

// GetMonitorScrapePoolList 获取监控采集池列表
func (s *scrapePoolDAO) GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	var pools []*model.MonitorScrapePool
	var count int64

	query := s.db.WithContext(ctx).Model(&model.MonitorScrapePool{})

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

	// 获取总数
	if err := query.Count(&count).Error; err != nil {
		s.l.Error("获取 MonitorScrapePool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	if err := query.Offset((req.Page - 1) * req.Size).Limit(req.Size).Find(&pools).Error; err != nil {
		s.l.Error("获取 MonitorScrapePool 记录失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// CreateMonitorScrapePool 创建监控采集池
func (s *scrapePoolDAO) CreateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error {
	// 检查是否已存在相同名称的pool
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.MonitorScrapePool{}).
		Where("name = ?", pool.Name).
		Count(&count).Error; err != nil {
		s.l.Error("检查 MonitorScrapePool 是否存在失败", zap.Error(err))
		return err
	}

	if count > 0 {
		return fmt.Errorf("pool已存在,请勿重复创建")
	}

	if err := s.db.WithContext(ctx).Create(pool).Error; err != nil {
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

	if err := s.db.WithContext(ctx).First(&pool, id).Error; err != nil {
		s.l.Error("根据 ID 获取 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &pool, nil
}

// UpdateMonitorScrapePool 更新监控采集池
func (s *scrapePoolDAO) UpdateMonitorScrapePool(ctx context.Context, req *model.UpdateMonitorScrapePoolReq) error {
	if req.ID <= 0 {
		s.l.Error("UpdateMonitorScrapePool 失败: ID 为 0", zap.Any("pool", req))
		return fmt.Errorf("monitorScrapePool 的 ID 必须设置且非零")
	}

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"name":                    req.Name,
			"prometheus_instances":    req.PrometheusInstances,
			"alert_manager_instances": req.AlertManagerInstances,
			"scrape_interval":         req.ScrapeInterval,
			"scrape_timeout":          req.ScrapeTimeout,
			"remote_timeout_seconds":  req.RemoteTimeoutSeconds,
			"support_alert":           req.SupportAlert,
			"support_record":          req.SupportRecord,
			"external_labels":         req.ExternalLabels,
			"remote_write_url":        req.RemoteWriteUrl,
			"remote_read_url":         req.RemoteReadUrl,
			"alert_manager_url":       req.AlertManagerUrl,
			"rule_file_path":          req.RuleFilePath,
			"record_file_path":        req.RecordFilePath,
			"user_id":                 req.UserID,
		}).Error; err != nil {
		s.l.Error("更新 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", req.ID))
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
		Where("id = ?", poolId).
		Delete(&model.MonitorScrapePool{})

	if result.Error != nil {
		s.l.Error("删除 MonitorScrapePool 失败", zap.Error(result.Error), zap.Int("poolId", poolId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapePool 失败: %w", poolId, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorScrapePool 或已被删除", poolId)
	}

	return nil
}

// GetMonitorScrapePoolSupportedAlert 获取支持警报的监控采集池
func (s *scrapePoolDAO) GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, int64, error) {
	var pools []*model.MonitorScrapePool
	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("support_alert = ?", true).
		Count(&count).Error; err != nil {
		s.l.Error("获取支持警报的 MonitorScrapePool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).
		Where("support_alert = ?", true).
		Find(&pools).Error; err != nil {
		s.l.Error("获取支持警报的 MonitorScrapePool 失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// GetMonitorScrapePoolSupportedRecord 获取支持记录规则的监控采集池
func (s *scrapePoolDAO) GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, int64, error) {
	var pools []*model.MonitorScrapePool
	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("support_record = ?", true).
		Count(&count).Error; err != nil {
		s.l.Error("获取支持记录规则的 MonitorScrapePool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := s.db.WithContext(ctx).
		Where("support_record = ?", true).
		Find(&pools).Error; err != nil {
		s.l.Error("获取支持记录规则的 MonitorScrapePool 失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// CheckMonitorScrapePoolExists 检查监控采集池是否存在
func (s *scrapePoolDAO) CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error) {
	if scrapePool.Name == "" {
		return false, fmt.Errorf("scrapePool 或 name 不能为空")
	}

	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("name = ?", scrapePool.Name).
		Count(&count).Error; err != nil {
		s.l.Error("检查 MonitorScrapePool 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}
