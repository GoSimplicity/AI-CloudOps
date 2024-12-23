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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AliResourceHandler struct {
	service service.AliResourceService
}

func NewAliResourceHandler(service service.AliResourceService) *AliResourceHandler {
	return &AliResourceHandler{
		service: service,
	}
}

func (a *AliResourceHandler) RegisterRouters(server *gin.Engine) {
	aliResourceGroup := server.Group("/api/tree/ecs/ali/resource")
	aliResourceGroup.POST("/createAliResource", a.CreateAliEcsResource)
	aliResourceGroup.POST("/updateAliResource", a.UpdateAliEcsResource)
	aliResourceGroup.DELETE("/deleteAliResource/:id", a.DeleteAliEcsResource)
	aliResourceGroup.GET("/getResourceStatus/:id", a.GetResourceStatus)
}

func (a *AliResourceHandler) CreateAliEcsResource(ctx *gin.Context) {
	var req model.TerraformConfig

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	id, err := a.service.CreateResource(ctx, req)
	if err != nil {
		utils.ErrorWithMessage(ctx, "创建阿里云ECS资源失败: "+err.Error())
		return
	}

	utils.SuccessWithData(ctx, id)
}

func (a *AliResourceHandler) UpdateAliEcsResource(ctx *gin.Context) {
	var req model.TerraformConfig

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入")
		return
	}

	if req.ID == 0 {
		utils.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供有效的资源ID")
		return
	}

	if err := a.service.UpdateResource(ctx, req.ID, req); err != nil {
		utils.ErrorWithMessage(ctx, "更新阿里云ECS资源失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}

func (a *AliResourceHandler) DeleteAliEcsResource(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "资源ID必须为有效的整数")
		return
	}

	if err := a.service.DeleteResource(ctx, idInt); err != nil {
		utils.ErrorWithMessage(ctx, "删除阿里云ECS资源失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}

func (a *AliResourceHandler) GetResourceStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		utils.BadRequestWithDetails(ctx, "资源ID不能为空", "请提供有效的资源ID")
		return
	}

	task, err := a.service.GetTaskStatus(ctx, id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "获取资源状态失败: "+err.Error())
		return
	}

	utils.SuccessWithData(ctx, task)
}
