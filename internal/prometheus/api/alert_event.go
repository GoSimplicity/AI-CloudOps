package api

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

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

type AlertEventHandler struct {
	alertEventService alertEventService.AlertManagerEventService
	l                 *zap.Logger
}

func NewAlertEventHandler(l *zap.Logger, alertEventService alertEventService.AlertManagerEventService) *AlertEventHandler {
	return &AlertEventHandler{
		l:                 l,
		alertEventService: alertEventService,
	}
}

func (a *AlertEventHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	alertEvents := monitorGroup.Group("/alert_events")
	{
		alertEvents.GET("/", a.GetMonitorAlertEventList)          // 获取告警事件列表
		alertEvents.POST("/:id/silence", a.EventAlertSilence)     // 将指定告警事件设置为静默状态
		alertEvents.POST("/:id/claim", a.EventAlertClaim)         // 认领指定的告警事件
		alertEvents.POST("/:id/unSilence", a.EventAlertUnSilence) // 取消指定告警事件的静默状态
		alertEvents.POST("/silence", a.BatchEventAlertSilence)    // 批量设置告警事件为静默状态
	}
}

// GetMonitorAlertEventList 获取告警事件列表
func (a *AlertEventHandler) GetMonitorAlertEventList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := a.alertEventService.GetMonitorAlertEventList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// EventAlertSilence 将指定告警事件设置为静默状态
func (a *AlertEventHandler) EventAlertSilence(ctx *gin.Context) {
	var silence model.AlertEventSilenceRequest

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := ctx.ShouldBind(&silence); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := a.alertEventService.EventAlertSilence(ctx, intId, &silence, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EventAlertClaim 认领指定的告警事件
func (a *AlertEventHandler) EventAlertClaim(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := a.alertEventService.EventAlertClaim(ctx, intId, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EventAlertUnSilence 取消指定告警事件的静默状态
func (a *AlertEventHandler) EventAlertUnSilence(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := a.alertEventService.EventAlertClaim(ctx, intId, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchEventAlertSilence 批量设置告警事件为静默状态
func (a *AlertEventHandler) BatchEventAlertSilence(ctx *gin.Context) {
	var req model.BatchEventAlertSilenceRequest

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := a.alertEventService.BatchEventAlertSilence(ctx, &req, uc.Uid); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
