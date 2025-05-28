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
	"net/http"
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
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
		statsGroup.GET("/overview", h.GetOverview)
		statsGroup.GET("/trend", h.GetTrend)
		statsGroup.GET("/category", h.GetCategoryStats)
		statsGroup.GET("/performance", h.GetPerformanceStats)
		statsGroup.GET("/user", h.GetUserStats)
		statsGroup.GET("/export", h.ExportStats)
	}
}

// GetOverview 获取工单总览统计数据
func (h *StatisticsHandler) GetOverview(ctx *gin.Context) {
	req := &model.OverviewStatsReq{}

	// 解析可选的日期参数
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.StartDate = &startDate
		}
	}

	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.EndDate = &endDate
		}
	}

	result, err := h.service.GetOverview(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetTrend 获取工单趋势统计数据
func (h *StatisticsHandler) GetTrend(ctx *gin.Context) {
	req := &model.TrendStatsReq{}

	// 解析必需的日期参数
	startDateStr := ctx.Query("start_date")
	if startDateStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "开始日期不能为空",
			"data":    nil,
		})
		return
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "开始日期格式错误，请使用 YYYY-MM-DD 格式",
			"data":    nil,
		})
		return
	}
	req.StartDate = startDate

	endDateStr := ctx.Query("end_date")
	if endDateStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "结束日期不能为空",
			"data":    nil,
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "结束日期格式错误，请使用 YYYY-MM-DD 格式",
			"data":    nil,
		})
		return
	}
	req.EndDate = endDate

	// 解析统计维度参数
	req.Dimension = ctx.DefaultQuery("dimension", "day")

	// 解析可选的分类ID参数
	if categoryIDStr := ctx.Query("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.Atoi(categoryIDStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "分类ID格式错误",
				"data":    nil,
			})
			return
		} else {
			req.CategoryID = &categoryID
		}
	}

	result, err := h.service.GetTrend(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetCategoryStats 获取按分类统计的工单数据
func (h *StatisticsHandler) GetCategoryStats(ctx *gin.Context) {
	req := &model.CategoryStatsReq{}

	// 解析可选的日期参数
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.StartDate = &startDate
		}
	}

	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.EndDate = &endDate
		}
	}

	// 解析可选的top参数
	if topStr := ctx.Query("top"); topStr != "" {
		if top, err := strconv.Atoi(topStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "top参数格式错误",
				"data":    nil,
			})
			return
		} else {
			req.Top = top
		}
	}

	result, err := h.service.GetCategoryStats(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetPerformanceStats 获取操作员绩效统计数据
func (h *StatisticsHandler) GetPerformanceStats(ctx *gin.Context) {
	req := &model.PerformanceStatsReq{}

	// 解析可选的日期参数
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.StartDate = &startDate
		}
	}

	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.EndDate = &endDate
		}
	}

	// 解析可选的用户ID参数
	if userIDStr := ctx.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.Atoi(userIDStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "用户ID格式错误",
				"data":    nil,
			})
			return
		} else {
			req.UserID = &userID
		}
	}

	// 解析可选的top参数
	if topStr := ctx.Query("top"); topStr != "" {
		if top, err := strconv.Atoi(topStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "top参数格式错误",
				"data":    nil,
			})
			return
		} else {
			req.Top = top
		}
	}

	result, err := h.service.GetPerformanceStats(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// GetUserStats 获取特定用户的统计数据
func (h *StatisticsHandler) GetUserStats(ctx *gin.Context) {
	req := &model.UserStatsReq{}

	// 解析必需的用户ID参数
	userIDStr := ctx.Query("user_id")
	if userIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID不能为空",
			"data":    nil,
		})
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID格式错误",
			"data":    nil,
		})
		return
	}
	req.UserID = &userID

	// 解析可选的日期参数
	if startDateStr := ctx.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.StartDate = &startDate
		}
	}

	if endDateStr := ctx.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，请使用 YYYY-MM-DD 格式",
				"data":    nil,
			})
			return
		} else {
			req.EndDate = &endDate
		}
	}

	result, err := h.service.GetUserStats(ctx.Request.Context(), req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// ExportStats 导出统计数据（占位方法）
func (h *StatisticsHandler) ExportStats(ctx *gin.Context) {
	// TODO: 实现统计数据导出功能
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "导出功能开发中",
		"data":    nil,
	})
}
