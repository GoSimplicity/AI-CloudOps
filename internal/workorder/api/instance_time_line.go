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
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type InstanceTimeLineHandler struct {
	service service.WorkorderInstanceTimeLineService
}

func NewInstanceTimeLineHandler(service service.WorkorderInstanceTimeLineService) *InstanceTimeLineHandler {
	return &InstanceTimeLineHandler{
		service: service,
	}
}

func (h *InstanceTimeLineHandler) RegisterRouters(server *gin.Engine) {
	timelineGroup := server.Group("/api/workorder/instance/timeline")
	{
		timelineGroup.POST("/create", h.CreateInstanceTimeLine)
		timelineGroup.PUT("/update/:id", h.UpdateInstanceTimeLine)
		timelineGroup.DELETE("/delete/:id", h.DeleteInstanceTimeLine)
		timelineGroup.GET("/list", h.ListInstanceTimeLine)
		timelineGroup.GET("/detail/:id", h.DetailInstanceTimeLine)
	}
}

// CreateInstanceTimeLine 创建工单时间线记录
// @Summary 创建工单时间线记录
// @Description 为指定工单实例创建新的时间线记录
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderInstanceTimelineReq true "创建时间线记录请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/timeline/create [post]
// CreateInstanceTimeLine 创建工单时间线记录
func (h *InstanceTimeLineHandler) CreateInstanceTimeLine(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceTimelineReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.CreateInstanceTimeLine(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateInstanceTimeLine 更新工单时间线记录
// @Summary 更新工单时间线记录
// @Description 更新指定的工单时间线记录信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "时间线记录ID"
// @Param request body model.UpdateWorkorderInstanceTimelineReq true "更新时间线记录请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/timeline/update/{id} [put]
// UpdateInstanceTimeLine 更新工单时间线记录
func (h *InstanceTimeLineHandler) UpdateInstanceTimeLine(ctx *gin.Context) {
	var req model.UpdateWorkorderInstanceTimelineReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.UpdateInstanceTimeLine(ctx, &req, user.Uid)
	})
}

// DeleteInstanceTimeLine 删除工单时间线记录
// @Summary 删除工单时间线记录
// @Description 删除指定的工单时间线记录
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "时间线记录ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/timeline/delete/{id} [delete]
// DeleteInstanceTimeLine 删除工单时间线记录
func (h *InstanceTimeLineHandler) DeleteInstanceTimeLine(ctx *gin.Context) {
	var req model.DeleteWorkorderInstanceTimelineReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.DeleteInstanceTimeLine(ctx, req.ID, user.Uid)
	})
}

// DetailInstanceTimeLine 获取工单时间线记录详情
// @Summary 获取工单时间线记录详情
// @Description 获取指定工单时间线记录的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "时间线记录ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/timeline/detail/{id} [get]
// DetailInstanceTimeLine 获取工单时间线记录详情
func (h *InstanceTimeLineHandler) DetailInstanceTimeLine(ctx *gin.Context) {
	var req model.DetailWorkorderInstanceTimelineReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.GetInstanceTimeLine(ctx, req.ID)
	})
}

// ListInstanceTimeLine 获取工单时间线记录列表
// @Summary 获取工单时间线记录列表
// @Description 分页获取工单时间线记录列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param instanceId query int false "工单实例ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/timeline/list [get]
// ListInstanceTimeLine 获取工单时间线记录列表
func (h *InstanceTimeLineHandler) ListInstanceTimeLine(ctx *gin.Context) {
	var req model.ListWorkorderInstanceTimelineReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListInstanceTimeLine(ctx, &req)
	})
}
