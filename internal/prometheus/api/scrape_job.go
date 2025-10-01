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
)

type ScrapeJobHandler struct {
	scrapeJobService scrapeJobService.ScrapeJobService
}

func NewScrapeJobHandler(scrapeJobService scrapeJobService.ScrapeJobService) *ScrapeJobHandler {
	return &ScrapeJobHandler{
		scrapeJobService: scrapeJobService,
	}
}

func (h *ScrapeJobHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")
	{
		monitorGroup.GET("/scrape_jobs/list", h.GetMonitorScrapeJobList)
		monitorGroup.GET("/scrape_jobs/detail/:id", h.GetMonitorScrapeJobDetail)
		monitorGroup.POST("/scrape_jobs/create", h.CreateMonitorScrapeJob)
		monitorGroup.PUT("/scrape_jobs/update/:id", h.UpdateMonitorScrapeJob)
		monitorGroup.DELETE("/scrape_jobs/delete/:id", h.DeleteMonitorScrapeJob)
	}
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
func (h *ScrapeJobHandler) GetMonitorScrapeJobList(ctx *gin.Context) {
	var req model.GetMonitorScrapeJobListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.scrapeJobService.GetMonitorScrapeJobList(ctx, &req)
	})
}

// CreateMonitorScrapeJob 创建监控采集 Job
func (h *ScrapeJobHandler) CreateMonitorScrapeJob(ctx *gin.Context) {
	var req model.CreateMonitorScrapeJobReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.scrapeJobService.CreateMonitorScrapeJob(ctx, &req)
	})
}

// UpdateMonitorScrapeJob 更新监控采集 Job
func (h *ScrapeJobHandler) UpdateMonitorScrapeJob(ctx *gin.Context) {
	var req model.UpdateMonitorScrapeJobReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.scrapeJobService.UpdateMonitorScrapeJob(ctx, &req)
	})
}

// DeleteMonitorScrapeJob 删除监控采集 Job
func (h *ScrapeJobHandler) DeleteMonitorScrapeJob(ctx *gin.Context) {
	var req model.DeleteMonitorScrapeJobReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.scrapeJobService.DeleteMonitorScrapeJob(ctx, req.ID)
	})
}

// GetMonitorScrapeJobDetail 获取监控采集 Job 详情
func (h *ScrapeJobHandler) GetMonitorScrapeJobDetail(ctx *gin.Context) {
	var req model.GetMonitorScrapeJobDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.scrapeJobService.GetMonitorScrapeJobDetail(ctx, &req)
	})
}
