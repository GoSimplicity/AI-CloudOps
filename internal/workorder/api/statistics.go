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
package api

import (
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type StatisticsHandler struct {
	service service.StatisticsService
}

func NewStatisticsHandler(service service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{
		service: service,
	}
}

func (h *StatisticsHandler) RegisterRouters(server *gin.Engine) {
	statsGroup := server.Group("/api/workorder/statistics")
	{
		statsGroup.GET("/overview", h.GetOverview)             // 概览统计
		statsGroup.GET("/trend", h.GetTrend)                   // 趋势统计
		statsGroup.GET("/category", h.GetCategoryStats)        // 分类统计
		statsGroup.GET("/user", h.GetUserStats)                // 用户统计
		statsGroup.GET("/template", h.GetTemplateStats)        // 模板统计
		statsGroup.GET("/status", h.GetStatusDistribution)     // 状态分布
		statsGroup.GET("/priority", h.GetPriorityDistribution) // 优先级分布
	}
}

// GetOverview 获取工单总览统计
func (h *StatisticsHandler) GetOverview(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetOverview(ctx, req)
	})
}

// GetTrend 获取工单趋势统计
func (h *StatisticsHandler) GetTrend(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	// 趋势统计需要维度参数
	dimension := ctx.DefaultQuery("dimension", "day")
	if !isValidDimension(dimension) {
		utils.ErrorWithMessage(ctx, "维度参数只能是 day、week 或 month")
		return
	}
	req.Dimension = dimension

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetTrend(ctx, req)
	})
}

// GetCategoryStats 获取分类统计
func (h *StatisticsHandler) GetCategoryStats(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetCategoryStats(ctx, req)
	})
}

// GetUserStats 获取用户统计
func (h *StatisticsHandler) GetUserStats(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetUserStats(ctx, req)
	})
}

// GetTemplateStats 获取模板统计
func (h *StatisticsHandler) GetTemplateStats(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetTemplateStats(ctx, req)
	})
}

// GetStatusDistribution 获取状态分布
func (h *StatisticsHandler) GetStatusDistribution(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetStatusDistribution(ctx, req)
	})
}

// GetPriorityDistribution 获取优先级分布
func (h *StatisticsHandler) GetPriorityDistribution(ctx *gin.Context) {
	req := h.parseStatsRequest(ctx)
	if req == nil {
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return h.service.GetPriorityDistribution(ctx, req)
	})
}

// parseStatsRequest 统一解析统计请求参数
func (h *StatisticsHandler) parseStatsRequest(ctx *gin.Context) *model.StatsReq {
	req := &model.StatsReq{}

	// 解析日期参数
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		startDate, err := parseTimeRFC3339(startDateStr)
		if err != nil {
			utils.ErrorWithMessage(ctx, "开始日期格式错误，请使用 RFC3339 格式")
			return nil
		}
		req.StartDate = &startDate
	}

	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		endDate, err := parseTimeRFC3339(endDateStr)
		if err != nil {
			utils.ErrorWithMessage(ctx, "结束日期格式错误，请使用 RFC3339 格式")
			return nil
		}
		req.EndDate = &endDate
	}

	// 验证日期范围
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		utils.ErrorWithMessage(ctx, "开始日期不能晚于结束日期")
		return nil
	}

	// 解析筛选参数
	if categoryIDStr := ctx.Query("category_id"); categoryIDStr != "" {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			utils.ErrorWithMessage(ctx, "分类ID必须是数字")
			return nil
		}
		req.CategoryID = &categoryID
	}

	if userIDStr := ctx.Query("user_id"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			utils.ErrorWithMessage(ctx, "用户ID必须是数字")
			return nil
		}
		req.UserID = &userID
	}

	if status := ctx.Query("status"); status != "" {
		req.Status = &status
	}

	if priority := ctx.Query("priority"); priority != "" {
		req.Priority = &priority
	}

	// 解析排行数量参数
	if topStr := ctx.Query("top"); topStr != "" {
		top, err := strconv.Atoi(topStr)
		if err != nil || top <= 0 || top > 50 {
			utils.ErrorWithMessage(ctx, "top参数必须是1-50之间的数字")
			return nil
		}
		req.Top = top
	} else {
		req.Top = 10 // 默认值
	}

	// 解析排序字段
	if sortBy := ctx.Query("sort_by"); sortBy != "" {
		if !isValidSortBy(sortBy) {
			utils.ErrorWithMessage(ctx, "sort_by参数只能是 count、completion_rate 或 avg_process_time")
			return nil
		}
		req.SortBy = sortBy
	}

	return req
}

// parseTimeRFC3339 解析RFC3339格式的时间字符串
func parseTimeRFC3339(timeStr string) (time.Time, error) {
	// 尝试多种格式解析时间
	// 先尝试完整的RFC3339格式
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// 再尝试带时区的格式
	t, err = time.Parse("2006-01-02T15:04:05Z07:00", timeStr)
	if err == nil {
		return t, nil
	}

	// 再尝试不带时区的格式
	t, err = time.Parse("2006-01-02T15:04:05", timeStr)
	if err == nil {
		return t, nil
	}

	// 最后尝试只有日期的格式
	return time.Parse("2006-01-02", timeStr)
}

// isValidDimension 验证维度参数
func isValidDimension(dimension string) bool {
	validDimensions := map[string]bool{
		"day":   true,
		"week":  true,
		"month": true,
	}
	return validDimensions[dimension]
}

// isValidSortBy 验证排序字段参数
func isValidSortBy(sortBy string) bool {
	validSortBy := map[string]bool{
		"count":            true,
		"completion_rate":  true,
		"avg_process_time": true,
	}
	return validSortBy[sortBy]
}
