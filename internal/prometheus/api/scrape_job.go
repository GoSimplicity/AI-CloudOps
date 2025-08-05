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

func (s *ScrapeJobHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	scrapeJobs := monitorGroup.Group("/scrape_jobs")
	{
		scrapeJobs.GET("/list", s.GetMonitorScrapeJobList)
		scrapeJobs.GET("/detail/:id", s.GetMonitorScrapeJobDetail)
		scrapeJobs.POST("/create", s.CreateMonitorScrapeJob)
		scrapeJobs.PUT("/update/:id", s.UpdateMonitorScrapeJob)
		scrapeJobs.DELETE("/delete/:id", s.DeleteMonitorScrapeJob)
	}
}

// GetMonitorScrapeJobList 获取监控采集 Job 列表
// @Summary 获取采集任务列表
// @Description 获取所有监控采集任务的分页列表
// @Tags 采集管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/scrape_jobs/list [get]
// @Security BearerAuth
func (s *ScrapeJobHandler) GetMonitorScrapeJobList(ctx *gin.Context) {
	var req model.GetMonitorScrapeJobListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.scrapeJobService.GetMonitorScrapeJobList(ctx, &req)
	})
}

// CreateMonitorScrapeJob 创建监控采集 Job
// @Summary 创建采集任务
// @Description 创建新的监控采集任务配置
// @Tags 采集管理
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorScrapeJobReq true "创建采集任务请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/scrape_jobs/create [post]
// @Security BearerAuth
func (s *ScrapeJobHandler) CreateMonitorScrapeJob(ctx *gin.Context) {
	var req model.CreateMonitorScrapeJobReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.scrapeJobService.CreateMonitorScrapeJob(ctx, &req)
	})
}

// UpdateMonitorScrapeJob 更新监控采集 Job
// @Summary 更新采集任务
// @Description 更新指定的监控采集任务配置
// @Tags 采集管理
// @Accept json
// @Produce json
// @Param id path int true "采集任务ID"
// @Param request body model.UpdateMonitorScrapeJobReq true "更新采集任务请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/scrape_jobs/update/{id} [put]
// @Security BearerAuth
func (s *ScrapeJobHandler) UpdateMonitorScrapeJob(ctx *gin.Context) {
	var req model.UpdateMonitorScrapeJobReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.scrapeJobService.UpdateMonitorScrapeJob(ctx, &req)
	})
}

// DeleteMonitorScrapeJob 删除监控采集 Job
// @Summary 删除采集任务
// @Description 删除指定ID的监控采集任务
// @Tags 采集管理
// @Accept json
// @Produce json
// @Param id path int true "采集任务ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/scrape_jobs/delete/{id} [delete]
// @Security BearerAuth
func (s *ScrapeJobHandler) DeleteMonitorScrapeJob(ctx *gin.Context) {
	var req model.DeleteMonitorScrapeJobReq

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

// GetMonitorScrapeJobDetail 获取监控采集 Job 详情
// @Summary 获取采集任务详情
// @Description 根据ID获取指定监控采集任务的详细信息
// @Tags 采集管理
// @Accept json
// @Produce json
// @Param id path int true "采集任务ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/scrape_jobs/detail/{id} [get]
// @Security BearerAuth
func (s *ScrapeJobHandler) GetMonitorScrapeJobDetail(ctx *gin.Context) {
	var req model.GetMonitorScrapeJobDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.scrapeJobService.GetMonitorScrapeJobDetail(ctx, &req)
	})
}
