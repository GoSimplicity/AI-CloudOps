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
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/gin-gonic/gin"
)

type AlertEventHandler struct {
	svc alertEventService.AlertManagerEventService
}

func NewAlertEventHandler(svc alertEventService.AlertManagerEventService) *AlertEventHandler {
	return &AlertEventHandler{
		svc: svc,
	}
}

func (a *AlertEventHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	alertEvents := monitorGroup.Group("/alert_events")
	{
		alertEvents.GET("/list", a.GetMonitorAlertEventList)
		alertEvents.POST("/silence/:id", a.EventAlertSilence)
		alertEvents.POST("/claim/:id", a.EventAlertClaim)
		alertEvents.POST("/unsilence/:id", a.EventAlertUnSilence)
	}
}

// GetMonitorAlertEventList 获取告警事件列表
// @Summary 获取告警事件列表
// @Description 获取监控告警事件列表，支持分页和条件查询
// @Tags 告警事件管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param status query string false "告警状态"
// @Param severity query string false "告警级别"
// @Success 200 {object} utils.ApiResponse "成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_events/list [get]
// @Security BearerAuth
func (a *AlertEventHandler) GetMonitorAlertEventList(ctx *gin.Context) {
	var req model.GetMonitorAlertEventListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.GetMonitorAlertEventList(ctx, &req)
	})
}

// EventAlertSilence 将指定告警事件设置为静默状态
// @Summary 告警事件静默
// @Description 将指定的告警事件设置为静默状态，静默期间不会发送通知
// @Tags 告警事件管理
// @Accept json
// @Produce json
// @Param id path int true "告警事件ID"
// @Param data body model.EventAlertSilenceReq true "静默配置"
// @Success 200 {object} utils.ApiResponse "成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_events/silence/{id} [post]
// @Security BearerAuth
func (a *AlertEventHandler) EventAlertSilence(ctx *gin.Context) {
	var req model.EventAlertSilenceReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.EventAlertSilence(ctx, &req)
	})
}

// EventAlertClaim 认领指定的告警事件
// @Summary 认领告警事件
// @Description 认领指定的告警事件，表示该告警由当前用户处理
// @Tags 告警事件管理
// @Accept json
// @Produce json
// @Param id path int true "告警事件ID"
// @Param data body model.EventAlertClaimReq true "认领信息"
// @Success 200 {object} utils.ApiResponse "成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_events/claim/{id} [post]
// @Security BearerAuth
func (a *AlertEventHandler) EventAlertClaim(ctx *gin.Context) {
	var req model.EventAlertClaimReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.EventAlertClaim(ctx, &req)
	})
}

// EventAlertUnSilence 取消指定告警事件的静默状态
// @Summary 取消告警事件静默
// @Description 取消指定告警事件的静默状态，恢复告警通知
// @Tags 告警事件管理
// @Accept json
// @Produce json
// @Param id path int true "告警事件ID"
// @Param data body model.EventAlertUnSilenceReq true "取消静默信息"
// @Success 200 {object} utils.ApiResponse "成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_events/unsilence/{id} [post]
// @Security BearerAuth
func (a *AlertEventHandler) EventAlertUnSilence(ctx *gin.Context) {
	var req model.EventAlertUnSilenceReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.EventAlertUnSilence(ctx, &req)
	})
}
