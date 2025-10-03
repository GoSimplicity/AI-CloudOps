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
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type OnDutyGroupHandler struct {
	alertOnDutyService alert.AlertManagerOnDutyService
}

func NewOnDutyGroupHandler(alertOnDutyService alert.AlertManagerOnDutyService) *OnDutyGroupHandler {
	return &OnDutyGroupHandler{
		alertOnDutyService: alertOnDutyService,
	}
}

func (h *OnDutyGroupHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")
	{
		monitorGroup.GET("/onduty_groups/list", h.GetMonitorOnDutyGroupList)
		monitorGroup.POST("/onduty_groups/create", h.CreateMonitorOnDutyGroup)
		monitorGroup.POST("/onduty_groups/changes", h.CreateMonitorOnDutyGroupChange)
		monitorGroup.GET("/onduty_groups/changes/:id", h.GetMonitorOnDutyGroupChangeList)
		monitorGroup.PUT("/onduty_groups/update/:id", h.UpdateMonitorOnDutyGroup)
		monitorGroup.DELETE("/onduty_groups/delete/:id", h.DeleteMonitorOnDutyGroup)
		monitorGroup.GET("/onduty_groups/detail/:id", h.GetMonitorOnDutyGroup)
		monitorGroup.GET("/onduty_groups/future_plan/:id", h.GetMonitorOnDutyGroupFuturePlan)
		monitorGroup.GET("/onduty_groups/history/:id", h.GetMonitorOnDutyHistory)
	}
}

// GetMonitorOnDutyGroupList 获取值班组列表
func (h *OnDutyGroupHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertOnDutyService.GetMonitorOnDutyGroupList(ctx, &req)
	})
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (h *OnDutyGroupHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.CreateMonitorOnDutyGroupReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertOnDutyService.CreateMonitorOnDutyGroup(ctx, &req)
	})
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
func (h *OnDutyGroupHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	var req model.CreateMonitorOnDutyGroupChangeReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertOnDutyService.CreateMonitorOnDutyGroupChange(ctx, &req)
	})
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (h *OnDutyGroupHandler) UpdateMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.UpdateMonitorOnDutyGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertOnDutyService.UpdateMonitorOnDutyGroup(ctx, &req)
	})
}

// DeleteMonitorOnDutyGroup 删除指定的值班组
func (h *OnDutyGroupHandler) DeleteMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.DeleteMonitorOnDutyGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertOnDutyService.DeleteMonitorOnDutyGroup(ctx, &req)
	})
}

// GetMonitorOnDutyGroup 获取指定的值班组信息
func (h *OnDutyGroupHandler) GetMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertOnDutyService.GetMonitorOnDutyGroup(ctx, &req)
	})
}

// GetMonitorOnDutyGroupFuturePlan 获取指定值班组的未来值班计划
func (h *OnDutyGroupHandler) GetMonitorOnDutyGroupFuturePlan(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupFuturePlanReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertOnDutyService.GetMonitorOnDutyGroupFuturePlan(ctx, &req)
	})
}

// GetMonitorOnDutyHistory 获取值班历史记录
func (h *OnDutyGroupHandler) GetMonitorOnDutyHistory(ctx *gin.Context) {
	var req model.GetMonitorOnDutyHistoryReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.OnDutyGroupID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertOnDutyService.GetMonitorOnDutyHistory(ctx, &req)
	})
}

// GetMonitorOnDutyGroupChangeList 获取值班组换班记录列表
func (h *OnDutyGroupHandler) GetMonitorOnDutyGroupChangeList(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupChangeListReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.OnDutyGroupID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertOnDutyService.GetMonitorOnDutyGroupChangeList(ctx, &req)
	})
}
