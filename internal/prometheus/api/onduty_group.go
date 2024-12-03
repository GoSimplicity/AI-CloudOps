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
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
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
		onDutyGroups.GET("/list", o.GetMonitorOnDutyGroupList)               // 获取值班组列表
		onDutyGroups.POST("/create", o.CreateMonitorOnDutyGroup)             // 创建新的值班组
		onDutyGroups.POST("/changes", o.CreateMonitorOnDutyGroupChange)      // 创建值班组的换班记录
		onDutyGroups.POST("/update", o.UpdateMonitorOnDutyGroup)             // 更新值班组信息
		onDutyGroups.DELETE("/:id", o.DeleteMonitorOnDutyGroup)              // 删除指定的值班组
		onDutyGroups.GET("/:id", o.GetMonitorOnDutyGroup)                    // 获取指定的值班组信息
		onDutyGroups.POST("/future_plan", o.GetMonitorOnDutyGroupFuturePlan) // 获取指定值班组的未来值班计划
	}
}

// GetMonitorOnDutyGroupList 获取值班组列表
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	searchName := ctx.Query("name")

	list, err := o.alertOnDutyService.GetMonitorOnDutyGroupList(ctx, &searchName)
	if err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "获取值班组列表失败")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	var onDutyGroup model.MonitorOnDutyGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := ctx.ShouldBind(&onDutyGroup); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	onDutyGroup.UserID = uc.Uid

	if err := o.alertOnDutyService.CreateMonitorOnDutyGroup(ctx, &onDutyGroup); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	var onDutyGroupChange model.MonitorOnDutyChange

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBind(&onDutyGroupChange); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	onDutyGroupChange.UserID = uc.Uid

	if err := o.alertOnDutyService.CreateMonitorOnDutyGroupChange(ctx, &onDutyGroupChange); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (o *OnDutyGroupHandler) UpdateMonitorOnDutyGroup(ctx *gin.Context) {
	var onDutyGroup model.MonitorOnDutyGroup

	if err := ctx.ShouldBind(&onDutyGroup); err != nil {
		apiresponse.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	if err := o.alertOnDutyService.UpdateMonitorOnDutyGroup(ctx, &onDutyGroup); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (o *OnDutyGroupHandler) DeleteMonitorOnDutyGroup(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := o.alertOnDutyService.DeleteMonitorOnDutyGroup(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetMonitorOnDutyGroup 获取指定的值班组信息
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroup(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	group, err := o.alertOnDutyService.GetMonitorOnDutyGroup(ctx, intId)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, group)
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组的未来值班计划
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupFuturePlan(ctx *gin.Context) {
	var req struct {
		Id        int    `json:"id"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	plans, err := o.alertOnDutyService.GetMonitorOnDutyGroupFuturePlan(ctx, req.Id, req.StartTime, req.EndTime)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, plans)
}
