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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	treeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
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
	treeDao treeDao.TreeNodeDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewPrometheusScrapeService(dao scrapeJobDao.ScrapeJobDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO, treeDao treeDao.TreeNodeDAO) ScrapeJobService {
	return &scrapeJobService{
		dao:     dao,
		userDao: userDao,
		treeDao: treeDao,
		l:       l,
		cache:   cache,
	}
}
func (s *scrapeJobService) GetMonitorScrapeJobList(ctx context.Context, search *string) ([]*model.MonitorScrapeJob, error) {
	// 获取作业列表
	jobs, err := pkg.HandleList(ctx, search,
		s.dao.SearchMonitorScrapeJobsByName,
		s.dao.GetAllMonitorScrapeJobs)
	if err != nil {
		s.l.Error("获取抓取作业列表失败", zap.Error(err))
		return nil, err
	}

	// 提前分配好容量,避免频繁扩容
	treeNodeIDMap := make(map[int]struct{}, len(jobs))

	// 收集所有需要查询的树节点ID和用户ID
	userIDMap := make(map[int]struct{}, len(jobs))
	for _, job := range jobs {
		userIDMap[job.UserID] = struct{}{}
		for _, idStr := range job.TreeNodeIDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				s.l.Error("转换树节点ID失败",
					zap.String("id", idStr),
					zap.Error(err))
				continue
			}
			treeNodeIDMap[id] = struct{}{}
		}
	}

	// 批量获取用户信息
	userIDs := make([]int, 0, len(userIDMap))
	for id := range userIDMap {
		userIDs = append(userIDs, id)
	}
	users, err := s.userDao.GetUserByIDs(ctx, userIDs)
	if err != nil {
		s.l.Error("批量获取用户信息失败", zap.Error(err))
		return nil, err
	}

	// 构建用户ID到用户信息的映射
	userMap := make(map[int]*model.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	// 填充用户信息
	for _, job := range jobs {
		if user, ok := userMap[job.UserID]; ok {
			if user.RealName == "" {
				job.CreateUserName = user.Username
			} else {
				job.CreateUserName = user.RealName
			}
		} else {
			// 如果找不到对应的用户信息,设置一个默认值
			job.CreateUserName = "未知用户"
		}
	}

	// 将map转为slice
	treeNodeIDs := make([]int, 0, len(treeNodeIDMap))
	for id := range treeNodeIDMap {
		treeNodeIDs = append(treeNodeIDs, id)
	}

	// 如果没有需要查询的树节点ID,直接返回
	if len(treeNodeIDs) == 0 {
		return jobs, nil
	}

	// 检查treeDao是否为nil
	if s.treeDao == nil {
		s.l.Error("treeDao未初始化")
		return jobs, errors.New("treeDao未初始化")
	}

	// 批量获取树节点信息
	treeNodes, err := s.treeDao.GetByIDs(ctx, treeNodeIDs)
	if err != nil {
		s.l.Error("批量获取树节点信息失败", zap.Error(err))
		return jobs, err // 返回jobs而不是nil,避免完全失败
	}

	// 构建ID到名称的映射
	treeNodeNameMap := make(map[int]string, len(treeNodes))
	for _, node := range treeNodes {
		treeNodeNameMap[node.ID] = node.Title
	}

	// 填充作业的树节点名称
	for _, job := range jobs {
		job.TreeNodeNames = make([]string, 0, len(job.TreeNodeIDs))
		for _, idStr := range job.TreeNodeIDs {
			id, _ := strconv.Atoi(idStr)
			if name, ok := treeNodeNameMap[id]; ok {
				job.TreeNodeNames = append(job.TreeNodeNames, name)
			}
		}
	}

	return jobs, nil
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
