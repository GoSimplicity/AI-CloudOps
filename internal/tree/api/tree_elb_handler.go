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

type TreeElbHandler struct {
	elbService service.TreeElbService
}

func NewTreeElbHandler(elbService service.TreeElbService) *TreeElbHandler {
	return &TreeElbHandler{
		elbService: elbService,
	}
}

func (h *TreeElbHandler) RegisterRouters(server *gin.Engine) {
	elbGroup := server.Group("/api/tree/elb")
	{
		elbGroup.POST("/list", h.ListElbResources)
		elbGroup.POST("/detail/:id", h.GetElbDetail)
		elbGroup.POST("/create", h.CreateElbResource)
		elbGroup.POST("/update/:id", h.UpdateElb)
		elbGroup.DELETE("/delete/:id", h.DeleteElb)
		elbGroup.POST("/start/:id", h.StartElb)
		elbGroup.POST("/stop/:id", h.StopElb)
		elbGroup.POST("/restart/:id", h.RestartElb)
		elbGroup.POST("/resize/:id", h.ResizeElb)
		elbGroup.POST("/bind_servers", h.BindServersToElb)
		elbGroup.POST("/unbind_servers", h.UnbindServersFromElb)
		elbGroup.POST("/health_check/:id", h.ConfigureHealthCheck)
	}
}

// ListElbResources 获取ELB实例列表
func (h *TreeElbHandler) ListElbResources(ctx *gin.Context) {
	var req model.ListElbResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.elbService.ListElbResources(ctx, &req)
	})
}

// GetElbDetail 获取ELB实例详情
func (h *TreeElbHandler) GetElbDetail(ctx *gin.Context) {
	var req model.GetElbDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.elbService.GetElbDetail(ctx, &req)
	})
}

// CreateElbResource 创建ELB实例
func (h *TreeElbHandler) CreateElbResource(ctx *gin.Context) {
	var req model.CreateElbResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.CreateElbResource(ctx, &req)
	})
}

// DeleteElb 删除ELB实例
func (h *TreeElbHandler) DeleteElb(ctx *gin.Context) {
	var req model.DeleteElbReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.DeleteElb(ctx, &req)
	})
}

// UpdateElb 更新ELB实例
func (h *TreeElbHandler) UpdateElb(ctx *gin.Context) {
	var req model.UpdateElbReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.UpdateElb(ctx, &req)
	})
}

// StartElb 启动ELB实例
func (h *TreeElbHandler) StartElb(ctx *gin.Context) {
	var req model.StartElbReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.StartElb(ctx, &req)
	})
}

// StopElb 停止ELB实例
func (h *TreeElbHandler) StopElb(ctx *gin.Context) {
	var req model.StopElbReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.StopElb(ctx, &req)
	})
}

// RestartElb 重启ELB实例
func (h *TreeElbHandler) RestartElb(ctx *gin.Context) {
	var req model.RestartElbReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.RestartElb(ctx, &req)
	})
}

// ResizeElb 调整ELB实例规格
func (h *TreeElbHandler) ResizeElb(ctx *gin.Context) {
	var req model.ResizeElbReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.ResizeElb(ctx, &req)
	})
}

// BindServersToElb 绑定服务器到ELB实例
func (h *TreeElbHandler) BindServersToElb(ctx *gin.Context) {
	var req model.BindServersToElbReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.BindServersToElb(ctx, &req)
	})
}

// UnbindServersFromElb 解绑服务器从ELB实例
func (h *TreeElbHandler) UnbindServersFromElb(ctx *gin.Context) {
	var req model.UnbindServersFromElbReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.UnbindServersFromElb(ctx, &req)
	})
}

// ConfigureHealthCheck 配置健康检查
func (h *TreeElbHandler) ConfigureHealthCheck(ctx *gin.Context) {
	var req model.ConfigureHealthCheckReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.ConfigureHealthCheck(ctx, &req)
	})
}
