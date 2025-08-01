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

type FormDesignHandler struct {
	service service.FormDesignService
}

func NewFormDesignHandler(service service.FormDesignService) *FormDesignHandler {
	return &FormDesignHandler{
		service: service,
	}
}

func (h *FormDesignHandler) RegisterRouters(server *gin.Engine) {
	formDesignGroup := server.Group("/api/workorder/form-design")
	{
		formDesignGroup.POST("/create", h.CreateFormDesign)
		formDesignGroup.PUT("/update/:id", h.UpdateFormDesign)
		formDesignGroup.DELETE("/delete/:id", h.DeleteFormDesign)
		formDesignGroup.GET("/list", h.ListFormDesign)
		formDesignGroup.GET("/detail/:id", h.DetailFormDesign)
	}
}

// CreateFormDesign 创建表单设计
// @Summary 创建工单表单设计
// @Description 创建新的工单表单设计配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderFormDesignReq true "创建表单设计请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/form-design/create [post]
func (h *FormDesignHandler) CreateFormDesign(ctx *gin.Context) {
	var req model.CreateWorkorderFormDesignReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateFormDesign(ctx, &req)
	})
}

// UpdateFormDesign 更新表单设计
// @Summary 更新工单表单设计
// @Description 更新指定的工单表单设计配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "表单设计ID"
// @Param request body model.UpdateWorkorderFormDesignReq true "更新表单设计请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/form-design/update/{id} [put]
func (h *FormDesignHandler) UpdateFormDesign(ctx *gin.Context) {
	var req model.UpdateWorkorderFormDesignReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateFormDesign(ctx, &req)
	})
}

// DeleteFormDesign 删除表单设计
// @Summary 删除工单表单设计
// @Description 删除指定的工单表单设计配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "表单设计ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/form-design/delete/{id} [delete]
func (h *FormDesignHandler) DeleteFormDesign(ctx *gin.Context) {
	var req model.DeleteWorkorderFormDesignReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteFormDesign(ctx, req.ID)
	})
}

// ListFormDesign 获取表单设计列表
// @Summary 获取工单表单设计列表
// @Description 分页获取工单表单设计配置列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param name query string false "表单设计名称"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/form-design/list [get]
func (h *FormDesignHandler) ListFormDesign(ctx *gin.Context) {
	var req model.ListWorkorderFormDesignReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListFormDesign(ctx, &req)
	})
}

// DetailFormDesign 获取表单设计详情
// @Summary 获取工单表单设计详情
// @Description 获取指定工单表单设计的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "表单设计ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/form-design/detail/{id} [get]
func (h *FormDesignHandler) DetailFormDesign(ctx *gin.Context) {
	var req model.DetailWorkorderFormDesignReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetFormDesign(ctx, req.ID)
	})
}
