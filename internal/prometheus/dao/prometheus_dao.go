package dao

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
	"time"
)

type PrometheusDao interface {
	GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error)
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, poolId int) error
	SearchMonitorScrapePoolsByName(ctx context.Context, name string) ([]*model.MonitorScrapePool, error)

	GetAllMonitorScrapeJobs(ctx context.Context) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error)
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, jobId int) error
	SearchMonitorScrapeJobsByName(ctx context.Context, name string) ([]*model.MonitorScrapeJob, error)

	GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error
	SearchMonitorAlertManagerPoolByName(ctx context.Context, name string) ([]*model.MonitorAlertManagerPool, error)
	GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error)
	GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error)
	GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error)
	GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error)

	GetAllMonitorOndutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	GetMonitorOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	SearchMonitorOnDutyGroupByName(ctx context.Context, name string) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyGroupChange *model.MonitorOnDutyChange) error
	GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime time.Time) ([]*model.MonitorOnDutyChange, error)
	GetMonitorSendGroupByOnDutyGroupId(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, error)

	SearchMonitorSendGroupByName(ctx context.Context, name string) ([]*model.MonitorSendGroup, error)
	GetMonitorSendGroupList(ctx context.Context) ([]*model.MonitorSendGroup, error)
	GetMonitorSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error)
	CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	DeleteMonitorSendGroup(ctx context.Context, id int) error
	SearchMonitorAlertRuleByName(ctx context.Context, name string) ([]*model.MonitorAlertRule, error)
	GetMonitorAlertRuleList(ctx context.Context) ([]*model.MonitorAlertRule, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error
	DeleteMonitorAlertRule(ctx context.Context, ruleID int) error
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

// GetAllMonitorScrapePool 获取所有 MonitorScrapePool 记录
func (p *prometheusDao) GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).Find(&pools).Error; err != nil {
		p.l.Error("获取所有 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	if len(pools) == 0 {
		p.l.Info("未找到任何 MonitorScrapePool 记录")
	}

	return pools, nil
}

// CreateMonitorScrapePool 在数据库中创建一个新的 MonitorScrapePool 记录
func (p *prometheusDao) CreateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error {
	if pool == nil {
		p.l.Error("CreateMonitorScrapePool 失败: pool 为 nil")
		return fmt.Errorf("monitorScrapePool 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(pool).Error; err != nil {
		p.l.Error("创建 MonitorScrapePool 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorScrapePoolById 根据 ID 获取 MonitorScrapePool 记录
func (p *prometheusDao) GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	if id <= 0 {
		p.l.Error("GetMonitorScrapePoolById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var pool model.MonitorScrapePool
	if err := p.db.WithContext(ctx).First(&pool, id).Error; err != nil {
		p.l.Error("GetMonitorScrapePoolById 获取记录失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &pool, nil
}

// UpdateMonitorScrapePool 更新现有的 MonitorScrapePool 记录
func (p *prometheusDao) UpdateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error {
	if pool == nil {
		p.l.Error("UpdateMonitorScrapePool 失败: pool 为 nil")
		return fmt.Errorf("monitorScrapePool 不能为空")
	}

	if pool.ID == 0 {
		p.l.Error("UpdateMonitorScrapePool 失败: ID 为 0", zap.Any("pool", pool))
		return fmt.Errorf("monitorScrapePool 的 ID 必须设置且非零")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", pool.ID).
		Updates(pool)

	if result.Error != nil {
		p.l.Error("UpdateMonitorScrapePool 更新记录失败", zap.Error(result.Error), zap.Int("id", pool.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorScrapePool 未找到要更新的记录", zap.Int("id", pool.ID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorScrapePool", pool.ID)
	}

	return nil
}

// DeleteMonitorScrapePool 根据 ID 删除 MonitorScrapePool 记录
func (p *prometheusDao) DeleteMonitorScrapePool(ctx context.Context, poolId int) error {
	if poolId <= 0 {
		p.l.Error("DeleteMonitorScrapePool 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return fmt.Errorf("无效的 poolId: %d", poolId)
	}

	result := p.db.WithContext(ctx).
		Delete(&model.MonitorScrapePool{}, poolId)

	if result.Error != nil {
		p.l.Error("DeleteMonitorScrapePool 删除记录失败", zap.Error(result.Error), zap.Int("poolId", poolId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapePool 失败: %w", poolId, result.Error)
	}

	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorScrapePool 未找到要删除的记录", zap.Int("poolId", poolId))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorScrapePool", poolId)
	}

	return nil
}

// SearchMonitorScrapePoolsByName 通过名称搜索 MonitorScrapePool
func (p *prometheusDao) SearchMonitorScrapePoolsByName(ctx context.Context, name string) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&pools).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetAllMonitorScrapeJobs 获取所有 MonitorScrapeJob 记录
func (p *prometheusDao) GetAllMonitorScrapeJobs(ctx context.Context) ([]*model.MonitorScrapeJob, error) {
	var jobs []*model.MonitorScrapeJob

	if err := p.db.WithContext(ctx).Find(&jobs).Error; err != nil {
		p.l.Error("获取所有 MonitorScrapeJob 失败", zap.Error(err))
		return nil, err
	}

	if len(jobs) == 0 {
		p.l.Info("未找到任何 MonitorScrapeJob 记录")
	}

	return jobs, nil
}

// CreateMonitorScrapeJob 在数据库中创建一个新的 MonitorScrapeJob 记录
func (p *prometheusDao) CreateMonitorScrapeJob(ctx context.Context, job *model.MonitorScrapeJob) error {
	if job == nil {
		p.l.Error("CreateMonitorScrapeJob 失败: job 为 nil")
		return fmt.Errorf("monitorScrapeJob 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(job).Error; err != nil {
		p.l.Error("创建 MonitorScrapeJob 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorScrapeJobsByPoolId 根据 poolId 获取启用的 MonitorScrapeJob 记录
func (p *prometheusDao) GetMonitorScrapeJobsByPoolId(ctx context.Context, poolId int) ([]*model.MonitorScrapeJob, error) {
	if poolId <= 0 {
		p.l.Error("GetMonitorScrapeJobsByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var jobs []*model.MonitorScrapeJob
	if err := p.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
		Find(&jobs).Error; err != nil {
		p.l.Error("GetMonitorScrapeJobsByPoolId 获取记录失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return jobs, nil
}

// UpdateMonitorScrapeJob 更新现有的 MonitorScrapeJob 记录
func (p *prometheusDao) UpdateMonitorScrapeJob(ctx context.Context, job *model.MonitorScrapeJob) error {
	if job == nil {
		p.l.Error("UpdateMonitorScrapeJob 失败: job 为 nil")
		return fmt.Errorf("monitorScrapeJob 不能为空")
	}

	if job.ID == 0 {
		p.l.Error("UpdateMonitorScrapeJob 失败: ID 为 0", zap.Any("job", job))
		return fmt.Errorf("monitorScrapeJob 的 ID 必须设置且非零")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("id = ?", job.ID).
		Updates(job)

	if result.Error != nil {
		p.l.Error("UpdateMonitorScrapeJob 更新记录失败", zap.Error(result.Error), zap.Int("id", job.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorScrapeJob 未找到要更新的记录", zap.Int("id", job.ID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorScrapeJob", job.ID)
	}

	return nil
}

// DeleteMonitorScrapeJob 根据 ID 删除 MonitorScrapeJob 记录
func (p *prometheusDao) DeleteMonitorScrapeJob(ctx context.Context, jobId int) error {
	if jobId <= 0 {
		p.l.Error("DeleteMonitorScrapeJob 失败: 无效的 jobId", zap.Int("jobId", jobId))
		return fmt.Errorf("无效的 jobId: %d", jobId)
	}

	result := p.db.WithContext(ctx).
		Delete(&model.MonitorScrapeJob{}, jobId)

	if result.Error != nil {
		p.l.Error("DeleteMonitorScrapeJob 删除记录失败", zap.Error(result.Error), zap.Int("jobId", jobId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapeJob 失败: %w", jobId, result.Error)
	}

	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorScrapeJob 未找到要删除的记录", zap.Int("jobId", jobId))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorScrapeJob", jobId)
	}

	return nil
}

// SearchMonitorScrapeJobsByName 通过名称搜索 MonitorScrapeJob
func (p *prometheusDao) SearchMonitorScrapeJobsByName(ctx context.Context, name string) ([]*model.MonitorScrapeJob, error) {
	var jobs []*model.MonitorScrapeJob

	if err := p.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&jobs).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorScrapeJob 失败", zap.Error(err))
		return nil, err
	}

	return jobs, nil
}

// GetAllAlertManagerPools 获取所有 MonitorAlertManagerPool 记录
func (p *prometheusDao) GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error) {
	var pools []*model.MonitorAlertManagerPool

	if err := p.db.WithContext(ctx).Find(&pools).Error; err != nil {
		p.l.Error("获取所有 MonitorAlertManagerPool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetMonitorSendGroupByPoolId 根据 poolId 获取 MonitorSendGroup 记录
func (p *prometheusDao) GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error) {
	if poolId <= 0 {
		p.l.Error("GetMonitorSendGroupByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var sendGroups []*model.MonitorSendGroup
	if err := p.db.WithContext(ctx).
		Where("pool_id = ?", poolId).
		Find(&sendGroups).Error; err != nil {
		p.l.Error("GetMonitorSendGroupByPoolId 获取记录失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return sendGroups, nil
}

// GetMonitorScrapePoolSupportedAlert 获取支持警报的 MonitorScrapePool 记录
func (p *prometheusDao) GetMonitorScrapePoolSupportedAlert(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).
		Where("support_alert = ?", true).
		Find(&pools).Error; err != nil {
		p.l.Error("获取支持警报的 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetMonitorScrapePoolSupportedRecord 获取支持记录规则的 MonitorScrapePool 记录
func (p *prometheusDao) GetMonitorScrapePoolSupportedRecord(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).
		Where("support_record = ?", true).
		Find(&pools).Error; err != nil {
		p.l.Error("获取支持记录规则的 MonitorScrapePool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// GetMonitorAlertRuleByPoolId 根据 poolId 获取启用的 MonitorAlertRule 记录
func (p *prometheusDao) GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error) {
	if poolId <= 0 {
		p.l.Error("GetMonitorAlertRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var alertRules []*model.MonitorAlertRule
	if err := p.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
		Find(&alertRules).Error; err != nil {
		p.l.Error("GetMonitorAlertRuleByPoolId 获取记录失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return alertRules, nil
}

// GetMonitorRecordRuleByPoolId 根据 poolId 获取启用的 MonitorRecordRule 记录
func (p *prometheusDao) GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error) {
	if poolId <= 0 {
		p.l.Error("GetMonitorRecordRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var recordRules []*model.MonitorRecordRule
	if err := p.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
		Find(&recordRules).Error; err != nil {
		p.l.Error("GetMonitorRecordRuleByPoolId 获取记录失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return recordRules, nil
}

// GetAllMonitorOndutyGroup 获取所有 MonitorOnDutyGroup 记录
func (p *prometheusDao) GetAllMonitorOndutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var groups []*model.MonitorOnDutyGroup

	if err := p.db.WithContext(ctx).Find(&groups).Error; err != nil {
		p.l.Error("获取所有 MonitorOnDutyGroup 失败", zap.Error(err))
		return nil, err
	}

	return groups, nil
}

// CreateMonitorOnDutyGroup 在数据库中创建一个新的 MonitorOnDutyGroup 记录
func (p *prometheusDao) CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	if group == nil {
		p.l.Error("CreateMonitorOnDutyGroup 失败: group 为 nil")
		return fmt.Errorf("monitorOnDutyGroup 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(group).Error; err != nil {
		p.l.Error("创建 MonitorOnDutyGroup 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorOnDutyGroupById 根据 ID 获取 MonitorOnDutyGroup 记录
func (p *prometheusDao) GetMonitorOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	if id <= 0 {
		p.l.Error("GetMonitorOnDutyGroupById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var group model.MonitorOnDutyGroup
	if err := p.db.WithContext(ctx).First(&group, id).Error; err != nil {
		p.l.Error("GetMonitorOnDutyGroupById 获取记录失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &group, nil
}

// CreateMonitorOnDutyGroupChange 在数据库中创建一个新的 MonitorOnDutyChange 记录
func (p *prometheusDao) CreateMonitorOnDutyGroupChange(ctx context.Context, change *model.MonitorOnDutyChange) error {
	if change == nil {
		p.l.Error("CreateMonitorOnDutyGroupChange 失败: change 为 nil")
		return fmt.Errorf("monitorOnDutyGroupChange 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(change).Error; err != nil {
		p.l.Error("创建 MonitorOnDutyGroupChange 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorOnDutyChangesByGroupAndTimeRange 获取指定值班组在指定时间范围内的值班计划变更
func (p *prometheusDao) GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime time.Time) ([]*model.MonitorOnDutyChange, error) {
	if groupID <= 0 {
		p.l.Error("GetMonitorOnDutyChangesByGroupAndTimeRange 失败: 无效的 groupID", zap.Int("groupID", groupID))
		return nil, fmt.Errorf("无效的 groupID: %d", groupID)
	}

	var changes []*model.MonitorOnDutyChange
	if err := p.db.WithContext(ctx).
		Where("on_duty_group_id = ?", groupID).
		Where("date >= ?", startTime).
		Where("date <= ?", endTime).
		Find(&changes).Error; err != nil {
		p.l.Error("GetMonitorOnDutyChangesByGroupAndTimeRange 获取变更记录失败", zap.Error(err), zap.Int("groupID", groupID))
		return nil, err
	}

	return changes, nil
}

// UpdateMonitorOnDutyGroup 更新现有的 MonitorOnDutyGroup 记录
func (p *prometheusDao) UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	if group == nil {
		p.l.Error("UpdateMonitorOnDutyGroup 失败: group 为 nil")
		return fmt.Errorf("monitorOnDutyGroup 不能为空")
	}

	if group.ID == 0 {
		p.l.Error("UpdateMonitorOnDutyGroup 失败: ID 为 0", zap.Any("group", group))
		return fmt.Errorf("monitorOnDutyGroup 的 ID 必须设置且非零")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("id = ?", group.ID).
		Updates(group)

	if result.Error != nil {
		p.l.Error("UpdateMonitorOnDutyGroup 更新记录失败", zap.Error(result.Error), zap.Int("id", group.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorOnDutyGroup 未找到要更新的记录", zap.Int("id", group.ID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorOnDutyGroup", group.ID)
	}

	return nil
}

// GetMonitorSendGroupByOnDutyGroupId 根据 onDutyGroupID 获取 MonitorSendGroup 列表
func (p *prometheusDao) GetMonitorSendGroupByOnDutyGroupId(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, error) {
	if onDutyGroupID <= 0 {
		p.l.Error("GetMonitorSendGroupByOnDutyGroupId 失败: 无效的 onDutyGroupID", zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, fmt.Errorf("无效的 onDutyGroupID: %d", onDutyGroupID)
	}

	var sendGroups []*model.MonitorSendGroup
	if err := p.db.WithContext(ctx).
		Where("on_duty_group_id = ?", onDutyGroupID).
		Find(&sendGroups).Error; err != nil {
		p.l.Error("GetMonitorSendGroupByOnDutyGroupId 获取发送组失败", zap.Error(err), zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, err
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

// CreateMonitorAlertManagerPool 在数据库中创建一个新的 MonitorAlertManagerPool 记录
func (p *prometheusDao) CreateMonitorAlertManagerPool(ctx context.Context, pool *model.MonitorAlertManagerPool) error {
	if pool == nil {
		p.l.Error("CreateMonitorAlertManagerPool 失败: pool 为 nil")
		return fmt.Errorf("monitorAlertManagerPool 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(pool).Error; err != nil {
		p.l.Error("创建 MonitorAlertManagerPool 失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorAlertManagerPool 更新现有的 MonitorAlertManagerPool 记录
func (p *prometheusDao) UpdateMonitorAlertManagerPool(ctx context.Context, pool *model.MonitorAlertManagerPool) error {
	if pool == nil {
		p.l.Error("UpdateMonitorAlertManagerPool 失败: pool 为 nil")
		return fmt.Errorf("monitorAlertManagerPool 不能为空")
	}

	if pool.ID == 0 {
		p.l.Error("UpdateMonitorAlertManagerPool 失败: ID 为 0", zap.Any("pool", pool))
		return fmt.Errorf("monitorAlertManagerPool 的 ID 必须设置且非零")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ?", pool.ID).
		Updates(pool)

	if result.Error != nil {
		p.l.Error("UpdateMonitorAlertManagerPool 更新记录失败", zap.Error(result.Error), zap.Int("id", pool.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorAlertManagerPool 未找到要更新的记录", zap.Int("id", pool.ID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorAlertManagerPool", pool.ID)
	}

	return nil
}

func (p *prometheusDao) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	if id <= 0 {
		p.l.Error("DeleteMonitorAlertManagerPool 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := p.db.WithContext(ctx).
		Delete(&model.MonitorAlertManagerPool{}, id)

	if result.Error != nil {
		p.l.Error("DeleteMonitorAlertManagerPool 删除记录失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertManagerPool 失败: %w", id, result.Error)
	}

	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorAlertManagerPool 未找到要删除的记录", zap.Int("id", id))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorAlertManagerPool", id)
	}

	return nil
}

// SearchMonitorAlertManagerPoolByName 通过名称搜索 MonitorAlertManagerPool
func (p *prometheusDao) SearchMonitorAlertManagerPoolByName(ctx context.Context, name string) ([]*model.MonitorAlertManagerPool, error) {
	var pools []*model.MonitorAlertManagerPool

	if err := p.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&pools).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorAlertManagerPool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// SearchMonitorOnDutyGroupByName 通过名称搜索 MonitorOnDutyGroup
func (p *prometheusDao) SearchMonitorOnDutyGroupByName(ctx context.Context, name string) ([]*model.MonitorOnDutyGroup, error) {
	var groups []*model.MonitorOnDutyGroup

	if err := p.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&groups).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorOnDutyGroup 失败", zap.Error(err))
		return nil, err
	}

	return groups, nil
}

// SearchMonitorSendGroupByName 通过名称搜索 MonitorSendGroup
func (p *prometheusDao) SearchMonitorSendGroupByName(ctx context.Context, name string) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	if err := p.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&sendGroups).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorSendGroup 失败", zap.Error(err))
		return nil, err
	}

	return sendGroups, nil
}

// GetMonitorSendGroupList 获取所有 MonitorSendGroup 记录
func (p *prometheusDao) GetMonitorSendGroupList(ctx context.Context) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	if err := p.db.WithContext(ctx).Find(&sendGroups).Error; err != nil {
		p.l.Error("获取所有 MonitorSendGroup 失败", zap.Error(err))
		return nil, err
	}

	if len(sendGroups) == 0 {
		p.l.Info("未找到任何 MonitorSendGroup 记录")
	}

	return sendGroups, nil
}

// GetMonitorSendGroupById 根据 ID 获取 MonitorSendGroup 记录
func (p *prometheusDao) GetMonitorSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error) {
	if id <= 0 {
		p.l.Error("GetMonitorSendGroupById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var sendGroup model.MonitorSendGroup
	if err := p.db.WithContext(ctx).First(&sendGroup, id).Error; err != nil {
		p.l.Error("GetMonitorSendGroupById 获取记录失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &sendGroup, nil
}

// CreateMonitorSendGroup 在数据库中创建一个新的 MonitorSendGroup 记录
func (p *prometheusDao) CreateMonitorSendGroup(ctx context.Context, sendGroup *model.MonitorSendGroup) error {
	if sendGroup == nil {
		p.l.Error("CreateMonitorSendGroup 失败: sendGroup 为 nil")
		return fmt.Errorf("monitorSendGroup 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(sendGroup).Error; err != nil {
		p.l.Error("创建 MonitorSendGroup 失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorSendGroup 更新现有的 MonitorSendGroup 记录
func (p *prometheusDao) UpdateMonitorSendGroup(ctx context.Context, sendGroup *model.MonitorSendGroup) error {
	if sendGroup == nil {
		p.l.Error("UpdateMonitorSendGroup 失败: sendGroup 为 nil")
		return fmt.Errorf("monitorSendGroup 不能为空")
	}

	if sendGroup.ID == 0 {
		p.l.Error("UpdateMonitorSendGroup 失败: ID 为 0", zap.Any("sendGroup", sendGroup))
		return fmt.Errorf("monitorSendGroup 的 ID 必须设置且非零")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("id = ?", sendGroup.ID).
		Updates(sendGroup)

	if result.Error != nil {
		p.l.Error("UpdateMonitorSendGroup 更新记录失败", zap.Error(result.Error), zap.Int("id", sendGroup.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorSendGroup 未找到要更新的记录", zap.Int("id", sendGroup.ID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorSendGroup", sendGroup.ID)
	}

	return nil
}

func (p *prometheusDao) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	if id <= 0 {
		p.l.Error("DeleteMonitorSendGroup 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := p.db.WithContext(ctx).
		Delete(&model.MonitorSendGroup{}, id)

	if result.Error != nil {
		p.l.Error("DeleteMonitorSendGroup 删除记录失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorSendGroup 失败: %w", id, result.Error)
	}

	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorSendGroup 未找到要删除的记录", zap.Int("id", id))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorSendGroup", id)
	}

	return nil
}

// SearchMonitorAlertRuleByName 通过名称搜索 MonitorAlertRule
func (p *prometheusDao) SearchMonitorAlertRuleByName(ctx context.Context, name string) ([]*model.MonitorAlertRule, error) {
	var alertRules []*model.MonitorAlertRule

	if err := p.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&alertRules).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorAlertRule 失败", zap.Error(err))
		return nil, err
	}

	return alertRules, nil
}

// GetMonitorAlertRuleList 获取所有 MonitorAlertRule 记录
func (p *prometheusDao) GetMonitorAlertRuleList(ctx context.Context) ([]*model.MonitorAlertRule, error) {
	var alertRules []*model.MonitorAlertRule

	if err := p.db.WithContext(ctx).Find(&alertRules).Error; err != nil {
		p.l.Error("获取所有 MonitorAlertRule 失败", zap.Error(err))
		return nil, err
	}

	if len(alertRules) == 0 {
		p.l.Info("未找到任何 MonitorAlertRule 记录")
	}

	return alertRules, nil
}

// CreateMonitorAlertRule 在数据库中创建一个新的 MonitorAlertRule 记录
func (p *prometheusDao) CreateMonitorAlertRule(ctx context.Context, alertRule *model.MonitorAlertRule) error {
	if alertRule == nil {
		p.l.Error("CreateMonitorAlertRule 失败: alertRule 为 nil")
		return fmt.Errorf("monitorAlertRule 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(alertRule).Error; err != nil {
		p.l.Error("创建 MonitorAlertRule 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorAlertRuleById 根据 ID 获取 MonitorAlertRule 记录
func (p *prometheusDao) GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error) {
	if id <= 0 {
		p.l.Error("GetMonitorAlertRuleById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertRule model.MonitorAlertRule
	if err := p.db.WithContext(ctx).First(&alertRule, id).Error; err != nil {
		p.l.Error("GetMonitorAlertRuleById 获取记录失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertRule, nil
}

// UpdateMonitorAlertRule 更新现有的 MonitorAlertRule 记录
func (p *prometheusDao) UpdateMonitorAlertRule(ctx context.Context, alertRule *model.MonitorAlertRule) error {
	if alertRule == nil {
		p.l.Error("UpdateMonitorAlertRule 失败: alertRule 为 nil")
		return fmt.Errorf("monitorAlertRule 不能为空")
	}

	if alertRule.ID == 0 {
		p.l.Error("UpdateMonitorAlertRule 失败: ID 为 0", zap.Any("alertRule", alertRule))
		return fmt.Errorf("monitorAlertRule 的 ID 必须设置且非零")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", alertRule.ID).
		Updates(alertRule)

	if result.Error != nil {
		p.l.Error("UpdateMonitorAlertRule 更新记录失败", zap.Error(result.Error), zap.Int("id", alertRule.ID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("UpdateMonitorAlertRule 未找到要更新的记录", zap.Int("id", alertRule.ID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorAlertRule", alertRule.ID)
	}

	return nil
}

// EnableSwitchMonitorAlertRule 启用或禁用 MonitorAlertRule
func (p *prometheusDao) EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		p.l.Error("EnableSwitchMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	// 假设 MonitorAlertRule 有一个 "enable" 字段
	result := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", ruleID).
		Update("enable", gorm.Expr("NOT enable"))

	if result.Error != nil {
		p.l.Error("EnableSwitchMonitorAlertRule 更新记录失败", zap.Error(result.Error), zap.Int("ruleID", ruleID))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("EnableSwitchMonitorAlertRule 未找到要更新的记录", zap.Int("ruleID", ruleID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorAlertRule", ruleID)
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量启用或禁用 MonitorAlertRule
func (p *prometheusDao) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		p.l.Error("BatchEnableSwitchMonitorAlertRule 失败: ruleIDs 为空")
		return fmt.Errorf("ruleIDs 不能为空")
	}

	result := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id IN ?", ruleIDs).
		Update("enable", gorm.Expr("NOT enable"))

	if result.Error != nil {
		p.l.Error("BatchEnableSwitchMonitorAlertRule 更新记录失败", zap.Error(result.Error), zap.Ints("ruleIDs", ruleIDs))
		return result.Error
	}

	if result.RowsAffected == 0 {
		p.l.Warn("BatchEnableSwitchMonitorAlertRule 未找到要更新的记录", zap.Ints("ruleIDs", ruleIDs))
		return fmt.Errorf("未找到任何指定 ID 的 MonitorAlertRule")
	}

	return nil
}

// DeleteMonitorAlertRule 根据 ruleID 删除 MonitorAlertRule 记录
func (p *prometheusDao) DeleteMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		p.l.Error("DeleteMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := p.db.WithContext(ctx).
		Delete(&model.MonitorAlertRule{}, ruleID)

	if result.Error != nil {
		p.l.Error("DeleteMonitorAlertRule 删除记录失败", zap.Error(result.Error), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertRule 失败: %w", ruleID, result.Error)
	}

	if result.RowsAffected == 0 {
		p.l.Warn("DeleteMonitorAlertRule 未找到要删除的记录", zap.Int("ruleID", ruleID))
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorAlertRule", ruleID)
	}

	return nil
}
