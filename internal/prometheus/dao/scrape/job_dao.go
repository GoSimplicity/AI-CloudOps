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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScrapeJobDAO interface {
	GetMonitorScrapeJobList(ctx context.Context, req *model.GetMonitorScrapeJobListReq) ([]*model.MonitorScrapeJob, int64, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error)
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, jobId int) error
	GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error)
	CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error)
	CheckMonitorInstanceExists(ctx context.Context, poolID int) (bool, error)
}

type scrapeJobDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewScrapeJobDAO(db *gorm.DB, l *zap.Logger) ScrapeJobDAO {
	return &scrapeJobDAO{
		db: db,
		l:  l,
	}
}

// GetMonitorScrapeJobList 获取监控采集作业列表
func (s *scrapeJobDAO) GetMonitorScrapeJobList(ctx context.Context, req *model.GetMonitorScrapeJobListReq) ([]*model.MonitorScrapeJob, int64, error) {
	var jobs []*model.MonitorScrapeJob
	var total int64

	query := s.db.WithContext(ctx).Model(&model.MonitorScrapeJob{})

	offset := (req.Page - 1) * req.Size
	limit := req.Size
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.PoolID > 0 {
		query = query.Where("pool_id = ?", req.PoolID)
	}

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	if err := query.Count(&total).Error; err != nil {
		s.l.Error("计算监控采集作业总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := query.Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		s.l.Error("获取监控采集作业列表失败", zap.Error(err))
		return nil, 0, err
	}

	return jobs, total, nil
}

// CreateMonitorScrapeJob 创建监控采集作业
func (s *scrapeJobDAO) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if err := s.db.WithContext(ctx).Create(monitorScrapeJob).Error; err != nil {
		s.l.Error("创建 MonitorScrapeJob 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorScrapeJobsByPoolId 获取监控采集作业列表
func (s *scrapeJobDAO) GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error) {
	if poolId <= 0 {
		s.l.Error("GetMonitorScrapeJobsByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var jobs []*model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).
		Where("enable = ?", 1).
		Where("pool_id = ?", poolId).
		Find(&jobs).Error; err != nil {
		s.l.Error("获取 MonitorScrapeJob 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return jobs, nil
}

// UpdateMonitorScrapeJob 更新监控采集作业
func (s *scrapeJobDAO) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if monitorScrapeJob.ID <= 0 {
		s.l.Error("UpdateMonitorScrapeJob 失败: ID 无效", zap.Any("job", monitorScrapeJob))
		return fmt.Errorf("monitorScrapeJob 的 ID 必须大于 0")
	}

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("id = ?", monitorScrapeJob.ID).
		Updates(monitorScrapeJob).Error; err != nil {
		s.l.Error("更新 MonitorScrapeJob 失败", zap.Error(err), zap.Int("id", monitorScrapeJob.ID))
		return err
	}

	return nil
}

// DeleteMonitorScrapeJob 删除监控采集作业
func (s *scrapeJobDAO) DeleteMonitorScrapeJob(ctx context.Context, jobId int) error {
	if jobId <= 0 {
		s.l.Error("DeleteMonitorScrapeJob 失败: 无效的 jobId", zap.Int("jobId", jobId))
		return fmt.Errorf("无效的 jobId: %d", jobId)
	}

	result := s.db.WithContext(ctx).
		Where("id = ?", jobId).
		Delete(&model.MonitorScrapeJob{})

	if err := result.Error; err != nil {
		s.l.Error("删除 MonitorScrapeJob 失败", zap.Error(err), zap.Int("jobId", jobId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapeJob 失败: %w", jobId, err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到ID为 %d 的记录或已被删除", jobId)
	}

	return nil
}

// CheckMonitorScrapeJobExists 检查监控采集作业是否存在
func (s *scrapeJobDAO) CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("名称不能为空")
	}

	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("name = ?", name).
		Count(&count).Error; err != nil {
		s.l.Error("检查 MonitorScrapeJob 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetMonitorScrapeJobById 获取监控采集作业
func (s *scrapeJobDAO) GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error) {
	if id <= 0 {
		s.l.Error("GetMonitorScrapeJobById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var scrapeJob model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).
		Where("id = ?", id).
		First(&scrapeJob).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到ID为 %d 的记录", id)
		}
		s.l.Error("获取 MonitorScrapeJob 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &scrapeJob, nil
}

// CheckMonitorInstanceExists 检查监控实例是否存在
func (s *scrapeJobDAO) CheckMonitorInstanceExists(ctx context.Context, poolID int) (bool, error) {
	if poolID <= 0 {
		return false, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", poolID).
		Count(&count).Error; err != nil {
		s.l.Error("检查监控实例是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}
