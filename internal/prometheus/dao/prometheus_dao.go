package dao

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

	GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, error)
	GetMonitorAlertEventList(ctx context.Context) ([]*model.MonitorAlertEvent, error)
	EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error
	GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error)
	UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error

	SearchMonitorRecordRuleByName(ctx context.Context, name string) ([]*model.MonitorRecordRule, error)
	GetMonitorRecordRuleList(ctx context.Context) ([]*model.MonitorRecordRule, error)
	CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error)
	UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, ruleID int) error
	EnableSwitchMonitorRecordRule(ctx context.Context, ruleID int) error
	GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, error)

	CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error)
	CheckMonitorSendGroupNameExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error)
	CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error)
	CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error)
	CheckMonitorRecordRuleExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error)
	CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error)
	CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error)
	CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error)
	GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error)
	CheckMonitorOnDutyGroupExists(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (bool, error)
	CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error)
	GetMonitorAlertManagerPoolById(ctx context.Context, id int) (*model.MonitorAlertManagerPool, error)
}

type prometheusDao struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

// NewPrometheusDAO 创建一个新的 PrometheusDAO 实例
func NewPrometheusDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) PrometheusDao {
	return &prometheusDao{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

// GetAllMonitorScrapePool 从数据库中获取所有 MonitorScrapePool 记录
func (p *prometheusDao) GetAllMonitorScrapePool(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	var pools []*model.MonitorScrapePool

	if err := p.db.WithContext(ctx).Find(&pools).Error; err != nil {
		p.l.Error("获取所有 MonitorScrapePool 记录失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// CreateMonitorScrapePool 在数据库中创建一个新的 MonitorScrapePool 记录
func (p *prometheusDao) CreateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error {
	if pool == nil {
		p.l.Error("CreateMonitorScrapePool 失败：pool 为 nil")
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
		p.l.Error("GetMonitorScrapePoolById 失败：无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID：%d", id)
	}

	var pool model.MonitorScrapePool
	if err := p.db.WithContext(ctx).First(&pool, id).Error; err != nil {
		p.l.Error("根据 ID 获取 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &pool, nil
}

// UpdateMonitorScrapePool 更新现有的 MonitorScrapePool 记录
func (p *prometheusDao) UpdateMonitorScrapePool(ctx context.Context, pool *model.MonitorScrapePool) error {
	if pool == nil {
		p.l.Error("UpdateMonitorScrapePool 失败：pool 为 nil")
		return fmt.Errorf("monitorScrapePool 不能为空")
	}

	if pool.ID == 0 {
		p.l.Error("UpdateMonitorScrapePool 失败：ID 为 0", zap.Any("pool", pool))
		return fmt.Errorf("monitorScrapePool 的 ID 必须设置且非零")
	}

	// 使用 Updates 方法时，应当使用非零值结构体，以避免更新零值字段
	if err := p.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", pool.ID).
		Updates(pool).Error; err != nil {
		p.l.Error("更新 MonitorScrapePool 失败", zap.Error(err), zap.Int("id", pool.ID))
		return err
	}

	return nil
}

// DeleteMonitorScrapePool 根据 ID 删除 MonitorScrapePool 记录
func (p *prometheusDao) DeleteMonitorScrapePool(ctx context.Context, poolId int) error {
	if poolId <= 0 {
		p.l.Error("DeleteMonitorScrapePool 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return fmt.Errorf("无效的 poolId: %d", poolId)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorScrapePool{}, poolId)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorScrapePool 失败", zap.Error(err), zap.Int("poolId", poolId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapePool 失败: %w", poolId, err)
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
		p.l.Error("获取 MonitorScrapeJob 失败", zap.Error(err), zap.Int("poolId", poolId))
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

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("id = ?", job.ID).
		Updates(job).Error; err != nil {
		p.l.Error("更新 MonitorScrapeJob 失败", zap.Error(err), zap.Int("id", job.ID))
		return err
	}

	return nil
}

// DeleteMonitorScrapeJob 根据 ID 删除 MonitorScrapeJob 记录
func (p *prometheusDao) DeleteMonitorScrapeJob(ctx context.Context, jobId int) error {
	if jobId <= 0 {
		p.l.Error("DeleteMonitorScrapeJob 失败: 无效的 jobId", zap.Int("jobId", jobId))
		return fmt.Errorf("无效的 jobId: %d", jobId)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorScrapeJob{}, jobId)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorScrapeJob 失败", zap.Error(err), zap.Int("jobId", jobId))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorScrapeJob 失败: %w", jobId, err)
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
		p.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("poolId", poolId))
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
		p.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("poolId", poolId))
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
		p.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("poolId", poolId))
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
		p.l.Error("获取 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &group, nil
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

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("id = ?", group.ID).
		Updates(group).Error; err != nil {
		p.l.Error("更新 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("id", group.ID))
		return err
	}

	return nil
}

// DeleteMonitorOnDutyGroup 删除指定 ID 的 MonitorOnDutyGroup
func (p *prometheusDao) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	if id <= 0 {
		p.l.Error("DeleteMonitorOnDutyGroup 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorOnDutyGroup{}, id)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("ID", id))
		return err
	}

	return nil
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
		Where("date BETWEEN ? AND ?", startTime, endTime).
		Find(&changes).Error; err != nil {
		p.l.Error("获取值班计划变更失败", zap.Error(err), zap.Int("groupID", groupID))
		return nil, err
	}

	return changes, nil
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
		p.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, err
	}

	return sendGroups, nil
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

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ?", pool.ID).
		Updates(pool).Error; err != nil {
		p.l.Error("更新 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", pool.ID))
		return err
	}

	return nil
}

// DeleteMonitorAlertManagerPool 根据 ID 删除 MonitorAlertManagerPool 记录
func (p *prometheusDao) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	if id <= 0 {
		p.l.Error("DeleteMonitorAlertManagerPool 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorAlertManagerPool{}, id)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertManagerPool 失败: %w", id, err)
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
		p.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
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

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("id = ?", sendGroup.ID).
		Updates(sendGroup).Error; err != nil {
		p.l.Error("更新 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", sendGroup.ID))
		return err
	}

	return nil
}

// DeleteMonitorSendGroup 根据 ID 删除 MonitorSendGroup 记录
func (p *prometheusDao) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	if id <= 0 {
		p.l.Error("DeleteMonitorSendGroup 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorSendGroup{}, id)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorSendGroup 失败: %w", id, err)
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
		p.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", id))
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

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", alertRule.ID).
		Updates(alertRule).Error; err != nil {
		p.l.Error("更新 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", alertRule.ID))
		return err
	}

	return nil
}

// EnableSwitchMonitorAlertRule 启用或禁用 MonitorAlertRule
func (p *prometheusDao) EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		p.l.Error("EnableSwitchMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", ruleID).
		Update("enable", gorm.Expr("NOT enable")).Error; err != nil {
		p.l.Error("更新 MonitorAlertRule 状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量启用或禁用 MonitorAlertRule
func (p *prometheusDao) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		p.l.Error("BatchEnableSwitchMonitorAlertRule 失败: ruleIDs 为空")
		return fmt.Errorf("ruleIDs 不能为空")
	}

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id IN ?", ruleIDs).
		Update("enable", gorm.Expr("NOT enable")).Error; err != nil {
		p.l.Error("批量更新 MonitorAlertRule 状态失败", zap.Error(err), zap.Ints("ruleIDs", ruleIDs))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 根据 ruleID 删除 MonitorAlertRule 记录
func (p *prometheusDao) DeleteMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		p.l.Error("DeleteMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorAlertRule{}, ruleID)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorAlertRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertRule 失败: %w", ruleID, err)
	}

	return nil
}

// GetMonitorAlertEventById 根据 ID 获取 MonitorAlertEvent 记录
func (p *prometheusDao) GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		p.l.Error("GetMonitorAlertEventById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent
	if err := p.db.WithContext(ctx).First(&alertEvent, id).Error; err != nil {
		p.l.Error("获取 MonitorAlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

// SearchMonitorAlertEventByName 通过名称搜索 MonitorAlertEvent
func (p *prometheusDao) SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, error) {
	var alertEvents []*model.MonitorAlertEvent

	if err := p.db.WithContext(ctx).
		Where("alert_name LIKE ?", "%"+name+"%").
		Find(&alertEvents).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorAlertEvent 失败", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return alertEvents, nil
}

// GetMonitorAlertEventList 获取所有 MonitorAlertEvent 记录
func (p *prometheusDao) GetMonitorAlertEventList(ctx context.Context) ([]*model.MonitorAlertEvent, error) {
	var alertEvents []*model.MonitorAlertEvent

	if err := p.db.WithContext(ctx).Find(&alertEvents).Error; err != nil {
		p.l.Error("获取 MonitorAlertEvent 列表失败", zap.Error(err))
		return nil, err
	}

	return alertEvents, nil
}

// EventAlertClaim 更新事件的认领信息
func (p *prometheusDao) EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error {
	if event == nil {
		p.l.Error("EventAlertClaim 失败: event 为 nil")
		return fmt.Errorf("event 不能为空")
	}

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("id = ?", event.ID).
		Updates(event).Error; err != nil {
		p.l.Error("EventAlertClaim 更新失败", zap.Error(err), zap.Int("id", event.ID))
		return err
	}

	return nil
}

// GetAlertEventByID 根据 ID 获取 MonitorAlertEvent 记录
func (p *prometheusDao) GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		p.l.Error("GetAlertEventByID 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent
	if err := p.db.WithContext(ctx).First(&alertEvent, id).Error; err != nil {
		p.l.Error("获取 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

// GetAlertPoolByID 根据 poolID 获取 MonitorAlertManagerPool 记录
func (p *prometheusDao) GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error) {
	if poolID <= 0 {
		p.l.Error("GetAlertPoolByID 失败: 无效的 poolID", zap.Int("poolID", poolID))
		return nil, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var alertPool model.MonitorAlertManagerPool
	if err := p.db.WithContext(ctx).Where("id = ?", poolID).First(&alertPool).Error; err != nil {
		p.l.Error("获取 AlertPool 失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, err
	}

	return &alertPool, nil
}

// UpdateAlertEvent 更新 AlertEvent 记录
func (p *prometheusDao) UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error {
	if alertEvent == nil {
		p.l.Error("UpdateAlertEvent 失败: alertEvent 为 nil")
		return fmt.Errorf("alertEvent 不能为空")
	}

	if err := p.db.WithContext(ctx).Save(alertEvent).Error; err != nil {
		p.l.Error("更新 AlertEvent 失败", zap.Error(err), zap.Int("id", alertEvent.ID))
		return err
	}

	return nil
}

// SearchMonitorRecordRuleByName 通过名称搜索 MonitorRecordRule
func (p *prometheusDao) SearchMonitorRecordRuleByName(ctx context.Context, name string) ([]*model.MonitorRecordRule, error) {
	if name == "" {
		return nil, fmt.Errorf("name 不能为空")
	}

	var recordRules []*model.MonitorRecordRule

	if err := p.db.WithContext(ctx).
		Where("name LIKE ?", "%"+name+"%").
		Find(&recordRules).Error; err != nil {
		p.l.Error("通过名称搜索 MonitorRecordRule 失败", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return recordRules, nil
}

// GetMonitorRecordRuleList 获取所有 MonitorRecordRule 记录
func (p *prometheusDao) GetMonitorRecordRuleList(ctx context.Context) ([]*model.MonitorRecordRule, error) {
	var recordRules []*model.MonitorRecordRule

	if err := p.db.WithContext(ctx).Find(&recordRules).Error; err != nil {
		p.l.Error("获取所有 MonitorRecordRule 失败", zap.Error(err))
		return nil, err
	}

	return recordRules, nil
}

// CreateMonitorRecordRule 在数据库中创建一个新的 MonitorRecordRule 记录
func (p *prometheusDao) CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
	if recordRule == nil {
		p.l.Error("CreateMonitorRecordRule 失败: recordRule 为 nil")
		return fmt.Errorf("monitorRecordRule 不能为空")
	}

	if err := p.db.WithContext(ctx).Create(recordRule).Error; err != nil {
		p.l.Error("创建 MonitorRecordRule 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorRecordRuleById 根据 ID 获取 MonitorRecordRule 记录
func (p *prometheusDao) GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error) {
	if id <= 0 {
		p.l.Error("GetMonitorRecordRuleById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var recordRule model.MonitorRecordRule
	if err := p.db.WithContext(ctx).First(&recordRule, id).Error; err != nil {
		p.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &recordRule, nil
}

// UpdateMonitorRecordRule 更新现有的 MonitorRecordRule 记录
func (p *prometheusDao) UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
	if recordRule == nil {
		p.l.Error("UpdateMonitorRecordRule 失败: recordRule 为 nil")
		return fmt.Errorf("monitorRecordRule 不能为空")
	}

	if recordRule.ID == 0 {
		p.l.Error("UpdateMonitorRecordRule 失败: ID 为 0", zap.Any("recordRule", recordRule))
		return fmt.Errorf("monitorRecordRule 的 ID 必须设置且非零")
	}

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", recordRule.ID).
		Updates(recordRule).Error; err != nil {
		p.l.Error("更新 MonitorRecordRule 失败", zap.Error(err), zap.Int("id", recordRule.ID))
		return err
	}

	return nil
}

// DeleteMonitorRecordRule 根据 ruleID 删除 MonitorRecordRule 记录
func (p *prometheusDao) DeleteMonitorRecordRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		p.l.Error("DeleteMonitorRecordRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := p.db.WithContext(ctx).Delete(&model.MonitorRecordRule{}, ruleID)
	if err := result.Error; err != nil {
		p.l.Error("删除 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorRecordRule 失败: %w", ruleID, err)
	}

	return nil
}

// EnableSwitchMonitorRecordRule 启用或禁用 MonitorRecordRule
func (p *prometheusDao) EnableSwitchMonitorRecordRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		p.l.Error("EnableSwitchMonitorRecordRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	// 获取当前规则的状态
	var rule model.MonitorRecordRule
	if err := p.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", ruleID).
		First(&rule).Error; err != nil {
		p.l.Error("查询 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	// 切换状态，1->2 或 2->1·
	newEnable := 1
	if rule.Enable == 1 {
		newEnable = 2
	} else if rule.Enable == 2 {
		newEnable = 1
	}

	// 更新状态
	if err := p.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", ruleID).
		Update("enable", newEnable).Error; err != nil {
		p.l.Error("更新 MonitorRecordRule 状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// GetAssociatedResourcesBySendGroupId 根据 sendGroupId 获取关联的 MonitorScrapePool 资源
func (p *prometheusDao) GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, error) {
	if sendGroupId <= 0 {
		p.l.Error("GetAssociatedResourcesBySendGroupId 失败: 无效的 sendGroupId", zap.Int("sendGroupId", sendGroupId))
		return nil, fmt.Errorf("无效的 sendGroupId: %d", sendGroupId)
	}

	var scrapePools []*model.MonitorAlertRule

	if err := p.db.WithContext(ctx).
		Where("send_group_id = ?", sendGroupId).
		Find(&scrapePools).Error; err != nil {
		p.l.Error("获取关联资源失败", zap.Error(err), zap.Int("sendGroupId", sendGroupId))
		return nil, err
	}

	return scrapePools, nil
}

// 以下为检查存在性的方法

func (p *prometheusDao) CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("id = ?", sendGroup.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorSendGroupNameExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("name = ?", sendGroup.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", alertRule.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("name = ?", alertRule.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorRecordRuleExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", recordRule.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("name = ?", recordRule.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorScrapePoolExists(ctx context.Context, scrapePool *model.MonitorScrapePool) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorScrapePool{}).
		Where("id = ?", scrapePool.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorScrapeJobExists(ctx context.Context, name string) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorScrapeJob{}).
		Where("name = ?", name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) GetMonitorScrapeJobById(ctx context.Context, id int) (*model.MonitorScrapeJob, error) {
	if id <= 0 {
		p.l.Error("GetMonitorScrapeJobById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var scrapeJob model.MonitorScrapeJob

	if err := p.db.WithContext(ctx).Where("id = ?", id).First(&scrapeJob).Error; err != nil {
		p.l.Error("获取 MonitorScrapeJob 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &scrapeJob, nil
}

func (p *prometheusDao) CheckMonitorOnDutyGroupExists(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("id = ?", onDutyGroup.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error) {
	var count int64

	if err := p.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ?", alertManagerPool.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *prometheusDao) GetMonitorAlertManagerPoolById(ctx context.Context, id int) (*model.MonitorAlertManagerPool, error) {
	if id <= 0 {
		p.l.Error("GetMonitorAlertManagerPoolById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertManagerPool model.MonitorAlertManagerPool

	if err := p.db.WithContext(ctx).Where("id = ?", id).First(&alertManagerPool).Error; err != nil {
		p.l.Error("获取 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertManagerPool, nil
}
