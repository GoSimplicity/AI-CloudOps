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
		scrapeJobs.GET("/list", s.GetMonitorScrapeJobList)
		scrapeJobs.POST("/create", s.CreateMonitorScrapeJob)
		scrapeJobs.POST("/update", s.UpdateMonitorScrapeJob)
		scrapeJobs.DELETE("/:id", s.DeleteMonitorScrapeJob)
	}
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (s *ScrapeJobHandler) GetMonitorScrapeJobList(ctx *gin.Context) {
	var listReq model.ListReq

	utils.HandleRequest(ctx, &listReq, func() (interface{}, error) {
		return s.scrapeJobService.GetMonitorScrapeJobList(ctx, &listReq)
	})
}

// CreateMonitorScrapeJob 创建监控采集 Job
func (s *ScrapeJobHandler) CreateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob model.MonitorScrapeJob

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	monitorScrapeJob.UserID = uc.Uid

	utils.HandleRequest(ctx, &monitorScrapeJob, func() (interface{}, error) {
		return nil, s.scrapeJobService.CreateMonitorScrapeJob(ctx, &monitorScrapeJob)
	})
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (s *ScrapeJobHandler) UpdateMonitorScrapeJob(ctx *gin.Context) {
	var monitorScrapeJob model.MonitorScrapeJob

	utils.HandleRequest(ctx, &monitorScrapeJob, func() (interface{}, error) {
		return nil, s.scrapeJobService.UpdateMonitorScrapeJob(ctx, &monitorScrapeJob)
	})
}

// DeleteMonitorScrapeJob 删除监控采集 Job
func (s *ScrapeJobHandler) DeleteMonitorScrapeJob(ctx *gin.Context) {
	var req model.DeleteMonitorScrapeJobRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.scrapeJobService.DeleteMonitorScrapeJob(ctx, req.ID)
	})
}
