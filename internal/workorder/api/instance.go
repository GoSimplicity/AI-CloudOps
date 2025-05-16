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
		// 工单实例基本操作
		instanceGroup.POST("/create", h.CreateInstance)       // 创建工单实例
		instanceGroup.POST("/update", h.UpdateInstance)       // 更新工单实例
		instanceGroup.DELETE("/delete/:id", h.DeleteInstance) // 删除工单实例

		// 工单流程操作
		instanceGroup.POST("/approve", h.ApproveInstance)   // 审批工单
		instanceGroup.POST("/action", h.ActionInstance)     // 处理工单（完成/拒绝/取消）
		instanceGroup.POST("/comment", h.CommentInstance)   // 添加评论
		instanceGroup.POST("/transfer", h.TransferInstance) // 转交工单

		// 查询操作
		instanceGroup.POST("/list", h.ListInstance)             // 获取工单列表
		instanceGroup.POST("/detail", h.DetailInstance)         // 获取工单详情
		instanceGroup.POST("/my", h.MyInstance)                 // 获取我的工单
		instanceGroup.POST("/statistics", h.InstanceStatistics) // 获取工单统计
	}
}

func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {
	var req model.CreateInstanceReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateInstance(ctx, req, user.Uid, user.Username)
	})
}

func (h *InstanceHandler) UpdateInstance(ctx *gin.Context) {
	var req model.UpdateInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateInstance(ctx, req)
	})
}

func (h *InstanceHandler) ApproveInstance(ctx *gin.Context) {
	var req model.InstanceFlowReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.Action = "approve" // 确保操作类型为审批
		return nil, h.service.ProcessInstanceFlow(ctx, req, user.Uid, user.Username)
	})
}

func (h *InstanceHandler) ActionInstance(ctx *gin.Context) {
	var req model.InstanceFlowReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.ProcessInstanceFlow(ctx, req, user.Uid, user.Username)
	})
}

func (h *InstanceHandler) TransferInstance(ctx *gin.Context) {
	var req model.InstanceFlowReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.Action = "transfer" // 确保操作类型为转交
		return nil, h.service.ProcessInstanceFlow(ctx, req, user.Uid, user.Username)
	})
}

func (h *InstanceHandler) CommentInstance(ctx *gin.Context) {
	var req model.InstanceCommentReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CommentInstance(ctx, req, user.Uid, user.Username)
	})
}

func (h *InstanceHandler) ListInstance(ctx *gin.Context) {
	var req model.ListInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListInstance(ctx, req)
	})
}

func (h *InstanceHandler) MyInstance(ctx *gin.Context) {
	var req model.ListInstanceReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 根据当前用户角色决定查询条件
		// 如果是处理人，查询分配给我的工单
		req.AssigneeID = user.Uid
		return h.service.ListInstance(ctx, req)
	})
}

func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {
	var req model.DetailInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailInstance(ctx, req.ID)
	})
}

func (h *InstanceHandler) DeleteInstance(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteInstance(ctx, id)
	})
}

func (h *InstanceHandler) InstanceStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetInstanceStatistics(ctx)
	})
}
