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
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeJobService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ScrapeJobHandler struct {
	scrapeJobService scrapeJobService.ScrapeJobService
	l                *zap.Logger
}

func NewScrapeJobHandler(l *zap.Logger, scrapeJobService scrapeJobService.ScrapeJobService) *ScrapeJobHandler {
	return &ScrapeJobHandler{
		l:                l,
		scrapeJobService: scrapeJobService,
	}
}

func (s *ScrapeJobHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	scrapeJobs := monitorGroup.Group("/scrape_jobs")
	{
		scrapeJobs.GET("/list", s.GetMonitorScrapeJobList)   // 获取监控采集 Job 列表
		scrapeJobs.POST("/create", s.CreateMonitorScrapeJob) // 创建监控采集 Job
		scrapeJobs.POST("/update", s.UpdateMonitorScrapeJob) // 更新监控采集 Job
		scrapeJobs.DELETE("/:id", s.DeleteMonitorScrapeJob)  // 删除监控采集 Job
	}
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (s *ScrapeJobHandler) GetMonitorScrapeJobList(ctx *gin.Context) {
	search := ctx.Query("search")
	list, err := s.scrapeJobService.GetMonitorScrapeJobList(ctx, &search)
	if err != nil {
		utils.ErrorWithDetails(ctx, err, "获取监控采集 Job 列表失败")
		return
	}

	utils.SuccessWithData(ctx, list)
}

// CreateMonitorScrapeJob 创建监控采集 Job
func (s *ScrapeJobHandler) CreateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob model.MonitorScrapeJob

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&monitorScrapeJob); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	monitorScrapeJob.UserID = uc.Uid

	if err := s.scrapeJobService.CreateMonitorScrapeJob(ctx, &monitorScrapeJob); err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	utils.Success(ctx)
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (s *ScrapeJobHandler) UpdateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob model.MonitorScrapeJob

	if err := ctx.ShouldBind(&monitorScrapeJob); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := s.scrapeJobService.UpdateMonitorScrapeJob(ctx, &monitorScrapeJob); err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	utils.Success(ctx)
}

// DeleteMonitorScrapeJob 删除监控采集 Job
func (s *ScrapeJobHandler) DeleteMonitorScrapeJob(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := s.scrapeJobService.DeleteMonitorScrapeJob(ctx, intId); err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	utils.Success(ctx)
}
