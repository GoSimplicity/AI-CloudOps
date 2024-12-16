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
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type EcsHandler struct {
	service service.EcsService
	l       *zap.Logger
}

func NewEcsHandler(service service.EcsService, logger *zap.Logger) *EcsHandler {
	return &EcsHandler{
		service: service,
		l:       logger,
	}
}

func (e *EcsHandler) RegisterRouters(server *gin.Engine) {
	ecsGroup := server.Group("/api/tree/ecs")
	ecsGroup.GET("/getEcsUnbindList", e.GetEcsUnbindList)
	ecsGroup.GET("/getEcsList", e.GetEcsList)
	ecsGroup.POST("/bindEcs", e.BindEcs)
	ecsGroup.POST("/unBindEcs", e.UnBindEcs)
}

func (e *EcsHandler) GetEcsUnbindList(ctx *gin.Context) {
	ecs, err := e.service.GetEcsUnbindList(ctx)
	if err != nil {
		e.l.Error("get unbind ecs failed", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "获取未绑定的ECS实例列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, ecs)
}

func (e *EcsHandler) GetEcsList(ctx *gin.Context) {
	ecs, err := e.service.GetEcsList(ctx)
	if err != nil {
		e.l.Error("get ecs list failed", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "获取ECS实例列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, ecs)
}

func (e *EcsHandler) BindEcs(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if len(req.ResourceIds) == 0 {
		apiresponse.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供要绑定的ECS实例ID")
		return
	}

	if req.NodeId == 0 {
		apiresponse.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要绑定到的节点ID")
		return
	}

	if err := e.service.BindEcs(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		e.l.Error("bind ecs failed", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "绑定ECS实例失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (e *EcsHandler) UnBindEcs(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if len(req.ResourceIds) == 0 {
		apiresponse.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供要解绑的ECS实例ID")
		return
	}

	if req.NodeId == 0 {
		apiresponse.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要解绑的节点ID")
		return
	}

	if err := e.service.UnBindEcs(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		e.l.Error("unbind ecs failed", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, "解绑ECS实例失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}
