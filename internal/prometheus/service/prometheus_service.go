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

// GetMonitorScrapePoolList 获取监控抓取池列表，并根据可选的搜索参数进行过滤
func (p *prometheusService) GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error) {
	poolList, err := p.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		p.l.Error("failed to get all monitor scrape pool", zap.Error(err))
		return nil, err
	}

	if search == nil {
		return poolList, nil
	}

	// 初始化过滤后的抓取池列表
	var filteredPools []*model.MonitorScrapePool

	// 遍历所有抓取池，并根据名称进行过滤
	for _, pool := range poolList {
		if pool.Name == *search {
			filteredPools = append(filteredPools, pool)
		}
	}

	return filteredPools, nil
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
	jobList, err := p.dao.GetAllMonitorScrapeJobs(ctx)
	if err != nil {
		p.l.Error("failed to get all monitor scrape pool", zap.Error(err))
		return nil, err
	}

	if search == nil || *search == "" {
		return jobList, nil
	}

	var filteredJobs []*model.MonitorScrapeJob

	for _, job := range jobList {
		if job.Name == *search {
			filteredJobs = append(filteredJobs, job)
		}
	}

	return filteredJobs, nil
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
	groupList, err := p.dao.GetAllMonitorOndutyGroup(ctx)
	if err != nil {
		p.l.Error("failed to get all monitor onduty group", zap.Error(err))
		return nil, err
	}

	if searchName == nil || *searchName == "" {
		return groupList, nil
	}

	var filteredGroups []*model.MonitorOnDutyGroup

	for _, group := range groupList {
		if group.Name == *searchName {
			filteredGroups = append(filteredGroups, group)
		}
	}

	return filteredGroups, nil
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
		p.l.Error("解析开始时间错误", zap.String("startTime", startTimeStr), zap.Error(err))
		return nil, err
	}

	// 解析结束时间
	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		p.l.Error("解析结束时间错误", zap.String("endTime", endTimeStr), zap.Error(err))
		return nil, err
	}

	// 验证时间范围
	if endTime.Before(startTime) {
		errMsg := "结束时间不能早于开始时间"
		p.l.Error(errMsg, zap.String("startTime", startTimeStr), zap.String("endTime", endTimeStr))
		return nil, errors.New(errMsg)
	}

	// 获取值班组信息
	onDutyGroup, err := p.dao.GetMonitorOnDutyGroupById(ctx, id)
	if err != nil {
		p.l.Error("获取值班组信息失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	if onDutyGroup == nil {
		errMsg := "值班组不存在"
		p.l.Error(errMsg, zap.Int("id", id))
		return nil, errors.New(errMsg)
	}

	// 获取指定时间范围内的值班计划变更
	changes, err := p.dao.GetMonitorOnDutyChangesByGroupAndTimeRange(ctx, id, startTime, endTime)
	if err != nil {
		p.l.Error("获取值班计划变更失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return changes, nil
}
