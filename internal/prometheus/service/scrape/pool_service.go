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

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type ScrapePoolService interface {
	GetMonitorScrapePoolList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorScrapePool, error)
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

// GetMonitorScrapePoolList 获取抓取池列表
func (s *scrapePoolService) GetMonitorScrapePoolList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorScrapePool, error) {
	if listReq.Search != "" {
		pools, err := s.dao.SearchMonitorScrapePoolsByName(ctx, listReq.Search)
		if err != nil {
			s.l.Error("搜索抓取池列表失败", zap.String("search", listReq.Search), zap.Error(err))
			return nil, err
		}
		return pools, nil
	}

	// 分页处理
	offset := (listReq.Page - 1) * listReq.Size
	limit := listReq.Size

	pools, err := s.dao.GetMonitorScrapePoolList(ctx, offset, limit)
	if err != nil {
		s.l.Error("获取抓取池列表失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

// CreateMonitorScrapePool 创建抓取池
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

	pools, err := s.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败：获取抓取池时出错", zap.Error(err))
		return err
	}

	// 检查新的抓取池 IP 是否已存在
	if err := utils.CheckPoolIpExists(pools, monitorScrapePool); err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败", zap.Error(err))
		return err
	}

	// 创建抓取池
	if err := s.dao.CreateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		s.l.Error("创建抓取池失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorScrapePool 更新抓取池
func (s *scrapePoolService) UpdateMonitorScrapePool(ctx context.Context, monitorScrapePool *model.MonitorScrapePool) error {
	// 检查 ID 是否有效
	if monitorScrapePool.ID <= 0 {
		return errors.New("无效的抓取池ID")
	}

	// 先获取原有的抓取池信息
	oldPool, err := s.dao.GetMonitorScrapePoolById(ctx, monitorScrapePool.ID)
	if err != nil {
		s.l.Error("更新抓取池失败：获取原有抓取池信息出错", zap.Error(err))
		return err
	}

	// 如果名称发生变化,需要检查新名称是否已存在
	if oldPool.Name != monitorScrapePool.Name {
		exists, err := s.dao.CheckMonitorScrapePoolExists(ctx, monitorScrapePool)
		if err != nil {
			s.l.Error("更新抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
			return err
		}

		if exists {
			return errors.New("抓取池名称已存在")
		}
	}

	pools, err := s.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败：获取抓取池时出错", zap.Error(err))
		return err
	}

	// 检查新的抓取池 IP 是否已被其他池使用
	if err := utils.CheckPoolIpExists(pools, monitorScrapePool); err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败", zap.Error(err))
		return err
	}

	// 更新抓取池
	if err := s.dao.UpdateMonitorScrapePool(ctx, monitorScrapePool); err != nil {
		s.l.Error("更新抓取池失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorScrapePool 删除抓取池
func (s *scrapePoolService) DeleteMonitorScrapePool(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("无效的抓取池ID")
	}

	// 检查抓取池是否存在
	_, err := s.dao.GetMonitorScrapePoolById(ctx, id)
	if err != nil {
		s.l.Error("删除抓取池失败：获取抓取池信息出错", zap.Error(err))
		return err
	}

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

	return nil
}
