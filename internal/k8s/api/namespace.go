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
)

type K8sNamespaceHandler struct {
	namespaceService service.NamespaceService
}

func NewK8sNamespaceHandler(namespaceService service.NamespaceService) *K8sNamespaceHandler {
	return &K8sNamespaceHandler{
		namespaceService: namespaceService,
	}
}

func (k *K8sNamespaceHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/namespaces/:cluster_id/list", k.ListNamespaces)
		k8sGroup.POST("/namespaces/:cluster_id/create", k.CreateNamespace)
		k8sGroup.DELETE("/namespaces/:cluster_id/:name/delete", k.DeleteNamespace)
		k8sGroup.GET("/namespaces/:cluster_id/:name/details", k.GetNamespaceDetails)
		k8sGroup.PUT("/namespaces/:cluster_id/:name/update", k.UpdateNamespace)
	}
}

func (k *K8sNamespaceHandler) CreateNamespace(ctx *gin.Context) {
	var req model.K8sNamespaceCreateReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.CreateNamespace(ctx, &req)
	})
}

func (k *K8sNamespaceHandler) DeleteNamespace(ctx *gin.Context) {
	var req model.K8sNamespaceDeleteReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Name = name

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.namespaceService.DeleteNamespace(ctx, &req)
	})
}

func (k *K8sNamespaceHandler) GetNamespaceDetails(ctx *gin.Context) {
	var req model.K8sNamespaceGetDetailsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.namespaceService.GetNamespaceDetails(ctx, &req)
	})
}

func (k *K8sNamespaceHandler) UpdateNamespace(ctx *gin.Context) {
	var req model.K8sNamespaceUpdateReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.namespaceService.UpdateNamespace(ctx, &req)
	})
}

func (k *K8sNamespaceHandler) ListNamespaces(ctx *gin.Context) {
	var req model.K8sNamespaceListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.namespaceService.ListNamespaces(ctx, &req)
	})
}
