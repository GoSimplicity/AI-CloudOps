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
	alertService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AlertRuleHandler struct {
	svc alertService.AlertManagerRuleService
}

func NewAlertRuleHandler(svc alertService.AlertManagerRuleService) *AlertRuleHandler {
	return &AlertRuleHandler{
		svc: svc,
	}
}

func (a *AlertRuleHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	alertRules := monitorGroup.Group("/alert_rules")
	{
		alertRules.GET("/list", a.GetMonitorAlertRuleList)
		alertRules.GET("/detail/:id", a.GetMonitorAlertRule)
		alertRules.POST("/promql_check", a.PromqlExprCheck)
		alertRules.POST("/create", a.CreateMonitorAlertRule)
		alertRules.PUT("/update/:id", a.UpdateMonitorAlertRule)
		alertRules.DELETE("/delete/:id", a.DeleteMonitorAlertRule)
	}
}

// CreateMonitorAlertRule 创建新的告警规则
// @Summary 创建告警规则
// @Description 创建新的监控告警规则配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorAlertRuleReq true "创建告警规则请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_rules/create [post]
// @Security BearerAuth
func (a *AlertRuleHandler) CreateMonitorAlertRule(ctx *gin.Context) {
	var req model.CreateMonitorAlertRuleReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.CreateMonitorAlertRule(ctx, &req)
	})
}

// UpdateMonitorAlertRule 更新现有的告警规则
// @Summary 更新告警规则
// @Description 更新指定的监控告警规则配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "告警规则ID"
// @Param request body model.UpdateMonitorAlertRuleReq true "更新告警规则请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_rules/update/{id} [put]
// @Security BearerAuth
func (a *AlertRuleHandler) UpdateMonitorAlertRule(ctx *gin.Context) {
	var req model.UpdateMonitorAlertRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.UpdateMonitorAlertRule(ctx, &req)
	})
}

// DeleteMonitorAlertRule 删除指定的告警规则
// @Summary 删除告警规则
// @Description 删除指定ID的监控告警规则
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "告警规则ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_rules/delete/{id} [delete]
// @Security BearerAuth
func (a *AlertRuleHandler) DeleteMonitorAlertRule(ctx *gin.Context) {
	var req model.DeleteMonitorAlertRuleRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.DeleteMonitorAlertRule(ctx, &req)
	})
}

// GetMonitorAlertRuleList 获取告警规则列表
// @Summary 获取告警规则列表
// @Description 获取所有监控告警规则的分页列表
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_rules/list [get]
// @Security BearerAuth
func (a *AlertRuleHandler) GetMonitorAlertRuleList(ctx *gin.Context) {
	var req model.GetMonitorAlertRuleListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.GetMonitorAlertRuleList(ctx, &req)
	})
}

// PromqlExprCheck 检查 PromQL 表达式的合法性
// @Summary 验证PromQL表达式
// @Description 检查和验证PromQL表达式的语法正确性
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param request body model.PromqlAlertRuleExprCheckReq true "PromQL表达式检查请求参数"
// @Success 200 {object} utils.ApiResponse "检查成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_rules/promql_check [post]
// @Security BearerAuth
func (a *AlertRuleHandler) PromqlExprCheck(ctx *gin.Context) {
	var promql model.PromqlAlertRuleExprCheckReq

	utils.HandleRequest(ctx, &promql, func() (interface{}, error) {
		return a.svc.PromqlExprCheck(ctx, &promql)
	})
}

// GetMonitorAlertRule 获取指定的告警规则详情
// @Summary 获取告警规则详情
// @Description 根据ID获取指定告警规则的详细信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "告警规则ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/alert_rules/detail/{id} [get]
// @Security BearerAuth
func (a *AlertRuleHandler) GetMonitorAlertRule(ctx *gin.Context) {
	var req model.GetMonitorAlertRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.GetMonitorAlertRule(ctx, &req)
	})
}
