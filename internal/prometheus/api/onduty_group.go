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

func (o *OnDutyGroupHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	// 值班组相关路由
	onDutyGroups := monitorGroup.Group("/onduty_groups")
	{
		onDutyGroups.GET("/list", o.GetMonitorOnDutyGroupList)
		onDutyGroups.POST("/create", o.CreateMonitorOnDutyGroup)
		onDutyGroups.POST("/changes", o.CreateMonitorOnDutyGroupChange)
		onDutyGroups.GET("/changes/:id", o.GetMonitorOnDutyGroupChangeList)
		onDutyGroups.PUT("/update/:id", o.UpdateMonitorOnDutyGroup)
		onDutyGroups.DELETE("/delete/:id", o.DeleteMonitorOnDutyGroup)
		onDutyGroups.GET("/detail/:id", o.GetMonitorOnDutyGroup)
		onDutyGroups.GET("/future_plan/:id", o.GetMonitorOnDutyGroupFuturePlan)
		onDutyGroups.GET("/history/:id", o.GetMonitorOnDutyHistory)
	}
}

// GetMonitorOnDutyGroupList 获取值班组列表
// @Summary 获取值班组列表
// @Description 获取所有值班组的分页列表
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/list [get]
// @Security BearerAuth
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupList(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return o.alertOnDutyService.GetMonitorOnDutyGroupList(ctx, &req)
	})
}

// CreateMonitorOnDutyGroup 创建新的值班组
// @Summary 创建值班组
// @Description 创建新的值班组配置
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorOnDutyGroupReq true "创建值班组请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/create [post]
// @Security BearerAuth
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroup(ctx *gin.Context) {
	var req model.CreateMonitorOnDutyGroupReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, o.alertOnDutyService.CreateMonitorOnDutyGroup(ctx, &req)
	})
}

// CreateMonitorOnDutyGroupChange 创建值班组的换班记录
// @Summary 创建换班记录
// @Description 为指定的值班组创建换班记录
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorOnDutyGroupChangeReq true "创建换班记录请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/changes [post]
// @Security BearerAuth
func (o *OnDutyGroupHandler) CreateMonitorOnDutyGroupChange(ctx *gin.Context) {
	var req model.CreateMonitorOnDutyGroupChangeReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, o.alertOnDutyService.CreateMonitorOnDutyGroupChange(ctx, &req)
	})
}

// UpdateMonitorOnDutyGroup 更新值班组信息
// @Summary 更新值班组
// @Description 更新指定的值班组配置信息
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path int true "值班组ID"
// @Param request body model.UpdateMonitorOnDutyGroupReq true "更新值班组请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/update/{id} [put]
// @Security BearerAuth
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
// @Summary 删除值班组
// @Description 删除指定ID的值班组
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path int true "值班组ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/delete/{id} [delete]
// @Security BearerAuth
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
// @Summary 获取值班组详情
// @Description 根据ID获取指定值班组的详细信息
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path int true "值班组ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/detail/{id} [get]
// @Security BearerAuth
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
// @Summary 获取值班计划
// @Description 获取指定值班组的未来值班计划安排
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path int true "值班组ID"
// @Param days query int false "获取天数" default(30)
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/future_plan/{id} [get]
// @Security BearerAuth
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

// GetMonitorOnDutyHistory 获取值班历史记录
// @Summary 获取值班历史
// @Description 获取指定值班组的历史值班记录
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path int true "值班组ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/history/{id} [get]
// @Security BearerAuth
func (o *OnDutyGroupHandler) GetMonitorOnDutyHistory(ctx *gin.Context) {
	var req model.GetMonitorOnDutyHistoryReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.OnDutyGroupID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return o.alertOnDutyService.GetMonitorOnDutyHistory(ctx, &req)
	})
}

// GetMonitorOnDutyGroupChangeList 获取值班组换班记录列表
// @Summary 获取换班记录列表
// @Description 获取指定值班组的换班记录列表
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path int true "值班组ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/onduty_groups/changes/{id} [get]
// @Security BearerAuth
func (o *OnDutyGroupHandler) GetMonitorOnDutyGroupChangeList(ctx *gin.Context) {
	var req model.GetMonitorOnDutyGroupChangeListReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.OnDutyGroupID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return o.alertOnDutyService.GetMonitorOnDutyGroupChangeList(ctx, &req)
	})
}
