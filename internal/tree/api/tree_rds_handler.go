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
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type TreeRdsHandler struct {
	rdsService service.TreeRdsService
}

func NewTreeRdsHandler(rdsService service.TreeRdsService) *TreeRdsHandler {
	return &TreeRdsHandler{
		rdsService: rdsService,
	}
}

func (h *TreeRdsHandler) RegisterRouters(server *gin.Engine) {
	rdsGroup := server.Group("/api/tree/rds")
	{
		rdsGroup.POST("/list", h.ListRdsResources)
		rdsGroup.POST("/detail/:id", h.GetRdsDetail)
		rdsGroup.POST("/create", h.CreateRdsResource)
		rdsGroup.POST("/update/:id", h.UpdateRds)
		rdsGroup.POST("/start/:id", h.StartRds)
		rdsGroup.POST("/stop/:id", h.StopRds)
		rdsGroup.POST("/restart/:id", h.RestartRds)
		rdsGroup.DELETE("/delete/:id", h.DeleteRds)
		rdsGroup.POST("/resize/:id", h.ResizeRds)
		rdsGroup.POST("/backup/:id", h.BackupRds)
		rdsGroup.POST("/restore/:id", h.RestoreRds)
		rdsGroup.POST("/reset_password/:id", h.ResetRdsPassword)
		rdsGroup.POST("/renew/:id", h.RenewRds)
	}
}

// ListRdsResources 获取RDS实例列表
func (h *TreeRdsHandler) ListRdsResources(ctx *gin.Context) {
	var req model.ListRdsResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.rdsService.ListRdsResources(ctx, &req)
	})
}

// GetRdsDetail 获取RDS实例详情
func (h *TreeRdsHandler) GetRdsDetail(ctx *gin.Context) {
	var req model.GetRdsDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.rdsService.GetRdsDetail(ctx, &req)
	})
}

// CreateRdsResource 创建RDS实例
func (h *TreeRdsHandler) CreateRdsResource(ctx *gin.Context) {
	var req model.CreateRdsResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.CreateRdsResource(ctx, &req)
	})
}

// StartRds 启动RDS实例
func (h *TreeRdsHandler) StartRds(ctx *gin.Context) {
	var req model.StartRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.StartRds(ctx, &req)
	})
}

// StopRds 停止RDS实例
func (h *TreeRdsHandler) StopRds(ctx *gin.Context) {
	var req model.StopRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.StopRds(ctx, &req)
	})
}

// RestartRds 重启RDS实例
func (h *TreeRdsHandler) RestartRds(ctx *gin.Context) {
	var req model.RestartRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.RestartRds(ctx, &req)
	})
}

// DeleteRds 删除RDS实例
func (h *TreeRdsHandler) DeleteRds(ctx *gin.Context) {
	var req model.DeleteRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.DeleteRds(ctx, &req)
	})
}

// UpdateRds 更新RDS实例
func (h *TreeRdsHandler) UpdateRds(ctx *gin.Context) {
	var req model.UpdateRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.UpdateRds(ctx, &req)
	})
}

// ResizeRds 调整RDS实例规格
func (h *TreeRdsHandler) ResizeRds(ctx *gin.Context) {
	var req model.ResizeRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.ResizeRds(ctx, &req)
	})
}

// BackupRds 备份RDS实例
func (h *TreeRdsHandler) BackupRds(ctx *gin.Context) {
	var req model.BackupRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.BackupRds(ctx, &req)
	})
}

// RestoreRds 恢复RDS实例
func (h *TreeRdsHandler) RestoreRds(ctx *gin.Context) {
	var req model.RestoreRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.RestoreRds(ctx, &req)
	})
}

// ResetRdsPassword 重置RDS实例密码
func (h *TreeRdsHandler) ResetRdsPassword(ctx *gin.Context) {
	var req model.ResetRdsPasswordReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.ResetRdsPassword(ctx, &req)
	})
}

// RenewRds 续费RDS实例
func (h *TreeRdsHandler) RenewRds(ctx *gin.Context) {
	var req model.RenewRdsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.RenewRds(ctx, &req)
	})
}
