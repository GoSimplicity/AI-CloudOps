package scrape

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

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
)

type ScrapeJobService interface {
	GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, id int) error
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

func (s *scrapeJobService) GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error) {
	return pkg.HandleList(ctx, search,
		s.dao.SearchMonitorScrapeJobsByName, // 搜索函数
		s.dao.GetAllMonitorScrapeJobs)       // 获取所有函数
}

func (s *scrapeJobService) CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	// 检查抓取作业是否已存在
	exists, err := s.dao.CheckMonitorScrapeJobExists(ctx, monitorScrapeJob.Name)
	if err != nil {
		s.l.Error("创建抓取作业失败：检查抓取作业是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("抓取作业已存在")
	}

	// 创建抓取作业
	if err := s.dao.CreateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		s.l.Error("创建抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := s.cache.MonitorCacheManager(ctx); err != nil {
		s.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	s.l.Info("创建抓取作业成功", zap.Int("id", monitorScrapeJob.ID))
	return nil
}

func (s *scrapeJobService) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
	// 更新抓取作业
	if err := s.dao.UpdateMonitorScrapeJob(ctx, monitorScrapeJob); err != nil {
		s.l.Error("更新抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := s.cache.MonitorCacheManager(ctx); err != nil {
		s.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	s.l.Info("更新抓取作业成功", zap.Int("id", monitorScrapeJob.ID))

	return nil
}

func (s *scrapeJobService) DeleteMonitorScrapeJob(ctx context.Context, id int) error {
	// 删除抓取作业
	if err := s.dao.DeleteMonitorScrapeJob(ctx, id); err != nil {
		s.l.Error("删除抓取作业失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := s.cache.MonitorCacheManager(ctx); err != nil {
		s.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	s.l.Info("删除抓取作业成功", zap.Int("id", id))
	return nil
}
