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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ScrapeJobDAO interface {
	GetMonitorScrapeJobList(ctx context.Context, offset, limit int) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error)
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, jobId int) error
	SearchMonitorScrapeJobsByName(ctx context.Context, name string) ([]*model.MonitorScrapeJob, error)
	GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error)
	CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error)
	CheckMonitorInstanceExists(ctx context.Context, poolID int) (bool, error)
	GetMonitorScrapeJobTotal(ctx context.Context) (int, error)
}

type scrapeJobDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewScrapeJobDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) ScrapeJobDAO {
	return &scrapeJobDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

// GetMonitorScrapeJobList 获取监控采集作业列表
func (s *scrapeJobDAO) GetMonitorScrapeJobList(ctx context.Context, offset, limit int) ([]*model.MonitorScrapeJob, error) {
	if offset < 0 {
		return nil, fmt.Errorf("offset不能为负数")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit必须大于0")
	}

	var jobs []*model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		s.l.Error("获取监控采集作业列表失败", zap.Error(err))
		return nil, err
	}

	return jobs, nil
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
		Updates(map[string]interface{}{
			"name":                        monitorScrapeJob.Name,
			"enable":                      monitorScrapeJob.Enable,
			"service_discovery_type":      monitorScrapeJob.ServiceDiscoveryType,
			"metrics_path":                monitorScrapeJob.MetricsPath,
			"scheme":                      monitorScrapeJob.Scheme,
			"scrape_interval":             monitorScrapeJob.ScrapeInterval,
			"scrape_timeout":              monitorScrapeJob.ScrapeTimeout,
			"pool_id":                     monitorScrapeJob.PoolID,
			"relabel_configs_yaml_string": monitorScrapeJob.RelabelConfigsYamlString,
			"refresh_interval":            monitorScrapeJob.RefreshInterval,
			"port":                        monitorScrapeJob.Port,
			"ip_address":                  monitorScrapeJob.IpAddress,
			"kube_config_file_path":       monitorScrapeJob.KubeConfigFilePath,
			"tls_ca_file_path":            monitorScrapeJob.TlsCaFilePath,
			"tls_ca_content":              monitorScrapeJob.TlsCaContent,
			"bearer_token":                monitorScrapeJob.BearerToken,
			"bearer_token_file":           monitorScrapeJob.BearerTokenFile,
			"kubernetes_sd_role":          monitorScrapeJob.KubernetesSdRole,
			"updated_at":                  monitorScrapeJob.UpdatedAt,
		}).Error; err != nil {
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

// SearchMonitorScrapeJobsByName 通过名称搜索监控采集作业
func (s *scrapeJobDAO) SearchMonitorScrapeJobsByName(ctx context.Context, name string) ([]*model.MonitorScrapeJob, error) {
	if name == "" {
		return nil, fmt.Errorf("搜索名称不能为空")
	}

	var jobs []*model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&jobs).Error; err != nil {
		s.l.Error("通过名称搜索 MonitorScrapeJob 失败", zap.Error(err))
		return nil, err
	}

	return jobs, nil
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

// GetMonitorScrapeJobTotal 获取监控采集作业总数
func (s *scrapeJobDAO) GetMonitorScrapeJobTotal(ctx context.Context) (int, error) {
	var count int64

	if err := s.db.WithContext(ctx).Model(&model.MonitorScrapeJob{}).Count(&count).Error; err != nil {
		s.l.Error("获取监控采集作业总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
