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
	alertService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
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
		alertRules.GET("/list", a.GetMonitorAlertRuleList)
		alertRules.POST("/promql_check", a.PromqlExprCheck)
		alertRules.POST("/create", a.CreateMonitorAlertRule)
		alertRules.POST("/update/:id", a.UpdateMonitorAlertRule)
		alertRules.POST("/enable/:id", a.EnableSwitchMonitorAlertRule)
		alertRules.DELETE("/delete/:id", a.DeleteMonitorAlertRule)
	}
}

// CreateMonitorAlertRule 创建新的告警规则
func (a *AlertRuleHandler) CreateMonitorAlertRule(ctx *gin.Context) {
	var alertRule model.MonitorAlertRule

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	alertRule.UserID = uc.Uid

	utils.HandleRequest(ctx, &alertRule, func() (interface{}, error) {
		return nil, a.alertRuleService.CreateMonitorAlertRule(ctx, &alertRule)
	})
}

// UpdateMonitorAlertRule 更新现有的告警规则
func (a *AlertRuleHandler) UpdateMonitorAlertRule(ctx *gin.Context) {
	var req model.MonitorAlertRule

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.alertRuleService.UpdateMonitorAlertRule(ctx, &req)
	})
}

// EnableSwitchMonitorAlertRule 切换告警规则的启用状态
func (a *AlertRuleHandler) EnableSwitchMonitorAlertRule(ctx *gin.Context) {
	var req model.EnableSwitchMonitorAlertRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.alertRuleService.EnableSwitchMonitorAlertRule(ctx, req.ID)
	})
}

// DeleteMonitorAlertRule 删除指定的告警规则
func (a *AlertRuleHandler) DeleteMonitorAlertRule(ctx *gin.Context) {
	var req model.DeleteMonitorAlertRuleRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.alertRuleService.DeleteMonitorAlertRule(ctx, req.ID)
	})
}

// GetMonitorAlertRuleList 获取告警规则列表
func (a *AlertRuleHandler) GetMonitorAlertRuleList(ctx *gin.Context) {
	var listReq model.ListReq

	utils.HandleRequest(ctx, &listReq, func() (interface{}, error) {
		return a.alertRuleService.GetMonitorAlertRuleList(ctx, &listReq)
	})
}

// PromqlExprCheck 检查 PromQL 表达式的合法性
func (a *AlertRuleHandler) PromqlExprCheck(ctx *gin.Context) {
	var promql model.PromqlExprCheckReq

	if err := ctx.ShouldBindJSON(&promql); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	exist, err := a.alertRuleService.PromqlExprCheck(ctx, promql.PromqlExpr)
	if !exist || err != nil {
		utils.ErrorWithMessage(ctx, "PromQL 表达式不合法")
		return
	}

	utils.Success(ctx)
}
