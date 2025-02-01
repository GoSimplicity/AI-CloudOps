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

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
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
		scrapePools.GET("/list", s.GetMonitorScrapePoolList)
		scrapePools.GET("/all", s.GetMonitorScrapePoolAll)
		scrapePools.POST("/create", s.CreateMonitorScrapePool)
		scrapePools.POST("/update", s.UpdateMonitorScrapePool)
		scrapePools.DELETE("/:id", s.DeleteMonitorScrapePool)
		scrapePools.GET("/total", s.GetMonitorScrapePoolTotal)
	}
}

// GetMonitorScrapePoolList 获取监控采集池列表
func (s *ScrapePoolHandler) GetMonitorScrapePoolList(ctx *gin.Context) {
	var listReq model.ListReq

	if err := ctx.ShouldBindQuery(&listReq); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	list, err := s.scrapePoolService.GetMonitorScrapePoolList(ctx, &listReq)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, list)
}

// CreateMonitorScrapePool 创建监控采集池
func (s *ScrapePoolHandler) CreateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool model.MonitorScrapePool

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&monitorScrapePool); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapePool.UserID = uc.Uid
	if err := s.scrapePoolService.CreateMonitorScrapePool(ctx, &monitorScrapePool); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// UpdateMonitorScrapePool 更新监控采集池
func (s *ScrapePoolHandler) UpdateMonitorScrapePool(ctx *gin.Context) {
	var monitorScrapePool model.MonitorScrapePool

	if err := ctx.ShouldBind(&monitorScrapePool); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := s.scrapePoolService.UpdateMonitorScrapePool(ctx, &monitorScrapePool); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// DeleteMonitorScrapePool 删除监控采集池
func (s *ScrapePoolHandler) DeleteMonitorScrapePool(ctx *gin.Context) {
	id := ctx.Param("id")
	atom, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := s.scrapePoolService.DeleteMonitorScrapePool(ctx, atom); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// GetMonitorScrapePoolTotal 获取监控采集池总数
func (s *ScrapePoolHandler) GetMonitorScrapePoolTotal(ctx *gin.Context) {
	total, err := s.scrapePoolService.GetMonitorScrapePoolTotal(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	utils.SuccessWithData(ctx, total)
}

// GetMonitorScrapePoolAll 获取所有监控采集池
func (s *ScrapePoolHandler) GetMonitorScrapePoolAll(ctx *gin.Context) {
	all, err := s.scrapePoolService.GetMonitorScrapePoolAll(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	utils.SuccessWithData(ctx, all)
}
