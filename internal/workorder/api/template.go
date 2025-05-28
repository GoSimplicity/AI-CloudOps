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
		templateGroup.GET("/:id", h.DetailTemplate)
		templateGroup.POST("/:id/enable", h.EnableTemplate)
		templateGroup.POST("/:id/disable", h.DisableTemplate)
		templateGroup.GET("/process/:process_id", h.GetTemplatesByProcessID)
		templateGroup.GET("/category/:category_id", h.GetTemplatesByCategory)
		templateGroup.POST("/batch/status", h.BatchUpdateStatus)
		templateGroup.GET("/count", h.GetTemplateCount)
		templateGroup.GET("/check-name", h.CheckTemplateName)
	}
}

// CreateTemplate 创建模板
func (h *TemplateHandler) CreateTemplate(ctx *gin.Context) {
	var req model.CreateTemplateReq
	user := ctx.MustGet("user").(utils.UserClaims)
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTemplate(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateTemplate 更新模板
func (h *TemplateHandler) UpdateTemplate(ctx *gin.Context) {
	var req model.UpdateTemplateReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateTemplate(ctx, &req)
	})
}

// DeleteTemplate 删除模板
func (h *TemplateHandler) DeleteTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteTemplate(ctx, id)
	})
}

// ListTemplate 获取模板列表
func (h *TemplateHandler) ListTemplate(ctx *gin.Context) {
	var req model.ListTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListTemplate(ctx, &req)
	})
}

// DetailTemplate 获取模板详情
func (h *TemplateHandler) DetailTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.DetailTemplate(ctx, id)
	})
}

// EnableTemplate 启用模板
func (h *TemplateHandler) EnableTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	user := ctx.MustGet("user").(utils.UserClaims)
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.EnableTemplate(ctx, id, user.Uid)
	})
}

// DisableTemplate 禁用模板
func (h *TemplateHandler) DisableTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	user := ctx.MustGet("user").(utils.UserClaims)
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DisableTemplate(ctx, id, user.Uid)
	})
}

// GetTemplatesByProcessID 根据流程ID获取模板列表
func (h *TemplateHandler) GetTemplatesByProcessID(ctx *gin.Context) {
	processIDStr := ctx.Param("process_id")
	processID, err := strconv.Atoi(processIDStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的流程ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTemplatesByProcessID(ctx, processID)
	})
}

// GetTemplatesByCategory 根据分类ID获取模板列表
func (h *TemplateHandler) GetTemplatesByCategory(ctx *gin.Context) {
	categoryIDStr := ctx.Param("category_id")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的分类ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTemplatesByCategory(ctx, categoryID)
	})
}

// BatchUpdateStatus 批量更新状态
func (h *TemplateHandler) BatchUpdateStatus(ctx *gin.Context) {
	var req struct {
		IDs    []int `json:"ids" binding:"required"`
		Status int8  `json:"status" binding:"required,oneof=0 1"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BatchUpdateStatus(ctx, req.IDs, req.Status)
	})
}

// GetTemplateCount 获取模板总数
func (h *TemplateHandler) GetTemplateCount(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTemplateCount(ctx)
	})
}

// CheckTemplateName 检查模板名称是否存在
func (h *TemplateHandler) CheckTemplateName(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		utils.ErrorWithMessage(ctx, "模板名称不能为空")
		return
	}

	excludeIDStr := ctx.Query("exclude_id")
	excludeID := 0
	if excludeIDStr != "" {
		var err error
		excludeID, err = strconv.Atoi(excludeIDStr)
		if err != nil {
			utils.ErrorWithMessage(ctx, "无效的排除ID")
			return
		}
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		exists, err := h.service.IsTemplateNameExists(ctx, name, excludeID)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"exists": exists}, nil
	})
}
