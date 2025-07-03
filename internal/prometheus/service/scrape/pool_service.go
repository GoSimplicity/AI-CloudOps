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
	GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) (model.ListResp[*model.MonitorScrapePool], error)
	CreateMonitorScrapePool(ctx context.Context, req *model.CreateMonitorScrapePoolReq) error
	UpdateMonitorScrapePool(ctx context.Context, req *model.UpdateMonitorScrapePoolReq) error
	DeleteMonitorScrapePool(ctx context.Context, req *model.DeleteMonitorScrapePoolReq) error
	GetMonitorScrapePoolTotal(ctx context.Context) (int, error)
	GetMonitorScrapePoolAll(ctx context.Context) (model.ListResp[*model.MonitorScrapePool], error)
	GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error)
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
func (s *scrapePoolService) GetMonitorScrapePoolList(ctx context.Context, req *model.GetMonitorScrapePoolListReq) (model.ListResp[*model.MonitorScrapePool], error) {
	var pools []*model.MonitorScrapePool
	var count int64
	var err error

	pools, count, err = s.dao.GetMonitorScrapePoolList(ctx, req)
	if err != nil {
		s.l.Error("获取抓取池列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorScrapePool]{}, err
	}

	// 填充创建用户信息
	for _, pool := range pools {
		if pool.UserID > 0 {
			user, err := s.userDao.GetUserByID(ctx, pool.UserID)
			if err != nil {
				s.l.Error("获取创建用户名失败", zap.Int("userId", pool.UserID), zap.Error(err))
				continue
			}
			if user != nil {
				if user.RealName == "" {
					pool.CreateUserName = user.Username
				} else {
					pool.CreateUserName = user.RealName
				}
			}
		}
	}

	return model.ListResp[*model.MonitorScrapePool]{
		Items: pools,
		Total: count,
	}, nil
}

// GetMonitorScrapePoolById 根据ID获取抓取池
func (s *scrapePoolService) GetMonitorScrapePoolById(ctx context.Context, id int) (*model.MonitorScrapePool, error) {
	if id <= 0 {
		return nil, errors.New("无效的抓取池ID")
	}

	pool, err := s.dao.GetMonitorScrapePoolById(ctx, id)
	if err != nil {
		s.l.Error("获取抓取池详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return pool, nil
}

// CreateMonitorScrapePool 创建抓取池
func (s *scrapePoolService) CreateMonitorScrapePool(ctx context.Context, req *model.CreateMonitorScrapePoolReq) error {
	if req.Name == "" {
		return errors.New("抓取池名称不能为空")
	}

	// 检查抓取池是否已存在
	exists, err := s.dao.CheckMonitorScrapePoolExists(ctx, &model.MonitorScrapePool{
		Name: req.Name,
	})
	if err != nil {
		s.l.Error("创建抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("抓取池已存在")
	}

	pools, _, err := s.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败：获取抓取池时出错", zap.Error(err))
		return err
	}

	pool := &model.MonitorScrapePool{
		Name:                  req.Name,
		PrometheusInstances:   req.PrometheusInstances,
		AlertManagerInstances: req.AlertManagerInstances,
		UserID:                req.UserID,
		ScrapeInterval:        req.ScrapeInterval,
		ScrapeTimeout:         req.ScrapeTimeout,
		RemoteTimeoutSeconds:  req.RemoteTimeoutSeconds,
		SupportAlert:          req.SupportAlert,
		SupportRecord:         req.SupportRecord,
		ExternalLabels:        req.ExternalLabels,
		RemoteWriteUrl:        req.RemoteWriteUrl,
		RemoteReadUrl:         req.RemoteReadUrl,
		AlertManagerUrl:       req.AlertManagerUrl,
		RuleFilePath:          req.RuleFilePath,
		RecordFilePath:        req.RecordFilePath,
	}

	// 检查新的抓取池 IP 是否已存在
	if err := utils.CheckPoolIpExists(pools, pool); err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败", zap.Error(err))
		return err
	}

	// 创建抓取池
	if err := s.dao.CreateMonitorScrapePool(ctx, pool); err != nil {
		s.l.Error("创建抓取池失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorScrapePool 更新抓取池
func (s *scrapePoolService) UpdateMonitorScrapePool(ctx context.Context, req *model.UpdateMonitorScrapePoolReq) error {
	// 检查 ID 是否有效
	if req.ID <= 0 {
		return errors.New("无效的抓取池ID")
	}

	// 先获取原有的抓取池信息
	oldPool, err := s.dao.GetMonitorScrapePoolById(ctx, req.ID)
	if err != nil {
		s.l.Error("更新抓取池失败：获取原有抓取池信息出错", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	if oldPool == nil {
		return errors.New("抓取池不存在")
	}

	// 如果名称发生变化,需要检查新名称是否已存在
	if oldPool.Name != req.Name {
		exists, err := s.dao.CheckMonitorScrapePoolExists(ctx, &model.MonitorScrapePool{
			Name: req.Name,
		})
		if err != nil {
			s.l.Error("更新抓取池失败：检查抓取池是否存在时出错", zap.Error(err))
			return err
		}

		if exists {
			return errors.New("抓取池名称已存在")
		}
	}

	pools, _, err := s.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败：获取抓取池时出错", zap.Error(err))
		return err
	}

	pool := &model.MonitorScrapePool{
		Model:                 model.Model{ID: req.ID},
		Name:                  req.Name,
		PrometheusInstances:   req.PrometheusInstances,
		AlertManagerInstances: req.AlertManagerInstances,
		UserID:                req.UserID,
		ScrapeInterval:        req.ScrapeInterval,
		ScrapeTimeout:         req.ScrapeTimeout,
		RemoteTimeoutSeconds:  req.RemoteTimeoutSeconds,
		SupportAlert:          req.SupportAlert,
		SupportRecord:         req.SupportRecord,
		ExternalLabels:        req.ExternalLabels,
		RemoteWriteUrl:        req.RemoteWriteUrl,
		RemoteReadUrl:         req.RemoteReadUrl,
		AlertManagerUrl:       req.AlertManagerUrl,
		RuleFilePath:          req.RuleFilePath,
		RecordFilePath:        req.RecordFilePath,
	}

	// 检查新的抓取池 IP 是否已被其他池使用
	if err := utils.CheckPoolIpExists(pools, pool); err != nil {
		s.l.Error("检查抓取池 IP 是否存在失败", zap.Error(err))
		return err
	}

	// 更新抓取池
	if err := s.dao.UpdateMonitorScrapePool(ctx, req); err != nil {
		s.l.Error("更新抓取池失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorScrapePool 删除抓取池
func (s *scrapePoolService) DeleteMonitorScrapePool(ctx context.Context, req *model.DeleteMonitorScrapePoolReq) error {
	if req.ID <= 0 {
		return errors.New("无效的抓取池ID")
	}

	// 检查抓取池是否存在
	pool, err := s.dao.GetMonitorScrapePoolById(ctx, req.ID)
	if err != nil {
		s.l.Error("删除抓取池失败：获取抓取池信息出错", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	if pool == nil {
		return errors.New("抓取池不存在")
	}

	// 检查抓取池是否有相关的抓取作业
	jobs, err := s.jobDao.GetMonitorScrapeJobsByPoolId(ctx, req.ID)
	if err != nil {
		s.l.Error("删除抓取池失败：获取抓取作业时出错", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	if len(jobs) > 0 {
		return errors.New("抓取池存在相关抓取作业，无法删除")
	}

	// 删除抓取池
	if err := s.dao.DeleteMonitorScrapePool(ctx, req.ID); err != nil {
		s.l.Error("删除抓取池失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorScrapePoolTotal 获取监控采集池总数
func (s *scrapePoolService) GetMonitorScrapePoolTotal(ctx context.Context) (int, error) {
	total, err := s.dao.GetMonitorScrapePoolTotal(ctx)
	if err != nil {
		s.l.Error("获取监控采集池总数失败", zap.Error(err))
	}
	return total, err
}

// GetMonitorScrapePoolAll 获取所有监控采集池
func (s *scrapePoolService) GetMonitorScrapePoolAll(ctx context.Context) (model.ListResp[*model.MonitorScrapePool], error) {
	pools, count, err := s.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		s.l.Error("获取所有监控采集池失败", zap.Error(err))
		return model.ListResp[*model.MonitorScrapePool]{}, err
	}

	// 填充创建用户信息
	for _, pool := range pools {
		if pool.UserID > 0 {
			user, err := s.userDao.GetUserByID(ctx, pool.UserID)
			if err != nil {
				s.l.Error("获取创建用户名失败", zap.Int("userId", pool.UserID), zap.Error(err))
				continue
			}
			if user != nil {
				if user.RealName == "" {
					pool.CreateUserName = user.Username
				} else {
					pool.CreateUserName = user.RealName
				}
			}
		}
	}

	return model.ListResp[*model.MonitorScrapePool]{
		Items: pools,
		Total: count,
	}, nil
}
