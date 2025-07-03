/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package scrape

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type ScrapeJobService interface {
	GetMonitorScrapeJobList(ctx context.Context, req *model.GetMonitorScrapeJobListReq) (model.ListResp[*model.MonitorScrapeJob], error)
	CreateMonitorScrapeJob(ctx context.Context, req *model.CreateMonitorScrapeJobReq) error
	UpdateMonitorScrapeJob(ctx context.Context, req *model.UpdateMonitorScrapeJobReq) error
	DeleteMonitorScrapeJob(ctx context.Context, id int) error
	GetMonitorScrapeJobTotal(ctx context.Context) (int, error)
}

type scrapeJobService struct {
	dao     scrapeJobDao.ScrapeJobDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewPrometheusScrapeService(dao scrapeJobDao.ScrapeJobDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) ScrapeJobService {
	return &scrapeJobService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (s *scrapeJobService) GetMonitorScrapeJobList(ctx context.Context, req *model.GetMonitorScrapeJobListReq) (model.ListResp[*model.MonitorScrapeJob], error) {
	var (
		jobs  []*model.MonitorScrapeJob
		total int64
		err   error
	)

	jobs, total, err = s.dao.GetMonitorScrapeJobList(ctx, req)
	if err != nil {
		s.l.Error("获取抓取作业列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorScrapeJob]{}, err
	}

	// 填充用户信息
	if err := s.buildUserInfo(ctx, jobs); err != nil {
		s.l.Error("填充用户信息失败", zap.Error(err))
		return model.ListResp[*model.MonitorScrapeJob]{}, err
	}

	return model.ListResp[*model.MonitorScrapeJob]{
		Items: jobs,
		Total: total,
	}, nil
}

// CreateMonitorScrapeJob 创建监控采集 Job
func (s *scrapeJobService) CreateMonitorScrapeJob(ctx context.Context, req *model.CreateMonitorScrapeJobReq) error {
	monitorScrapeJob := &model.MonitorScrapeJob{
		Name:                     req.Name,
		PoolID:                   req.PoolID,
		UserID:                   req.UserID,
		Enable:                   req.Enable,
		ServiceDiscoveryType:     req.ServiceDiscoveryType,
		MetricsPath:              req.MetricsPath,
		Scheme:                   req.Scheme,
		ScrapeInterval:           req.ScrapeInterval,
		ScrapeTimeout:            req.ScrapeTimeout,
		RefreshInterval:          req.RefreshInterval,
		Port:                     req.Port,
		IpAddress:                req.IpAddress,
		KubeConfigFilePath:       req.KubeConfigFilePath,
		TlsCaFilePath:            req.TlsCaFilePath,
		TlsCaContent:             req.TlsCaContent,
		BearerToken:              req.BearerToken,
		BearerTokenFile:          req.BearerTokenFile,
		KubernetesSdRole:         req.KubernetesSdRole,
		RelabelConfigsYamlString: req.RelabelConfigsYamlString,
	}

	// 检查抓取作业是否已存在
	exists, err := s.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob.Name)
	if err != nil {
		s.l.Error("创建抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("抓取作业已存在")
	}

	// 检查采集池是否存在
	poolExists, err := s.dao.CheckMonitorInstanceExists(ctx, monitorScrapeJob.PoolID)
	if err != nil {
		s.l.Error("创建抓取作业失败：检查采集池是否存在时出错", zap.Error(err))
		return err
	}

	if !poolExists {
		return errors.New("采集池不存在")
	}

	// 创建抓取作业
	if err := s.dao.CreateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		s.l.Error("创建抓取作业失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (s *scrapeJobService) UpdateMonitorScrapeJob(ctx context.Context, req *model.UpdateMonitorScrapeJobReq) error {
	monitorScrapeJob := &model.MonitorScrapeJob{
		Model:                    model.Model{ID: req.ID},
		Name:                     req.Name,
		Enable:                   req.Enable,
		ServiceDiscoveryType:     req.ServiceDiscoveryType,
		MetricsPath:              req.MetricsPath,
		Scheme:                   req.Scheme,
		ScrapeInterval:           req.ScrapeInterval,
		ScrapeTimeout:            req.ScrapeTimeout,
		PoolID:                   req.PoolID,
		RelabelConfigsYamlString: req.RelabelConfigsYamlString,
		RefreshInterval:          req.RefreshInterval,
		Port:                     req.Port,
		IpAddress:                req.IpAddress,
		KubeConfigFilePath:       req.KubeConfigFilePath,
		TlsCaFilePath:            req.TlsCaFilePath,
		TlsCaContent:             req.TlsCaContent,
		BearerToken:              req.BearerToken,
		BearerTokenFile:          req.BearerTokenFile,
		KubernetesSdRole:         req.KubernetesSdRole,
	}

	// 检查 ID 是否有效
	if monitorScrapeJob.ID <= 0 {
		return errors.New("无效的抓取作业ID")
	}

	// 先获取原有的抓取作业信息
	oldJob, err := s.dao.GetMonitorScrapeJobById(ctx, monitorScrapeJob.ID)
	if err != nil {
		s.l.Error("更新抓取作业失败：获取原有抓取作业信息出错", zap.Error(err))
		return err
	}

	// 如果名称发生变化,需要检查新名称是否已存在
	if oldJob.Name != monitorScrapeJob.Name {
		exists, err := s.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob.Name)
		if err != nil {
			s.l.Error("更新抓取作业失败：检查抓取作业名称是否存在时出错", zap.Error(err))
			return err
		}
		if exists {
			return errors.New("抓取作业名称已存在")
		}
	}

	// 更新抓取作业
	if err := s.dao.UpdateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		s.l.Error("更新抓取作业失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorScrapeJob 删除监控采集 Job
func (s *scrapeJobService) DeleteMonitorScrapeJob(ctx context.Context, id int) error {
	// 检查抓取作业是否存在
	_, err := s.dao.GetMonitorScrapeJobById(ctx, id)
	if err != nil {
		s.l.Error("删除抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}

	// 删除抓取作业
	if err := s.dao.DeleteMonitorScrapeJob(ctx, id); err != nil {
		s.l.Error("删除抓取作业失败", zap.Error(err))
		return err
	}

	return nil
}

// buildUserInfo 构建用户信息
func (s *scrapeJobService) buildUserInfo(ctx context.Context, jobs []*model.MonitorScrapeJob) error {
	if len(jobs) == 0 {
		return nil
	}

	// 收集唯一用户ID
	userIDs := make([]int, 0, len(jobs))
	seen := make(map[int]bool)
	for _, job := range jobs {
		if !seen[job.UserID] {
			userIDs = append(userIDs, job.UserID)
			seen[job.UserID] = true
		}
	}

	// 批量获取用户信息
	users, err := s.userDao.GetUserByIDs(ctx, userIDs)
	if err != nil {
		s.l.Error("批量获取用户信息失败", zap.Error(err))
	}

	// 构建用户映射
	userMap := make(map[int]string, len(users))
	for _, user := range users {
		if user.RealName != "" {
			userMap[user.ID] = user.RealName
		} else {
			userMap[user.ID] = user.Username
		}
	}

	// 填充用户名
	for _, job := range jobs {
		job.CreateUserName = userMap[job.UserID]
		if job.CreateUserName == "" {
			job.CreateUserName = "未知用户"
		}
	}

	return nil
}

// GetMonitorScrapeJobTotal 获取监控采集作业总数
func (s *scrapeJobService) GetMonitorScrapeJobTotal(ctx context.Context) (int, error) {
	return s.dao.GetMonitorScrapeJobTotal(ctx)
}
