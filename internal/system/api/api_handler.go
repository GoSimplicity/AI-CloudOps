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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type ApiHandler struct {
	svc service.ApiService
}

func NewApiHandler(svc service.ApiService) *ApiHandler {
	return &ApiHandler{
		svc: svc,
	}
}

func (h *ApiHandler) RegisterRouters(server *gin.Engine) {
	apiGroup := server.Group("/api/apis")

	apiGroup.GET("/list", h.ListApis)
	apiGroup.POST("/create", h.CreateAPI)
	apiGroup.PUT("/update/:id", h.UpdateAPI)
	apiGroup.DELETE("/delete/:id", h.DeleteAPI)
	apiGroup.GET("/detail/:id", h.DetailAPI)
	apiGroup.GET("/statistics", h.GetApiStatistics)
}

// ListApis 获取API列表
// @Summary 获取API列表
// @Description 分页获取系统中的API列表
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param name query string false "API名称模糊搜索"
// @Param path query string false "API路径模糊搜索"
// @Success 200 {object} utils.ApiResponse{data=[]model.Api} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/apis/list [get]
func (a *ApiHandler) ListApis(ctx *gin.Context) {
	var req model.ListApisRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.ListApis(ctx, &req)
	})
}

// CreateAPI 创建新的API
// @Summary 创建API
// @Description 创建新的API接口信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.CreateApiRequest true "创建API请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/apis/create [post]
func (a *ApiHandler) CreateAPI(ctx *gin.Context) {
	var req model.CreateApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.CreateApi(ctx, &req)
	})
}

// UpdateAPI 更新API信息
// @Summary 更新API
// @Description 更新指定API的信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "API ID"
// @Param request body model.UpdateApiRequest true "更新API请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/apis/update/{id} [put]
func (a *ApiHandler) UpdateAPI(ctx *gin.Context) {
	var req model.UpdateApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.UpdateApi(ctx, &req)
	})
}

// DeleteAPI 删除API
// @Summary 删除API
// @Description 根据ID删除指定的API
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "API ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/apis/delete/{id} [delete]
func (a *ApiHandler) DeleteAPI(ctx *gin.Context) {
	var req model.DeleteApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, a.svc.DeleteApi(ctx, req.ID)
	})
}

// DetailAPI 获取API详情
// @Summary 获取API详情
// @Description 根据ID获取指定API的详细信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "API ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/apis/detail/{id} [get]
func (a *ApiHandler) DetailAPI(ctx *gin.Context) {
	var req model.GetApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return a.svc.GetApiById(ctx, id)
	})
}

// GetApiStatistics 获取API统计
// @Summary 获取API统计信息
// @Description 获取系统API相关的统计数据
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/apis/statistics [get]
func (a *ApiHandler) GetApiStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return a.svc.GetApiStatistics(ctx)
	})
}
