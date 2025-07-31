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
func (h *WorkorderProcessHandler) CreateWorkorderProcess(ctx *gin.Context) {
	var req model.CreateWorkorderProcessReq

	user := ctx.MustGet("user").(utils.UserClaims)
	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateWorkorderProcess(ctx, &req)
	})
}

// UpdateWorkorderProcess 更新工单流程
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
func (h *WorkorderProcessHandler) ListWorkorderProcess(ctx *gin.Context) {
	var req model.ListWorkorderProcessReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListWorkorderProcess(ctx, &req)
	})
}

// DetailWorkorderProcess 获取工单流程详情
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
