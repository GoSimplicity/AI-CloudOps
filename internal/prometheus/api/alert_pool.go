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

func (a *AlertPoolHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	alertManagerPools := monitorGroup.Group("/alert_manager_pools")
	{
		alertManagerPools.GET("/list", a.GetMonitorAlertManagerPoolList)
		alertManagerPools.POST("/create", a.CreateMonitorAlertManagerPool)
		alertManagerPools.PUT("/update/:id", a.UpdateMonitorAlertManagerPool)
		alertManagerPools.DELETE("/delete/:id", a.DeleteMonitorAlertManagerPool)
		alertManagerPools.GET("/detail/:id", a.GetMonitorAlertManagerPool)
	}
}

// GetMonitorAlertManagerPoolList 获取 AlertManager 集群池列表
// @Summary 获取AlertManager集群池列表
// @Description 获取所有AlertManager集群池的分页列表
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_manager_pools/list [get]
// @Security BearerAuth
func (a *AlertPoolHandler) GetMonitorAlertManagerPoolList(ctx *gin.Context) {
	var req model.GetMonitorAlertManagerPoolListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.GetMonitorAlertManagerPoolList(ctx, &req)
	})
}

// CreateMonitorAlertManagerPool 创建新的 AlertManager 集群池
// @Summary 创建AlertManager集群池
// @Description 创建新的AlertManager集群池配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorAlertManagerPoolReq true "创建请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_manager_pools/create [post]
// @Security BearerAuth
func (a *AlertPoolHandler) CreateMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.CreateMonitorAlertManagerPoolReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.CreateMonitorAlertManagerPool(ctx, &req)
	})
}

// UpdateMonitorAlertManagerPool 更新现有的 AlertManager 集群池
// @Summary 更新AlertManager集群池
// @Description 更新指定的AlertManager集群池配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "集群池ID"
// @Param request body model.UpdateMonitorAlertManagerPoolReq true "更新请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_manager_pools/update/{id} [put]
// @Security BearerAuth
func (a *AlertPoolHandler) UpdateMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.UpdateMonitorAlertManagerPoolReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.UpdateMonitorAlertManagerPool(ctx, &req)
	})
}

// DeleteMonitorAlertManagerPool 删除指定的 AlertManager 集群池
// @Summary 删除AlertManager集群池
// @Description 删除指定ID的AlertManager集群池
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "集群池ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_manager_pools/delete/{id} [delete]
// @Security BearerAuth
func (a *AlertPoolHandler) DeleteMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.DeleteMonitorAlertManagerPoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.DeleteMonitorAlertManagerPool(ctx, &req)
	})
}

// GetMonitorAlertManagerPool 获取指定的AlertManager集群池详情
// @Summary 获取AlertManager集群池详情
// @Description 根据ID获取指定AlertManager集群池的详细信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "集群池ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_manager_pools/detail/{id} [get]
// @Security BearerAuth
func (a *AlertPoolHandler) GetMonitorAlertManagerPool(ctx *gin.Context) {
	var req model.GetMonitorAlertManagerPoolReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.GetMonitorAlertManagerPool(ctx, &req)
	})
}
