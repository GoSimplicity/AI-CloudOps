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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sNamespaceHandler struct {
	logger           *zap.Logger
	namespaceService admin.NamespaceService
}

func NewK8sNamespaceHandler(logger *zap.Logger, namespaceService admin.NamespaceService) *K8sNamespaceHandler {
	return &K8sNamespaceHandler{
		logger:           logger,
		namespaceService: namespaceService,
	}
}

func (k *K8sNamespaceHandler) RegisterRouters(router *gin.Engine) {
	k8sGroup := router.Group("/api/k8s")
	namespaces := k8sGroup.Group("/namespaces")
	{
		namespaces.GET("/list", k.GetClusterNamespacesForCascade)      // 获取级联选择的命名空间列表
		namespaces.GET("/select/:id", k.GetClusterNamespacesForSelect) // 获取用于选择的命名空间列表
		namespaces.POST("/create", k.CreateNamespace)                  // 创建新的命名空间
		namespaces.DELETE("/delete/:id", k.DeleteNamespace)            // 删除指定的命名空间
		namespaces.GET("/:id", k.GetNamespaceDetails)                  // 获取指定命名空间的详情
		namespaces.POST("/update", k.UpdateNamespace)                  // 更新指定命名空间
		namespaces.GET("/:id/resources", k.GetNamespaceResources)      // 获取命名空间中的资源
		namespaces.GET("/:id/events", k.GetNamespaceEvents)            // 获取命名空间事件
	}
}

// GetClusterNamespacesForCascade 获取级联选择的命名空间列表
// @Summary 获取级联选择的命名空间列表
// @Description 获取所有集群的命名空间列表，用于级联选择器展示
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]model.ClusterNamespaces} "获取成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/list [get]
// @Security BearerAuth
func (k *K8sNamespaceHandler) GetClusterNamespacesForCascade(ctx *gin.Context) {
	namespaces, err := k.namespaceService.GetClusterNamespacesList(ctx)
	if err != nil {
		k.logger.Error("Failed to get cascade namespaces", zap.Error(err))
		utils.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	utils.SuccessWithData(ctx, namespaces)
}

// GetClusterNamespacesForSelect 获取用于选择的命名空间列表
// @Summary 获取指定集群的命名空间列表
// @Description 根据集群ID获取该集群下的所有命名空间列表，用于选择器展示
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.Namespace} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/select/{id} [get]
// @Security BearerAuth
func (k *K8sNamespaceHandler) GetClusterNamespacesForSelect(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespaces, err := k.namespaceService.GetClusterNamespacesById(ctx, id)
	if err != nil {
		utils.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	utils.SuccessWithData(ctx, namespaces)
}

// CreateNamespace 创建新的命名空间
// @Summary 创建新的命名空间
// @Description 在指定的Kubernetes集群中创建新的命名空间
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param request body model.CreateNamespaceRequest true "创建命名空间请求"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/create [post]
// @Security BearerAuth
func (k *K8sNamespaceHandler) CreateNamespace(ctx *gin.Context) {
	var req model.CreateNamespaceRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.CreateNamespace(ctx, req)
	})
}

// DeleteNamespace 删除指定的命名空间
// @Summary 删除指定的命名空间
// @Description 从指定的Kubernetes集群中删除命名空间
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param name query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sNamespaceHandler) DeleteNamespace(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespaceName := ctx.Query("name")
	if namespaceName == "" {
		utils.BadRequestError(ctx, "命名空间名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.namespaceService.DeleteNamespace(ctx, namespaceName, id)
	})
}

// GetNamespaceDetails 获取指定命名空间的详情
// @Summary 获取命名空间详情
// @Description 获取指定集群中某个命名空间的详细信息
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param name query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/{id} [get]
// @Security BearerAuth
func (k *K8sNamespaceHandler) GetNamespaceDetails(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespaceName := ctx.Query("name")
	if namespaceName == "" {
		utils.BadRequestError(ctx, "命名空间名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.namespaceService.GetNamespaceDetails(ctx, namespaceName, id)
	})
}

// UpdateNamespace 更新指定命名空间
// @Summary 更新命名空间
// @Description 更新指定集群中某个命名空间的标签和注解
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param request body model.UpdateNamespaceRequest true "更新命名空间请求"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/update [post]
// @Security BearerAuth
func (k *K8sNamespaceHandler) UpdateNamespace(ctx *gin.Context) {
	var req model.UpdateNamespaceRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.UpdateNamespace(ctx, req)
	})
}

// GetNamespaceResources 获取指定命名空间中的资源
// @Summary 获取命名空间资源
// @Description 获取指定集群中某个命名空间下的所有资源列表
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param name query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.Resource} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/{id}/resources [get]
// @Security BearerAuth
func (k *K8sNamespaceHandler) GetNamespaceResources(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespaceName := ctx.Query("name")
	if namespaceName == "" {
		utils.BadRequestError(ctx, "命名空间名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.namespaceService.GetNamespaceResources(ctx, namespaceName, id)
	})
}

// GetNamespaceEvents 获取指定命名空间中的事件
// @Summary 获取命名空间事件
// @Description 获取指定集群中某个命名空间下的所有事件列表
// @Tags 命名空间管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param name query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.Event} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/namespaces/{id}/events [get]
// @Security BearerAuth
func (k *K8sNamespaceHandler) GetNamespaceEvents(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespaceName := ctx.Query("name")
	if namespaceName == "" {
		utils.BadRequestError(ctx, "命名空间名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.namespaceService.GetNamespaceEvents(ctx, namespaceName, id)
	})
}
