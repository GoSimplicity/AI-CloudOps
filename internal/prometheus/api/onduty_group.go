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

type OnDutyGroupHandler struct {
	alertOnDutyService alertEventService.AlertManagerOnDutyService
	l                  *zap.Logger
}

func NewOnDutyGroupHandler(l *zap.Logger, alertOnDutyService alertEventService.AlertManagerOnDutyService) *OnDutyGroupHandler {
	return &OnDutyGroupHandler{
		l:                  l,
		alertOnDutyService: alertOnDutyService,
	}
}

func (o *OnDutyGroupHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	// 值班组相关路由
	onDutyGroups := monitorGroup.Group("/onDuty_groups")
	{
		onDutyGroups.GET("/list", o.GetMonitorOnDutyGroupList)
		onDutyGroups.POST("/create", o.CreateMonitorOnDutyGroup)
		onDutyGroups.POST("/changes", o.CreateMonitorOnDutyGroupChange)
		onDutyGroups.POST("/update", o.UpdateMonitorOnDutyGroup)
		onDutyGroups.DELETE("/:id", o.DeleteMonitorOnDutyGroup)
		onDutyGroups.GET("/:id", o.GetMonitorOnDutyGroup)
		onDutyGroups.GET("/future_plan", o.GetMonitorOnDutyGroupFuturePlan)
		onDutyGroups.GET("/total", o.GetMonitorOnDutyGroupTotal)
	}
}

// GetMonitorOnDutyGroupList 获取值班组列表
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	var listReq model.ListReq

	if err := ctx.ShouldBindQuery(&listReq); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	list, err := o.alertOnDutyService.GetMonitorOnDutyGroupList(ctx, &listReq)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, list)
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	var onDutyGroup model.MonitorOnDutyGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&onDutyGroup); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	onDutyGroup.UserID = uc.Uid

	if err := o.alertOnDutyService.CreateMonitorOnDutyGroup(ctx, &onDutyGroup); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	var onDutyGroupChange model.MonitorOnDutyChange

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&onDutyGroupChange); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	onDutyGroupChange.UserID = uc.Uid

	if err := o.alertOnDutyService.CreateMonitorOnDutyGroupChange(ctx, &onDutyGroupChange); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (o *OnDutyGroupHandler) UpdateMonitorOnDutyGroup(ctx *gin.Context) {
	var onDutyGroup model.MonitorOnDutyGroup

	if err := ctx.ShouldBind(&onDutyGroup); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := o.alertOnDutyService.UpdateMonitorOnDutyGroup(ctx, &onDutyGroup); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (o *OnDutyGroupHandler) DeleteMonitorOnDutyGroup(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := o.alertOnDutyService.DeleteMonitorOnDutyGroup(ctx, intId); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// GetMonitorOnDutyGroup 获取指定的值班组信息
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroup(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	group, err := o.alertOnDutyService.GetMonitorOnDutyGroup(ctx, intId)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, group)
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组的未来值班计划
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupFuturePlan(ctx *gin.Context) {
	var req struct {
		Id        int    `json:"id" form:"id" binding:"required"`
		StartTime string `json:"start_time" form:"start_time" binding:"required"` // 例如
		EndTime   string `json:"end_time" form:"end_time" binding:"required"`
	}

	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	plans, err := o.alertOnDutyService.GetMonitorOnDutyGroupFuturePlan(ctx, req.Id, req.StartTime, req.EndTime)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, plans)
}

// GetMonitorOnDutyGroupTotal 获取监控告警事件总数
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupTotal(ctx *gin.Context) {
	total, err := o.alertOnDutyService.GetMonitorOnDutyTotal(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	utils.SuccessWithData(ctx, total)
}
