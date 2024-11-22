package api

import (
	"strconv"

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

type EcsResourceHandler struct {
	service service.EcsResourceService
}

func NewEcsResourceHandler(service service.EcsResourceService) *EcsResourceHandler {
	return &EcsResourceHandler{
		service: service,
	}
}

func (r *EcsResourceHandler) RegisterRouters(server *gin.Engine) {
	ecsResourceGroup := server.Group("/api/tree/ecs/resource")
	ecsResourceGroup.POST("/createEcsResource", r.CreateEcsResource)
	ecsResourceGroup.POST("/updateEcsResource", r.UpdateEcsResource)
	ecsResourceGroup.DELETE("/deleteEcsResource/:id", r.DeleteEcsResource)
	ecsResourceGroup.GET("/getAllResourceByType", r.GetAllResourceByType)
}

func (r *EcsResourceHandler) GetAllResourceByType(ctx *gin.Context) {
	resourceType := ctx.Query("type")
	if resourceType == "" || (resourceType != "ecs" && resourceType != "elb" && resourceType != "rds") {
		apiresponse.BadRequestWithDetails(ctx, "资源类型错误", "资源类型必须为ecs、elb或rds之一")
		return
	}

	nid := ctx.Query("nid")
	if nid == "" {
		apiresponse.BadRequestWithDetails(ctx, "节点ID为空", "请提供有效的节点ID")
		return
	}
	nodeId, err := strconv.Atoi(nid)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, "节点ID格式错误", "节点ID必须为有效的整数")
		return
	}

	p := ctx.DefaultQuery("page", "1")
	s := ctx.DefaultQuery("size", "10")
	page, err := strconv.Atoi(p)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, "页码格式错误", "页码必须为有效的整数")
		return
	}
	size, err := strconv.Atoi(s)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, "分页大小格式错误", "分页大小必须为有效的整数")
		return
	}

	resource, err := r.service.GetAllResourcesByType(ctx, nodeId, resourceType, page, size)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取资源列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, resource)
}

func (r *EcsResourceHandler) CreateEcsResource(ctx *gin.Context) {
	var req model.ResourceEcs

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if err := r.service.CreateEcsResource(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "创建ECS资源失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (r *EcsResourceHandler) UpdateEcsResource(ctx *gin.Context) {
	var req model.ResourceEcs

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if err := r.service.UpdateEcsResource(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "更新ECS资源失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (r *EcsResourceHandler) DeleteEcsResource(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, "资源ID格式错误", "资源ID必须为有效的整数")
		return
	}

	if err := r.service.DeleteEcsResource(ctx, idInt); err != nil {
		apiresponse.ErrorWithMessage(ctx, "删除ECS资源失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}
