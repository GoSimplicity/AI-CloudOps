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

type InstanceHandler struct {
	service service.InstanceService
}

func NewInstanceHandler(service service.InstanceService) *InstanceHandler {
	return &InstanceHandler{
		service: service,
	}
}

func (h *InstanceHandler) RegisterRouters(server *gin.Engine) {
	instanceGroup := server.Group("/api/workorder/instance")
	{
		instanceGroup.POST("/create", h.CreateInstance)
		instanceGroup.PUT("/update/:id", h.UpdateInstance)
		instanceGroup.DELETE("/delete/:id", h.DeleteInstance)
		instanceGroup.GET("/list", h.ListInstance)
		instanceGroup.GET("/detail/:id", h.DetailInstance)
		instanceGroup.POST("/transfer/:id", h.TransferInstance)
		instanceGroup.GET("/my", h.GetMyInstances)
	}
}

// CreateInstance 创建工单实例
func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {
	var req model.CreateInstanceReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.CreateInstance(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateInstance 更新工单实例
func (h *InstanceHandler) UpdateInstance(ctx *gin.Context) {
	var req model.UpdateInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.UpdateInstance(ctx, &req, user.Uid)
	})
}

// DeleteInstance 删除工单实例
func (h *InstanceHandler) DeleteInstance(ctx *gin.Context) {
	var req model.DeleteInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.DeleteInstance(ctx, req.ID, user.Uid)
	})
}

// DetailInstance 获取工单实例详情
func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {
	var req model.DetailInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.GetInstance(ctx, req.ID)
	})
}

// ListInstance 获取工单实例列表
func (h *InstanceHandler) ListInstance(ctx *gin.Context) {
	var req model.ListInstanceReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListInstance(ctx, &req)
	})
}

// GetMyInstances 获取我的工单
func (h *InstanceHandler) GetMyInstances(ctx *gin.Context) {
	var req model.MyInstanceReq

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.GetMyInstances(ctx, &req, user.Uid)
	})
}

// TransferInstance 转移工单
func (h *InstanceHandler) TransferInstance(ctx *gin.Context) {
	var req model.TransferInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.InstanceID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.TransferInstance(ctx, req.InstanceID, user.Uid, req.AssigneeID, req.Comment)
	})
}
