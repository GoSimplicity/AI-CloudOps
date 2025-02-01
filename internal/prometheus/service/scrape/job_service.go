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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	treeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type ScrapeJobService interface {
	GetMonitorScrapeJobList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorScrapeJob, error)
	CreateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error
	DeleteMonitorScrapeJob(ctx context.Context, id int) error
	GetMonitorScrapeJobTotal(ctx context.Context) (int, error)
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

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (s *scrapeJobService) GetMonitorScrapeJobList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorScrapeJob, error) {
	var (
		jobs []*model.MonitorScrapeJob
		err  error
	)

	// 搜索处理
	if listReq.Search != "" {
		jobs, err = s.dao.SearchMonitorScrapeJobsByName(ctx, listReq.Search)
		if err != nil {
			s.l.Error("搜索抓取作业列表失败", zap.String("search", listReq.Search), zap.Error(err))
			return nil, err
		}
	} else {
		// 分页处理
		offset := (listReq.Page - 1) * listReq.Size
		limit := listReq.Size

		jobs, err = s.dao.GetMonitorScrapeJobList(ctx, offset, limit)
		if err != nil {
			s.l.Error("获取抓取作业列表失败", zap.Error(err))
			return nil, err
		}
	}

	// 填充用户信息
	if err := s.buildUserInfo(ctx, jobs); err != nil {
		s.l.Error("填充用户信息失败", zap.Error(err))
		return nil, err
	}

	// 填充树节点信息
	if err := s.buildTreeNodeInfo(ctx, jobs); err != nil {
		s.l.Error("填充树节点信息失败", zap.Error(err))
		return jobs, err
	}

	return jobs, nil
}

// CreateMonitorScrapeJob 创建监控采集 Job
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

	return nil
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (s *scrapeJobService) UpdateMonitorScrapeJob(ctx context.Context, monitorScrapeJob *model.MonitorScrapeJob) error {
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

// buildTreeNodeInfo 构建树节点信息
func (s *scrapeJobService) buildTreeNodeInfo(ctx context.Context, jobs []*model.MonitorScrapeJob) error {
	if len(jobs) == 0 || s.treeDao == nil {
		return nil
	}

	// 收集唯一树节点ID
	nodeIDs := make([]int, 0)
	seen := make(map[int]bool)
	for _, job := range jobs {
		for _, idStr := range job.TreeNodeIDs {
			if idStr == "" {
				continue
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				s.l.Error("转换树节点ID失败", zap.String("id", idStr), zap.Error(err))
				continue
			}
			if !seen[id] {
				nodeIDs = append(nodeIDs, id)
				seen[id] = true
			}
		}
	}

	if len(nodeIDs) == 0 {
		return nil
	}

	// 批量获取节点信息
	nodes, err := s.treeDao.GetByIDs(ctx, nodeIDs)
	if err != nil {
		s.l.Error("批量获取树节点信息失败", zap.Error(err))
		return err
	}

	// 构建节点映射
	nodeMap := make(map[int]string, len(nodes))
	for _, node := range nodes {
		nodeMap[node.ID] = node.Title
	}

	// 填充节点名称
	for _, job := range jobs {
		names := make([]string, 0, len(job.TreeNodeIDs))
		for _, idStr := range job.TreeNodeIDs {
			if id, err := strconv.Atoi(idStr); err == nil {
				if name := nodeMap[id]; name != "" {
					names = append(names, name)
				}
			}
		}
		job.TreeNodeNames = names
	}

	return nil
}

// GetMonitorScrapeJobTotal 获取监控采集作业总数
func (s *scrapeJobService) GetMonitorScrapeJobTotal(ctx context.Context) (int, error) {
	return s.dao.GetMonitorScrapeJobTotal(ctx)
}
