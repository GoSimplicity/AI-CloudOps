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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AlertPoolHandler struct {
	svc alertEventService.AlertManagerPoolService
}

func NewAlertPoolHandler(svc alertEventService.AlertManagerPoolService) *AlertPoolHandler {
	return &AlertPoolHandler{
		svc: svc,
	}
}

func (h *AlertPoolHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")
	{
		monitorGroup.GET("/alert_manager_pools/list", h.GetMonitorAlertManagerPoolList)
		monitorGroup.POST("/alert_manager_pools/create", h.CreateMonitorAlertManagerPool)
		monitorGroup.PUT("/alert_manager_pools/update/:id", h.UpdateMonitorAlertManagerPool)
		monitorGroup.DELETE("/alert_manager_pools/delete/:id", h.DeleteMonitorAlertManagerPool)
		monitorGroup.GET("/alert_manager_pools/detail/:id", h.GetMonitorAlertManagerPool)
	}
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表
func (h *AlertPoolHandler) GetMonitorAlertManagerPoolList(ctx *gin.Context) {
	var req model.GetMonitorAlertManagerPoolListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetMonitorAlertManagerPoolList(ctx, &req)
	})
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
func (h *AlertPoolHandler) CreateMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.CreateMonitorAlertManagerPoolReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateMonitorAlertManagerPool(ctx, &req)
	})
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
func (h *AlertPoolHandler) UpdateMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.UpdateMonitorAlertManagerPoolReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.UpdateMonitorAlertManagerPool(ctx, &req)
	})
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (h *AlertPoolHandler) DeleteMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.DeleteMonitorAlertManagerPoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteMonitorAlertManagerPool(ctx, &req)
	})
}

// GetMonitorAlertManagerPool 获取指定的AlertManager集群池详情
func (h *AlertPoolHandler) GetMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.GetMonitorAlertManagerPoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetMonitorAlertManagerPool(ctx, &req)
	})
}
