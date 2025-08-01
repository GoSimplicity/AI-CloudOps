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

type CategoryGroupHandler struct {
	service service.CategoryGroupService
}

func NewCategoryGroupHandler(service service.CategoryGroupService) *CategoryGroupHandler {
	return &CategoryGroupHandler{
		service: service,
	}
}

func (h *CategoryGroupHandler) RegisterRouters(server *gin.Engine) {
	categoryGroup := server.Group("/api/workorder/category")
	{
		categoryGroup.POST("/create", h.CreateCategory)
		categoryGroup.PUT("/update/:id", h.UpdateCategory)
		categoryGroup.DELETE("/delete/:id", h.DeleteCategory)
		categoryGroup.GET("/list", h.ListCategory)
		categoryGroup.GET("/detail/:id", h.DetailCategory)
	}
}

// CreateCategory 创建工单分类
// @Summary 创建工单分类
// @Description 创建新的工单分类
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderCategoryReq true "创建分类请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/category/create [post]
func (h *CategoryGroupHandler) CreateCategory(ctx *gin.Context) {
	var req model.CreateWorkorderCategoryReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateCategory(ctx, &req)
	})
}

// UpdateCategory 更新工单分类
// @Summary 更新工单分类
// @Description 更新指定的工单分类信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Param request body model.UpdateWorkorderCategoryReq true "更新分类请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/category/update/{id} [put]
func (h *CategoryGroupHandler) UpdateCategory(ctx *gin.Context) {
	var req model.UpdateWorkorderCategoryReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateCategory(ctx, &req)
	})
}

// DeleteCategory 删除工单分类
// @Summary 删除工单分类
// @Description 删除指定的工单分类
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/category/delete/{id} [delete]
func (h *CategoryGroupHandler) DeleteCategory(ctx *gin.Context) {
	var req model.DeleteWorkorderCategoryReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteCategory(ctx, req.ID)
	})
}

// ListCategory 获取工单分类列表
// @Summary 获取工单分类列表
// @Description 分页获取工单分类列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse{data=[]model.WorkorderCategory} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/category/list [get]
func (h *CategoryGroupHandler) ListCategory(ctx *gin.Context) {
	var req model.ListWorkorderCategoryReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListCategory(ctx, req)
	})
}

// DetailCategory 获取工单分类详情
// @Summary 获取工单分类详情
// @Description 根据ID获取工单分类的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "分类ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/category/detail/{id} [get]
func (h *CategoryGroupHandler) DetailCategory(ctx *gin.Context) {
	var req model.DetailWorkorderCategoryReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetCategory(ctx, req.ID)
	})
}
