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
	"go.uber.org/zap"
)

type AlertPoolHandler struct {
	alertPoolService alertEventService.AlertManagerPoolService
	l                *zap.Logger
}

func NewAlertPoolHandler(l *zap.Logger, alertPoolService alertEventService.AlertManagerPoolService) *AlertPoolHandler {
	return &AlertPoolHandler{
		l:                l,
		alertPoolService: alertPoolService,
	}
}

func (a *AlertPoolHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	alertManagerPools := monitorGroup.Group("/alertManager_pools")
	{
		alertManagerPools.GET("/list", a.GetMonitorAlertManagerPoolList)
		alertManagerPools.POST("/create", a.CreateMonitorAlertManagerPool)
		alertManagerPools.PUT("/update/:id", a.UpdateMonitorAlertManagerPool)
		alertManagerPools.DELETE("/delete/:id", a.DeleteMonitorAlertManagerPool)
	}
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表
func (a *AlertPoolHandler) GetMonitorAlertManagerPoolList(ctx *gin.Context) {
	var req model.GetMonitorAlertManagerPoolListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.alertPoolService.GetMonitorAlertManagerPoolList(ctx, &req)
	})
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
func (a *AlertPoolHandler) CreateMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.CreateMonitorAlertManagerPoolReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.alertPoolService.CreateMonitorAlertManagerPool(ctx, &req)
	})
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
func (a *AlertPoolHandler) UpdateMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.UpdateMonitorAlertManagerPoolReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.alertPoolService.UpdateMonitorAlertManagerPool(ctx, &req)
	})
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
func (a *AlertPoolHandler) DeleteMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.DeleteMonitorAlertManagerPoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.alertPoolService.DeleteMonitorAlertManagerPool(ctx, req.ID)
	})
}
