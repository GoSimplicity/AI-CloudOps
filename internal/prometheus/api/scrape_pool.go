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

func (h *ScrapePoolHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")
	{
		monitorGroup.GET("/scrape_pools/list", h.GetMonitorScrapePoolList)
		monitorGroup.POST("/scrape_pools/create", h.CreateMonitorScrapePool)
		monitorGroup.PUT("/scrape_pools/update/:id", h.UpdateMonitorScrapePool)
		monitorGroup.DELETE("/scrape_pools/delete/:id", h.DeleteMonitorScrapePool)
		monitorGroup.GET("/scrape_pools/detail/:id", h.GetMonitorScrapePoolDetail)
	}
}

// GetMonitorScrapePoolList 获取监控采集池列表
func (h *ScrapePoolHandler) GetMonitorScrapePoolList(ctx *gin.Context) {
	var req model.GetMonitorScrapePoolListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.scrapePoolService.GetMonitorScrapePoolList(ctx, &req)
	})
}

// CreateMonitorScrapePool 创建监控采集池
func (h *ScrapePoolHandler) CreateMonitorScrapePool(ctx *gin.Context) {
	var req model.CreateMonitorScrapePoolReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.scrapePoolService.CreateMonitorScrapePool(ctx, &req)
	})
}

// UpdateMonitorScrapePool 更新监控采集池
func (h *ScrapePoolHandler) UpdateMonitorScrapePool(ctx *gin.Context) {
	var req model.UpdateMonitorScrapePoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.scrapePoolService.UpdateMonitorScrapePool(ctx, &req)
	})
}

// DeleteMonitorScrapePool 删除监控采集池
func (h *ScrapePoolHandler) DeleteMonitorScrapePool(ctx *gin.Context) {
	var req model.DeleteMonitorScrapePoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.scrapePoolService.DeleteMonitorScrapePool(ctx, &req)
	})
}

// GetMonitorScrapePoolDetail 获取监控采集池详情
func (h *ScrapePoolHandler) GetMonitorScrapePoolDetail(ctx *gin.Context) {
	var req model.GetMonitorScrapePoolDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.scrapePoolService.GetMonitorScrapePoolDetail(ctx, &req)
	})
}
