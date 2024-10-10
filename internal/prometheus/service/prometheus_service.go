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

	return p.dao.CreateMonitorScrapePool(ctx, monitorScrapePool)
}

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

	// 检查抓取池是否已经存在
	pools, err := p.dao.GetAllMonitorScrapePool(ctx)
	if pkg.CheckPoolIpExists(pool, pools) {
		return errors.New("scrape pool ip exists")
	}

	return p.dao.UpdateMonitorScrapePool(ctx, monitorScrapePool)
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

	return p.dao.DeleteMonitorScrapePool(ctx, id)
}

func (p *prometheusService) GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error) {
	if search != nil && *search != "" {
		// 在dao层进行名称搜索
		return p.dao.SearchMonitorScrapeJobsByName(ctx, *search)
	}

	return p.dao.GetAllMonitorScrapeJobs(ctx)
}

func (p *prometheusService) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	return p.dao.CreateMonitorScrapeJob(ctx, monitorScrapeJob)
}

func (p *prometheusService) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	return p.dao.UpdateMonitorScrapeJob(ctx, monitorScrapeJob)
}

func (p *prometheusService) DeleteMonitorScrapeJob(ctx context.Context, id int) error {
	return p.dao.DeleteMonitorScrapeJob(ctx, id)
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

func (p *prometheusService) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	sendGroup, err := p.dao.GetMonitorSendGroupByOnDutyGroupId(ctx, id)
	if !errors.Is(err, gorm.ErrRecordNotFound) || sendGroup != nil {
		p.l.Error("failed to get monitor send group by onduty group id", zap.Error(err))
		return err
	}

	return p.dao.DeleteMonitorOnDutyGroup(ctx, id)

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

	return p.dao.CreateMonitorAlertManagerPool(ctx, monitorAlertManagerPool)

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

	return p.dao.UpdateMonitorAlertManagerPool(ctx, monitorAlertManagerPool)
}

func (p *prometheusService) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	sendGroup, _ := p.dao.GetMonitorSendGroupByPoolId(ctx, id)

	if sendGroup != nil || len(sendGroup) > 0 {
		return errors.New("该实例下存在发送组，无法删除")
	}

	return p.dao.DeleteMonitorAlertManagerPool(ctx, id)
}
