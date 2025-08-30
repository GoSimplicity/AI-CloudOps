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
		k8sGroup.GET("/namespaces/:cluster_id/list", k.ListNamespaces)               // 获取命名空间列表
		k8sGroup.POST("/namespaces/:cluster_id/create", k.CreateNamespace)           // 创建新的命名空间
		k8sGroup.DELETE("/namespaces/:cluster_id/:name/delete", k.DeleteNamespace)   // 删除指定的命名空间
		k8sGroup.GET("/namespaces/:cluster_id/:name/details", k.GetNamespaceDetails) // 获取指定命名空间的详情
		k8sGroup.PUT("/namespaces/:cluster_id/:name/update", k.UpdateNamespace)      // 更新指定命名空间
	}
}

// CreateNamespace 创建新的命名空间
func (k *K8sNamespaceHandler) CreateNamespace(ctx *gin.Context) {
	var req model.K8sNamespaceCreateReq

	clusterID, err := utils.GetParamCustomName(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterIDInt

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.CreateNamespace(ctx, &req)
	})
}

// DeleteNamespace 删除指定的命名空间
func (k *K8sNamespaceHandler) DeleteNamespace(ctx *gin.Context) {
	var req model.K8sNamespaceDeleteReq

	clusterID, err := utils.GetParamCustomName(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterIDInt
	req.Name = name

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.namespaceService.DeleteNamespace(ctx, &req)
	})
}

// GetNamespaceDetails 获取指定命名空间的详情
func (k *K8sNamespaceHandler) GetNamespaceDetails(ctx *gin.Context) {
	var req model.K8sNamespaceGetDetailsReq

	clusterID, err := utils.GetParamCustomName(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterIDInt
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.namespaceService.GetNamespaceDetails(ctx, &req)
	})
}

// UpdateNamespace 更新指定命名空间
func (k *K8sNamespaceHandler) UpdateNamespace(ctx *gin.Context) {
	var req model.K8sNamespaceUpdateReq

	clusterID, err := utils.GetParamCustomName(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterIDInt
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.UpdateNamespace(ctx, &req)
	})
}

// ListNamespaces 获取命名空间列表
func (k *K8sNamespaceHandler) ListNamespaces(ctx *gin.Context) {
	var req model.K8sNamespaceListReq

	clusterID, err := utils.GetParamCustomName(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterIDInt, err := strconv.Atoi(clusterID)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterIDInt

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.namespaceService.ListNamespaces(ctx, &req)
	})
}
