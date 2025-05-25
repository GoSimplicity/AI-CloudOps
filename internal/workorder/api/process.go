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
		processGroup.GET("/:id/detail", h.DetailProcess)
		processGroup.GET("/:id/relations", h.GetProcessWithRelations)
		processGroup.POST("/:id/publish", h.PublishProcess)
		processGroup.POST("/:id/clone", h.CloneProcess)
		processGroup.PATCH("/:id/status", h.UpdateProcessStatus)
		processGroup.GET("/:id/validate", h.ValidateProcess)
		processGroup.GET("/published", h.GetPublishedProcesses)
		processGroup.GET("/form-design/:formDesignID", h.GetProcessesByFormDesignID)
		processGroup.GET("/category/:categoryID", h.GetProcessesByCategoryID)
		processGroup.POST("/batch-status", h.BatchUpdateProcessStatus)
		processGroup.GET("/check-name", h.CheckProcessNameExists)
		processGroup.POST("/batch-get", h.GetProcessesByIDs)
	}
}

func (h *ProcessHandler) CreateProcess(ctx *gin.Context) {
	var req model.CreateProcessReq
	user := ctx.MustGet("user").(utils.UserClaims)
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateProcess(ctx, &req, user.Uid, user.Username)
	})
}

func (h *ProcessHandler) UpdateProcess(ctx *gin.Context) {
	var req model.UpdateProcessReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id

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

	// 从查询参数中获取分页信息
	if pageStr := ctx.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}
	if sizeStr := ctx.Query("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			req.Size = size
		}
	}

	// 从查询参数中获取其他过滤条件
	if name := ctx.Query("name"); name != "" {
		req.Name = &name
	}
	if categoryIDStr := ctx.Query("categoryID"); categoryIDStr != "" {
		if categoryID, err := strconv.Atoi(categoryIDStr); err == nil {
			req.CategoryID = &categoryID
		}
	}
	if statusStr := ctx.Query("status"); statusStr != "" {
		if status, err := strconv.Atoi(statusStr); err == nil {
			statusInt8 := int8(status)
			req.Status = &statusInt8
		}
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.ListProcess(ctx, &req)
	})
}

func (h *ProcessHandler) GetProcess(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.DetailProcess(ctx, id, user.Uid)
	})
}

func (h *ProcessHandler) DetailProcess(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.DetailProcess(ctx, id, user.Uid)
	})
}

func (h *ProcessHandler) GetProcessWithRelations(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetProcessWithRelations(ctx, id)
	})
}

func (h *ProcessHandler) PublishProcess(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req := &model.PublishProcessReq{ID: id}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.PublishProcess(ctx, req)
	})
}

func (h *ProcessHandler) CloneProcess(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	var req model.CloneProcessReq
	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.CloneProcess(ctx, &req, user.Uid)
	})
}

func (h *ProcessHandler) UpdateProcessStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	var req struct {
		Status int8 `json:"status" binding:"required"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateProcessStatus(ctx, id, req.Status)
	})
}

func (h *ProcessHandler) ValidateProcess(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.ValidateProcess(ctx, id, user.Uid)
	})
}

func (h *ProcessHandler) GetPublishedProcesses(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetPublishedProcesses(ctx)
	})
}

func (h *ProcessHandler) GetProcessesByFormDesignID(ctx *gin.Context) {
	formDesignIDStr := ctx.Param("formDesignID")
	formDesignID, err := strconv.Atoi(formDesignIDStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的表单设计ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetProcessesByFormDesignID(ctx, formDesignID)
	})
}

func (h *ProcessHandler) GetProcessesByCategoryID(ctx *gin.Context) {
	categoryIDStr := ctx.Param("categoryID")
	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的分类ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetProcessesByCategoryID(ctx, categoryID)
	})
}

func (h *ProcessHandler) BatchUpdateProcessStatus(ctx *gin.Context) {
	var req struct {
		IDs    []int `json:"ids" binding:"required"`
		Status int8  `json:"status" binding:"required"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BatchUpdateProcessStatus(ctx, req.IDs, req.Status)
	})
}

func (h *ProcessHandler) CheckProcessNameExists(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		utils.ErrorWithMessage(ctx, "流程名称不能为空")
		return
	}

	var excludeID []int
	if excludeIDStr := ctx.Query("excludeID"); excludeIDStr != "" {
		if id, err := strconv.Atoi(excludeIDStr); err == nil {
			excludeID = []int{id}
		}
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		exists, err := h.service.CheckProcessNameExists(ctx, name, excludeID...)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"exists": exists}, nil
	})
}

func (h *ProcessHandler) GetProcessesByIDs(ctx *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetProcessesByIDs(ctx, req.IDs)
	})
}
