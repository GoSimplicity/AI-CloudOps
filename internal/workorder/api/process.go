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

type WorkorderProcessHandler struct {
	service service.WorkorderProcessService
}

func NewWorkorderProcessHandler(service service.WorkorderProcessService) *WorkorderProcessHandler {
	return &WorkorderProcessHandler{
		service: service,
	}
}

func (h *WorkorderProcessHandler) RegisterRouters(server *gin.Engine) {
	processGroup := server.Group("/api/workorder/process")
	{
		processGroup.POST("/create", h.CreateWorkorderProcess)
		processGroup.PUT("/update/:id", h.UpdateWorkorderProcess)
		processGroup.DELETE("/delete/:id", h.DeleteWorkorderProcess)
		processGroup.GET("/list", h.ListWorkorderProcess)
		processGroup.GET("/detail/:id", h.DetailWorkorderProcess)
	}
}

// CreateWorkorderProcess 创建工单流程
// @Summary 创建工单流程
// @Description 创建新的工单流程配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderProcessReq true "创建工单流程请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/process/create [post]
func (h *WorkorderProcessHandler) CreateWorkorderProcess(ctx *gin.Context) {
	var req model.CreateWorkorderProcessReq

	user := ctx.MustGet("user").(utils.UserClaims)
	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateWorkorderProcess(ctx, &req)
	})
}

// UpdateWorkorderProcess 更新工单流程
// @Summary 更新工单流程
// @Description 更新指定的工单流程配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单流程ID"
// @Param request body model.UpdateWorkorderProcessReq true "更新工单流程请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/process/update/{id} [put]
func (h *WorkorderProcessHandler) UpdateWorkorderProcess(ctx *gin.Context) {
	var req model.UpdateWorkorderProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateWorkorderProcess(ctx, &req)
	})
}

// DeleteWorkorderProcess 删除工单流程
// @Summary 删除工单流程
// @Description 删除指定的工单流程
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单流程ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/process/delete/{id} [delete]
func (h *WorkorderProcessHandler) DeleteWorkorderProcess(ctx *gin.Context) {
	var req model.DeleteWorkorderProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteWorkorderProcess(ctx, req.ID)
	})
}

// ListWorkorderProcess 获取工单流程列表
// @Summary 获取工单流程列表
// @Description 分页获取工单流程列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse{data=[]model.WorkorderProcess} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/process/list [get]
func (h *WorkorderProcessHandler) ListWorkorderProcess(ctx *gin.Context) {
	var req model.ListWorkorderProcessReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListWorkorderProcess(ctx, &req)
	})
}

// DetailWorkorderProcess 获取工单流程详情
// @Summary 获取工单流程详情
// @Description 根据ID获取工单流程的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单流程ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/process/detail/{id} [get]
func (h *WorkorderProcessHandler) DetailWorkorderProcess(ctx *gin.Context) {
	var req model.DetailWorkorderProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailWorkorderProcess(ctx, req.ID)
	})
}
