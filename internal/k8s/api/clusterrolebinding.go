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

type K8sClusterRoleBindingHandler struct {
	clusterRoleBindingService service.ClusterRoleBindingService
}

func NewK8sClusterRoleBindingHandler(clusterRoleBindingService service.ClusterRoleBindingService) *K8sClusterRoleBindingHandler {
	return &K8sClusterRoleBindingHandler{
		clusterRoleBindingService: clusterRoleBindingService,
	}
}

func (h *K8sClusterRoleBindingHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/clusterrolebinding/:cluster_id/list", h.GetClusterRoleBindingList)
		k8sGroup.GET("/clusterrolebinding/:cluster_id/:name/detail", h.GetClusterRoleBindingDetails)
		k8sGroup.GET("/clusterrolebinding/:cluster_id/:name/detail/yaml", h.GetClusterRoleBindingYaml)
		k8sGroup.POST("/clusterrolebinding/:cluster_id/create", h.CreateClusterRoleBinding)
		k8sGroup.POST("/clusterrolebinding/:cluster_id/create/yaml", h.CreateClusterRoleBindingByYaml)
		k8sGroup.PUT("/clusterrolebinding/:cluster_id/:name/update", h.UpdateClusterRoleBinding)
		k8sGroup.PUT("/clusterrolebinding/:cluster_id/:name/update/yaml", h.UpdateClusterRoleBindingYaml)
		k8sGroup.DELETE("/clusterrolebinding/:cluster_id/:name/delete", h.DeleteClusterRoleBinding)
	}
}

func (h *K8sClusterRoleBindingHandler) GetClusterRoleBindingList(ctx *gin.Context) {
	var req model.GetClusterRoleBindingListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.clusterRoleBindingService.GetClusterRoleBindingList(ctx, &req)
	})
}

func (h *K8sClusterRoleBindingHandler) GetClusterRoleBindingDetails(ctx *gin.Context) {
	var req model.GetClusterRoleBindingDetailsReq

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
		return h.clusterRoleBindingService.GetClusterRoleBindingDetails(ctx, &req)
	})
}

func (h *K8sClusterRoleBindingHandler) CreateClusterRoleBinding(ctx *gin.Context) {
	var req model.CreateClusterRoleBindingReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterRoleBindingService.CreateClusterRoleBinding(ctx, &req)
	})
}

// CreateClusterRoleBindingByYaml 通过YAML创建ClusterRoleBinding
func (h *K8sClusterRoleBindingHandler) CreateClusterRoleBindingByYaml(ctx *gin.Context) {
	var req model.CreateClusterRoleBindingByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterRoleBindingService.CreateClusterRoleBindingByYaml(ctx, &req)
	})
}

func (h *K8sClusterRoleBindingHandler) UpdateClusterRoleBinding(ctx *gin.Context) {
	var req model.UpdateClusterRoleBindingReq

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
		return nil, h.clusterRoleBindingService.UpdateClusterRoleBinding(ctx, &req)
	})
}

func (h *K8sClusterRoleBindingHandler) DeleteClusterRoleBinding(ctx *gin.Context) {
	var req model.DeleteClusterRoleBindingReq

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
		return nil, h.clusterRoleBindingService.DeleteClusterRoleBinding(ctx, &req)
	})
}

func (h *K8sClusterRoleBindingHandler) GetClusterRoleBindingYaml(ctx *gin.Context) {
	var req model.GetClusterRoleBindingYamlReq

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
		return h.clusterRoleBindingService.GetClusterRoleBindingYaml(ctx, &req)
	})
}

func (h *K8sClusterRoleBindingHandler) UpdateClusterRoleBindingYaml(ctx *gin.Context) {
	var req model.UpdateClusterRoleBindingByYamlReq

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
		return nil, h.clusterRoleBindingService.UpdateClusterRoleBindingYaml(ctx, &req)
	})
}
