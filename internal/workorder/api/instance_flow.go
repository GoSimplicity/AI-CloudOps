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

type InstanceFlowHandler struct {
	flowService service.InstanceFlowService
}

func NewInstanceFlowHandler(flowService service.InstanceFlowService) *InstanceFlowHandler {
	return &InstanceFlowHandler{
		flowService: flowService,
	}
}

func (h *InstanceFlowHandler) RegisterRouters(server *gin.Engine) {
	flowGroup := server.Group("/api/workorder/instance/flow")
	{
		flowGroup.POST("/create", h.CreateInstanceFlow)
		flowGroup.GET("/list", h.ListInstanceFlows)
		flowGroup.GET("/detail/:id", h.DetailInstanceFlow)
	}
}

// CreateInstanceFlow 创建工单流转记录
// @Summary 创建工单流转记录
// @Description 为指定工单实例创建新的流转记录
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderInstanceFlowReq true "创建流转记录请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/flow/create [post]
// 创建工单流转记录
func (h *InstanceFlowHandler) CreateInstanceFlow(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceFlowReq

	user := ctx.MustGet("user").(utils.UserClaims)
	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.flowService.CreateInstanceFlow(ctx, &req)
	})
}

// ListInstanceFlows 获取工单流转记录列表
// @Summary 获取工单流转记录列表
// @Description 分页获取工单流转记录列表
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
// @Router /api/workorder/instance/flow/list [get]
// 获取工单流转记录列表
func (h *InstanceFlowHandler) ListInstanceFlows(ctx *gin.Context) {
	var req model.ListWorkorderInstanceFlowReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.flowService.ListInstanceFlows(ctx, &req)
	})
}

// DetailInstanceFlow 获取工单流转记录详情
// @Summary 获取工单流转记录详情
// @Description 获取指定工单流转记录的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "流转记录ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/flow/detail/{id} [get]
// 获取工单流转记录详情
func (h *InstanceFlowHandler) DetailInstanceFlow(ctx *gin.Context) {
	var req model.DetailWorkorderInstanceFlowReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.flowService.DetailInstanceFlow(ctx, req.ID)
	})
}
