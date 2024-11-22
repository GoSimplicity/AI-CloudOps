package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
)

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

type RdsHandler struct {
	service service.RdsService
}

func NewRdsHandler(service service.RdsService) *RdsHandler {
	return &RdsHandler{
		service: service,
	}
}

func (r *RdsHandler) RegisterRouters(server *gin.Engine) {
	rdsGroup := server.Group("/api/tree/rds")
	rdsGroup.GET("/getRdsUnbindList", r.GetRdsUnbindList)
	rdsGroup.GET("/getRdsList", r.GetRdsList)
	rdsGroup.POST("/bindRds", r.BindRds)
	rdsGroup.POST("/unBindRds", r.UnBindRds)
}

func (r *RdsHandler) GetRdsUnbindList(ctx *gin.Context) {
	rds, err := r.service.GetRdsUnbindList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取未绑定的RDS实例列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, rds)
}

func (r *RdsHandler) GetRdsList(ctx *gin.Context) {
	rds, err := r.service.GetRdsList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取RDS实例列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, rds)
}

func (r *RdsHandler) BindRds(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if len(req.ResourceIds) == 0 {
		apiresponse.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供要绑定的RDS实例ID")
		return
	}

	if req.NodeId == 0 {
		apiresponse.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要绑定到的节点ID")
		return
	}

	if err := r.service.BindRds(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "绑定RDS实例失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (r *RdsHandler) UnBindRds(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if len(req.ResourceIds) == 0 {
		apiresponse.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供要解绑的RDS实例ID")
		return
	}

	if req.NodeId == 0 {
		apiresponse.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要解绑的节点ID")
		return
	}

	if err := r.service.UnBindRds(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "解绑RDS实例失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}
