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
		processGroup.POST("/", h.CreateProcess)
		processGroup.PUT("/:id", h.UpdateProcess)
		processGroup.DELETE("/:id", h.DeleteProcess)
		processGroup.GET("/", h.ListProcess)
		processGroup.GET("/:id", h.GetProcess)
		processGroup.POST("/:id/publish", h.PublishProcess)
		processGroup.POST("/:id/clone", h.CloneProcess)
		processGroup.GET("/:id/validate", h.ValidateProcess)
	}
}

func (h *ProcessHandler) CreateProcess(ctx *gin.Context) {
	var req model.CreateProcessReq
	user := ctx.MustGet("user").(utils.UserClaims) // Get user claims
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateProcess(ctx, &req, user.Uid, user.Username) // Pass user info
	})
}

func (h *ProcessHandler) UpdateProcess(ctx *gin.Context) {
	var req model.UpdateProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateProcess(ctx, &req)
	})
}

func (h *ProcessHandler) DeleteProcess(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteProcess(ctx, id)
	})
}

func (h *ProcessHandler) ListProcess(ctx *gin.Context) {
	var req model.ListProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListProcess(ctx, req)
	})
}

func (h *ProcessHandler) DetailProcess(ctx *gin.Context) {
	var req model.DetailProcessReq

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailProcess(ctx, req.ID, user.Uid)
	})
}

func (h *ProcessHandler) PublishProcess(ctx *gin.Context) {
	var req model.PublishProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.PublishProcess(ctx, req)
	})
}

func (h *ProcessHandler) CloneProcess(ctx *gin.Context) {
	var req model.CloneProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CloneProcess(ctx, req)
	})
}
