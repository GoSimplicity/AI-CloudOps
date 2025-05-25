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
	GetOverview(ctx context.Context, req *model.OverviewStatsReq) (*model.OverviewStatsResp, error)
	GetTrend(ctx context.Context, req *model.TrendStatsReq) (*model.TrendStatsResp, error)
	GetCategoryStats(ctx context.Context, req *model.CategoryStatsReq) ([]model.CategoryStatsResp, error)
	GetPerformanceStats(ctx context.Context, req *model.PerformanceStatsReq) (*model.PerformanceStatsResp, error)
	GetUserStats(ctx context.Context, req *model.UserStatsReq) (*model.UserStatsResp, error)
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

// GetOverview 获取工单总览统计数据
func (s *statisticsService) GetOverview(ctx context.Context, req *model.OverviewStatsReq) (*model.OverviewStatsResp, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	s.logger.Info("开始获取工单总览统计数据",
		zap.Timep("startDate", req.StartDate),
		zap.Timep("endDate", req.EndDate))

	// 验证日期范围
	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	overviewData, err := s.statisticsDAO.GetOverviewStats(ctx, req.StartDate, req.EndDate)
	if err != nil {
		s.logger.Error("获取工单总览统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取总览统计数据失败: %w", err)
	}

	s.logger.Info("工单总览统计数据获取成功",
		zap.Int64("totalCount", overviewData.TotalCount),
		zap.Int64("completedCount", overviewData.CompletedCount),
		zap.Float64("completionRate", overviewData.CompletionRate))

	return overviewData, nil
}

// GetTrend 获取工单趋势统计数据
func (s *statisticsService) GetTrend(ctx context.Context, req *model.TrendStatsReq) (*model.TrendStatsResp, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	s.logger.Info("开始获取工单趋势统计数据",
		zap.Time("startDate", req.StartDate),
		zap.Time("endDate", req.EndDate),
		zap.String("dimension", req.Dimension),
		zap.Intp("categoryID", req.CategoryID))

	// 验证统计维度
	if err := s.validateDimension(req.Dimension); err != nil {
		return nil, err
	}

	// 验证日期范围（注意这里是必填的time.Time，不是指针）
	if req.StartDate.IsZero() || req.EndDate.IsZero() {
		return nil, fmt.Errorf("开始日期和结束日期不能为空")
	}

	if req.StartDate.After(req.EndDate) {
		return nil, fmt.Errorf("开始日期不能晚于结束日期")
	}

	// 验证日期范围不能超过一年
	if req.EndDate.Sub(req.StartDate) > 365*24*time.Hour {
		return nil, fmt.Errorf("查询时间范围不能超过一年")
	}

	trendData, err := s.statisticsDAO.GetInstanceTrendStats(ctx, req.StartDate, req.EndDate, req.Dimension, req.CategoryID)
	if err != nil {
		s.logger.Error("获取工单趋势统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取趋势统计数据失败: %w", err)
	}

	s.logger.Info("工单趋势统计数据获取成功",
		zap.Int("dataPoints", len(trendData.Dates)))

	return trendData, nil
}

// GetCategoryStats 获取按分类统计的工单数据
func (s *statisticsService) GetCategoryStats(ctx context.Context, req *model.CategoryStatsReq) ([]model.CategoryStatsResp, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	s.logger.Info("开始获取按分类统计的工单数据",
		zap.Timep("startDate", req.StartDate),
		zap.Timep("endDate", req.EndDate),
		zap.Int("top", req.Top))

	// 验证日期范围
	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	// 验证top参数
	if req.Top > 0 && req.Top < 5 {
		return nil, fmt.Errorf("top参数不能小于5")
	}
	if req.Top > 20 {
		return nil, fmt.Errorf("top参数不能大于20")
	}

	// 将top参数转换为指针
	var topPtr *int
	if req.Top > 0 {
		topPtr = &req.Top
	}

	// 调用DAO方法，注意返回类型是[]model.CategoryStatsResp
	categoryItems, err := s.statisticsDAO.GetWorkloadByCategory(ctx, req.StartDate, req.EndDate, topPtr)
	if err != nil {
		s.logger.Error("获取按分类统计的工单数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取分类统计数据失败: %w", err)
	}

	s.logger.Info("按分类统计的工单数据获取成功",
		zap.Int("categoryCount", len(categoryItems)))

	return categoryItems, nil
}

// GetPerformanceStats 获取操作员绩效统计数据
func (s *statisticsService) GetPerformanceStats(ctx context.Context, req *model.PerformanceStatsReq) (*model.PerformanceStatsResp, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	s.logger.Info("开始获取操作员绩效统计数据",
		zap.Timep("startDate", req.StartDate),
		zap.Timep("endDate", req.EndDate),
		zap.Intp("userID", req.UserID),
		zap.Int("top", req.Top))

	// 验证日期范围
	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	// 验证参数
	if req.UserID != nil && *req.UserID <= 0 {
		return nil, fmt.Errorf("用户ID必须大于0")
	}

	if req.Top > 0 && req.Top < 5 {
		return nil, fmt.Errorf("top参数不能小于5")
	}
	if req.Top > 50 {
		return nil, fmt.Errorf("top参数不能大于50")
	}

	// 将top参数转换为指针
	var topPtr *int
	if req.Top > 0 {
		topPtr = &req.Top
	}

	// 调用DAO方法获取原始数据
	rawPerformanceItems, err := s.statisticsDAO.GetOperatorPerformance(ctx, req.StartDate, req.EndDate, req.UserID, topPtr)
	if err != nil {
		s.logger.Error("获取操作员绩效统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取绩效统计数据失败: %w", err)
	}

	// 准备返回结果
	result := &model.PerformanceStatsResp{
		Items: make([]model.PerformanceStatsItem, 0, len(rawPerformanceItems)),
	}

	// 初始化统计变量
	var totalAssigned, totalCompleted, totalOverdue int64
	var totalResponseTime, totalProcessingTime float64
	var validResponseTimeCount, validProcessingTimeCount int

	// 丰富用户信息并计算总计
	for _, rawItem := range rawPerformanceItems {
		item := model.PerformanceStatsItem{
			UserID:            rawItem.UserID,
			AssignedCount:     rawItem.AssignedCount,
			CompletedCount:    rawItem.CompletedCount,
			CompletionRate:    rawItem.CompletionRate,
			AvgResponseTime:   rawItem.AvgResponseTime,
			AvgProcessingTime: rawItem.AvgProcessingTime,
			OverdueCount:      rawItem.OverdueCount,
		}

		// 尝试获取用户名称
		if user, err := s.userDAO.GetUserByID(ctx, rawItem.UserID); err != nil {
			s.logger.Warn("获取用户信息失败",
				zap.Int("userID", rawItem.UserID),
				zap.Error(err))
			// 设置默认用户名
			item.UserName = fmt.Sprintf("用户ID_%d", rawItem.UserID)
		} else {
			item.UserName = user.Username
		}

		result.Items = append(result.Items, item)

		// 累计统计数据
		totalAssigned += rawItem.AssignedCount
		totalCompleted += rawItem.CompletedCount
		totalOverdue += rawItem.OverdueCount

		// 累计响应时间和处理时间（排除无效值）
		if rawItem.AvgResponseTime > 0 {
			totalResponseTime += rawItem.AvgResponseTime
			validResponseTimeCount++
		}
		if rawItem.AvgProcessingTime > 0 {
			totalProcessingTime += rawItem.AvgProcessingTime
			validProcessingTimeCount++
		}
	}

	// 设置总计字段
	result.TotalAssigned = totalAssigned
	result.TotalCompleted = totalCompleted
	result.TotalOverdue = totalOverdue
	result.AssignedCount = totalAssigned
	result.CompletedCount = totalCompleted

	// 计算总完成率
	if totalAssigned > 0 {
		result.CompletionRate = float64(totalCompleted) / float64(totalAssigned) * 100
	}

	// 计算平均响应时间和处理时间
	if validResponseTimeCount > 0 {
		result.AvgResponseTime = totalResponseTime / float64(validResponseTimeCount)
	}
	if validProcessingTimeCount > 0 {
		result.AvgProcessingTime = totalProcessingTime / float64(validProcessingTimeCount)
	}

	s.logger.Info("操作员绩效统计数据获取成功",
		zap.Int("operatorCount", len(result.Items)))

	return result, nil
}

// GetUserStats 获取特定用户的统计数据
func (s *statisticsService) GetUserStats(ctx context.Context, req *model.UserStatsReq) (*model.UserStatsResp, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	if req.UserID == nil || *req.UserID <= 0 {
		return nil, fmt.Errorf("用户ID不能为空且必须大于0")
	}

	s.logger.Info("开始获取用户统计数据",
		zap.Timep("startDate", req.StartDate),
		zap.Timep("endDate", req.EndDate),
		zap.Int("userID", *req.UserID))

	// 验证日期范围
	if err := s.validateDateRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	// 验证用户是否存在
	user, err := s.userDAO.GetUserByID(ctx, *req.UserID)
	if err != nil {
		s.logger.Error("获取用户信息失败", zap.Int("userID", *req.UserID), zap.Error(err))
		return nil, fmt.Errorf("用户不存在或获取用户信息失败: %w", err)
	}

	userStatsData, err := s.statisticsDAO.GetStatsByUser(ctx, req.StartDate, req.EndDate, req.UserID)
	if err != nil {
		s.logger.Error("获取用户统计数据失败", zap.Error(err))
		return nil, fmt.Errorf("获取用户统计数据失败: %w", err)
	}

	// 注意：UserStatsResp模型中没有UserName字段，所以不需要设置
	// 如果需要，可以在模型中添加UserName字段

	s.logger.Info("用户统计数据获取成功",
		zap.Int("userID", *req.UserID),
		zap.String("userName", user.Username),
		zap.Int64("createdCount", userStatsData.CreatedCount),
		zap.Int64("completedCount", userStatsData.CompletedCount))

	return userStatsData, nil
}

// validateDateRange 验证日期范围
func (s *statisticsService) validateDateRange(startDate, endDate *time.Time) error {
	if startDate != nil && endDate != nil {
		if startDate.After(*endDate) {
			return fmt.Errorf("开始日期不能晚于结束日期")
		}

		// 限制查询时间范围，避免性能问题
		if endDate.Sub(*startDate) > 365*24*time.Hour {
			return fmt.Errorf("查询时间范围不能超过一年")
		}
	}

	return nil
}

// validateDimension 验证统计维度
func (s *statisticsService) validateDimension(dimension string) error {
	validDimensions := map[string]bool{
		"day":   true,
		"week":  true,
		"month": true,
	}

	if !validDimensions[dimension] {
		return fmt.Errorf("无效的统计维度: %s，支持的维度为: day, week, month", dimension)
	}

	return nil
}
