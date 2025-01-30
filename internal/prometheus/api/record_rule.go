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

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RecordRuleHandler struct {
	alertRecordService alertEventService.AlertManagerRecordService
	l                  *zap.Logger
}

func NewRecordRuleHandler(l *zap.Logger, alertRecordService alertEventService.AlertManagerRecordService) *RecordRuleHandler {
	return &RecordRuleHandler{
		l:                  l,
		alertRecordService: alertRecordService,
	}
}

func (r *RecordRuleHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	recordRules := monitorGroup.Group("/record_rules")
	{
		recordRules.GET("/list", r.GetMonitorRecordRuleList)
		recordRules.POST("/create", r.CreateMonitorRecordRule)
		recordRules.POST("/update", r.UpdateMonitorRecordRule)
		recordRules.DELETE("/:id", r.DeleteMonitorRecordRule)
		recordRules.DELETE("/", r.BatchDeleteMonitorRecordRule)
		recordRules.POST("/:id/enable", r.EnableSwitchMonitorRecordRule)
		recordRules.POST("/enable", r.BatchEnableSwitchMonitorRecordRule)
		recordRules.GET("/total", r.GetMonitorRecordRuleTotal)
	}
}

// GetMonitorRecordRuleList 获取预聚合规则列表
func (r *RecordRuleHandler) GetMonitorRecordRuleList(ctx *gin.Context) {
	var listReq model.ListReq

	if err := ctx.ShouldBindQuery(&listReq); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	list, err := r.alertRecordService.GetMonitorRecordRuleList(ctx, &listReq)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, list)
}

// CreateMonitorRecordRule 创建新的预聚合规则
func (r *RecordRuleHandler) CreateMonitorRecordRule(ctx *gin.Context) {
	var recordRule model.MonitorRecordRule

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&recordRule); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	recordRule.UserID = uc.Uid

	if err := r.alertRecordService.CreateMonitorRecordRule(ctx, &recordRule); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// UpdateMonitorRecordRule 更新现有的预聚合规则
func (r *RecordRuleHandler) UpdateMonitorRecordRule(ctx *gin.Context) {
	var recordRule model.MonitorRecordRule

	if err := ctx.ShouldBind(&recordRule); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := r.alertRecordService.UpdateMonitorRecordRule(ctx, &recordRule); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// DeleteMonitorRecordRule 删除指定的预聚合规则
func (r *RecordRuleHandler) DeleteMonitorRecordRule(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := r.alertRecordService.DeleteMonitorRecordRule(ctx, intId); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// BatchDeleteMonitorRecordRule 批量删除预聚合规则
func (r *RecordRuleHandler) BatchDeleteMonitorRecordRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBind(&req); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := r.alertRecordService.BatchDeleteMonitorRecordRule(ctx, req.IDs); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// EnableSwitchMonitorRecordRule 切换预聚合规则的启用状态
func (r *RecordRuleHandler) EnableSwitchMonitorRecordRule(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := r.alertRecordService.EnableSwitchMonitorRecordRule(ctx, intId); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// BatchEnableSwitchMonitorRecordRule 批量切换预聚合规则的启用状态
func (r *RecordRuleHandler) BatchEnableSwitchMonitorRecordRule(ctx *gin.Context) {
	var req model.BatchRequest

	if err := ctx.ShouldBind(&req); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := r.alertRecordService.BatchEnableSwitchMonitorRecordRule(ctx, req.IDs); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// GetMonitorRecordRuleTotal 获取监控告警事件总数
func (r *RecordRuleHandler) GetMonitorRecordRuleTotal(ctx *gin.Context) {
	total, err := r.alertRecordService.GetMonitorRecordRuleTotal(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	utils.SuccessWithData(ctx, total)
}
