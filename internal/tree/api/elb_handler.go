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

type ElbHandler struct {
	service service.ElbService
}

func NewElbHandler(service service.ElbService) *ElbHandler {
	return &ElbHandler{
		service: service,
	}
}

func (e *ElbHandler) RegisterRouters(server *gin.Engine) {
	elbGroup := server.Group("/api/tree/elb")

	// ELB相关路由
	elbGroup.GET("/getElbUnbindList", e.GetElbUnbindList) // 获取未绑定的ELB实例列表
	elbGroup.GET("/getElbList", e.GetElbList)             // 获取ELB实例列表
	elbGroup.POST("/bindElb", e.BindElb)                  // 绑定ELB实例
	elbGroup.POST("/unBindElb", e.UnBindElb)              // 解绑ELB实例
}

func (e *ElbHandler) GetElbUnbindList(ctx *gin.Context) {
	elb, err := e.service.GetElbUnbindList(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "获取未绑定的ELB实例列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(ctx, elb)
}

func (e *ElbHandler) GetElbList(ctx *gin.Context) {
	elb, err := e.service.GetElbList(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "获取ELB实例列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(ctx, elb)
}

func (e *ElbHandler) BindElb(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if len(req.ResourceIds) == 0 {
		utils.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供要绑定的ELB实例ID")
		return
	}

	if req.NodeId == 0 {
		utils.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要绑定到的节点ID")
		return
	}

	if err := e.service.BindElb(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		utils.ErrorWithMessage(ctx, "绑定ELB实例失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}

func (e *ElbHandler) UnBindElb(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if len(req.ResourceIds) == 0 {
		utils.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供要解绑的ELB实例ID")
		return
	}

	if req.NodeId == 0 {
		utils.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要解绑的节点ID")
		return
	}

	if err := e.service.UnBindElb(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		utils.ErrorWithMessage(ctx, "解绑ELB实例失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}
