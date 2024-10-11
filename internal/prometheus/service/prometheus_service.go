package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"github.com/prometheus/alertmanager/types"
	promModel "github.com/prometheus/common/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sync"
	"time"
)

type PrometheusService interface {
	GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, id int) error

	GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, id int) error

	GetMonitorPrometheusYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string
	GetMonitorAlertManagerYaml(ctx context.Context, ip string) string

	GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTime string, endTime string) ([]*model.MonitorOnDutyChange, error)

	GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error

	GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error)
	CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	DeleteMonitorSendGroup(ctx context.Context, id int) error
	GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error)
	PromqlExprCheck(ctx context.Context, expr string) (bool, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error
	DeleteMonitorAlertRule(ctx context.Context, id int) error
	BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error

	GetMonitorAlertEventList(ctx context.Context, searchName *string) ([]*model.MonitorAlertEvent, error)
	EventAlertSilence(ctx context.Context, id int, event *model.AlertEventSilenceRequest, userId int) error
	EventAlertClaim(ctx context.Context, id int, userId int) error
	BatchEventAlertSilence(ctx context.Context, ids *model.BatchEventAlertSilenceRequest, userId int) error
	GetMonitorRecordRuleList(ctx context.Context, searchName *string) ([]*model.MonitorRecordRule, error)
	CreateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error
	UpdateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, id int) error
	BatchDeleteMonitorRecordRule(ctx context.Context, ids []int) error
	EnableSwitchMonitorRecordRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorRecordRule(ctx context.Context, ids []int) error
}

type prometheusService struct {
	dao     dao.PrometheusDao
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewPrometheusService(dao dao.PrometheusDao, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) PrometheusService {
	return &prometheusService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

// GetMonitorScrapePoolList 获取监控抓取池列表，可选根据名称过滤
func (p *prometheusService) GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error) {
	return pkg.HandleList(ctx, search,
		p.dao.SearchMonitorScrapePoolsByName, // 搜索函数
		p.dao.GetAllMonitorScrapePool)        // 获取所有函数
}

// CreateMonitorScrapePool 创建新的监控抓取池
func (p *prometheusService) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 检查抓取池 IP 是否已存在
	exists, err := p.dao.CheckMonitorScrapePoolExists(ctx, monitorScrapePool)
	if err != nil {
		p.l.Error("创建抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取池 IP 已存在")
	}

	// 创建抓取池
	if err := p.dao.CreateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		p.l.Error("创建抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建抓取池成功", zap.Int("id", monitorScrapePool.ID))
	return nil
}

// UpdateMonitorScrapePool 更新现有的监控抓取池
func (p *prometheusService) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 确保要更新的抓取池存在
	existingPool, err := p.dao.GetMonitorScrapePoolById(ctx, monitorScrapePool.ID)
	if err != nil {
		p.l.Error("更新抓取池失败：获取抓取池时出错", zap.Error(err))
		return err
	}
	if existingPool == nil {
		return errors.New("抓取池不存在")
	}

	// 检查新的抓取池 IP 是否已存在
	exists, err := p.dao.CheckMonitorScrapePoolExists(ctx, monitorScrapePool)
	if err != nil {
		p.l.Error("更新抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取池 IP 已存在")
	}

	// 更新抓取池
	if err := p.dao.UpdateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		p.l.Error("更新抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("更新抓取池成功", zap.Int("id", monitorScrapePool.ID))
	return nil
}

// DeleteMonitorScrapePool 删除指定的监控抓取池
func (p *prometheusService) DeleteMonitorScrapePool(ctx context.Context, id int) error {
	// 检查抓取池是否有相关的抓取作业
	jobs, err := p.dao.GetMonitorScrapeJobsByPoolId(ctx, id)
	if err != nil {
		p.l.Error("删除抓取池失败：获取抓取作业时出错", zap.Error(err))
		return err
	}
	if len(jobs) > 0 {
		return errors.New("抓取池存在相关抓取作业，无法删除")
	}

	// 删除抓取池
	if err := p.dao.DeleteMonitorScrapePool(ctx, id); err != nil {
		p.l.Error("删除抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除抓取池成功", zap.Int("id", id))
	return nil
}

// GetMonitorScrapeJobList 获取监控抓取作业列表，可选根据名称过滤
func (p *prometheusService) GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error) {
	return pkg.HandleList(ctx, search,
		p.dao.SearchMonitorScrapeJobsByName, // 搜索函数
		p.dao.GetAllMonitorScrapeJobs)       // 获取所有函数
}

// CreateMonitorScrapeJob 创建新的监控抓取作业
func (p *prometheusService) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	// 检查抓取作业是否已存在
	exists, err := p.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob)
	if err != nil {
		p.l.Error("创建抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取作业已存在")
	}

	// 创建抓取作业
	if err := p.dao.CreateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		p.l.Error("创建抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建抓取作业成功", zap.Int("id", monitorScrapeJob.ID))
	return nil
}

// UpdateMonitorScrapeJob 更新现有的监控抓取作业
func (p *prometheusService) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	// 确保要更新的抓取作业存在
	existingJob, err := p.dao.GetMonitorScrapeJobById(ctx, monitorScrapeJob.ID)
	if err != nil {
		p.l.Error("更新抓取作业失败：获取抓取作业时出错", zap.Error(err))
		return err
	}
	if existingJob == nil {
		return errors.New("抓取作业不存在")
	}

	// 检查新的抓取作业名称是否已存在
	exists, err := p.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob)
	if err != nil {
		p.l.Error("更新抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取作业名称已存在")
	}

	// 更新抓取作业
	if err := p.dao.UpdateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		p.l.Error("更新抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("更新抓取作业成功", zap.Int("id", monitorScrapeJob.ID))
	return nil
}

// DeleteMonitorScrapeJob 删除指定的监控抓取作业
func (p *prometheusService) DeleteMonitorScrapeJob(ctx context.Context, id int) error {
	// 删除抓取作业
	if err := p.dao.DeleteMonitorScrapeJob(ctx, id); err != nil {
		p.l.Error("删除抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除抓取作业成功", zap.Int("id", id))
	return nil
}

// GetMonitorPrometheusYaml 获取 Prometheus 主配置 YAML
func (p *prometheusService) GetMonitorPrometheusYaml(ctx context.Context, ip string) string {
	return p.cache.GetPrometheusMainConfigByIP(ip)
}

// GetMonitorPrometheusAlertRuleYaml 获取 Prometheus 告警规则 YAML
func (p *prometheusService) GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string {
	return p.cache.GetPrometheusAlertRuleConfigYamlByIp(ip)
}

// GetMonitorPrometheusRecordYaml 获取 Prometheus 记录规则 YAML
func (p *prometheusService) GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string {
	return p.cache.GetPrometheusRecordRuleConfigYamlByIp(ip)
}

// GetMonitorAlertManagerYaml 获取 AlertManager 主配置 YAML
func (p *prometheusService) GetMonitorAlertManagerYaml(ctx context.Context, ip string) string {
	return p.cache.GetAlertManagerMainConfigYamlByIP(ip)
}

// GetMonitorOnDutyGroupList 获取值班组列表，可选根据名称过滤
func (p *prometheusService) GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorOnDutyGroupByName, // 搜索函数
		p.dao.GetAllMonitorOndutyGroup)       // 获取所有函数
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (p *prometheusService) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	// 检查值班组是否已存在
	exists, err := p.dao.CheckMonitorOnDutyGroupExists(ctx, monitorOnDutyGroup)
	if err != nil {
		p.l.Error("创建值班组失败：检查值班组是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("值班组已存在")
	}

	// 创建值班组
	if err := p.dao.CreateMonitorOnDutyGroup(ctx, monitorOnDutyGroup); err != nil {
		p.l.Error("创建值班组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建值班组成功", zap.Int("id", monitorOnDutyGroup.ID))
	return nil
}

// CreateMonitorOnDutyGroupChange 创建值班组变更记录
func (p *prometheusService) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error {
	// 验证值班组 ID 是否有效
	if monitorOnDutyChange.OnDutyGroupID == 0 {
		p.l.Error("创建值班组变更失败：值班组 ID 为空")
		return errors.New("值班组 ID 为空")
	}

	// 确保值班组存在
	_, err := p.dao.GetMonitorOnDutyGroupById(ctx, monitorOnDutyChange.OnDutyGroupID)
	if err != nil {
		p.l.Error("创建值班组变更失败：获取值班组时出错", zap.Error(err))
		return err
	}

	// 获取原始用户信息
	originUser, err := p.userDao.GetUserByID(ctx, monitorOnDutyChange.OriginUserID)
	if err != nil {
		p.l.Error("创建值班组变更失败：获取原始用户时出错", zap.Error(err))
		return err
	}

	// 获取目标用户信息
	targetUser, err := p.userDao.GetUserByID(ctx, monitorOnDutyChange.OnDutyUserID)
	if err != nil {
		p.l.Error("创建值班组变更失败：获取目标用户时出错", zap.Error(err))
		return err
	}

	// 更新用户 ID
	monitorOnDutyChange.OriginUserID = originUser.ID
	monitorOnDutyChange.OnDutyUserID = targetUser.ID

	// 创建值班组变更记录
	if err := p.dao.CreateMonitorOnDutyGroupChange(ctx, monitorOnDutyChange); err != nil {
		p.l.Error("创建值班组变更失败", zap.Error(err))
		return err
	}

	p.l.Info("创建值班组变更记录成功", zap.Int("onDutyGroupID", monitorOnDutyChange.OnDutyGroupID))
	return nil
}

// UpdateMonitorOnDutyGroup 更新现有的值班组
func (p *prometheusService) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	// 确保值班组存在
	existingGroup, err := p.dao.GetMonitorOnDutyGroupById(ctx, monitorOnDutyGroup.ID)
	if err != nil {
		p.l.Error("更新值班组失败：获取值班组时出错", zap.Error(err))
		return err
	}
	if existingGroup == nil {
		return errors.New("值班组不存在")
	}

	// 获取成员用户信息
	users, err := p.userDao.GetUserByUsernames(ctx, monitorOnDutyGroup.UserNames)
	if err != nil {
		p.l.Error("更新值班组失败：获取用户时出错", zap.Error(err))
		return err
	}

	// 更新成员
	monitorOnDutyGroup.Members = users

	// 更新值班组
	if err := p.dao.UpdateMonitorOnDutyGroup(ctx, monitorOnDutyGroup); err != nil {
		p.l.Error("更新值班组失败", zap.Error(err))
		return err
	}

	p.l.Info("更新值班组成功", zap.Int("id", monitorOnDutyGroup.ID))
	return nil
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (p *prometheusService) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	// 检查值班组是否有关联的发送组
	sendGroups, err := p.dao.GetMonitorSendGroupByOnDutyGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("删除值班组失败：获取关联发送组时出错", zap.Error(err))
		return err
	}
	if len(sendGroups) > 0 {
		return errors.New("值班组存在关联发送组，无法删除")
	}

	// 删除值班组
	if err := p.dao.DeleteMonitorOnDutyGroup(ctx, id); err != nil {
		p.l.Error("删除值班组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除值班组成功", zap.Int("id", id))
	return nil
}

// GetMonitorOnDutyGroup 获取指定的值班组
func (p *prometheusService) GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	group, err := p.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		p.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	if group == nil {
		return nil, errors.New("值班组不存在")
	}
	return group, nil
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组在指定时间范围内的值班计划变更
func (p *prometheusService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) ([]*model.MonitorOnDutyChange, error) {
	// 解析开始时间
	startTime, err := time.Parse("2006-01-02", startTimeStr)
	if err != nil {
		p.l.Error("获取未来值班计划失败：开始时间格式错误", zap.String("startTime", startTimeStr), zap.Error(err))
		return nil, errors.New("开始时间格式错误")
	}

	// 解析结束时间
	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		p.l.Error("获取未来值班计划失败：结束时间格式错误", zap.String("endTime", endTimeStr), zap.Error(err))
		return nil, errors.New("结束时间格式错误")
	}

	// 验证时间范围
	if endTime.Before(startTime) {
		errMsg := "结束时间不能早于开始时间"
		p.l.Error(errMsg, zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
		return nil, errors.New(errMsg)
	}

	// 确保值班组存在
	if _, err := p.dao.GetMonitorOnDutyGroupById(ctx, id); err != nil {
		p.l.Error("获取未来值班计划失败：根据 ID 获取值班组时出错", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 获取指定时间范围内的值班计划变更
	changes, err := p.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, id, startTime, endTime)
	if err != nil {
		p.l.Error("获取未来值班计划失败：获取值班计划变更时出错", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return changes, nil
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表，可选根据名称过滤
func (p *prometheusService) GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorAlertManagerPoolByName, // 搜索函数
		p.dao.GetAllAlertManagerPools)             // 获取所有函数
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
func (p *prometheusService) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	// 检查 AlertManager IP 是否已存在
	exists, err := p.dao.CheckMonitorAlertManagerPoolExists(ctx, monitorAlertManagerPool)
	if err != nil {
		p.l.Error("创建 AlertManager 集群池失败：检查是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("AlertManager 集群池 IP 已存在")
	}

	// 创建 AlertManager 集群池
	if err := p.dao.CreateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		p.l.Error("创建 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("创建 AlertManager 集群池成功", zap.Int("id", monitorAlertManagerPool.ID))
	return nil
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
func (p *prometheusService) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	// 确保 AlertManager 集群池存在
	existingPool, err := p.dao.GetMonitorAlertManagerPoolById(ctx, monitorAlertManagerPool.ID)
	if err != nil {
		p.l.Error("更新 AlertManager 集群池失败：获取集群池时出错", zap.Error(err))
		return err
	}
	if existingPool == nil {
		return errors.New("AlertManager 集群池不存在")
	}

	// 检查新的 AlertManager IP 是否已存在
	exists, err := p.dao.CheckMonitorAlertManagerPoolExists(ctx, monitorAlertManagerPool)
	if err != nil {
		p.l.Error("更新 AlertManager 集群池失败：检查是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("AlertManager 集群池 IP 已存在")
	}

	// 更新 AlertManager 集群池
	if err := p.dao.UpdateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		p.l.Error("更新 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("更新 AlertManager 集群池成功", zap.Int("id", monitorAlertManagerPool.ID))
	return nil
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (p *prometheusService) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	// 检查 AlertManager 集群池是否有关联的发送组
	sendGroups, err := p.dao.GetMonitorSendGroupByPoolId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("删除 AlertManager 集群池失败：获取关联发送组时出错", zap.Error(err))
		return err
	}
	if len(sendGroups) > 0 {
		return errors.New("AlertManager 集群池存在关联发送组，无法删除")
	}

	// 删除 AlertManager 集群池
	if err := p.dao.DeleteMonitorAlertManagerPool(ctx, id); err != nil {
		p.l.Error("删除 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	p.l.Info("删除 AlertManager 集群池成功", zap.Int("id", id))
	return nil
}

// GetMonitorSendGroupList 获取发送组列表，支持按名称搜索
func (p *prometheusService) GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorSendGroupByName,
		p.dao.GetMonitorSendGroupList)
}

// CreateMonitorSendGroup 创建发送组
func (p *prometheusService) CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 检查发送组是否已存在
	exists, err := p.dao.CheckMonitorSendGroupExists(ctx, monitorSendGroup)
	if err != nil {
		p.l.Error("创建发送组失败：检查发送组是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("发送组已存在")
	}

	// 创建发送组
	if err := p.dao.CreateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		p.l.Error("创建发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorSendGroup 更新发送组
func (p *prometheusService) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 确保发送组存在
	existingGroup, err := p.dao.GetMonitorSendGroupById(ctx, monitorSendGroup.ID)
	if err != nil {
		p.l.Error("更新发送组失败：获取发送组时出错", zap.Error(err))
		return err
	}
	if existingGroup == nil {
		return errors.New("发送组不存在")
	}

	// 检查发送组名称是否重复
	exists, err := p.dao.CheckMonitorSendGroupNameExists(ctx, monitorSendGroup)
	if err != nil {
		p.l.Error("更新发送组失败：检查发送组名称时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("发送组名称已存在")
	}

	// 更新发送组
	if err := p.dao.UpdateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		p.l.Error("更新发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorSendGroup 删除发送组
func (p *prometheusService) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	// 检查发送组是否有关联的资源
	associatedResources, err := p.dao.GetAssociatedResourcesBySendGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("删除发送组失败：获取关联资源时出错", zap.Error(err))
		return err
	}
	if len(associatedResources) > 0 {
		return errors.New("发送组存在关联资源，无法删除")
	}

	// 删除发送组
	if err := p.dao.DeleteMonitorSendGroup(ctx, id); err != nil {
		p.l.Error("删除发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorAlertRuleList 获取告警规则列表，支持按名称搜索
func (p *prometheusService) GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorAlertRuleByName,
		p.dao.GetMonitorAlertRuleList)
}

// PromqlExprCheck 检查 PromQL 表达式是否有效
func (p *prometheusService) PromqlExprCheck(ctx context.Context, expr string) (bool, error) {
	return pkg.PromqlExprCheck(expr)
}

// CreateMonitorAlertRule 创建告警规则
func (p *prometheusService) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则是否已存在
	exists, err := p.dao.CheckMonitorAlertRuleExists(ctx, monitorAlertRule)
	if err != nil {
		p.l.Error("创建告警规则失败：检查告警规则是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("告警规则已存在")
	}

	// 创建告警规则
	if err := p.dao.CreateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		p.l.Error("创建告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorAlertRule 更新告警规则
func (p *prometheusService) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 确保告警规则存在
	existingRule, err := p.dao.GetMonitorAlertRuleById(ctx, monitorAlertRule.ID)
	if err != nil {
		p.l.Error("更新告警规则失败：获取告警规则时出错", zap.Error(err))
		return err
	}
	if existingRule == nil {
		return errors.New("告警规则不存在")
	}

	// 检查告警规则名称是否重复
	exists, err := p.dao.CheckMonitorAlertRuleNameExists(ctx, monitorAlertRule)
	if err != nil {
		p.l.Error("更新告警规则失败：检查告警规则名称时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("告警规则名称已存在")
	}

	// 更新告警规则
	if err := p.dao.UpdateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		p.l.Error("更新告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (p *prometheusService) EnableSwitchMonitorAlertRule(ctx context.Context, id int) error {
	if err := p.dao.EnableSwitchMonitorAlertRule(ctx, id); err != nil {
		p.l.Error("切换告警规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则的启用状态
func (p *prometheusService) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error {
	// 批量切换告警规则状态
	if err := p.dao.BatchEnableSwitchMonitorAlertRule(ctx, ids); err != nil {
		p.l.Error("批量切换告警规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 删除告警规则
func (p *prometheusService) DeleteMonitorAlertRule(ctx context.Context, id int) error {
	// 删除告警规则
	if err := p.dao.DeleteMonitorAlertRule(ctx, id); err != nil {
		p.l.Error("删除告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteMonitorAlertRule 批量删除告警规则
func (p *prometheusService) BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.DeleteMonitorAlertRule(ctx, id); err != nil {
			// 记录错误但继续删除其他规则
			p.l.Error("批量删除告警规则失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除告警规则 ID %d 失败: %v", id, err)
		}
	}
	return nil
}

// GetMonitorAlertEventList 获取告警事件列表，支持按名称搜索
func (p *prometheusService) GetMonitorAlertEventList(ctx context.Context, searchName *string) ([]*model.MonitorAlertEvent, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorAlertEventByName,
		p.dao.GetMonitorAlertEventList)
}

// EventAlertSilence 设置单个告警事件为静默状态
func (p *prometheusService) EventAlertSilence(ctx context.Context, id int, event *model.AlertEventSilenceRequest, userId int) error {
	// 验证 ID 是否有效
	if id <= 0 {
		p.l.Error("设置静默失败：无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	// 获取 AlertEvent
	alertEvent, err := p.dao.GetAlertEventByID(ctx, id)
	if err != nil {
		p.l.Error("设置静默失败：无法获取 AlertEvent", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 获取用户信息
	user, err := p.userDao.GetUserByID(ctx, userId)
	if err != nil {
		p.l.Error("设置静默失败：无效的 userId", zap.Int("userId", userId))
		return fmt.Errorf("无效的 userId: %d", userId)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(event.Time)
	if err != nil {
		p.l.Error("设置静默失败：解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 构建匹配器
	matchers, err := pkg.BuildMatchers(alertEvent, p.l, event.UseName)
	if err != nil {
		p.l.Error("设置静默失败：构建匹配器错误", zap.Error(err))
		return err
	}

	// 创建 Silence 对象
	silence := types.Silence{
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(time.Duration(duration)),
		CreatedBy: user.RealName,
		Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
	}

	// 序列化 Silence 对象为 JSON
	silenceData, err := json.Marshal(silence)
	if err != nil {
		p.l.Error("设置静默失败：序列化 Silence 对象失败", zap.Error(err))
		return fmt.Errorf("序列化失败: %v", err)
	}

	// 获取 AlertManager 地址
	alertPool, err := p.dao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
	if err != nil {
		p.l.Error("设置静默失败：无法获取 AlertPool", zap.Error(err))
		return err
	}

	if len(alertPool.AlertManagerInstances) == 0 {
		p.l.Error("设置静默失败：AlertManager 实例为空", zap.Int("poolID", alertPool.ID))
		return fmt.Errorf("AlertManager 实例为空")
	}

	alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
	alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

	// 发送 Silence 请求到 AlertManager
	silenceID, err := pkg.SendSilenceRequest(ctx, p.l, alertUrl, silenceData)
	if err != nil {
		p.l.Error("设置静默失败：发送 Silence 请求失败", zap.Error(err))
		return fmt.Errorf("发送 Silence 请求失败: %v", err)
	}

	// 更新 AlertEvent 状态为已静默
	alertEvent.Status = "已屏蔽"
	alertEvent.SilenceID = silenceID
	if err := p.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
		p.l.Error("设置静默失败：更新 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("更新 AlertEvent 失败: %v", err)
	}

	p.l.Info("设置静默成功", zap.Int("id", id), zap.String("silenceID", silenceID))
	return nil
}

// EventAlertClaim 认领告警事件
func (p *prometheusService) EventAlertClaim(ctx context.Context, id int, userId int) error {
	// 获取告警事件
	event, err := p.dao.GetMonitorAlertEventById(ctx, id)
	if err != nil {
		p.l.Error("认领告警事件失败：获取告警事件时出错", zap.Error(err))
		return err
	}

	// 更新认领用户
	event.RenLingUserID = userId

	// 更新数据库
	if err := p.dao.EventAlertClaim(ctx, event); err != nil {
		p.l.Error("认领告警事件失败：更新告警事件时出错", zap.Error(err))
		return err
	}

	p.l.Info("认领告警事件成功", zap.Int("id", id), zap.Int("userId", userId))
	return nil
}

// BatchEventAlertSilence 批量设置告警事件为静默状态
func (p *prometheusService) BatchEventAlertSilence(ctx context.Context, request *model.BatchEventAlertSilenceRequest, userId int) error {
	// 输入验证
	if request == nil || len(request.IDs) == 0 {
		p.l.Error("批量设置静默失败：未提供事件ID")
		return fmt.Errorf("未提供事件ID")
	}

	// 获取用户信息
	user, err := p.userDao.GetUserByID(ctx, userId)
	if err != nil {
		p.l.Error("批量设置静默失败：无效的 userId", zap.Int("userId", userId), zap.Error(err))
		return fmt.Errorf("无效的 userId: %d", userId)
	}

	// 解析持续时间
	duration, err := promModel.ParseDuration(request.Time)
	if err != nil {
		p.l.Error("批量设置静默失败：解析持续时间错误", zap.Error(err))
		return fmt.Errorf("无效的持续时间: %v", err)
	}

	// 初始化等待组和错误收集
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	// 定义信号量以限制并发数量（例如，最多 10 个并发 goroutine）
	sem := make(chan struct{}, 10)

	for _, id := range request.IDs {
		if id <= 0 {
			p.l.Error("批量设置静默跳过：无效的 ID", zap.Int("id", id))
			mu.Lock()
			errs = append(errs, fmt.Errorf("无效的 ID: %d", id))
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量
		go func(eventID int) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 获取 AlertEvent
			alertEvent, err := p.dao.GetAlertEventByID(ctx, eventID)
			if err != nil {
				p.l.Error("批量设置静默失败：无法获取 AlertEvent", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 构建匹配器
			matchers, err := pkg.BuildMatchers(alertEvent, p.l, request.UseName)
			if err != nil {
				p.l.Error("批量设置静默失败：构建匹配器错误", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 创建 Silence 对象
			silence := types.Silence{
				Matchers:  matchers,
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(time.Duration(duration)),
				CreatedBy: user.RealName,
				Comment:   fmt.Sprintf("eventId: %v 操作人: %v 静默时间: %v", alertEvent.ID, user.RealName, duration),
			}

			// 序列化 Silence 对象为 JSON
			silenceData, err := json.Marshal(silence)
			if err != nil {
				p.l.Error("批量设置静默失败：序列化 Silence 对象失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 获取 AlertManager 地址
			alertPool, err := p.dao.GetAlertPoolByID(ctx, alertEvent.SendGroup.PoolID)
			if err != nil {
				p.l.Error("批量设置静默失败：无法获取 AlertPool", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			if len(alertPool.AlertManagerInstances) == 0 {
				p.l.Error("批量设置静默失败：AlertManager 实例为空", zap.Int("poolID", alertPool.ID), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: AlertManager 实例为空", eventID))
				mu.Unlock()
				return
			}

			alertAddr := fmt.Sprintf("http://%v:9093", alertPool.AlertManagerInstances[0])
			alertUrl := fmt.Sprintf("%s/api/v1/silences", alertAddr)

			// 发送 Silence 请求到 AlertManager
			silenceID, err := pkg.SendSilenceRequest(ctx, p.l, alertUrl, silenceData)
			if err != nil {
				p.l.Error("批量设置静默失败：发送 Silence 请求失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			// 更新 AlertEvent 状态为已静默
			alertEvent.Status = "已屏蔽"
			alertEvent.SilenceID = silenceID
			if err := p.dao.UpdateAlertEvent(ctx, alertEvent); err != nil {
				p.l.Error("批量设置静默失败：更新 AlertEvent 失败", zap.Error(err), zap.Int("id", eventID))
				mu.Lock()
				errs = append(errs, fmt.Errorf("事件 ID %d: %v", eventID, err))
				mu.Unlock()
				return
			}

			p.l.Info("批量设置静默成功", zap.Int("id", eventID), zap.String("silenceID", silenceID))
		}(id)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 关闭信号量通道
	close(sem)

	if len(errs) > 0 {
		// 聚合错误
		errMsg := "批量设置静默过程中遇到以下错误："
		for _, e := range errs {
			errMsg += "\n" + e.Error()
		}
		p.l.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	p.l.Info("批量设置静默成功处理所有事件")
	return nil
}

// GetMonitorRecordRuleList 获取记录规则列表，支持按名称搜索
func (p *prometheusService) GetMonitorRecordRuleList(ctx context.Context, searchName *string) ([]*model.MonitorRecordRule, error) {
	return pkg.HandleList(ctx, searchName,
		p.dao.SearchMonitorRecordRuleByName,
		p.dao.GetMonitorRecordRuleList)
}

// CreateMonitorRecordRule 创建记录规则
func (p *prometheusService) CreateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error {
	// 检查记录规则是否已存在
	exists, err := p.dao.CheckMonitorRecordRuleExists(ctx, monitorRecordRule)
	if err != nil {
		p.l.Error("创建记录规则失败：检查记录规则是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("记录规则已存在")
	}

	// 创建记录规则
	if err := p.dao.CreateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		p.l.Error("创建记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorRecordRule 更新记录规则
func (p *prometheusService) UpdateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error {
	// 确保记录规则存在
	existingRule, err := p.dao.GetMonitorRecordRuleById(ctx, monitorRecordRule.ID)
	if err != nil {
		p.l.Error("更新记录规则失败：获取记录规则时出错", zap.Error(err))
		return err
	}
	if existingRule == nil {
		return errors.New("记录规则不存在")
	}

	// 检查记录规则名称是否重复
	exists, err := p.dao.CheckMonitorRecordRuleNameExists(ctx, monitorRecordRule)
	if err != nil {
		p.l.Error("更新记录规则失败：检查记录规则名称时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("记录规则名称已存在")
	}

	// 更新记录规则
	if err := p.dao.UpdateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		p.l.Error("更新记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorRecordRule 删除记录规则
func (p *prometheusService) DeleteMonitorRecordRule(ctx context.Context, id int) error {
	// 删除记录规则
	if err := p.dao.DeleteMonitorRecordRule(ctx, id); err != nil {
		p.l.Error("删除记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteMonitorRecordRule 批量删除记录规则
func (p *prometheusService) BatchDeleteMonitorRecordRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.DeleteMonitorRecordRule(ctx, id); err != nil {
			// 记录错误但继续删除其他规则
			p.l.Error("批量删除记录规则失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除记录规则 ID %d 失败: %v", id, err)
		}
	}
	return nil
}

// EnableSwitchMonitorRecordRule 切换记录规则的启用状态
func (p *prometheusService) EnableSwitchMonitorRecordRule(ctx context.Context, id int) error {
	if err := p.dao.EnableSwitchMonitorRecordRule(ctx, id); err != nil {
		p.l.Error("切换记录规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorRecordRule 批量切换记录规则的启用状态
func (p *prometheusService) BatchEnableSwitchMonitorRecordRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.EnableSwitchMonitorRecordRule(ctx, id); err != nil {
			p.l.Error("批量切换记录规则状态失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("切换记录规则 ID %d 状态失败: %v", id, err)
		}
	}
	return nil
}
