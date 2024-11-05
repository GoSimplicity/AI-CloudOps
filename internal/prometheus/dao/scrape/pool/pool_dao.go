package pool

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type ScrapePoolDAO interface {
	GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error)
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, poolId int) error
	SearchMonitorScrapePoolsByName(ctx context.Context, name string) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error)
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

func (s *scrapePoolDAO) GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).Find(&pools).Error; err != nil {
		s.l.Error("获取所有 MonitorScrapePool 记录失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (s *scrapePoolDAO) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	if monitorScrapePool == nil {
		s.l.Error("CreateMonitorScrapePool 失败：pool 为 nil")
		return fmt.Errorf("monitorScrapePool 不能为空")
	}

	if err := s.db.WithContext(ctx).Create(monitorScrapePool).Error; err != nil {
		s.l.Error("创建 MonitorScrapePool 失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *scrapePoolDAO) GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	if id <= 0 {
		s.l.Error("GetMonitorScrapePoolById 失败：无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID：%d", id)
	}

	var pool model.MonitorScrapePool
	if err := s.db.WithContext(ctx).First(&pool, id).Error; err != nil {
		s.l.Error("根据 ID 获取 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &pool, nil
}

func (s *scrapePoolDAO) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	if monitorScrapePool == nil {
		s.l.Error("UpdateMonitorScrapePool 失败：pool 为 nil")
		return fmt.Errorf("monitorScrapePool 不能为空")
	}

	if monitorScrapePool.ID == 0 {
		s.l.Error("UpdateMonitorScrapePool 失败：ID 为 0", zap.Any("pool", monitorScrapePool))
		return fmt.Errorf("monitorScrapePool 的 ID 必须设置且非零")
	}

	// 使用 Updates 方法时，应当使用非零值结构体，以避免更新零值字段
	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", monitorScrapePool.ID).
		Updates(monitorScrapePool).Error; err != nil {
		s.l.Error("更新 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", monitorScrapePool.ID))
		return err
	}

	return nil
}

func (s *scrapePoolDAO) DeleteMonitorScrapePool(ctx context.Context, poolId int) error {
	if poolId <= 0 {
		s.l.Error("DeleteMonitorScrapePool 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return fmt.Errorf("无效的 poolId: %d", poolId)
	}

	result := s.db.WithContext(ctx).Delete(&model.MonitorScrapePool{}, poolId)
	if err := result.Error; err != nil {
		s.l.Error("删除 MonitorScrapePool 失败", zap.Error(err), zap.Int("poolId", poolId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapePool 失败: %w", poolId, err)
	}

	return nil
}

func (s *scrapePoolDAO) SearchMonitorScrapePoolsByName(ctx context.Context, name string) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&pools).Error; err != nil {
		s.l.Error("通过名称搜索 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (s *scrapePoolDAO) GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).
		Where("support_alert = ?", true).
		Find(&pools).Error; err != nil {
		s.l.Error("获取支持警报的 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (s *scrapePoolDAO) GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := s.db.WithContext(ctx).
		Where("support_record = ?", true).
		Find(&pools).Error; err != nil {
		s.l.Error("获取支持记录规则的 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (s *scrapePoolDAO) CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error) {
	var count int64

	if err := s.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("name = ?", scrapePool.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
