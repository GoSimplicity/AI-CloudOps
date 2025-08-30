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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sNamespaceHandler struct {
	logger           *zap.Logger
	namespaceService service.NamespaceService
}

func NewK8sNamespaceHandler(logger *zap.Logger, namespaceService service.NamespaceService) *K8sNamespaceHandler {
	return &K8sNamespaceHandler{
		logger:           logger,
		namespaceService: namespaceService,
	}
}

func (k *K8sNamespaceHandler) RegisterRouters(router *gin.Engine) {
	k8sGroup := router.Group("/api/k8s")
	{
		k8sGroup.GET("/namespaces/list", k.GetClusterNamespacesForCascade)      // 获取级联选择的命名空间列表
		k8sGroup.GET("/namespaces/select/:id", k.GetClusterNamespacesForSelect) // 获取用于选择的命名空间列表
		k8sGroup.POST("/namespaces/create", k.CreateNamespace)                  // 创建新的命名空间
		k8sGroup.DELETE("/namespaces/delete/:id", k.DeleteNamespace)            // 删除指定的命名空间
		k8sGroup.GET("/namespaces/:id", k.GetNamespaceDetails)                  // 获取指定命名空间的详情
		k8sGroup.POST("/namespaces/update", k.UpdateNamespace)                  // 更新指定命名空间
		k8sGroup.GET("/namespaces/:id/resources", k.GetNamespaceResources)      // 获取命名空间中的资源
		k8sGroup.GET("/namespaces/:id/events", k.GetNamespaceEvents)            // 获取命名空间事件
	}
}

// GetClusterNamespacesForCascade 获取级联选择的命名空间列表
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
func (k *K8sNamespaceHandler) CreateNamespace(ctx *gin.Context) {
	var req model.CreateNamespaceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.CreateNamespace(ctx, req)
	})
}

// DeleteNamespace 删除指定的命名空间
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
func (k *K8sNamespaceHandler) UpdateNamespace(ctx *gin.Context) {
	var req model.UpdateNamespaceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.UpdateNamespace(ctx, req)
	})
}

// GetNamespaceResources 获取指定命名空间中的资源
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
