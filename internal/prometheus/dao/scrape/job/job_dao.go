package job

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type ScrapeJobDAO interface {
	GetAllMonitorScrapeJobs(ctx context.Context) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error)
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, jobId int) error
	SearchMonitorScrapeJobsByName(ctx context.Context, name string) ([]*model.MonitorScrapeJob, error)
	CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error)
	GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error)
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

func (s *scrapeJobDAO) GetAllMonitorScrapeJobs(ctx context.Context) ([]*model.MonitorScrapeJob, error) {
	var jobs []*model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).Find(&jobs).Error; err != nil {
		s.l.Error("获取所有 MonitorScrapeJob 失败", zap.Error(err))
		return nil, err
	}

	return jobs, nil
}

func (s *scrapeJobDAO) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if monitorScrapeJob == nil {
		s.l.Error("CreateMonitorScrapeJob 失败: job 为 nil")
		return fmt.Errorf("monitorScrapeJob 不能为空")
	}

	if err := s.db.WithContext(ctx).Create(monitorScrapeJob).Error; err != nil {
		s.l.Error("创建 MonitorScrapeJob 失败", zap.Error(err))
		return err
	}

	return nil
}

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

func (s *scrapeJobDAO) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if monitorScrapeJob == nil {
		s.l.Error("UpdateMonitorScrapeJob 失败: job 为 nil")
		return fmt.Errorf("monitorScrapeJob 不能为空")
	}

	if monitorScrapeJob.ID == 0 {
		s.l.Error("UpdateMonitorScrapeJob 失败: ID 为 0", zap.Any("job", monitorScrapeJob))
		return fmt.Errorf("monitorScrapeJob 的 ID 必须设置且非零")
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

func (s *scrapeJobDAO) DeleteMonitorScrapeJob(ctx context.Context, jobId int) error {
	if jobId <= 0 {
		s.l.Error("DeleteMonitorScrapeJob 失败: 无效的 jobId", zap.Int("jobId", jobId))
		return fmt.Errorf("无效的 jobId: %d", jobId)
	}

	result := s.db.WithContext(ctx).Delete(&model.MonitorScrapeJob{}, jobId)
	if err := result.Error; err != nil {
		s.l.Error("删除 MonitorScrapeJob 失败", zap.Error(err), zap.Int("jobId", jobId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapeJob 失败: %w", jobId, err)
	}

	return nil
}

func (s *scrapeJobDAO) SearchMonitorScrapeJobsByName(ctx context.Context, name string) ([]*model.MonitorScrapeJob, error) {
	var jobs []*model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&jobs).Error; err != nil {
		s.l.Error("通过名称搜索 MonitorScrapeJob 失败", zap.Error(err))
		return nil, err
	}

	return jobs, nil
}

func (s *scrapeJobDAO) CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error) {
	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("name = ?", name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *scrapeJobDAO) GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error) {
	if id <= 0 {
		s.l.Error("GetMonitorScrapeJobById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var scrapeJob model.MonitorScrapeJob

	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&scrapeJob).Error; err != nil {
		s.l.Error("获取 MonitorScrapeJob 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &scrapeJob, nil
}
