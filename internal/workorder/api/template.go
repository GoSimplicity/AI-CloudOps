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
	}
}

// CreateTemplate 创建模板
// @Summary 创建工单模板
// @Description 创建新的工单模板
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderTemplateReq true "创建模板请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/template/create [post]
func (h *TemplateHandler) CreateTemplate(ctx *gin.Context) {
	var req model.CreateWorkorderTemplateReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CreateTemplate(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateTemplate 更新模板
// @Summary 更新工单模板
// @Description 更新指定的工单模板信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param request body model.UpdateWorkorderTemplateReq true "更新模板请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/template/update/{id} [put]
func (h *TemplateHandler) UpdateTemplate(ctx *gin.Context) {
	var req model.UpdateWorkorderTemplateReq

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
// @Summary 删除工单模板
// @Description 删除指定的工单模板
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/template/delete/{id} [delete]
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
// @Summary 获取工单模板列表
// @Description 分页获取工单模板列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse{data=[]model.WorkorderTemplate} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/template/list [get]
func (h *TemplateHandler) ListTemplate(ctx *gin.Context) {
	var req model.ListWorkorderTemplateReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListTemplate(ctx, &req)
	})
}

// DetailTemplate 获取模板详情
// @Summary 获取工单模板详情
// @Description 根据ID获取工单模板的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/template/detail/{id} [get]
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
