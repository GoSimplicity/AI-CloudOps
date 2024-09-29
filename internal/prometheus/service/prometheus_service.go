package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
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

	GetMonitorPrometheusYaml(ctx context.Context) (string, error)
	GetMonitorPrometheusAlertYaml(ctx context.Context) (string, error)
	GetMonitorPrometheusRecordYaml(ctx context.Context) (string, error)
	GetMonitorAlertManagerYaml(ctx context.Context) (string, error)
}

type prometheusService struct {
	l   *zap.Logger
	dao dao.PrometheusDao
}

func NewPrometheusService(dao dao.PrometheusDao) PrometheusService {
	return &prometheusService{
		dao: dao,
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

	if search == nil {
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

func (p *prometheusService) GetMonitorPrometheusYaml(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (p *prometheusService) GetMonitorPrometheusAlertYaml(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (p *prometheusService) GetMonitorPrometheusRecordYaml(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (p *prometheusService) GetMonitorAlertManagerYaml(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}
