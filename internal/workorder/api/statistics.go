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
		statsGroup.POST("/overview", h.GetOverview)
		statsGroup.POST("/trend", h.GetTrend)
		statsGroup.POST("/category", h.GetCategoryStats)
		statsGroup.POST("/performance", h.GetPerformanceStats)
		statsGroup.POST("/user", h.GetUserStats)
	}
}

func (h *StatisticsHandler) GetOverview(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetTrend(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetCategoryStats(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetPerformanceStats(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetUserStats(ctx *gin.Context) {

}
