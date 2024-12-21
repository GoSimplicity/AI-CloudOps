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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AlertRuleHandler struct {
	alertRuleService alertService.AlertManagerRuleService
	l                *zap.Logger
}

func NewAlertRuleHandler(l *zap.Logger, alertRuleService alertService.AlertManagerRuleService) *AlertRuleHandler {
	return &AlertRuleHandler{
		l:                l,
		alertRuleService: alertRuleService,
	}
}

func (a *AlertRuleHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	alertRules := monitorGroup.Group("/alert_rules")
	{
		alertRules.GET("/list", a.GetMonitorAlertRuleList)                    // 获取告警规则列表
		alertRules.POST("/promql_check", a.PromqlExprCheck)                   // 检查 PromQL 表达式的合法性
		alertRules.POST("/create", a.CreateMonitorAlertRule)                  // 创建新的告警规则
		alertRules.POST("/update", a.UpdateMonitorAlertRule)                  // 更新现有的告警规则
		alertRules.POST("/enable", a.EnableSwitchMonitorAlertRule)            // 切换告警规则的启用状态
		alertRules.POST("/batch_enable", a.BatchEnableSwitchMonitorAlertRule) // 批量切换告警规则的启用状态
		alertRules.DELETE("/:id", a.DeleteMonitorAlertRule)                   // 删除指定的告警规则
		alertRules.DELETE("/", a.BatchDeleteMonitorAlertRule)                 // 批量删除告警规则
	}
}

// CreateMonitorAlertRule 创建新的告警规则
func (a *AlertRuleHandler) CreateMonitorAlertRule(ctx *gin.Context) {
	var alertRule model.MonitorAlertRule

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&alertRule); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	alertRule.UserID = uc.Uid

	if err := a.alertRuleService.CreateMonitorAlertRule(ctx, &alertRule); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorAlertRule 更新现有的告警规则
func (a *AlertRuleHandler) UpdateMonitorAlertRule(ctx *gin.Context) {
	var alertRule model.MonitorAlertRule

	if err := ctx.ShouldBind(&alertRule); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := a.alertRuleService.UpdateMonitorAlertRule(ctx, &alertRule); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (a *AlertRuleHandler) EnableSwitchMonitorAlertRule(ctx *gin.Context) {
	var req model.IdRequest

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := a.alertRuleService.EnableSwitchMonitorAlertRule(ctx, req.ID); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则的启用状态
func (a *AlertRuleHandler) BatchEnableSwitchMonitorAlertRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := a.alertRuleService.BatchEnableSwitchMonitorAlertRule(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorAlertRule 删除指定的告警规则
func (a *AlertRuleHandler) DeleteMonitorAlertRule(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := a.alertRuleService.DeleteMonitorAlertRule(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchDeleteMonitorAlertRule 批量删除告警规则
func (a *AlertRuleHandler) BatchDeleteMonitorAlertRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := a.alertRuleService.BatchDeleteMonitorAlertRule(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorAlertRuleList 获取告警规则列表
func (a *AlertRuleHandler) GetMonitorAlertRuleList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := a.alertRuleService.GetMonitorAlertRuleList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// PromqlExprCheck 检查 PromQL 表达式的合法性
func (a *AlertRuleHandler) PromqlExprCheck(ctx *gin.Context) {
	var promql model.PromqlExprCheckReq

	if err := ctx.ShouldBind(&promql); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	exist, err := a.alertRuleService.PromqlExprCheck(ctx, promql.PromqlExpr)
	if !exist || err != nil {
		apiresponse.ErrorWithMessage(ctx, "PromQL 表达式不合法")
		return
	}

	apiresponse.Success(ctx)
}
