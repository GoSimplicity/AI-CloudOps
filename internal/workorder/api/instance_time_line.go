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

type InstanceTimeLineHandler struct {
	service service.WorkorderInstanceTimeLineService
}

func NewInstanceTimeLineHandler(service service.WorkorderInstanceTimeLineService) *InstanceTimeLineHandler {
	return &InstanceTimeLineHandler{
		service: service,
	}
}

func (h *InstanceTimeLineHandler) RegisterRouters(server *gin.Engine) {
	instanceGroup := server.Group("/api/workorder/instance")
	{
		instanceGroup.POST("/create", h.CreateInstanceTimeLine)
		instanceGroup.PUT("/update/:id", h.UpdateInstanceTimeLine)
		instanceGroup.DELETE("/delete/:id", h.DeleteInstanceTimeLine)
		instanceGroup.GET("/list", h.ListInstanceTimeLine)
		instanceGroup.GET("/detail/:id", h.DetailInstanceTimeLine)
	}
}

// CreateInstance 创建工单实例
func (h *InstanceTimeLineHandler) CreateInstanceTimeLine(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceTimelineReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.CreateInstanceTimeLine(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateInstance 更新工单实例
func (h *InstanceTimeLineHandler) UpdateInstanceTimeLine(ctx *gin.Context) {
	var req model.UpdateWorkorderInstanceTimelineReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.UpdateInstanceTimeLine(ctx, &req, user.Uid)
	})
}

// DeleteInstance 删除工单实例
func (h *InstanceTimeLineHandler) DeleteInstanceTimeLine(ctx *gin.Context) {
	var req model.DeleteWorkorderInstanceTimelineReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.DeleteInstanceTimeLine(ctx, req.ID, user.Uid)
	})
}

// DetailInstance 获取工单实例详情
func (h *InstanceTimeLineHandler) DetailInstanceTimeLine(ctx *gin.Context) {
	var req model.DetailWorkorderInstanceTimelineReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.GetInstanceTimeLine(ctx, req.ID)
	})
}

// ListInstance 获取工单实例列表
func (h *InstanceTimeLineHandler) ListInstanceTimeLine(ctx *gin.Context) {
	var req model.ListWorkorderInstanceTimelineReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListInstanceTimeLine(ctx, &req)
	})
}
