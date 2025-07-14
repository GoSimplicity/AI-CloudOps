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

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type StatisticsService interface {
	GetOverview(ctx context.Context, req *model.StatsReq) (*model.OverviewStats, error)
	GetTrend(ctx context.Context, req *model.StatsReq) (*model.TrendStats, error)
	GetCategoryStats(ctx context.Context, req *model.StatsReq) ([]model.CategoryStats, error)
	GetUserStats(ctx context.Context, req *model.StatsReq) ([]model.UserStats, error)
	GetTemplateStats(ctx context.Context, req *model.StatsReq) ([]model.TemplateStats, error)
	GetStatusDistribution(ctx context.Context, req *model.StatsReq) ([]model.StatusDistribution, error)
	GetPriorityDistribution(ctx context.Context, req *model.StatsReq) ([]model.PriorityDistribution, error)
}

type statisticsService struct {
	statisticsDAO dao.StatisticsDAO
	userDAO       userdao.UserDAO
	logger        *zap.Logger
}

func NewStatisticsService(statisticsDAO dao.StatisticsDAO, userDAO userdao.UserDAO, logger *zap.Logger) StatisticsService {
	return &statisticsService{
		statisticsDAO: statisticsDAO,
		userDAO:       userDAO,
		logger:        logger,
	}
}

// GetOverview 获取工单总览统计
func (s *statisticsService) GetOverview(ctx context.Context, req *model.StatsReq) (*model.OverviewStats, error) {
	s.logger.Info("获取工单总览统计",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate))

	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	overview, err := s.statisticsDAO.GetOverviewStats(ctx, req)
	if err != nil {
		s.logger.Error("获取总览统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取总览统计失败: %w", err)
	}

	s.logger.Info("总览统计获取成功",
		zap.Int64("totalCount", overview.TotalCount),
		zap.Float64("completionRate", overview.CompletionRate))

	return overview, nil
}

// GetTrend 获取工单趋势统计
func (s *statisticsService) GetTrend(ctx context.Context, req *model.StatsReq) (*model.TrendStats, error) {
	s.logger.Info("获取工单趋势统计",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate),
		zap.String("dimension", req.Dimension))

	if err := s.validateTrendRequest(req); err != nil {
		return nil, err
	}

	trend, err := s.statisticsDAO.GetTrendStats(ctx, req)
	if err != nil {
		s.logger.Error("获取趋势统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取趋势统计失败: %w", err)
	}

	s.logger.Info("趋势统计获取成功", zap.Int("dataPoints", len(trend.Dates)))
	return trend, nil
}

// GetCategoryStats 获取分类统计
func (s *statisticsService) GetCategoryStats(ctx context.Context, req *model.StatsReq) ([]model.CategoryStats, error) {
	s.logger.Info("获取分类统计",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate),
		zap.Int("top", req.Top))

	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	categories, err := s.statisticsDAO.GetCategoryStats(ctx, req)
	if err != nil {
		s.logger.Error("获取分类统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取分类统计失败: %w", err)
	}

	s.logger.Info("分类统计获取成功", zap.Int("categoryCount", len(categories)))
	return categories, nil
}

// GetUserStats 获取用户统计
func (s *statisticsService) GetUserStats(ctx context.Context, req *model.StatsReq) ([]model.UserStats, error) {
	s.logger.Info("获取用户统计",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate),
		zap.Any("userID", req.UserID))

	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	users, err := s.statisticsDAO.GetUserStats(ctx, req)
	if err != nil {
		s.logger.Error("获取用户统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取用户统计失败: %w", err)
	}

	// 批量补充用户信息，避免N+1查询问题
	if len(users) > 0 {
		// 收集所有用户ID
		userIDs := make([]int, 0, len(users))
		userIDSet := make(map[int]bool)
		for _, userStat := range users {
			if !userIDSet[userStat.UserID] {
				userIDs = append(userIDs, userStat.UserID)
				userIDSet[userStat.UserID] = true
			}
		}

		// 批量获取用户信息
		if len(userIDs) > 0 {
			userList, err := s.userDAO.GetUserByIDs(ctx, userIDs)
			if err != nil {
				s.logger.Warn("批量获取用户信息失败", zap.Error(err))
			} else {
				// 构建用户ID到用户名的映射
				userMap := make(map[int]string)
				for _, user := range userList {
					userMap[user.ID] = user.Username
				}

				// 填充用户名
				for i := range users {
					if userName, exists := userMap[users[i].UserID]; exists {
						users[i].UserName = userName
					}
				}
			}
		}
	}

	s.logger.Info("用户统计获取成功", zap.Int("userCount", len(users)))
	return users, nil
}

// GetTemplateStats 获取模板统计
func (s *statisticsService) GetTemplateStats(ctx context.Context, req *model.StatsReq) ([]model.TemplateStats, error) {
	s.logger.Info("获取模板统计",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate),
		zap.Any("categoryID", req.CategoryID))

	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	templates, err := s.statisticsDAO.GetTemplateStats(ctx, req)
	if err != nil {
		s.logger.Error("获取模板统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取模板统计失败: %w", err)
	}

	s.logger.Info("模板统计获取成功", zap.Int("templateCount", len(templates)))
	return templates, nil
}

// GetStatusDistribution 获取状态分布
func (s *statisticsService) GetStatusDistribution(ctx context.Context, req *model.StatsReq) ([]model.StatusDistribution, error) {
	s.logger.Info("获取状态分布",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate))

	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	distribution, err := s.statisticsDAO.GetStatusDistribution(ctx, req)
	if err != nil {
		s.logger.Error("获取状态分布失败", zap.Error(err))
		return nil, fmt.Errorf("获取状态分布失败: %w", err)
	}

	s.logger.Info("状态分布获取成功", zap.Int("statusCount", len(distribution)))
	return distribution, nil
}

// GetPriorityDistribution 获取优先级分布
func (s *statisticsService) GetPriorityDistribution(ctx context.Context, req *model.StatsReq) ([]model.PriorityDistribution, error) {
	s.logger.Info("获取优先级分布",
		zap.Any("startDate", req.StartDate),
		zap.Any("endDate", req.EndDate))

	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	distribution, err := s.statisticsDAO.GetPriorityDistribution(ctx, req)
	if err != nil {
		s.logger.Error("获取优先级分布失败", zap.Error(err))
		return nil, fmt.Errorf("获取优先级分布失败: %w", err)
	}

	s.logger.Info("优先级分布获取成功", zap.Int("priorityCount", len(distribution)))
	return distribution, nil
}

// ==================== 私有方法 ====================

// validateDateRange 验证日期范围
func (s *statisticsService) validateDateRange(startDate, endDate *time.Time) error {
	if startDate != nil && endDate != nil {
		if startDate.After(*endDate) {
			return fmt.Errorf("开始日期不能晚于结束日期")
		}

		// 限制查询范围不超过一年
		if endDate.Sub(*startDate) > 365*24*time.Hour {
			return fmt.Errorf("查询时间范围不能超过一年")
		}
	}

	return nil
}

// validateTrendRequest 验证趋势统计请求
func (s *statisticsService) validateTrendRequest(req *model.StatsReq) error {
	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return err
	}

	if req.Dimension == "" {
		return fmt.Errorf("趋势统计必须指定维度")
	}

	validDimensions := map[string]bool{
		"day": true, "week": true, "month": true,
	}
	if !validDimensions[req.Dimension] {
		return fmt.Errorf("无效的统计维度: %s", req.Dimension)
	}

	return nil
}
