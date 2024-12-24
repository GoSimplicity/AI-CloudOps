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
	"strconv"

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

	apiGroup.POST("/list", h.ListApis)
	apiGroup.POST("/create", h.CreateAPI)
	apiGroup.POST("/update", h.UpdateAPI)
	apiGroup.DELETE("/:id", h.DeleteAPI)
}

// ListApis 获取API列表
func (a *ApiHandler) ListApis(c *gin.Context) {
	var req model.ListApisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	// 调用service层获取API列表
	apis, total, err := a.svc.ListApis(c.Request.Context(), req.PageNumber, req.PageSize)
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, gin.H{
		"list":  apis,
		"total": total,
	})
}

// CreateAPI 创建新的API
func (a *ApiHandler) CreateAPI(c *gin.Context) {
	var req model.CreateApiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	// 构建API对象
	api := &model.Api{
		Name:        req.Name,
		Path:        req.Path,
		Method:      req.Method,
		Description: req.Description,
		Version:     req.Version,
		Category:    req.Category,
		IsPublic:    req.IsPublic,
	}

	if err := a.svc.CreateApi(c.Request.Context(), api); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// UpdateAPI 更新API信息
func (a *ApiHandler) UpdateAPI(c *gin.Context) {
	var req model.UpdateApiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	// 构建更新的API对象
	api := &model.Api{
		ID:          req.ID,
		Name:        req.Name,
		Path:        req.Path,
		Method:      req.Method,
		Description: req.Description,
		Version:     req.Version,
		Category:    req.Category,
		IsPublic:    req.IsPublic,
	}

	if err := a.svc.UpdateApi(c.Request.Context(), api); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// DeleteAPI 删除API
func (a *ApiHandler) DeleteAPI(c *gin.Context) {
	// 从URL参数中获取API ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.Error(c)
		return
	}

	if err := a.svc.DeleteApi(c.Request.Context(), id); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}
