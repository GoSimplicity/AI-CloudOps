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

func (h *StatisticsHandler) GetOverview(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []string{},
	})
}

func (h *StatisticsHandler) GetTrend(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []string{},
	})
}

func (h *StatisticsHandler) GetCategoryStats(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []string{},
	})
}

func (h *StatisticsHandler) GetPerformanceStats(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []string{},
	})
}

func (h *StatisticsHandler) GetUserStats(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []string{},
	})
}
