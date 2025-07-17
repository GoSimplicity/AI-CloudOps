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
		templateGroup.POST("/create", h.CreateTemplate)
		templateGroup.PUT("/update/:id", h.UpdateTemplate)
		templateGroup.DELETE("/delete/:id", h.DeleteTemplate)
		templateGroup.GET("/list", h.ListTemplate)
		templateGroup.GET("/detail/:id", h.DetailTemplate)
		templateGroup.POST("/clone/:id", h.CloneTemplate)
	}
}

// CreateTemplate 创建模板
func (h *TemplateHandler) CreateTemplate(ctx *gin.Context) {
	var req model.CreateTemplateReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CreateTemplate(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateTemplate 更新模板
func (h *TemplateHandler) UpdateTemplate(ctx *gin.Context) {
	var req model.UpdateTemplateReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的模板ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.UpdateTemplate(ctx, &req, user.Uid)
	})
}

// DeleteTemplate 删除模板
func (h *TemplateHandler) DeleteTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的模板ID")
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteTemplate(ctx, id, user.Uid)
	})
}

// ListTemplate 获取模板列表
func (h *TemplateHandler) ListTemplate(ctx *gin.Context) {
	var req model.ListTemplateReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListTemplate(ctx, &req)
	})
}

// DetailTemplate 获取模板详情
func (h *TemplateHandler) DetailTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的模板ID")
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.DetailTemplate(ctx, id, user.Uid)
	})
}

// CloneTemplate 克隆模板
func (h *TemplateHandler) CloneTemplate(ctx *gin.Context) {
	var req model.CloneTemplateReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的模板ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.CloneTemplate(ctx, &req, user.Uid)
	})
}
