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

type InstanceFlowHandler struct {
	flowService service.InstanceFlowService
}

func NewInstanceFlowHandler(flowService service.InstanceFlowService) *InstanceFlowHandler {
	return &InstanceFlowHandler{
		flowService: flowService,
	}
}

func (h *InstanceFlowHandler) RegisterRouters(server *gin.Engine) {
	flowGroup := server.Group("/api/workorder/instance/flow")
	{
		flowGroup.GET("/list", h.ListInstanceFlows)
		flowGroup.GET("/detail/:id", h.DetailInstanceFlow)
	}
}

// ListInstanceFlows 获取工单流转记录列表
// 获取工单流转记录列表
func (h *InstanceFlowHandler) ListInstanceFlows(ctx *gin.Context) {
	var req model.ListWorkorderInstanceFlowReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.flowService.ListInstanceFlows(ctx, &req)
	})
}

// DetailInstanceFlow 获取工单流转记录详情
// 获取工单流转记录详情
func (h *InstanceFlowHandler) DetailInstanceFlow(ctx *gin.Context) {
	var req model.DetailWorkorderInstanceFlowReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.flowService.DetailInstanceFlow(ctx, req.ID)
	})
}
