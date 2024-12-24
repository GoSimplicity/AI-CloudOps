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
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type ScrapePoolService interface {
	GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error)
	CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error
	DeleteMonitorScrapePool(ctx context.Context, id int) error
}

type scrapePoolService struct {
	dao     scrapeJobDao.ScrapePoolDAO
	jobDao  scrapeJobDao.ScrapeJobDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewPrometheusPoolService(dao scrapeJobDao.ScrapePoolDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO, jobDao scrapeJobDao.ScrapeJobDAO) ScrapePoolService {
	return &scrapePoolService{
		dao:     dao,
		jobDao:  jobDao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (s *scrapePoolService) GetMonitorScrapePoolList(ctx context.Context, search *string) ([]*model.MonitorScrapePool, error) {
	return pkg.HandleList(ctx, search,
		s.dao.SearchMonitorScrapePoolsByName, // 搜索函数
		s.dao.GetAllMonitorScrapePool)        // 获取所有函数
}

func (s *scrapePoolService) CreateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 检查抓取池是否已存在
	exists, err := s.dao.CheckMonitorScrapePoolExists(ctx, monitorScrapePool)
	if err != nil {
		s.l.Error("创建抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取池已存在")
	}

	// 创建抓取池
	if err := s.dao.CreateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		s.l.Error("创建抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := s.cache.MonitorCacheManager(ctx); err != nil {
		s.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	s.l.Info("创建抓取池成功", zap.Int("id", monitorScrapePool.ID))
	return nil
}

func (s *scrapePoolService) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 确保要更新的抓取池存在
	pools, err := s.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		s.l.Error("更新抓取池失败：获取抓取池时出错", zap.Error(err))
		return err
	}

	newPools := make([]*model.MonitorScrapePool, 0)

	for _, pool := range pools {
		if pool.ID == monitorScrapePool.ID {
			continue
		}

		if pool.Name == monitorScrapePool.Name {
			return errors.New("抓取池名称已存在")
		}

		newPools = append(newPools, pool)
	}

	// 检查新的抓取池 IP 是否已存在
	exists := pkg.CheckPoolIpExists(monitorScrapePool, newPools)
	if exists {
		return errors.New("抓取池 IP 已存在")
	}

	// 更新抓取池
	if err := s.dao.UpdateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		s.l.Error("更新抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := s.cache.MonitorCacheManager(ctx); err != nil {
		s.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	s.l.Info("更新抓取池成功", zap.Int("id", monitorScrapePool.ID))
	return nil
}

func (s *scrapePoolService) DeleteMonitorScrapePool(ctx context.Context, id int) error {
	// 检查抓取池是否有相关的抓取作业
	jobs, err := s.jobDao.GetMonitorScrapeJobsByPoolId(ctx, id)
	if err != nil {
		s.l.Error("删除抓取池失败：获取抓取作业时出错", zap.Error(err))
		return err
	}

	if len(jobs) > 0 {
		return errors.New("抓取池存在相关抓取作业，无法删除")
	}

	// 删除抓取池
	if err := s.dao.DeleteMonitorScrapePool(ctx, id); err != nil {
		s.l.Error("删除抓取池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := s.cache.MonitorCacheManager(ctx); err != nil {
		s.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	s.l.Info("删除抓取池成功", zap.Int("id", id))
	return nil
}
