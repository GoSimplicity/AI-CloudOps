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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ScrapePoolHandler struct {
	scrapePoolService scrapeJobService.ScrapePoolService
	l                 *zap.Logger
}

func NewScrapePoolHandler(l *zap.Logger, scrapePoolService scrapeJobService.ScrapePoolService) *ScrapePoolHandler {
	return &ScrapePoolHandler{
		l:                 l,
		scrapePoolService: scrapePoolService,
	}
}

func (s *ScrapePoolHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	scrapePools := monitorGroup.Group("/scrape_pools")
	{
		scrapePools.GET("/list", s.GetMonitorScrapePoolList)       // 获取监控采集池列表
		scrapePools.POST("/create", s.CreateMonitorScrapePool) // 创建监控采集池
		scrapePools.POST("/update", s.UpdateMonitorScrapePool) // 更新监控采集池
		scrapePools.DELETE("/:id", s.DeleteMonitorScrapePool)  // 删除监控采集池
	}
}

// GetMonitorScrapePoolList 获取监控采集池列表
func (s *ScrapePoolHandler) GetMonitorScrapePoolList(ctx *gin.Context) {
	search := ctx.Query("search")

	list, err := s.scrapePoolService.GetMonitorScrapePoolList(ctx, &search)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取监控采集池列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorScrapePool 创建监控采集池
func (s *ScrapePoolHandler) CreateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool model.MonitorScrapePool

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&monitorScrapePool); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapePool.UserID = uc.Uid
	if err := s.scrapePoolService.CreateMonitorScrapePool(ctx, &monitorScrapePool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorScrapePool 更新监控采集池
func (s *ScrapePoolHandler) UpdateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool model.MonitorScrapePool

	if err := ctx.ShouldBind(&monitorScrapePool); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := s.scrapePoolService.UpdateMonitorScrapePool(ctx, &monitorScrapePool); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorScrapePool 删除监控采集池
func (s *ScrapePoolHandler) DeleteMonitorScrapePool(ctx *gin.Context) {
	id := ctx.Param("id")
	atom, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := s.scrapePoolService.DeleteMonitorScrapePool(ctx, atom); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
