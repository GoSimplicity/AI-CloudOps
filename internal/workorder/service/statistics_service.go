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
	"fmt" // Added for error formatting

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao" // Added for userDAO
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap" // Added for logging
)

// StatisticsService 定义了统计相关的服务接口
type StatisticsService interface {
	GetOverview(ctx context.Context, req model.OverviewStatsReq) (*model.OverviewStatsResp, error)
	GetTrend(ctx context.Context, req model.TrendStatsReq) (*model.TrendStatsResp, error)
	GetCategoryStats(ctx context.Context, req model.CategoryStatsReq) (*model.CategoryStatsResp, error)
	GetPerformanceStats(ctx context.Context, req model.PerformanceStatsReq) (*model.PerformanceStatsResp, error)
	GetUserStats(ctx context.Context, req model.UserStatsReq) (*model.UserStatsResp, error)
}

// statisticsService 实现了 StatisticsService 接口
type statisticsService struct {
	statisticsDAO dao.StatisticsDAO
	userDAO       userdao.UserDAO // 用于获取用户信息，例如操作员名称
	logger        *zap.Logger
}

// NewStatisticsService 创建一个新的 StatisticsService 实例
func NewStatisticsService(statisticsDAO dao.StatisticsDAO, userDAO userdao.UserDAO, logger *zap.Logger) StatisticsService {
	return &statisticsService{
		statisticsDAO: statisticsDAO,
		userDAO:       userDAO,
		logger:        logger,
	}
}

// GetOverview 获取工单总览统计数据
func (s *statisticsService) GetOverview(ctx context.Context, req model.OverviewStatsReq) (*model.OverviewStatsResp, error) {
	s.logger.Info("开始获取工单总览统计数据", zap.Any("request", req))

	// 假设 DAO 返回一个包含所有需要字段的结构体，或者多个 DAO 调用组合数据
	// For now, assuming a single DAO call that returns a struct compatible with OverviewStatsResp or its components.
	// The actual DAO method `GetOverviewStats` might need to perform complex aggregations.
	overviewData, err := s.statisticsDAO.GetOverviewStats(ctx, req.StartDate, req.EndDate)
	if err != nil {
		s.logger.Error("获取工单总览统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取总览统计数据失败: %w", err)
	}

	// 直接转换或填充 OverviewStatsResp
	// This is a simplified mapping. In a real scenario, calculations like CompletionRate might happen here
	// if not done by the DAO.
	resp := &model.OverviewStatsResp{
		TotalCount:      overviewData.TotalCount,
		CompletedCount:  overviewData.CompletedCount,
		ProcessingCount: overviewData.ProcessingCount,
		PendingCount:    overviewData.PendingCount,
		OverdueCount:    overviewData.OverdueCount,
		CompletionRate:  overviewData.CompletionRate, // Assume DAO calculates this
		AvgProcessTime:  overviewData.AvgProcessTime,  // Assume DAO calculates this
		TodayCreated:    overviewData.TodayCreated,
		TodayCompleted:  overviewData.TodayCompleted,
	}

	s.logger.Info("工单总览统计数据获取成功")
	return resp, nil
}

// GetTrend 获取工单趋势统计数据
func (s *statisticsService) GetTrend(ctx context.Context, req model.TrendStatsReq) (*model.TrendStatsResp, error) {
	s.logger.Info("开始获取工单趋势统计数据", zap.Any("request", req))

	trendData, err := s.statisticsDAO.GetInstanceTrendStats(ctx, req.StartDate, req.EndDate, req.Dimension, req.CategoryID)
	if err != nil {
		s.logger.Error("获取工单趋势统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取趋势统计数据失败: %w", err)
	}

	// Directly map if DAO returns data in the required structure
	resp := &model.TrendStatsResp{
		Dates:            trendData.Dates,
		CreatedCounts:    trendData.CreatedCounts,
		CompletedCounts:  trendData.CompletedCounts,
		ProcessingCounts: trendData.ProcessingCounts,
	}

	s.logger.Info("工单趋势统计数据获取成功")
	return resp, nil
}

// GetCategoryStats 获取按分类统计的工单数据
func (s *statisticsService) GetCategoryStats(ctx context.Context, req model.CategoryStatsReq) (*model.CategoryStatsResp, error) {
	s.logger.Info("开始获取按分类统计的工单数据", zap.Any("request", req))

	// The DAO method `GetWorkloadByCategory` is expected to return a list of items
	// that directly map or can be easily converted to `model.CategoryStatsItem`.
	categoryItems, err := s.statisticsDAO.GetWorkloadByCategory(ctx, req.StartDate, req.EndDate, req.Top)
	if err != nil {
		s.logger.Error("获取按分类统计的工单数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取分类统计数据失败: %w", err)
	}

	resp := &model.CategoryStatsResp{
		Items: categoryItems, // Assuming direct mapping
	}

	s.logger.Info("按分类统计的工单数据获取成功")
	return resp, nil
}

// GetPerformanceStats 获取操作员绩效统计数据
func (s *statisticsService) GetPerformanceStats(ctx context.Context, req model.PerformanceStatsReq) (*model.PerformanceStatsResp, error) {
	s.logger.Info("开始获取操作员绩效统计数据", zap.Any("request", req))

	// DAO returns raw performance data, potentially without user names
	rawPerformanceItems, err := s.statisticsDAO.GetOperatorPerformance(ctx, req.StartDate, req.EndDate, req.UserID, req.Top)
	if err != nil {
		s.logger.Error("获取操作员绩效统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取绩效统计数据失败: %w", err)
	}

	performanceItems := make([]model.PerformanceStatsItem, 0, len(rawPerformanceItems))
	for _, rawItem := range rawPerformanceItems {
		item := model.PerformanceStatsItem{
			UserID:            rawItem.UserID,
			// UserName will be fetched below
			AssignedCount:     rawItem.AssignedCount,
			CompletedCount:    rawItem.CompletedCount,
			CompletionRate:    rawItem.CompletionRate,    // Assume DAO calculates this
			AvgResponseTime:   rawItem.AvgResponseTime,   // Assume DAO calculates this
			AvgProcessingTime: rawItem.AvgProcessingTime, // Assume DAO calculates this
			OverdueCount:      rawItem.OverdueCount,
		}
		// Fetch user name
		user, err := s.userDAO.GetUserByID(ctx, rawItem.UserID)
		if err != nil {
			s.logger.Warn("获取用户姓名失败，用于绩效统计", zap.Int("userID", rawItem.UserID), zap.Error(err))
			item.UserName = fmt.Sprintf("用户ID %d (姓名未找到)", rawItem.UserID)
		} else {
			item.UserName = user.Username // Assuming User model has Username
		}
		performanceItems = append(performanceItems, item)
	}

	resp := &model.PerformanceStatsResp{
		Items: performanceItems,
	}

	s.logger.Info("操作员绩效统计数据获取成功")
	return resp, nil
}

// GetUserStats 获取特定用户的统计数据
func (s *statisticsService) GetUserStats(ctx context.Context, req model.UserStatsReq) (*model.UserStatsResp, error) {
	s.logger.Info("开始获取用户统计数据", zap.Any("request", req))

	userStatsData, err := s.statisticsDAO.GetStatsByUser(ctx, req.StartDate, req.EndDate, req.UserID)
	if err != nil {
		s.logger.Error("获取用户统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取用户统计数据失败: %w", err)
	}

	// Directly map if DAO returns data in the required structure
	resp := &model.UserStatsResp{
		CreatedCount:      userStatsData.CreatedCount,
		AssignedCount:     userStatsData.AssignedCount,
		CompletedCount:    userStatsData.CompletedCount,
		PendingCount:      userStatsData.PendingCount,
		OverdueCount:      userStatsData.OverdueCount,
		AvgResponseTime:   userStatsData.AvgResponseTime,   // Assume DAO calculates
		AvgProcessingTime: userStatsData.AvgProcessingTime, // Assume DAO calculates
		SatisfactionScore: userStatsData.SatisfactionScore, // Assume DAO calculates or fetches
	}

	s.logger.Info("用户统计数据获取成功")
	return resp, nil
}
