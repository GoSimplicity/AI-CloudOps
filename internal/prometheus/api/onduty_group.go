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
	onDutyGroups := monitorGroup.Group("/onduty_groups")
	{
		onDutyGroups.GET("/list", o.GetMonitorOnDutyGroupList)
		onDutyGroups.POST("/create", o.CreateMonitorOnDutyGroup)
		onDutyGroups.POST("/changes", o.CreateMonitorOnDutyGroupChange)
		onDutyGroups.PUT("/update/:id", o.UpdateMonitorOnDutyGroup)
		onDutyGroups.DELETE("/delete/:id", o.DeleteMonitorOnDutyGroup)
		onDutyGroups.GET("/detail/:id", o.GetMonitorOnDutyGroup)
		onDutyGroups.GET("/future_plan/:id", o.GetMonitorOnDutyGroupFuturePlan)
	}
}

// GetMonitorOnDutyGroupList 获取值班组列表
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return o.alertOnDutyService.GetMonitorOnDutyGroupList(ctx, &req)
	})
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.CreateMonitorOnDutyGroupReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreatorName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, o.alertOnDutyService.CreateMonitorOnDutyGroup(ctx, &req)
	})
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	var req model.CreateMonitorOnDutyGroupChangeReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreatorName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, o.alertOnDutyService.CreateMonitorOnDutyGroupChange(ctx, &req)
	})
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (o *OnDutyGroupHandler) UpdateMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.UpdateMonitorOnDutyGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, o.alertOnDutyService.UpdateMonitorOnDutyGroup(ctx, &req)
	})
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (o *OnDutyGroupHandler) DeleteMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.DeleteMonitorOnDutyGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, o.alertOnDutyService.DeleteMonitorOnDutyGroup(ctx, &req)
	})
}

// GetMonitorOnDutyGroup 获取指定的值班组信息
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return o.alertOnDutyService.GetMonitorOnDutyGroup(ctx, &req)
	})
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组的未来值班计划
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupFuturePlan(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupFuturePlanReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return o.alertOnDutyService.GetMonitorOnDutyGroupFuturePlan(ctx, &req)
	})
}
