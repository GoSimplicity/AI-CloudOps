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
func (h *ApiHandler) ListApis(ctx *gin.Context) {
	var req model.ListApisRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListApis(ctx, &req)
	})
}

// CreateAPI 创建新的API
func (h *ApiHandler) CreateAPI(ctx *gin.Context) {
	var req model.CreateApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateApi(ctx, &req)
	})
}

// UpdateAPI 更新API信息
func (h *ApiHandler) UpdateAPI(ctx *gin.Context) {
	var req model.UpdateApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.UpdateApi(ctx, &req)
	})
}

// DeleteAPI 删除API
func (h *ApiHandler) DeleteAPI(ctx *gin.Context) {
	var req model.DeleteApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteApi(ctx, req.ID)
	})
}

// DetailAPI 获取API详情
func (h *ApiHandler) DetailAPI(ctx *gin.Context) {
	var req model.GetApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetApiById(ctx, id)
	})
}

// GetApiStatistics 获取API统计
func (h *ApiHandler) GetApiStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetApiStatistics(ctx)
	})
}
