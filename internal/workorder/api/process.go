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

type ProcessHandler struct {
	service service.ProcessService
}

func NewProcessHandler(service service.ProcessService) *ProcessHandler {
	return &ProcessHandler{
		service: service,
	}
}

func (h *ProcessHandler) RegisterRouters(server *gin.Engine) {
	processGroup := server.Group("/api/workorder/process")
	{
		processGroup.POST("/create", h.CreateProcess)
		processGroup.PUT("/update/:id", h.UpdateProcess)
		processGroup.DELETE("/delete/:id", h.DeleteProcess)
		processGroup.GET("/list", h.ListProcess)
		processGroup.GET("/detail/:id", h.DetailProcess)
		processGroup.GET("/relations/:id", h.GetProcessWithRelations)
		processGroup.POST("/publish/:id", h.PublishProcess)
		processGroup.POST("/clone/:id", h.CloneProcess)
	}
}

// CreateProcess 创建流程
func (h *ProcessHandler) CreateProcess(ctx *gin.Context) {
	var req model.CreateProcessReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.CreatorID = user.Uid
	req.CreatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateProcess(ctx, &req)
	})
}

// UpdateProcess 更新流程
func (h *ProcessHandler) UpdateProcess(ctx *gin.Context) {
	var req model.UpdateProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateProcess(ctx, &req)
	})
}

// DeleteProcess 删除流程
func (h *ProcessHandler) DeleteProcess(ctx *gin.Context) {
	var req model.DeleteProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteProcess(ctx, req.ID)
	})
}

// ListProcess 获取流程列表
func (h *ProcessHandler) ListProcess(ctx *gin.Context) {
	var req model.ListProcessReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListProcess(ctx, &req)
	})
}

// DetailProcess 获取流程详情
func (h *ProcessHandler) DetailProcess(ctx *gin.Context) {
	var req model.DetailProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailProcess(ctx, req.ID)
	})
}

// GetProcessWithRelations 获取流程关联信息
func (h *ProcessHandler) GetProcessWithRelations(ctx *gin.Context) {
	var req model.GetProcessWithRelationsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetProcessWithRelations(ctx, req.ID)
	})
}

// PublishProcess 发布流程
func (h *ProcessHandler) PublishProcess(ctx *gin.Context) {
	var req model.PublishProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.PublishProcess(ctx, req.ID)
	})
}

// CloneProcess 克隆流程
func (h *ProcessHandler) CloneProcess(ctx *gin.Context) {
	var req model.CloneProcessReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.CloneProcess(ctx, &req, user.Uid)
	})
}
