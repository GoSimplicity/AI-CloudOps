package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	if search != nil && *search != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorScrapePoolsByName(ctx, *search)
	}

	return p.dao.GetAllMonitorScrapePool(ctx)
}

func (p *prometheusService) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	pools, err := p.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		p.l.Error("failed to get all monitor scrape pool", zap.Error(err))
		return err
	}

	// 检查抓取池是否已经存在
	if pkg.CheckPoolIpExists(monitorScrapePool, pools) {
		return errors.New("scrape pool ip exists")
	}

	if err := p.dao.CreateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		p.l.Error("failed to create monitor scrape pool", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

// UpdateMonitorScrapePool 更新监控采集池
func (p *prometheusService) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 确保要更新的抓取池存在
	pool, err := p.dao.GetMonitorScrapePoolById(ctx, monitorScrapePool.ID)
	if err != nil {
		p.l.Error("failed to get monitor scrape pool by id", zap.Error(err))
		return err
	}

	if pool == nil {
		return errors.New("scrape pool not found")
	}

	// 检查新的抓取池 IP 是否已经存在
	pools, err := p.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		p.l.Error("failed to get all monitor scrape pools", zap.Error(err))
		return err
	}

	if pkg.CheckPoolIpExists(monitorScrapePool, pools) {
		return errors.New("scrape pool IP already exists")
	}

	if err := p.dao.UpdateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		p.l.Error("failed to update monitor scrape pool", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) DeleteMonitorScrapePool(ctx context.Context, id int) error {
	jobs, err := p.dao.GetMonitorScrapeJobsByPoolId(ctx, id)
	if err != nil {
		p.l.Error("failed to get monitor scrape jobs by pool id", zap.Error(err))
		return err
	}

	if len(jobs) > 0 {
		return errors.New("scrape pool has scrape jobs")
	}

	if err := p.dao.DeleteMonitorScrapePool(ctx, id); err != nil {
		p.l.Error("failed to delete monitor scrape pool", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error) {
	if search != nil && *search != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorScrapeJobsByName(ctx, *search)
	}

	return p.dao.GetAllMonitorScrapeJobs(ctx)
}

func (p *prometheusService) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if err := p.dao.CreateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		p.l.Error("failed to create monitor scrape job", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	if err := p.dao.UpdateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		p.l.Error("failed to update monitor scrape job", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) DeleteMonitorScrapeJob(ctx context.Context, id int) error {
	if err := p.dao.DeleteMonitorScrapeJob(ctx, id); err != nil {
		p.l.Error("failed to delete monitor scrape job", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) GetMonitorPrometheusYaml(_ context.Context, ip string) string {
	return p.cache.GetPrometheusMainConfigByIP(ip)
}

func (p *prometheusService) GetMonitorPrometheusAlertRuleYaml(_ context.Context, ip string) string {
	return p.cache.GetPrometheusAlertRuleConfigYamlByIp(ip)
}

func (p *prometheusService) GetMonitorPrometheusRecordYaml(_ context.Context, ip string) string {
	return p.cache.GetPrometheusRecordRuleConfigYamlByIp(ip)
}

func (p *prometheusService) GetMonitorAlertManagerYaml(_ context.Context, ip string) string {
	return p.cache.GetAlertManagerMainConfigYamlByIP(ip)
}

func (p *prometheusService) GetMonitorOnDutyGroupList(ctx context.Context, searchName *string) ([]*model.MonitorOnDutyGroup, error) {
	if searchName != nil && *searchName != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorOnDutyGroupByName(ctx, *searchName)
	}

	return p.dao.GetAllMonitorOndutyGroup(ctx)
}

func (p *prometheusService) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	return p.dao.CreateMonitorOnDutyGroup(ctx, monitorOnDutyGroup)
}

func (p *prometheusService) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error {
	if monitorOnDutyChange.OnDutyGroupID == 0 {
		return errors.New("on duty group id is empty")
	}

	_, err := p.dao.GetMonitorOnDutyGroupById(ctx, monitorOnDutyChange.OnDutyGroupID)
	if err != nil {
		p.l.Error("failed to get monitor onduty group by id", zap.Error(err))
		return err
	}

	originUser, err := p.userDao.GetUserByID(ctx, monitorOnDutyChange.OriginUserID)
	if err != nil {
		p.l.Error("failed to get user by id", zap.Error(err))
		return err
	}

	targetUser, err := p.userDao.GetUserByID(ctx, monitorOnDutyChange.OnDutyUserID)
	if err != nil {
		p.l.Error("failed to get user by id", zap.Error(err))
		return err
	}

	monitorOnDutyChange.OriginUserID = originUser.ID
	monitorOnDutyChange.OnDutyUserID = targetUser.ID

	return p.dao.CreateMonitorOnDutyGroupChange(ctx, monitorOnDutyChange)
}

func (p *prometheusService) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	_, err := p.dao.GetMonitorOnDutyGroupById(ctx, monitorOnDutyGroup.ID)
	if err != nil {
		p.l.Error("failed to get monitor onduty group by id", zap.Error(err))
		return err
	}

	users, err := p.userDao.GetUserByUsernames(ctx, monitorOnDutyGroup.UserNames)
	if err != nil {
		p.l.Error("failed to get user by username", zap.Error(err))
		return err
	}

	monitorOnDutyGroup.Members = users

	return p.dao.UpdateMonitorOnDutyGroup(ctx, monitorOnDutyGroup)
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (p *prometheusService) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	sendGroups, err := p.dao.GetMonitorSendGroupByOnDutyGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("failed to get monitor send groups by onduty group id", zap.Error(err))
		return err
	}

	// 仅当 sendGroups 不为空时，拒绝删除
	if len(sendGroups) > 0 {
		return errors.New("cannot delete on-duty group with existing send groups")
	}

	if err := p.dao.DeleteMonitorOnDutyGroup(ctx, id); err != nil {
		p.l.Error("failed to delete monitor on-duty group", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) GetMonitorOnDutyGroup(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	return p.dao.GetMonitorOnDutyGroupById(ctx, id)
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组在指定时间范围内的值班计划变更
func (p *prometheusService) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, id int, startTimeStr string, endTimeStr string) ([]*model.MonitorOnDutyChange, error) {
	// 解析开始时间
	startTime, err := time.Parse("2006-01-02", startTimeStr)
	if err != nil {
		p.l.Error("开始时间格式错误", zap.String("startTime", startTimeStr), zap.Error(err))
		return nil, errors.New("开始时间格式错误")
	}

	// 解析结束时间
	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		p.l.Error("结束时间格式错误", zap.String("endTime", endTimeStr), zap.Error(err))
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
		p.l.Error("根据 ID 获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 获取指定时间范围内的值班计划变更
	changes, err := p.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, id, startTime, endTime)
	if err != nil {
		p.l.Error("获取值班计划变更失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return changes, nil
}

func (p *prometheusService) GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error) {
	if searchName != nil && *searchName != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorAlertManagerPoolByName(ctx, *searchName)
	}

	return p.dao.GetAllAlertManagerPools(ctx)
}

func (p *prometheusService) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	alertList, err := p.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		p.l.Error("failed to get all alert manager pools", zap.Error(err))
		return err
	}

	if pkg.CheckAlertIpExists(monitorAlertManagerPool, alertList) {
		return errors.New("ip already exists")
	}

	if err := p.dao.CreateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		p.l.Error("failed to create monitor alert manager pool", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)

}

func (p *prometheusService) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	alertList, err := p.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		p.l.Error("failed to get all alert manager pools", zap.Error(err))
		return err
	}

	if pkg.CheckAlertIpExists(monitorAlertManagerPool, alertList) {
		return errors.New("ip already exists")
	}

	if err := p.dao.UpdateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		p.l.Error("failed to update monitor alert manager pool", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (p *prometheusService) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	sendGroups, err := p.dao.GetMonitorSendGroupByPoolId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("failed to get monitor send groups by pool id", zap.Error(err))
		return err
	}

	// 仅当 sendGroups 不为空时，拒绝删除
	if len(sendGroups) > 0 {
		return errors.New("cannot delete AlertManager pool with existing send groups")
	}

	if err := p.dao.DeleteMonitorAlertManagerPool(ctx, id); err != nil {
		p.l.Error("failed to delete monitor alert manager pool", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error) {
	if searchName != nil && *searchName != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorSendGroupByName(ctx, *searchName)
	}

	return p.dao.GetMonitorSendGroupList(ctx)
}

func (p *prometheusService) PromqlExprCheck(ctx context.Context, expr string) (bool, error) {
	return pkg.PromqlExprCheck(expr)
}

func (p *prometheusService) CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	if err := p.dao.CreateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		p.l.Error("failed to create monitor send group", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

// UpdateMonitorSendGroup 更新现有的发送组
func (p *prometheusService) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 确保发送组存在
	_, err := p.dao.GetMonitorSendGroupById(ctx, monitorSendGroup.ID)
	if err != nil {
		p.l.Error("failed to get monitor send group by id", zap.Error(err))
		return err
	}

	// 首先更新 DAO
	if err := p.dao.UpdateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		p.l.Error("failed to update monitor send group", zap.Error(err))
		return err
	}

	// 然后更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("failed to update cache", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorSendGroup 删除指定的发送组
func (p *prometheusService) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	sendGroups, err := p.dao.GetMonitorSendGroupByPoolId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		p.l.Error("failed to get monitor send groups by pool id", zap.Error(err))
		return err
	}

	// 仅当 sendGroups 不为空时，拒绝删除
	if len(sendGroups) > 0 {
		return errors.New("cannot delete send group with existing associations")
	}

	// 首先删除 DAO
	if err := p.dao.DeleteMonitorSendGroup(ctx, id); err != nil {
		p.l.Error("failed to delete monitor send group", zap.Error(err))
		return err
	}

	// 然后更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("failed to update cache", zap.Error(err))
		return err
	}

	return nil
}

func (p *prometheusService) GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error) {
	if searchName != nil && *searchName != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorAlertRuleByName(ctx, *searchName)
	}

	return p.dao.GetMonitorAlertRuleList(ctx)
}

func (p *prometheusService) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	if err := p.dao.CreateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		p.l.Error("failed to create monitor alert rule", zap.Error(err))
		return err
	}

	return p.cache.MonitorCacheManager(ctx)
}

// UpdateMonitorAlertRule 更新现有的告警规则
func (p *prometheusService) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 确保告警规则存在
	if _, err := p.dao.GetMonitorAlertRuleById(ctx, monitorAlertRule.ID); err != nil {
		p.l.Error("failed to get monitor alert rule by id", zap.Error(err))
		return err
	}

	// 首先更新 DAO
	if err := p.dao.UpdateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		p.l.Error("failed to update monitor alert rule", zap.Error(err))
		return err
	}

	// 然后更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("failed to update cache", zap.Error(err))
		return err
	}

	return nil
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (p *prometheusService) EnableSwitchMonitorAlertRule(ctx context.Context, id int) error {
	if err := p.dao.EnableSwitchMonitorAlertRule(ctx, id); err != nil {
		p.l.Error("failed to toggle monitor alert rule enable status", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("failed to update cache after toggling alert rule", zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则的启用状态
func (p *prometheusService) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error {
	if err := p.dao.BatchEnableSwitchMonitorAlertRule(ctx, ids); err != nil {
		p.l.Error("failed to batch toggle monitor alert rules enable status", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := p.cache.MonitorCacheManager(ctx); err != nil {
		p.l.Error("failed to update cache after batch toggling alert rules", zap.Error(err))
		return err
	}

	return nil
}

func (p *prometheusService) DeleteMonitorAlertRule(ctx context.Context, id int) error {
	if err := p.dao.DeleteMonitorAlertRule(ctx, id); err != nil {
		p.l.Error("failed to delete monitor alert rule", zap.Error(err))
		return err
	}
	return p.cache.MonitorCacheManager(ctx)
}

func (p *prometheusService) BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := p.dao.DeleteMonitorAlertRule(ctx, id); err != nil {
			p.l.Error("failed to delete monitor alert rule", zap.Error(err))
			return err
		}
	}

	return p.cache.MonitorCacheManager(ctx)
}
