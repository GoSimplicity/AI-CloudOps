package dao

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
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

	GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error)
	GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error)
	GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error)
	GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error)

	GetAllMonitorOndutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	GetMonitorOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyGroupChange *model.MonitorOnDutyChange) error
	GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime time.Time) ([]*model.MonitorOnDutyChange, error)
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	GetMonitorSendGroupByOnDutyGroupId(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, error)
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
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

func (p *prometheusDao) GetAllMonitorOndutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var ondutyGroups []*model.MonitorOnDutyGroup

	if err := p.db.WithContext(ctx).Find(&ondutyGroups).Error; err != nil {
		p.l.Error("GetAllMonitorOndutyGroup failed to get all onduty groups", zap.Error(err))
		return nil, err
	}

	return ondutyGroups, nil
}

func (p *prometheusDao) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	if monitorOnDutyGroup == nil {
		p.l.Error("CreateMonitorOnDutyGroup failed: monitorOnDutyGroup is nil")
		return fmt.Errorf("monitorOnDutyGroup cannot be nil")
	}

	if err := p.db.WithContext(ctx).Create(monitorOnDutyGroup).Error; err != nil {
		p.l.Error("CreateMonitorOnDutyGroup failed to create onduty group", zap.Error(err))
		return err
	}

	return nil
}

func (p *prometheusDao) GetMonitorOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	var ondutyGroup *model.MonitorOnDutyGroup

	if err := p.db.WithContext(ctx).Where("id = ?", id).First(&ondutyGroup).Error; err != nil {
		p.l.Error("GetMonitorOnDutyGroupById failed to get onduty group by id", zap.Error(err))
		return nil, err
	}

	return ondutyGroup, nil
}

func (p *prometheusDao) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyGroupChange *model.MonitorOnDutyChange) error {
	if monitorOnDutyGroupChange == nil {
		p.l.Error("CreateMonitorOnDutyGroupChange failed: monitorOnDutyGroupChange is nil")
		return fmt.Errorf("monitorOnDutyGroupChange cannot be nil")
	}

	if err := p.db.WithContext(ctx).Create(monitorOnDutyGroupChange).Error; err != nil {
		p.l.Error("CreateMonitorOnDutyGroupChange failed to create onduty group change", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorOnDutyChangesByGroupAndTimeRange 获取指定值班组在指定时间范围内的值班计划变更
func (p *prometheusDao) GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime time.Time) ([]*model.MonitorOnDutyChange, error) {
	var changes []*model.MonitorOnDutyChange

	if err := p.db.WithContext(ctx).Where("on_duty_group_id = ? AND date >= ? AND date <= ?", groupID, startTime, endTime).Find(&changes).Error; err != nil {
		p.l.Error("GetMonitorOnDutyChangesByGroupAndTimeRange failed to get onduty group changes", zap.Error(err))
		return nil, err
	}

	return changes, nil
}

// UpdateMonitorOnDutyGroup 更新 MonitorOnDutyGroup
func (p *prometheusDao) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	if monitorOnDutyGroup == nil {
		p.l.Error("UpdateMonitorOnDutyGroup failed: monitorOnDutyGroup is nil")
		return fmt.Errorf("monitorOnDutyGroup cannot be nil")
	}

	// 确保只更新指定的记录
	result := p.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("id = ?", monitorOnDutyGroup.ID).
		Updates(monitorOnDutyGroup)

	if result.Error != nil {
		p.l.Error("UpdateMonitorOnDutyGroup failed to update on-duty group", zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorOnDutyGroup: no rows affected", zap.Int("ID", monitorOnDutyGroup.ID))
		return fmt.Errorf("no on-duty group found with ID %d", monitorOnDutyGroup.ID)
	}

	return nil
}

// GetMonitorSendGroupByOnDutyGroupId 根据 onDutyGroupID 获取 MonitorSendGroup 列表
func (p *prometheusDao) GetMonitorSendGroupByOnDutyGroupId(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	result := p.db.WithContext(ctx).
		Where("on_duty_group_id = ?", onDutyGroupID).
		Find(&sendGroups)

	if result.Error != nil {
		p.l.Error("GetMonitorSendGroupByOnDutyGroupId failed to retrieve send groups",
			zap.Int("onDutyGroupID", onDutyGroupID),
			zap.Error(result.Error))
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Info("GetMonitorSendGroupByOnDutyGroupId: no send groups found",
			zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, nil
	}

	return sendGroups, nil
}

// DeleteMonitorOnDutyGroup 删除指定 ID 的 MonitorOnDutyGroup
func (p *prometheusDao) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	result := p.db.WithContext(ctx).
		Delete(&model.MonitorOnDutyGroup{}, id)

	if result.Error != nil {
		p.l.Error("DeleteMonitorOnDutyGroup failed to delete on-duty group",
			zap.Int("ID", id),
			zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorOnDutyGroup: no rows deleted",
			zap.Int("ID", id))
		return fmt.Errorf("no on-duty group found with ID %d", id)
	}

	return nil
}
