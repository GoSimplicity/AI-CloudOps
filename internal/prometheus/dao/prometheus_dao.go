package dao

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrometheusDao interface {
	GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error)
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, poolId int) error

	GetAllMonitorScrapeJobs(ctx context.Context) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error)
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, jobId int) error

	GetHttpSdApi(ctx context.Context, jobId int) (string, error)
	GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error)
	GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error)
	GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error)
	GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error)
}

type prometheusDao struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewPrometheusDAO(db *gorm.DB, l *zap.Logger) PrometheusDao {
	return &prometheusDao{
		db: db,
		l:  l,
	}
}

func (p *prometheusDao) GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var list []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).Find(&list).Error; err != nil {
		p.l.Error("failed to get all monitor scrape pool", zap.Error(err))
		return nil, err
	}

	if len(list) == 0 {
		p.l.Info("no monitor scrape pools found")
	}

	return list, nil
}

func (p *prometheusDao) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 确保 monitorScrapePool 不为 nil
	if monitorScrapePool == nil {
		p.l.Error("CreateMonitorScrapePool failed: monitorScrapePool is nil")
		return fmt.Errorf("monitorScrapePool cannot be nil")
	}

	if err := p.db.WithContext(ctx).Create(monitorScrapePool).Error; err != nil {
		p.l.Error("failed to create monitor scrape pool", zap.Error(err))
		return err
	}

	return nil
}

func (p *prometheusDao) GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	var monitorScrapePool *model.MonitorScrapePool

	// 确保 ID 是有效的（非零）
	if id <= 0 {
		p.l.Error("GetMonitorScrapePoolById failed: invalid ID", zap.Int("id", id))
		return nil, fmt.Errorf("invalid ID: %d", id)
	}

	if err := p.db.WithContext(ctx).Where("id = ?", id).First(&monitorScrapePool).Error; err != nil {
		p.l.Error("failed to get monitor scrape pool by id", zap.Error(err))
		return nil, err
	}

	return monitorScrapePool, nil
}

func (p *prometheusDao) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	if monitorScrapePool == nil {
		p.l.Error("UpdateMonitorScrapePool failed: monitorScrapePool is nil")
		return fmt.Errorf("monitorScrapePool cannot be nil")
	}

	// 确保 monitorScrapePool.ID 已设置
	if monitorScrapePool.ID == 0 {
		p.l.Error("UpdateMonitorScrapePool failed: ID is zero", zap.Any("monitorScrapePool", monitorScrapePool))
		return fmt.Errorf("monitorScrapePool ID must be set and non-zero")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).     // 明确指定模型
		Where("id = ?", monitorScrapePool.ID). // 根据 ID 过滤记录
		Updates(monitorScrapePool)             // 执行更新

	// 检查更新过程中是否有错误
	if result.Error != nil {
		p.l.Error("UpdateMonitorScrapePool failed to update record",
			zap.Error(result.Error),
			zap.Int("id", monitorScrapePool.ID))
		return result.Error
	}

	// 检查是否有记录被更新
	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorScrapePool found no records to update", zap.Int("id", monitorScrapePool.ID))
		return fmt.Errorf("no MonitorScrapePool found with ID %d", monitorScrapePool.ID)
	}

	return nil
}

func (p *prometheusDao) DeleteMonitorScrapePool(ctx context.Context, poolId int) error {
	// 确保 poolId 是有效的（非零）
	if poolId <= 0 {
		p.l.Error("DeleteMonitorScrapePool failed: invalid poolId", zap.Int("poolId", poolId))
		return fmt.Errorf("invalid poolId: %d", poolId)
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", poolId).
		Delete(&model.MonitorScrapePool{})

	// 检查删除过程中是否有错误
	if result.Error != nil {
		p.l.Error("DeleteMonitorScrapePool failed to delete record",
			zap.Error(result.Error),
			zap.Int("poolId", poolId))
		return fmt.Errorf("failed to delete monitor scrape pool with ID %d: %w", poolId, result.Error)
	}

	// 检查是否有记录被删除
	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorScrapePool found no records to delete",
			zap.Int("poolId", poolId))
		return fmt.Errorf("no monitor scrape pool found with ID %d", poolId)
	}

	return nil
}

func (p *prometheusDao) GetAllMonitorScrapeJobs(ctx context.Context) ([]*model.MonitorScrapeJob, error) {
	var scrapeJobs []*model.MonitorScrapeJob

	if err := p.db.WithContext(ctx).Find(&scrapeJobs).Error; err != nil {
		p.l.Error("GetAllMonitorScrapeJobs failed to get all scrape jobs", zap.Error(err))
		return nil, err
	}

	if len(scrapeJobs) == 0 {
		p.l.Info("no monitor scrape jobs found")
	}

	return scrapeJobs, nil
}

func (p *prometheusDao) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if monitorScrapeJob == nil {
		p.l.Error("CreateMonitorScrapeJob failed: monitorScrapeJob is nil")
		return fmt.Errorf("monitorScrapeJob cannot be nil")
	}

	if err := p.db.WithContext(ctx).Create(monitorScrapeJob).Error; err != nil {
		p.l.Error("CreateMonitorScrapeJob failed to create scrape job", zap.Error(err))
		return err
	}

	return nil
}

func (p *prometheusDao) GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error) {
	if poolId <= 0 {
		p.l.Error("GetMonitorScrapeJobsByPoolId failed: invalid poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("invalid poolId: %d", poolId)
	}

	var scrapeJobs []*model.MonitorScrapeJob

	if err := p.db.WithContext(ctx).Where("enable = 1 and pool_id = ?", poolId).Find(&scrapeJobs).Error; err != nil {
		p.l.Error("GetMonitorScrapeJobsByPoolId failed to get scrape jobs", zap.Error(err))
		return nil, err
	}

	return scrapeJobs, nil
}

func (p *prometheusDao) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if monitorScrapeJob == nil {
		p.l.Error("UpdateMonitorScrapeJob failed: monitorScrapeJob is nil")
		return fmt.Errorf("monitorScrapeJob cannot be nil")
	}

	// 确保 monitorScrapeJob.ID 已设置
	if monitorScrapeJob.ID == 0 {
		p.l.Error("UpdateMonitorScrapeJob failed: ID is zero", zap.Any("monitorScrapeJob", monitorScrapeJob))
		return fmt.Errorf("monitorScrapeJob ID must be set and non-zero")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).     // 明确指定模型
		Where("id = ?", monitorScrapeJob.ID). // 根据 ID 过滤记录
		Updates(monitorScrapeJob)             // 执行更新

	// 检查更新过程中是否有错误
	if result.Error != nil {
		p.l.Error("UpdateMonitorScrapeJob failed to update record",
			zap.Error(result.Error),
			zap.Int("id", monitorScrapeJob.ID))
		return result.Error
	}

	// 检查是否有记录被更新
	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorScrapeJob found no records to update", zap.Int("id", monitorScrapeJob.ID))
	}

	return nil
}

func (p *prometheusDao) DeleteMonitorScrapeJob(ctx context.Context, jobId int) error {
	if jobId <= 0 {
		p.l.Error("DeleteMonitorScrapeJob failed: invalid jobId", zap.Int("jobId", jobId))
		return fmt.Errorf("invalid jobId: %d", jobId)
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("id = ?", jobId).
		Delete(&model.MonitorScrapeJob{})

	// 检查删除过程中是否有错误
	if result.Error != nil {
		p.l.Error("DeleteMonitorScrapeJob failed to delete record",
			zap.Error(result.Error),
			zap.Int("jobId", jobId))
		return fmt.Errorf("failed to delete monitor scrape job with ID %d: %w", jobId, result.Error)
	}

	return nil
}

func (p *prometheusDao) GetHttpSdApi(ctx context.Context, jobId int) (string, error) {
	var scrapeJob *model.MonitorScrapeJob

	if err := p.db.WithContext(ctx).Where("id = ?", jobId).First(&scrapeJob).Error; err != nil {
		p.l.Error("GetHttpSdApi failed to get http sd api", zap.Error(err))
		return "", err
	}

	return scrapeJob.ServiceDiscoveryType, nil
}

func (p *prometheusDao) GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error) {
	var pools []*model.MonitorAlertManagerPool

	if err := p.db.WithContext(ctx).Find(&pools).Error; err != nil {
		p.l.Error("GetAllAlertManagerPools failed to get all alert manager pools", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (p *prometheusDao) GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	if err := p.db.WithContext(ctx).Where("pool_id = ?", poolId).Find(&sendGroups).Error; err != nil {
		p.l.Error("GetMonitorSendGroupByPoolId failed to get send groups", zap.Error(err))
		return nil, err
	}

	return sendGroups, nil
}

func (p *prometheusDao) GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).Where("support_alert = 1").Find(&pools).Error; err != nil {
		p.l.Error("GetMonitorScrapePoolSupportedAlert failed to get supported alert pools", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (p *prometheusDao) GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).Where("support_record = 1").Find(&pools).Error; err != nil {
		p.l.Error("GetMonitorScrapePoolSupportedRecord failed to get supported record pools", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (p *prometheusDao) GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error) {
	var alertRules []*model.MonitorAlertRule

	if err := p.db.WithContext(ctx).Where("enable = 1 and pool_id = ?", poolId).Find(&alertRules).Error; err != nil {
		p.l.Error("failed to get alert rules by pool id", zap.Error(err))
		return nil, err
	}

	return alertRules, nil
}

func (p *prometheusDao) GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error) {
	var recordRules []*model.MonitorRecordRule

	if err := p.db.WithContext(ctx).Where("enable = 1 and pool_id = ?", poolId).Find(&recordRules).Error; err != nil {
		p.l.Error("failed to get record rules by pool id", zap.Error(err))
		return nil, err
	}

	return recordRules, nil
}
