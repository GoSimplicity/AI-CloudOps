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

type TemplateHandler struct {
	service service.TemplateService
}

func NewTemplateHandler(service service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		service: service,
	}
}

func (h *TemplateHandler) RegisterRouters(server *gin.Engine) {
	templateGroup := server.Group("/api/workorder/template")
	{
		templateGroup.POST("/", h.CreateTemplate)
		templateGroup.PUT("/:id", h.UpdateTemplate)
		templateGroup.DELETE("/:id", h.DeleteTemplate)
		templateGroup.GET("/", h.ListTemplate)
		templateGroup.GET("/:id", h.GetTemplate)
		templateGroup.POST("/:id/enable", h.EnableTemplate)   // 新增启用功能
		templateGroup.POST("/:id/disable", h.DisableTemplate) // 新增禁用功能
	}
}

func (h *TemplateHandler) CreateTemplate(ctx *gin.Context) {
	var req model.CreateTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTemplate(ctx, &req)
	})
}

func (h *TemplateHandler) UpdateTemplate(ctx *gin.Context) {
	var req model.UpdateTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateTemplate(ctx, &req)
	})
}

func (h *TemplateHandler) DeleteTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteTemplate(ctx, id)
	})
}

func (h *TemplateHandler) ListTemplate(ctx *gin.Context) {
	var req model.ListTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListTemplate(ctx, &req)
	})
}

func (h *TemplateHandler) GetTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTemplate(ctx, id)
	})
}

// 新增启用模板方法
func (h *TemplateHandler) EnableTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.EnableTemplate(ctx, id)
	})
}

// 新增禁用模板方法
func (h *TemplateHandler) DisableTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DisableTemplate(ctx, id)
	})
}
